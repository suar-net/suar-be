package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Request struct {
	ID                 int             `json:"id"`
	UserID             *int            `json:"user_id"`
	ExecutedAt         time.Time       `json:"executed_at"`
	RequestMethod      string          `json:"request_method"`
	RequestURL         string          `json:"request_url"`
	RequestHeaders     json.RawMessage `json:"request_headers"`
	RequestBody        *string         `json:"request_body"`
	ResponseStatusCode *int            `json:"response_status_code"`
	ResponseHeaders    json.RawMessage `json:"response_headers"`
	ResponseBody       *string         `json:"response_body"`
	ResponseSize       *int64          `json:"response_size"`
	DurationMs         *int            `json:"duration_ms"`
}
