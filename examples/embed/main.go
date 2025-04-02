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

	// Create Frango instance (No SourceDir needed when only using embeds)
	php, err := frango.New(
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Add embedded files using AddEmbeddedLibrary.
	// It returns the temporary path where the file was written.
	// We use this temp path when creating the handler.
	tempIndexPath, err := php.AddEmbeddedLibrary(indexPhp, "php/index.php", "/index.php")
	assertNoError(err, "Add index.php")
	tempUserPath, err := php.AddEmbeddedLibrary(userPhp, "php/api/user.php", "/api/user.php")
	assertNoError(err, "Add user.php")
	tempItemsPath, err := php.AddEmbeddedLibrary(itemsPhp, "php/api/items.php", "/api/items.php")
	assertNoError(err, "Add items.php")

	// Create a standard HTTP mux for routing
	mux := http.NewServeMux()

	// --- Register PHP Handlers ---
	// Register routes pointing to the temporary paths of the embedded files.
	mux.Handle("/", php.HandlerFor("/", tempIndexPath))
	mux.Handle("/index", php.HandlerFor("/index", tempIndexPath)) // Clean URL
	mux.Handle("/index.php", php.HandlerFor("/index.php", tempIndexPath))

	mux.Handle("/api/user", php.HandlerFor("/api/user", tempUserPath))
	mux.Handle("/api/user.php", php.HandlerFor("/api/user.php", tempUserPath))

	mux.Handle("/api/items", php.HandlerFor("/api/items", tempItemsPath))
	mux.Handle("/api/items.php", php.HandlerFor("/api/items.php", tempItemsPath))

	// --- Register Go Handlers ---
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `", "source": "go"}`))
	})

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start server with the standard mux
	log.Printf("Embed example server starting on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Simple error helper
func assertNoError(err error, context string) {
	if err != nil {
		log.Fatalf("Error during setup (%s): %v", context, err)
	}
}
