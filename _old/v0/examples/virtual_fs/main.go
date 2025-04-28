package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/davidroman0O/frango"
)

//go:embed embedded templates
var embeddedFiles embed.FS

func main() {
	// Create Frango middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./web"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a standard Go ServeMux for routing
	mux := http.NewServeMux()

	// Create our virtual filesystem for demonstration
	fs := php.NewFS()

	// Add source directories
	err = fs.AddSourceDirectory("./web/pages", "/pages")
	if err != nil {
		log.Printf("Warning: Failed to add pages directory: %v", err)
	}

	err = fs.AddSourceDirectory("./web/api", "/api")
	if err != nil {
		log.Printf("Warning: Failed to add API directory: %v", err)
	}

	// Add embedded files
	err = fs.AddEmbeddedDirectory(embeddedFiles, "embedded", "/lib")
	if err != nil {
		log.Printf("Warning: Failed to add embedded files: %v", err)
	}

	err = fs.AddEmbeddedDirectory(embeddedFiles, "templates", "/templates")
	if err != nil {
		log.Printf("Warning: Failed to add embedded templates: %v", err)
	}

	// Create some virtual files for testing
	config := []byte(`<?php
// Virtual config file
$config = [
    'app_name' => 'Virtual FS Example',
    'version' => '1.0.0',
    'debug' => true,
];
`)
	err = fs.CreateVirtualFile("/config/app.php", config)
	if err != nil {
		log.Printf("Warning: Failed to create config file: %v", err)
	}

	// Create a simple index.php
	index := []byte(`<?php
header('Content-Type: text/plain');
echo "FRANGO MIDDLEWARE TEST\n";
echo "PHP Version: " . PHP_VERSION . "\n";
echo "Document Root: " . $_SERVER['DOCUMENT_ROOT'] . "\n";
echo "Timestamp: " . date('Y-m-d H:i:s') . "\n";
echo "\nThis is a PHP file served through the Frango middleware.\n";
?>`)
	err = fs.CreateVirtualFile("/index.php", index)
	if err != nil {
		log.Printf("Warning: Failed to create index file: %v", err)
	}

	// Create a handler function that will process PHP requests
	phpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Map URL path to a PHP file
		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.php" // Default to index.php for root path
		}

		// Use the VirtualFS's For method to get a handler for the requested file
		handler := fs.For(urlPath)
		handler.ServeHTTP(w, r)
	})

	// Simple handler for displaying server info (pure Go)
	infoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"server": "Frango Virtual FS Example",
			"message": "This endpoint is served directly by Go, without PHP"
		}`))
	})

	// Register our handlers with the ServeMux
	mux.Handle("/", phpHandler)      // Process all PHP requests
	mux.Handle("/info", infoHandler) // Pure Go endpoint

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get list of available files for logging
	availableFiles := fs.ListFiles()

	// Print available routes
	fmt.Printf("Server listening on http://localhost:%s\n\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  [ANY] / => /index.php (PHP through middleware)")
	fmt.Println("  [ANY] /pages/* => PHP files in web/pages/")
	fmt.Println("  [ANY] /api/* => PHP files in web/api/")
	fmt.Println("  [ANY] /lib/* => PHP files from embedded directory")
	fmt.Println("  [ANY] /templates/* => PHP files from embedded directory")
	fmt.Println("  [ANY] /config/app.php => Virtual PHP file")
	fmt.Println("  [GET] /info => Go-handled endpoint")

	fmt.Println("\nVirtual files in filesystem:")
	for _, file := range availableFiles {
		fmt.Printf("  %s\n", file)
	}

	// Start the server with our ServeMux
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
