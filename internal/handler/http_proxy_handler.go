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
		h.respondWithError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var dto model.DTORequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate the DTO
	if err := validate.Struct(&dto); err != nil {
		errMsg := ValidationError(err)
		h.respondWithError(w, http.StatusBadRequest, errMsg)
		return
	}

	// r.Context() carries deadlines, cancellation signals, and other request-scoped values.
	dtoResponse, err := h.service.ProcessRequest(r.Context(), &dto)
	if err != nil {
		h.logger.Printf("ERROR: %v", err) // Log the actual error

		// Check for specific error types to return appropriate status codes
		if errors.Is(err, service.ErrInvalidInput) {
			h.respondWithError(w, http.StatusBadRequest, err.Error())
			return
		} else if errors.Is(err, service.ErrRequestTimeout) {
			h.respondWithError(w, http.StatusGatewayTimeout, err.Error())
			return
		}

		// For any other error, return a generic 500
		h.respondWithError(w, http.StatusInternalServerError, "An internal error occurred")
		return
	}

	h.respondWithJSON(w, http.StatusOK, dtoResponse)
}

// helper function to send a JSON error message with a status code.
func (h *HTTPProxyHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON adalah fungsi helper untuk marshal payload ke JSON dan mengirimkannya.
func (h *HTTPProxyHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		// Avoid recursion - write error directly
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to marshal response"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
