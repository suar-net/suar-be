package handler

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// SetupRouter creates the main Chi router for the application.
// It takes the service and a logger as dependencies to inject into the handlers.
func SetupRouter(s HTTPProxyService, logger *log.Logger) *chi.Mux {
	// Create a new Chi router instance.
	r := chi.NewRouter()

	// --- Standard Middleware ---
	// Logger: Logs request details (method, path, latency, status). Very useful for debugging.
	r.Use(middleware.Logger)
	// Recoverer: Recovers from panics and returns a 500 error instead of crashing.
	r.Use(middleware.Recoverer)

	// --- CORS Middleware ---
	// This is critical for allowing your frontend (on a different domain) to communicate
	// with your backend.
	r.Use(cors.Handler(cors.Options{
		// IMPORTANT: For production, you should lock this down to your specific frontend's domain.
		// e.g., AllowedOrigins:   []string{"https://your-frontend.vercel.app"},
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any major browser
	}))

	// --- Route Definitions ---

	// Create an instance of our requester handler, injecting the service and logger.
	httpProxyHandler := NewHTTPProxyHandler(s, logger)

	// We'll group our API endpoints under a versioned path.
	r.Route("/api/v1", func(r chi.Router) {
		// Mount the handler for the specific endpoint.
		// We use r.Mount to delegate all methods, but our handler only accepts POST.
		// Alternatively, you could use r.Post("/request", httpProxyHandler.ServeHTTP)
		r.Mount("/request", httpProxyHandler)
	})

	return r
}
