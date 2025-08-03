package handler

import (
	"database/sql"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/suar-net/suar-be/internal/repository"
	"github.com/suar-net/suar-be/internal/service"
)

// SetupRouter creates the main Chi router for the application.
// add necessary service by inject it into the handlers.
func SetupRouter(
	repository repository.Repository,
	service service.Service,
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
	requestHandler := NewRequestHandelr(service.RequestService(), logger)
	authHandler := NewAuthHandler(service.AuthService(), logger)
	healthHandler := NewHealthHandler(db, logger)

	// --- Inisialisasi Middleware ---
	authMiddleware := NewAuthMiddleware(service.AuthService(), logger)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthcheck", healthHandler.Check)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/request", requestHandler.ServeHTTP)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

		})
	})

	return r
}
