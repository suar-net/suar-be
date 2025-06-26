package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

func newOutboundRequest(dto *model.DTORequest) (*OutboundRequest, error) {
	// HTTP Method Validation
	dto.Method = strings.ToUpper(dto.Method)
	if !allowedMethods[dto.Method] {
		return nil, fmt.Errorf("invalid or unsupported HTTP method: %s", dto.Method)
	}

	// URL Validation
	if dto.URL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}
	parsedURL, err := url.Parse(dto.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s. Only 'http' and 'https' are allowed", parsedURL.Scheme)
	}

	// Timeout Validation
	var timeout time.Duration
	if dto.Timeout <= 0 {
		timeout = defaultRequestTimeout
	} else {
		timeout = time.Duration(dto.Timeout) * time.Millisecond
	}
	if timeout > maxRequestTimeout {
		return nil, fmt.Errorf("timeout of %v exceeds the maximum allowed limit of %v", timeout, maxRequestTimeout)
	}

	headers := make(http.Header)
	for key, values := range dto.Headers {
		headers[key] = values
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

type Service struct {
	httpClient *http.Client
}

func NewService() *Service {
	return &Service{
		httpClient: &http.Client{},
	}
}

func (s *Service) ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error) {
	// Convert and validate the DTO to our internal request model.
	outboundRequest, err := newOutboundRequest(dto)
	if err != nil {
		return &model.DTOResponse{
			Error:     fmt.Sprintf("invalid request input: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	return s.Execute(ctx, outboundRequest)
}

func (s *Service) Execute(ctx context.Context, outboundRequest *OutboundRequest) (*model.DTOResponse, error) {
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
	if err != nil {
		return &model.DTOResponse{
			Error:     fmt.Sprintf("failed to execute request to target server: %v", err),
			Duration:  duration,
			Timestamp: startTime,
		}, nil
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
