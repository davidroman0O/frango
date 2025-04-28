// Example demonstrating rendering embedded PHP templates
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

//go:embed php/dashboard.php
var dashboardTemplate embed.FS

// Simple error helper
func assertNoError(err error, context string) {
	if err != nil {
		log.Fatalf("Error during setup (%s): %v", context, err)
	}
}

func main() {
	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	log.Printf("Starting server with development mode: %v", !*prodMode)

	// Create Frango instance (no SourceDir needed)
	php, err := frango.New(
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Add the embedded dashboard template using AddEmbeddedLibrary
	// This writes it to a temp location and makes it available to the cache.
	templateDiskPath, err := php.AddEmbeddedLibrary(dashboardTemplate, "php/dashboard.php", "/dashboard.php")
	assertNoError(err, "Add dashboard.php")

	// Create the render function
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		log.Println("Render function called - generating data")
		// ... (generate sample data - items, stats - same as before) ...
		items := []map[string]interface{}{
			{"id": 1, "name": "Widget Pro", "description": "The best widget ever made", "price": 19.99},
			{"id": 2, "name": "Super Gadget", "description": "A revolutionary gadget", "price": 29.99},
			{"id": 3, "name": "Amazing Product", "description": "You won't believe how amazing it is", "price": 39.99},
		}
		stats := map[string]interface{}{
			"total_users": 1250, "active_users": 867, "total_products": 342,
			"revenue": 12568.99, "conversion_rate": "3.2%",
		}

		data := map[string]interface{}{
			"title":      "Dashboard - Embedded PHP Rendering",
			"user":       map[string]interface{}{"name": "John Doe", "email": "john@example.com", "role": "Administrator"},
			"items":      items,
			"stats":      stats,
			"debug_info": map[string]interface{}{"timestamp": time.Now().Format(time.RFC3339), "values_set": true},
		}

		// Log each value for debugging
		for k, v := range data {
			jsonBytes, _ := json.Marshal(v)
			log.Printf("KEY %s = %s", k, string(jsonBytes))
		}
		return data
	}

	// Create mux and register route using the new Render method
	mux := http.NewServeMux()
	// Use the temporary disk path returned by AddEmbeddedLibrary
	mux.Handle("GET /", php.Render(templateDiskPath, renderFn))

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start the server
	log.Printf("Render Embed Example running on http://localhost:%s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
