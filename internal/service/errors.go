package service

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrRequestTimeout = errors.New("request timeout")
)
