package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Find web directory with automatic resolution
	webDir, err := frango.ResolveDirectory("www")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}

	// Create PHP middleware with functional options
	php, err := frango.New(
		frango.WithSourceDir(webDir),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Register the Redis API endpoint for the REST API
	// This uses SimpleRedis.php (pure PHP Redis client) for data storage
	php.HandlePHP("/api/redis", "api.php")

	// The root endpoint (/) is automatically handled by index.php
	// which displays Redis connection status and statistics

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start the HTTP server with PHP middleware as the handler
	log.Println("Starting Redis example server on :8082")
	log.Println("Open http://localhost:8082/ in your browser")

	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
