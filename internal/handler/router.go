package handler

import (
	"database/sql"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/suar-net/suar-be/internal/service"
)

// SetupRouter creates the main Chi router for the application.
// add necessary service by inject it into the handlers.
func SetupRouter(
	httpProxyService service.HTTPProxyService,
	db *sql.DB,
	logger *log.Logger,
) *chi.Mux {
	// Create a new Chi router instance.
	r := chi.NewRouter()

	// global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(
		cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		},
	))

	// --- Inisialisasi Semua Handler ---
	httpProxyHandler := NewHTTPProxyHandler(&httpProxyService, logger)
	healthHandler := NewHealthHandler(db, logger)

	// --- Definisi Rute ---
	r.Route("/api/v1", func(r chi.Router) {
		// Mount handler untuk endpoint yang berbeda
		r.Mount("/request", httpProxyHandler)
		r.Get("/healthcheck", healthHandler.Check)
	})

	return r
}
