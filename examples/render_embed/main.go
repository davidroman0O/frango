// Example demonstrating the HandleRenderEmbed function
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

// Embed the PHP files directly
//
//go:embed php/dashboard.php
var dashboardTemplate embed.FS

func main() {
	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	log.Printf("Starting server with development mode: %v", !*prodMode)

	// Create middleware
	php, err := frango.New(
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Register the dashboard template with embed.FS
	log.Printf("Embedding dashboard template from %s", "php/dashboard.php")

	// First add the file from embed
	targetPath := php.AddFromEmbed("/dashboard", dashboardTemplate, "php/dashboard.php")

	// Create the render function that will be used for both routes
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		log.Println("Render function called - generating data")

		// Generate some sample data
		items := []map[string]interface{}{
			{
				"id":          1,
				"name":        "Widget Pro",
				"description": "The best widget ever made",
				"price":       19.99,
			},
			{
				"id":          2,
				"name":        "Super Gadget",
				"description": "A revolutionary gadget",
				"price":       29.99,
			},
			{
				"id":          3,
				"name":        "Amazing Product",
				"description": "You won't believe how amazing it is",
				"price":       39.99,
			},
		}

		stats := map[string]interface{}{
			"total_users":     1250,
			"active_users":    867,
			"total_products":  342,
			"revenue":         12568.99,
			"conversion_rate": "3.2%",
		}

		// Create the data to pass to PHP (with debug output just in case)
		data := map[string]interface{}{
			"title": "Dashboard - Embedded PHP Rendering",
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"role":  "Administrator",
			},
			"items": items,
			"stats": stats,
			// Add debug info directly
			"debug_info": map[string]interface{}{
				"timestamp":  time.Now().Format(time.RFC3339),
				"values_set": true,
			},
		}

		// Log each value for debugging
		for k, v := range data {
			jsonBytes, _ := json.Marshal(v)
			log.Printf("KEY %s = %s", k, string(jsonBytes))
		}

		return data
	}

	// Register the render handler for the dashboard
	php.SetRenderHandler("/dashboard", renderFn)

	// Also make it accessible at the root for convenience
	php.HandlePHP("/", targetPath)
	php.SetRenderHandler("/", renderFn)

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
	if err := http.ListenAndServe(":"+*port, php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
