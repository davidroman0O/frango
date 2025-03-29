package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Find the web directory
	webDir, err := frango.ResolveDirectory("web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}
	log.Printf("Using web directory: %s", webDir)

	// Create the PHP middleware with development mode enabled
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Register some PHP endpoints
	php.HandlePHP("/api/users", "api/users.php")
	php.HandlePHP("/api/items", "api/items.php")

	// Handle a directory of PHP files
	if err := php.HandleDir("/pages", "pages"); err != nil {
		log.Printf("Warning: Could not register pages directory: %v", err)
	}

	// ===== Standard net/http usage =====
	// This shows how to use the middleware with standard Go net/http

	// Create a standard HTTP mux
	mux := http.NewServeMux()

	// Add native Go endpoints
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"time": "%s", "source": "go"}`, time.Now().Format(time.RFC3339))
	})

	// Use PHP middleware for PHP paths, allowing it to handle only PHP requests
	mux.Handle("/api/", php.Wrap(http.NotFoundHandler()))

	// Alternatively, mount the PHP middleware directly for a dedicated path
	mux.Handle("/php/", http.StripPrefix("/php", php))

	// For the root path, check PHP first, then fall back to a welcome page
	mux.Handle("/", php.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "Welcome to frango middleware example!")
	})))

	// ===== Example for Chi router =====
	// Note: This is mock code to show the pattern, not functional without the Chi import
	/*
		r := chi.NewRouter()

		// Mount PHP middleware at a specific path
		r.Mount("/php", php)

		// Use the middleware on specific routes
		r.Group(func(r chi.Router) {
			// All routes in this group will try PHP first
			r.Use(php.ForChi())

			r.Get("/api/data", apiHandler)
		})

		go http.ListenAndServe(":8083", r)
	*/

	// ===== Example for Gin =====
	// Note: This is mock code to show the pattern, not functional without the Gin import
	/*
		g := gin.New()
		g.Use(gin.Logger())
		g.Use(gin.Recovery())

		// Use PHP middleware on a group of routes
		apiGroup := g.Group("/api")
		apiGroup.Use(func(c *gin.Context) {
			// Check if PHP should handle this request
			req := c.Request
			if php.shouldHandlePHP(req) {
				php.ServeHTTP(c.Writer, req)
				c.Abort() // Stop further Gin handling
				return
			}
			c.Next() // Continue with Gin pipeline
		})

		go g.Run(":8084")
	*/

	// ===== Example for Echo =====
	// Note: This is mock code to show the pattern, not functional without the Echo import
	/*
		e := echo.New()

		// Add middleware that delegates to PHP middleware
		e.Use(php.ForEcho())

		go e.Start(":8085")
	*/

	// Setup graceful shutdown
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
	}()

	// Start the standard server
	log.Printf("Middleware example running on port 8082")
	log.Printf("Open http://localhost:8082/ in your browser")
	go func() {
		if err := http.ListenAndServe(":8082", mux); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown
	wg.Wait()
}
