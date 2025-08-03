package service

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrRequestTimeout = errors.New("request timeout")

	// Auth-related errors
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email is already taken")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrTokenExpired       = errors.New("token has expired")
)
