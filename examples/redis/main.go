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
	// Define web directory
	webDir := "www"

	// Create Frango instance
	php, err := frango.New(
		frango.WithSourceDir(webDir),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Create mux and register PHP handlers
	mux := http.NewServeMux()

	// Register the Redis API endpoint using the new For method
	// Assuming it handles relevant methods (GET/POST etc)
	mux.Handle("/api/redis", php.For("api.php"))

	// Register the root endpoint (index.php)
	mux.Handle("/", php.For("index.php"))

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start the HTTP server with the mux
	log.Println("Starting Redis example server on :8082")
	log.Printf("Using web directory: %s", php.SourceDir())
	log.Println("Open http://localhost:8082/ in your browser")

	if err := http.ListenAndServe(":8082", mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
