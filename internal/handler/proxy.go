package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/suar-net/suar-be/internal/model"
)

type ProxyService interface {
	ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error)
}

type RequesterHandler struct {
	service ProxyService
}

func NewRequesterHandler(s ProxyService) *RequesterHandler {
	return &RequesterHandler{
		service: s,
	}
}

func (h *RequesterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var dto model.DTORequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// r.Context() carries deadlines, cancellation signals, and other request-scoped values.
	dtoResponse, err := h.service.ProcessRequest(r.Context(), &dto)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "An internal error occurred")
		return
	}

	h.respondWithJSON(w, http.StatusOK, dtoResponse)
}

// helper function to send a JSON error message with a status code.
func (h *RequesterHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

// helper function to marshal a payload to JSON and send it.
func (h *RequesterHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
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
