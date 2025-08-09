package handler

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"
)

type HealthHandler struct {
	db     *sql.DB
	logger *log.Logger
}

func NewHealthHandler(db *sql.DB, logger *log.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Ping database untuk memeriksa koneksi
	if err := h.db.PingContext(ctx); err != nil {
		h.logger.Printf("Health check failed: database connection error: %v", err)

		respondWithError(w, http.StatusServiceUnavailable, "Database connection failed")
		return
	}

	data := map[string]string{
		"status":  "ok",
		"message": "Service is healthy and database connection is active",
	}
	respondWithJson(w, http.StatusOK, data)
}
