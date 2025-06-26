package main

import (
	"log"
	"net/http"

	// Import your internal packages
	"github.com/suar-net/suar-be/internal/handler"
	proxy "github.com/suar-net/suar-be/internal/service"
)

func main() {
	// --- 1. Initialize Dependencies ---
	// Create an instance of our core proxy/requester service.
	proxyService := proxy.NewService()

	// --- 2. Setup Router ---
	// Call the SetupRouter function from the handler package, injecting the service.
	// This gives the router and its handlers access to the business logic.
	router := handler.SetupRouter(proxyService)

	// --- 3. Start the Server ---
	port := ":8080"
	log.Printf("Starting server on http://localhost%s", port)

	// http.ListenAndServe starts the server. We wrap it in log.Fatal
	// so that if the server fails to start for any reason (e.g., port is busy),
	// the error will be logged, and the application will exit.
	err := http.ListenAndServe(port, router)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
