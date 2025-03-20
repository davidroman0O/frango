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

	gophp "github.com/davidroman0O/go-php"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Find the web directory using library's built-in function
	webDir, err := gophp.ResolveDirectory("examples/middleware/web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}

	// Also find the static directory
	staticDir, err := gophp.ResolveDirectory("examples/middleware/static")
	if err != nil {
		log.Fatalf("Error finding static directory: %v", err)
	}

	log.Printf("Using web directory: %s", webDir)
	log.Printf("Using static directory: %s", staticDir)

	// Setup options with the source directory
	options := gophp.StaticHandlerOptions(webDir)
	options.DevelopmentMode = !*prodMode

	// Create server instance
	server, err := gophp.NewServer(options)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Initialize server
	if err := server.Initialize(); err != nil {
		log.Fatalf("Error initializing server: %v", err)
	}

	// Create a standard HTTP mux
	mux := http.NewServeMux()

	// Add Go handlers
	mux.HandleFunc("/go/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>Hello from Go handler!</h1>")
	})

	mux.HandleFunc("/go/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"time": "%s", "source": "go"}`, time.Now().Format(time.RFC3339))
	})

	// Register PHP endpoints directly with the server
	server.RegisterEndpoint("/api/user", "api/user.php")
	server.RegisterEndpoint("/api/items", "api/items.php")

	// Handle PHP content under /php/ path
	mux.Handle("/php/", http.StripPrefix("/php", server))

	// Create static directory if it doesn't exist
	os.MkdirAll(staticDir, 0755)

	// Create sample static files for testing
	createSampleStaticFiles(staticDir)

	// Static file handling - make sure this comes before API middleware
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// For API paths, check PHP first then fall back to Go
	apiHandler := server.AsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler is called when no PHP file handles the request
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "Endpoint not found", "path": "%s", "method": "%s"}`,
			r.URL.Path, r.Method)
	}))
	mux.Handle("/api/", apiHandler)

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Shutdown()
		os.Exit(0)
	}()

	// Start server
	log.Printf("Middleware example server starting on port %s", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// createSampleStaticFiles creates sample static files for testing
func createSampleStaticFiles(staticDir string) {
	// Create a sample CSS file
	cssContent := `
body {
    font-family: Arial, sans-serif;
    background-color: #f0f0f0;
    color: #333;
}
.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}
`
	os.WriteFile(staticDir+"/style.css", []byte(cssContent), 0644)

	// Create a sample text file as image placeholder
	imgPlaceholder := "This is a placeholder for an image file."
	os.WriteFile(staticDir+"/image.jpg", []byte(imgPlaceholder), 0644)
}
