// File baru: internal/handler/health_handler.go
package handler

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"
)

// HealthHandler adalah handler untuk endpoint health check.
type HealthHandler struct {
	db     *sql.DB
	logger *log.Logger
}

// NewHealthHandler adalah constructor untuk HealthHandler.
func NewHealthHandler(db *sql.DB, logger *log.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// Check adalah http.HandlerFunc yang melakukan pengecekan kesehatan sistem.
// Saat ini hanya memeriksa koneksi database.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Ping database untuk memeriksa koneksi
	if err := h.db.PingContext(ctx); err != nil {
		h.logger.Printf("Health check failed: database connection error: %v", err)
		
		// Gunakan helper yang sudah kita buat!
		respondWithError(w, http.StatusServiceUnavailable, "Database connection failed")
		return
	}

	// Jika berhasil, kirim status OK
	data := map[string]string{
		"status":  "ok",
		"message": "Service is healthy and database connection is active",
	}
	respondWithJson(w, http.StatusOK, data)
}
