package main

import (
	"log"

	gophp "github.com/davidroman0O/gophp"
)

func main() {
	// Find web directory with automatic resolution
	webDir, err := gophp.ResolveDirectory("www")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}

	// Create server with functional options
	server, err := gophp.NewServer(
		gophp.WithSourceDir(webDir),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Register the Redis API endpoint for the REST API
	// This uses SimpleRedis.php (pure PHP Redis client) for data storage
	server.HandlePHP("/api/redis", "api.php")

	// The root endpoint (/) is automatically handled by index.php
	// which displays Redis connection status and statistics

	log.Println("Starting Redis example server on :8082")
	log.Println("Open http://localhost:8082/ in your browser")

	// Start serving PHP files
	if err := server.ListenAndServe(":8082"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
