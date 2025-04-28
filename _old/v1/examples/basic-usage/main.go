package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create a new frango middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./php"),    // Directory containing PHP files
		frango.WithDevelopmentMode(true), // Enable development mode
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown() // Always clean up resources when done

	// Create a standard HTTP server
	mux := http.NewServeMux()

	// Map routes to PHP files
	mux.Handle("/", php.For("index.php"))
	mux.Handle("/info", php.For("info.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
