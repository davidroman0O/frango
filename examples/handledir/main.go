package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Define the relative web directory path
	webDirRelative := "web"

	// Resolve the absolute path for the web directory
	// Note: We are essentially replicating resolveDirectory logic here,
	// perhaps frango.New should return the resolved path?
	absWebDir, err := filepath.Abs(webDirRelative)
	if err != nil {
		log.Fatalf("Error getting absolute path for %s: %v", webDirRelative, err)
	}
	// Check if it exists relative to CWD
	if _, statErr := os.Stat(absWebDir); os.IsNotExist(statErr) {
		// Try relative to caller (main.go)
		_, filename, _, ok := runtime.Caller(0) // Use Caller(0) for current file
		if ok {
			callerDir := filepath.Dir(filename)
			absWebDir = filepath.Join(callerDir, webDirRelative)
		} else {
			log.Fatalf("Could not resolve web directory: %s", webDirRelative)
		}
	}
	// Final check if resolved path exists
	if _, err := os.Stat(absWebDir); err != nil {
		log.Fatalf("Web directory not found at %s: %v", absWebDir, err)
	}
	log.Printf("Using web directory: %s", absWebDir)

	// Create Frango instance using the resolved absolute path
	php, err := frango.New(
		frango.WithSourceDir(absWebDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP mux
	mux := http.NewServeMux()

	// --- Use MapFileSystemRoutes to register directory contents ---

	// Register files in "pages" under "/pages", using the absolute path for DirFS
	pageRoutes, err := frango.MapFileSystemRoutes(php, os.DirFS(absWebDir), "pages", "/pages", nil)
	if err != nil {
		log.Fatalf("Error mapping pages directory: %v", err)
	}
	for _, route := range pageRoutes {
		muxPattern := route.Pattern
		if route.Method != "" {
			muxPattern = route.Method + " " + route.Pattern
		}
		mux.Handle(muxPattern, route.Handler)
	}

	// Register files in "api" under "/api", using the absolute path for DirFS
	apiRoutes, err := frango.MapFileSystemRoutes(php, os.DirFS(absWebDir), "api", "/api", nil)
	if err != nil {
		log.Fatalf("Error mapping API directory: %v", err)
	}
	for _, route := range apiRoutes {
		muxPattern := route.Pattern
		if route.Method != "" {
			muxPattern = route.Method + " " + route.Pattern
		}
		mux.Handle(muxPattern, route.Handler)
	}

	// Register the root handler explicitly if needed
	// Check if index.php exists in the absolute web dir
	if _, err := os.Stat(filepath.Join(absWebDir, "index.php")); err == nil {
		// HandlerFor needs path relative to SourceDir, which is absWebDir here.
		// So, just pass "index.php"
		mux.Handle("/", php.HandlerFor("/", "index.php"))
	} else {
		log.Println("Root index.php not found in web directory.")
	}

	// Optional: Add other Go handlers to the same mux
	// mux.HandleFunc("/go/route", myGoHandler)

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start standard Go HTTP server with the mux
	log.Printf("FileSystem Routes Example running on port %s", *port)
	log.Printf("Using web directory: %s", php.SourceDir())
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
