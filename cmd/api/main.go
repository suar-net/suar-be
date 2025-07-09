package main

import (
	"log"
	"net/http"
	"os"
	"time"

	// Import your internal packages
	"github.com/suar-net/suar-be/internal/handler"
	proxy "github.com/suar-net/suar-be/internal/service"
)

func main() {
	// Create a new logger
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// --- 1. Initialize Dependencies ---
	// Create an instance of our core proxy/requester service.
	proxyService := proxy.NewHTTPProxyService()

	// --- 2. Setup Router ---
	// Call the SetupRouter function from the handler package, injecting the service and logger.
	// This gives the router and its handlers access to the business logic.
	router := handler.SetupRouter(proxyService, logger)

	// --- 3. Start the Server ---
	// Get port from environment variable, with a fallback to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// We'll use a server instance for more control
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		// Good practice: Set timeouts to avoid resource leaks.
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Starting server on http://localhost:%s", port)

	// http.ListenAndServe starts the server. We wrap it in log.Fatal
	// so that if the server fails to start for any reason (e.g., port is busy),
	// the error will be logged, and the application will exit.
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
