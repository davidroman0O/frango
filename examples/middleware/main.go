package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Define web and static dirs
	webDir := "examples/middleware/web"
	staticDir := "examples/middleware/static"

	// Create Frango instance
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	absWebDir, _ := filepath.Abs(webDir)
	absStaticDir, _ := filepath.Abs(staticDir)
	log.Printf("Using web directory: %s", absWebDir)
	log.Printf("Using static directory: %s", absStaticDir)

	// Create a standard HTTP mux
	mux := http.NewServeMux()

	// --- Register Specific PHP Handlers ---
	// Use exact patterns including method where appropriate
	mux.Handle("GET /", php.For("index.php"))
	mux.Handle("GET /info", php.For("info.php"))

	mux.HandleFunc("GET /api/", http.NotFound) // Catches /api/nonexistent

	// Register specific methods for API endpoints to avoid conflict with GET /
	mux.Handle("GET /api/user", php.For("api/user.php"))
	mux.Handle("POST /api/user", php.For("api/user.php")) // Example: Allow POST too
	mux.Handle("GET /api/items", php.For("api/items.php"))
	mux.Handle("POST /api/items", php.For("api/items.php")) // Example: Allow POST too

	// --- Register Go Handlers ---
	// Explicitly register methods to avoid conflict with "GET /"
	mux.HandleFunc("GET /go/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>Hello from Go handler!</h1>")
	})
	mux.HandleFunc("GET /go/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"time": "%s", "source": "go"}`, time.Now().Format(time.RFC3339))
	})

	// --- Static File Handling ---
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		os.MkdirAll(staticDir, 0755)
		createSampleStaticFiles(staticDir)
	} else {
		log.Println("Static directory already exists.")
	}
	fileServer := http.FileServer(http.Dir(staticDir))
	// Register static handler specifically for GET requests
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// --- Ensure 404 for Non-Existent Paths ---
	// Create a wrapper handler to correctly handle 404s
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Special case for /php/ to always return 404 as an example
		if r.URL.Path == "/php/" {
			log.Printf("Specifically blocking path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Let the main mux handle the request
		mux.ServeHTTP(w, r)
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

	// Start server
	log.Printf("Middleware Example running on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, finalHandler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// createSampleStaticFiles creates sample static files for testing
func createSampleStaticFiles(staticDir string) {
	// Create a sample CSS file
	cssContent := `body { font-family: sans-serif; }`
	os.WriteFile(filepath.Join(staticDir, "style.css"), []byte(cssContent), 0644)
}
