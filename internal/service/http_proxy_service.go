package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/suar-net/suar-be/internal/model"
)

const (
	maxResponseBodySize   = 10 * 1024 * 1024 // 10 MB
	defaultRequestTimeout = 30 * time.Second
	maxRequestTimeout     = 90 * time.Second
)

var allowedMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodPatch:   true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

type OutboundRequest struct {
	Method  string
	URL     *url.URL
	Headers http.Header
	Body    []byte
	Timeout time.Duration
}

var blockedHeaders = map[string]bool{
	"Authorization":       true,
	"Cookie":              true,
	"Proxy-Authorization": true,
	"X-Forwarded-For":     true,
}

// isPrivateIP checks if a given IP address is private.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}
	ip4 := ip.To4()
	if ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}

func newOutboundRequest(dto *model.DTORequest) (*OutboundRequest, error) {
	// HTTP Method Validation
	dto.Method = strings.ToUpper(dto.Method)
	if !allowedMethods[dto.Method] {
		return nil, fmt.Errorf("%w: invalid or unsupported HTTP method: %s", ErrInvalidInput, dto.Method)
	}

	// URL Validation
	if dto.URL == "" {
		return nil, fmt.Errorf("%w: URL cannot be empty", ErrInvalidInput)
	}
	parsedURL, err := url.Parse(dto.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse URL: %v", ErrInvalidInput, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: invalid URL scheme: %s. Only 'http' and 'https' are allowed", ErrInvalidInput, parsedURL.Scheme)
	}

	// SSRF Protection: Disallow requests to private/local IP addresses
	ips, err := net.LookupIP(parsedURL.Hostname())
	if err != nil {
		return nil, fmt.Errorf("%w: could not resolve hostname: %v", ErrInvalidInput, err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return nil, fmt.Errorf("%w: requests to private IP addresses are not allowed", ErrInvalidInput)
		}
	}

	// Timeout Validation
	var timeout time.Duration
	if dto.Timeout <= 0 {
		timeout = defaultRequestTimeout
	} else {
		timeout = time.Duration(dto.Timeout) * time.Millisecond
	}
	if timeout > maxRequestTimeout {
		return nil, fmt.Errorf("%w: timeout of %v exceeds the maximum allowed limit of %v", ErrInvalidInput, timeout, maxRequestTimeout)
	}

	headers := make(http.Header)
	for key, values := range dto.Headers {
		if !blockedHeaders[http.CanonicalHeaderKey(key)] {
			headers[key] = values
		}
	}

	// Create the outbound request with validated data
	request := &OutboundRequest{
		Method:  dto.Method,
		URL:     parsedURL,
		Headers: headers,
		Body:    dto.Body,
		Timeout: timeout,
	}

	return request, nil
}

type HTTPProxyService struct {
	httpClient *http.Client
}

func NewHTTPProxyService() *HTTPProxyService {
	// Create a custom transport with optimized settings
	transport := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &HTTPProxyService{
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

func (s *HTTPProxyService) ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error) {
	// Convert and validate the DTO to our internal request model.
	outboundRequest, err := newOutboundRequest(dto)
	if err != nil {
		return nil, err // Propagate the error
	}

	return s.Execute(ctx, outboundRequest)
}

func (s *HTTPProxyService) Execute(ctx context.Context, outboundRequest *OutboundRequest) (*model.DTOResponse, error) {
	startTime := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, outboundRequest.Timeout)
	defer cancel()

	var bodyReader io.Reader
	if len(outboundRequest.Body) > 0 {
		bodyReader = bytes.NewReader(outboundRequest.Body)
	}

	httpRequest, err := http.NewRequestWithContext(
		reqCtx,
		outboundRequest.Method,
		outboundRequest.URL.String(),
		bodyReader,
	)
	if err != nil {
		return &model.DTOResponse{
			Error:     fmt.Sprintf("failed to create http request: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: startTime,
		}, nil
	}
	httpRequest.Header = outboundRequest.Headers

	httpResponse, err := s.httpClient.Do(httpRequest)
	duration := time.Since(startTime)
	// We check for context timeout error specifically
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrRequestTimeout, err)
		}
		return nil, fmt.Errorf("failed to execute request to target server: %w", err)
	}

	return httpResponseToDTOResponse(httpResponse, duration, startTime)
}

func httpResponseToDTOResponse(resp *http.Response, duration time.Duration, timestamp time.Time) (*model.DTOResponse, error) {
	defer resp.Body.Close()

	headers := make(map[string][]string)
	for key, values := range resp.Header {
		headers[key] = values
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: maxResponseBodySize}
	bodyBytes, err := io.ReadAll(limitedReader)

	dtoResponse := &model.DTOResponse{
		StatusCode: resp.StatusCode,
		Duration:   duration,
		Timestamp:  timestamp,
		Headers:    headers,
	}

	if err != nil {
		dtoResponse.Error = fmt.Sprintf("failed to read response body: %v", err)
		dtoResponse.Size = 0
		dtoResponse.Body = nil
	} else {
		dtoResponse.Size = int64(len(bodyBytes))
		dtoResponse.Body = bodyBytes
		// Check if response was truncated
		if limitedReader.N <= 0 {
			dtoResponse.Error = "response body truncated due to size limit"
		}
	}

	return dtoResponse, nil
}
