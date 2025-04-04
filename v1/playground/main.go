package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

//go:embed index.php
//go:embed users/*.php
//go:embed products/*.php
//go:embed nested/deep/path/*.php
//go:embed categories/**/*.php
var phpFiles embed.FS

func main() {
	// Initialize Frango middleware
	php, err := frango.New(
		frango.WithDevelopmentMode(true),
	)

	if err != nil {
		log.Fatalf("Failed to create frango instance: %v", err)
	}

	// Add embedded PHP files to the middleware
	php.AddEmbeddedDirectory(phpFiles, ".", "/")

	// Create a standard HTTP mux
	mux := http.NewServeMux()

	// Simple routes with automatic parameter extraction
	mux.Handle("/", php.For("/index.php"))
	mux.Handle("/users/", php.For("/users/{id}.php"))
	mux.Handle("/products/", php.For("/products/{id}.php"))
	mux.Handle("/nested/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/categories/", php.For("/categories/{category}/{subcategory}.php"))

	// Start the server
	log.Println("Starting playground server at http://localhost:8080")
	log.Println("Try these routes:")
	log.Println("  - / (Home page with debug info)")
	log.Println("  - /users/123 (User profile with ID parameter)")
	log.Println("  - /products/456?color=red (Product with ID and query param)")
	log.Println("  - /nested/deep/path (Deeply nested path)")
	log.Println("  - /categories/electronics/laptops (Multiple path parameters)")

	http.ListenAndServe(":8080", mux)
}
