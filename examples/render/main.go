// Example demonstrating the use of the Render function
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/davidroman0O/frango"
)

func main() {
	// Find the example directory with PHP files
	phpDir, err := frango.ResolveDirectory("php")
	if err != nil {
		log.Fatalf("Error finding PHP directory: %v", err)
	}

	log.Printf("PHP directory: %s", phpDir)

	// Create a new PHP server with the PHP directory as source
	server, err := frango.NewServer(
		frango.WithSourceDir(phpDir),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Register a render handler for the root path
	server.HandleRender("/", "template.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		now := time.Now()

		// Return variables to inject into the PHP template
		return map[string]interface{}{
			"title": "Go PHP Render Example - " + now.Format(time.RFC1123),
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"role":  "Administrator",
			},
			"items": []map[string]interface{}{
				{
					"name":        "Product 1",
					"description": "This is the first product",
					"price":       19.99,
				},
				{
					"name":        "Product 2",
					"description": "This is the second product",
					"price":       29.99,
				},
			},
		}
	})

	// Start the server
	log.Printf("Server started at http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", server))
}
