// Example demonstrating the use of the Render function
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/davidroman0O/frango"
)

func main() {
	// Enable verbose logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Find the example directory with PHP files
	phpDir, err := frango.ResolveDirectory("php")
	if err != nil {
		log.Fatalf("Error finding PHP directory: %v", err)
	}

	log.Printf("PHP directory: %s", phpDir)

	// Verify that the template file exists
	templatePath := strings.Join([]string{phpDir, "template.php"}, string(os.PathSeparator))
	if _, err := os.Stat(templatePath); err != nil {
		log.Fatalf("Template file not found at %s: %v", templatePath, err)
	} else {
		log.Printf("Template file found at %s", templatePath)
	}

	// Create a PHP middleware with the PHP directory as source
	php, err := frango.New(
		frango.WithSourceDir(phpDir),
		frango.WithDevelopmentMode(true),
		// Omit the logger option to use the default one
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Define a simple render function - use the RenderData type
	var renderFn frango.RenderData = func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		now := time.Now()
		log.Printf("Render function called for / path")

		// Debug data
		data := map[string]interface{}{
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

		// Debug output - print the render data
		fmt.Println("RENDER DATA FOR DEBUG:")
		for k, v := range data {
			fmt.Printf("  %s: %v\n", k, v)
		}

		return data
	}

	// Use HandleRender instead of separate HandlePHP and SetRenderHandler calls
	// This directly connects the template file with the render function
	php.HandleRender("/", "template.php", renderFn)

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start the server using standard Go HTTP server
	log.Printf("Server started at http://localhost:8082")
	log.Printf("Open http://localhost:8082/ in your browser")
	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
