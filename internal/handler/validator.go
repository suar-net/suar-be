package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidationError wraps the validators.ValidationErrors to provide a more user-friendly message.
func ValidationError(err error) string {
	if err == nil {
		return ""
	}

	validationErrors := err.(validator.ValidationErrors)
	var errorMsgs []string

	for _, e := range validationErrors {
		// Customize error messages for better feedback
		switch e.Tag() {
		case "required":
			errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' is required", e.Field()))
		case "url":
			errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' must be a valid URL", e.Field()))
		case "httpmethod":
			errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' must be a valid HTTP method", e.Field()))
		case "gte":
			errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' must be greater than or equal to %s", e.Field(), e.Param()))
		case "lte":
			errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' must be less than or equal to %s", e.Field(), e.Param()))
		default:
			errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' failed on the '%s' tag", e.Field(), e.Tag()))
		}
	}

	return strings.Join(errorMsgs, ", ")
}
