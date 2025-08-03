package model

import (
	"encoding/json"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Change incoming request body from JSON to http request format
type DTORequest struct {
	Method  string              `json:"method" validate:"required"`
	URL     string              `json:"url" validate:"required,url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
	Timeout int                 `json:"timeout" validate:"gte=0,lte=90000"` // 0 means default, max 90s
}

// Change incoming http response from complex object to simplified version
type DTOResponse struct {
	StatusCode int                 `json:"status_code"`
	Duration   time.Duration       `json:"duration"`
	Timestamp  time.Time           `json:"timestamp"`
	Size       int64               `json:"size"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body,omitempty"`
	Error      string              `json:"error,omitempty"`
}

type DTOUserRegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type DTOLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type DTOLoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type Claims struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}
