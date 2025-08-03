package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/suar-net/suar-be/internal/model"
	"github.com/suar-net/suar-be/internal/service"
)

type AuthHandler struct {
	authService service.IAuthService
	logger      *log.Logger
}

func NewAuthHandler(s service.IAuthService, l *log.Logger) *AuthHandler {
	return &AuthHandler{
		authService: s,
		logger:      l,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.DTOUserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := validate.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, ValidationError(err))
		return
	}

	user, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "already taken") {
			respondWithError(w, http.StatusConflict, err.Error())
		} else {
			h.logger.Printf("Error registering user: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to register user")
		}
		return
	}

	user.PasswordHash = ""
	respondWithJson(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.DTOLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := validate.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, ValidationError(err))
		return
	}

	resp, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			respondWithError(w, http.StatusUnauthorized, err.Error())
		} else {
			h.logger.Printf("Error logging in user: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to login user")
		}
		return
	}

	respondWithJson(w, http.StatusOK, resp)
}
