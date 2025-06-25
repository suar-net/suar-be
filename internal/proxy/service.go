package proxy

import (
	"net/http"
	"net/url"
	"time"
)

// OutboundRequest represents a request to be sent to an external service.
type OutboundRequest struct {
	Method  string
	URL     *url.URL
	Headers http.Header
	Body    []byte
	Timeout time.Duration
}
