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
//go:embed forms/*.php
//go:embed debug_panel.php
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

	// Form specific routes - explicit mappings for each form endpoint
	mux.Handle("/forms/form_display", php.For("/forms/form_display.php"))
	mux.Handle("/forms/post_display", php.For("/forms/post_display.php"))
	mux.Handle("/forms/get_display", php.For("/forms/get_display.php"))
	mux.Handle("/forms/upload_display", php.For("/forms/upload_display.php"))

	// For form submissions using the hyphenated convention
	mux.Handle("/forms/form-post", php.For("/forms/form-post.php"))
	mux.Handle("/forms/form-get", php.For("/forms/form-get.php"))
	mux.Handle("/forms/form-upload", php.For("/forms/form-upload.php"))

	// Debug pages
	mux.Handle("/debug.php", php.For("/debug.php"))
	mux.Handle("/forms/form_debug.php", php.For("/forms/form_debug.php"))

	// Default routes
	mux.Handle("/forms", php.For("/forms/index.php"))
	mux.Handle("/forms/", php.For("/forms/index.php"))
	mux.Handle("/users/", php.For("/users/{id}.php"))
	mux.Handle("/products/", php.For("/products/{id}.php"))
	mux.Handle("/nested/deep/path", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/path/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/categories/", php.For("/categories/{category}/{subcategory}.php"))

	// Root route
	mux.Handle("/", php.For("/index.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
