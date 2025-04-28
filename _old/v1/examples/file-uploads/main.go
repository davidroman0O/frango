package main

import (
	"log"
	"net/http"
	"os"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create uploads directory if it doesn't exist
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Create a new frango middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
		// Note: WithMount might not be available in the current version
		// For file uploads, PHP will use the uploadsDir configured in process-upload.php
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP server
	mux := http.NewServeMux()

	// Routes
	mux.Handle("/", php.For("index.php"))
	mux.Handle("/upload", php.For("upload.php"))
	mux.Handle("POST /process-upload", php.For("process-upload.php"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))
	mux.Handle("/view-files", php.For("view-files.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
