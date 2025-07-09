package model

import (
	"encoding/json"
	"time"
)

// DTO for incoming JSON requests.
type DTORequest struct {
	Method  string              `json:"method" validate:"required,httpmethod"`
	URL     string              `json:"url" validate:"required,url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
	Timeout int                 `json:"timeout" validate:"gte=0,lte=90000"` // 0 means default, max 90s
}

// DTO for outgoing JSON responses.
type DTOResponse struct {
	StatusCode int                 `json:"status_code"`
	Duration   time.Duration       `json:"duration"`
	Timestamp  time.Time           `json:"timestamp"`
	Size       int64               `json:"size"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body,omitempty"`
	Error      string              `json:"error,omitempty"`
}
