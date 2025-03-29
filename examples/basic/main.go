package main

import (
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

	// Create PHP middleware instance with functional options
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Register specific endpoints with explicit control over URL routing
	// Format: HandlePHP(pattern, phpFilePath)
	// - pattern: URL pattern that will be exposed to clients
	// - phpFilePath: Path to the PHP file (relative to web directory)

	// Standard endpoints
	php.HandlePHP("/api/user", "api/user.php")
	php.HandlePHP("/api/items", "api/items.php")

	// You can map the same PHP file to multiple URL paths
	php.HandlePHP("/api/users", "api/user.php") // Alias for the same file

	// You can register URLs with or without .php extension
	php.HandlePHP("/about", "about.php")     // Clean URL without .php
	php.HandlePHP("/about.php", "about.php") // Traditional URL with .php

	// Create clean URLs for index pages
	php.HandlePHP("/", "index.php") // Root maps to index.php

	// Create a standard HTTP mux for routing
	mux := http.NewServeMux()

	// Register a custom Go handler
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// Use the PHP middleware as the default handler
	// This routes all requests through PHP middleware, which will handle PHP files
	// and pass through non-PHP requests

	// Option 1: PHP handles everything first, falls back to mux for Go handlers
	handler := php.Wrap(mux)

	// Option 2: Use http.ServeMux and mount PHP directly for all paths
	// Comment out Option 1 and uncomment these lines to use this approach instead
	// mux.Handle("/", php)
	// handler := mux

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start server with our handlers
	log.Printf("Server starting on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
