package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv" // Import godotenv
	"github.com/suar-net/suar-be/internal/config"
	"github.com/suar-net/suar-be/internal/database"
	"github.com/suar-net/suar-be/internal/handler"
	"github.com/suar-net/suar-be/internal/service"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables from OS")
	}

	// Create a new logger
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// connect to database
	db, err := database.ConnectDB(cfg.DB)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	logger.Println("Succesfully connected to database")

	httpProxyService := service.NewHTTPProxyService()
	router := handler.SetupRouter(*httpProxyService, db, logger)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Printf("Server starting on port %s", cfg.Server.Port)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Cannot run server on port %s: %v", cfg.Server.Port, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Println("Shut down the server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		logger.Fatalf("Server shutdown failed: %v", err)
	}
	logger.Println("Server successfully shut down")
}
