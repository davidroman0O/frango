package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Find the web directory using the library's built-in function
	webDir, err := frango.ResolveDirectory("web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}
	log.Printf("Using web directory: %s", webDir)

	// Create server instance with functional options
	server, err := frango.NewServer(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Register specific endpoints with explicit control over URL routing
	// Format: HandlePHP(pattern, phpFilePath)
	// - pattern: URL pattern that will be exposed to clients
	// - phpFilePath: Path to the PHP file (relative to web directory)

	// Standard endpoints
	server.HandlePHP("/api/user", "api/user.php")
	server.HandlePHP("/api/items", "api/items.php")

	// You can map the same PHP file to multiple URL paths
	server.HandlePHP("/api/users", "api/user.php") // Alias for the same file

	// You can register URLs with or without .php extension
	server.HandlePHP("/about", "about.php")     // Clean URL without .php
	server.HandlePHP("/about.php", "about.php") // Traditional URL with .php

	// Create clean URLs for index pages
	server.HandlePHP("/", "index.php") // Root maps to index.php

	// Register a custom Go handler
	server.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `"}`))
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
	log.Printf("Server starting on port %s", *port)
	if err := server.ListenAndServe(context.Background(), ":"+*port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
