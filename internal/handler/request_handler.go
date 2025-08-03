package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/suar-net/suar-be/internal/model"
	"github.com/suar-net/suar-be/internal/service"
)

type RequestHandler struct {
	requestService service.IRequestService
	logger         *log.Logger
}

func NewRequestHandelr(s service.IRequestService, l *log.Logger) *RequestHandler {
	return &RequestHandler{
		requestService: s,
		logger:         l,
	}
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	dtoResponse, err := h.requestService.ProcessRequest(r.Context(), &dto)
	if err != nil {
		h.logger.Printf("ERROR: %v", err)

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
