package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/suar-net/suar-be/internal/model"
)

// OutboundRequest represents a request to be sent to an external service.
type OutboundRequest struct {
	Method  string
	URL     *url.URL
	Headers http.Header
	Body    []byte
	Timeout time.Duration
}

func newOutboundRequest(dto *model.DTORequest) (*OutboundRequest, error) {
	// Parse the URL from the DTO
	parsedURL, err := url.Parse(dto.URL)
	if err != nil {
		return nil, err
	}

	// Create the outbound request
	request := &OutboundRequest{
		Method:  dto.Method,
		URL:     parsedURL,
		Headers: http.Header(dto.Headers),
		Body:    dto.Body,
		Timeout: time.Duration(dto.Timeout) * time.Millisecond,
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

// Execute is the single public method for running a request.
// It orchestrates the creation, timeout handling, and execution.
func (s *Service) Execute(ctx context.Context, outboundRequest *OutboundRequest) (httpResponse *http.Response, err error) {
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
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpRequest.Header = outboundRequest.Headers

	response, err := s.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return response, nil
}
