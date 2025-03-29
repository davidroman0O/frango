package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

// Embed the PHP files directly
//
//go:embed php/index.php
var indexPhp embed.FS

//go:embed php/api/user.php
var userPhp embed.FS

//go:embed php/api/items.php
var itemsPhp embed.FS

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Create a PHP middleware with functional options
	php, err := frango.New(
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Add the PHP files from embedded filesystem
	indexPath := php.AddFromEmbed("/index.php", indexPhp, "php/index.php")
	userPath := php.AddFromEmbed("/api/user.php", userPhp, "php/api/user.php")
	itemsPath := php.AddFromEmbed("/api/items.php", itemsPhp, "php/api/items.php")

	// Explicitly register additional routes for these files
	php.HandlePHP("/", indexPath)      // Root path
	php.HandlePHP("/index", indexPath) // Without .php extension

	php.HandlePHP("/api/user", userPath)
	php.HandlePHP("/api/items", itemsPath)

	// Create a standard HTTP mux for routing
	mux := http.NewServeMux()

	// Register a custom Go handler
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `", "source": "go"}`))
	})

	// PHP middleware handles everything first, falls back to mux
	handler := php.Wrap(mux)

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start server with our handler
	log.Printf("Embed example server starting on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
