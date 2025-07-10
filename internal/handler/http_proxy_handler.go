package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/suar-net/suar-be/internal/model"
	service "github.com/suar-net/suar-be/internal/service"
)

// HTTPProxyService adalah interface yang mendefinisikan kontrak untuk service HTTP proxy.
// Handler bergantung pada interface ini, bukan pada implementasi konkretnya.
type HTTPProxyService interface {
	ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error)
}

// HTTPProxyHandler adalah struct yang mengimplementasikan http.Handler untuk fungsionalitas HTTP proxy.
type HTTPProxyHandler struct {
	service HTTPProxyService
	logger  *log.Logger
}

// NewHTTPProxyHandler adalah constructor untuk HTTPProxyHandler.
func NewHTTPProxyHandler(s HTTPProxyService, l *log.Logger) *HTTPProxyHandler {
	return &HTTPProxyHandler{
		service: s,
		logger:  l,
	}
}

func (h *HTTPProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var dto model.DTORequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate the DTO
	if err := validate.Struct(&dto); err != nil {
		errMsg := ValidationError(err)
		respondWithError(w, http.StatusBadRequest, errMsg)
		return
	}

	// r.Context() carries deadlines, cancellation signals, and other request-scoped values.
	dtoResponse, err := h.service.ProcessRequest(r.Context(), &dto)
	if err != nil {
		h.logger.Printf("ERROR: %v", err) // Log the actual error

		// Check for specific error types to return appropriate status codes
		if errors.Is(err, service.ErrInvalidInput) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		} else if errors.Is(err, service.ErrRequestTimeout) {
			respondWithError(w, http.StatusGatewayTimeout, err.Error())
			return
		}

		// For any other error, return a generic 500
		respondWithError(w, http.StatusInternalServerError, "An internal error occurred")
		return
	}

	respondWithJson(w, http.StatusOK, dtoResponse)
}
