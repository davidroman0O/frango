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
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Find the web directory using library's built-in function
	webDir, err := frango.ResolveDirectory("examples/middleware/web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}

	// Also find the static directory
	staticDir, err := frango.ResolveDirectory("examples/middleware/static")
	if err != nil {
		log.Fatalf("Error finding static directory: %v", err)
	}

	log.Printf("Using web directory: %s", webDir)
	log.Printf("Using static directory: %s", staticDir)

	// Create PHP middleware instance
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

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

	// Register PHP endpoints (these will be accessible via the wrapped handlers)
	php.HandlePHP("/api/user", "api/user.php")
	php.HandlePHP("/api/items", "api/items.php")

	// Handle PHP content under /php/ path
	mux.Handle("/php/", http.StripPrefix("/php", php))

	// Create static directory if it doesn't exist
	os.MkdirAll(staticDir, 0755)

	// Create sample static files for testing
	createSampleStaticFiles(staticDir)

	// Static file handling
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// For API paths, check PHP first then fall back to Go
	// Create a fallback handler for API paths
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler is called when no PHP file handles the request
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "Endpoint not found", "path": "%s", "method": "%s"}`,
			r.URL.Path, r.Method)
	})

	// Wrap the fallback handler with PHP middleware for /api paths
	mux.Handle("/api/", php.Wrap(apiHandler))

	// For root path, handle with PHP middleware first, then fall back to a welcome page
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Welcome to Frango Middleware Example</h1>"+
			"<p>Try these paths:</p>"+
			"<ul>"+
			"<li><a href='/php/index.php'>PHP Index</a></li>"+
			"<li><a href='/api/user'>API User (PHP)</a></li>"+
			"<li><a href='/go/hello'>Go Handler</a></li>"+
			"<li><a href='/static/style.css'>Static File</a></li>"+
			"</ul>")
	})

	mux.Handle("/", php.Wrap(rootHandler))

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start server
	log.Printf("Middleware example running on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
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
