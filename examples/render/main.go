// Example demonstrating the use of the Render function
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Enable verbose logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Define PHP source directory
	phpDir := "php"

	// Create Frango instance
	php, err := frango.New(
		frango.WithSourceDir(phpDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	log.Printf("Using PHP directory: %s", php.SourceDir())

	// Define a simple render function
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		now := time.Now()
		log.Printf("Render function called for path: %s", r.URL.Path)

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

	// Create mux and register route using RenderHandlerFor
	mux := http.NewServeMux()
	// Pattern includes method for Go 1.22+ mux
	pattern := "GET /"
	scriptPath := "template.php" // Relative to sourceDir
	mux.Handle(pattern, php.RenderHandlerFor(pattern, scriptPath, renderFn))

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
	log.Printf("Render Example Server started at http://localhost:%s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
