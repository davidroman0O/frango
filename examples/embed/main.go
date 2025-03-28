package main

import (
	"context"
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

	// Create a server with functional options
	server, err := frango.NewServer(
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Add the PHP files from embedded filesystem - simple and direct
	// Create files without registering endpoints
	indexPath := server.AddPHPFromEmbed("/index.php", indexPhp, "php/index.php")
	userPath := server.AddPHPFromEmbed("/api/user.php", userPhp, "php/api/user.php")
	itemsPath := server.AddPHPFromEmbed("/api/items.php", itemsPhp, "php/api/items.php")

	// Now explicitly register the endpoints
	server.HandlePHP("/", indexPath)          // Root path
	server.HandlePHP("/index", indexPath)     // Without .php extension
	server.HandlePHP("/index.php", indexPath) // With .php extension

	server.HandlePHP("/api/user", userPath)
	server.HandlePHP("/api/user.php", userPath)

	server.HandlePHP("/api/items", itemsPath)
	server.HandlePHP("/api/items.php", itemsPath)

	// Register a custom Go handler
	server.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `", "source": "go"}`))
	})

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Shutdown()
		os.Exit(0)
	}()

	// Start server (this blocks until the server is stopped)
	log.Printf("Embed example server starting on port %s", *port)
	if err := server.ListenAndServe(context.Background(), ":"+*port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
