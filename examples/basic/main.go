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

	// Define the web directory relative to the example's location
	webDir := "web"

	// Create Frango instance (PHP execution engine)
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP mux
	mux := http.NewServeMux()

	// --- Register PHP Handlers using the new For method ---
	// Now we only need to specify the pattern once - eliminating duplication

	// Standard endpoints (METHOD defaults to ANY if not specified)
	mux.Handle("/api/user", php.For("api/user.php"))
	mux.Handle("/api/items", php.For("api/items.php"))

	// Alias for the same file
	mux.Handle("/api/users", php.For("api/user.php"))

	// Clean URL without .php (Requires Go 1.22+ mux for good matching)
	mux.Handle("/about", php.For("about.php"))
	// Traditional URL with .php
	mux.Handle("/about.php", php.For("about.php"))

	// Root maps to index.php
	mux.Handle("/", php.For("index.php"))

	// --- Register Go Handlers ---
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `"}`))
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
	log.Printf("Basic Example server starting on port %s", *port)
	log.Printf("Using web directory: %s", php.SourceDir()) // Use getter if available, or access field if needed/public
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
