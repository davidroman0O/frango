//go:build nowatcher
// +build nowatcher

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/dunglas/frankenphp"
)

//go:embed pages/index.php
var indexFS embed.FS

//go:embed pages/demo.php
var demoFS embed.FS

//go:embed pages/dynamic.php
var dynamicFS embed.FS

//go:embed pages/stateful.php
var statefulFS embed.FS

// Global counter with mutex for concurrent access
var (
	apiCounter int
	counterMu  sync.Mutex
)

func extractFile(embedFS embed.FS, sourcePath string, targetDir string, targetFilename string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// Extract the file
	content, err := fs.ReadFile(embedFS, sourcePath)
	if err != nil {
		return err
	}

	// Write to target path
	targetPath := filepath.Join(targetDir, targetFilename)
	return os.WriteFile(targetPath, content, 0644)
}

func main() {
	// Port for the server
	port := "8082"

	// Create temporary directories for extracted files
	indexTempDir, err := os.MkdirTemp("", "php-index")
	if err != nil {
		log.Fatalf("Error creating temporary directory for index: %v", err)
	}
	defer os.RemoveAll(indexTempDir)

	demoTempDir, err := os.MkdirTemp("", "php-demo")
	if err != nil {
		log.Fatalf("Error creating temporary directory for demo: %v", err)
	}
	defer os.RemoveAll(demoTempDir)

	dynamicTempDir, err := os.MkdirTemp("", "php-dynamic")
	if err != nil {
		log.Fatalf("Error creating temporary directory for dynamic: %v", err)
	}
	defer os.RemoveAll(dynamicTempDir)

	statefulTempDir, err := os.MkdirTemp("", "php-stateful")
	if err != nil {
		log.Fatalf("Error creating temporary directory for stateful: %v", err)
	}
	defer os.RemoveAll(statefulTempDir)

	// Extract files to their respective temporary directories
	if err := extractFile(indexFS, "pages/index.php", indexTempDir, "index.php"); err != nil {
		log.Fatalf("Error extracting index.php: %v", err)
	}

	if err := extractFile(demoFS, "pages/demo.php", demoTempDir, "index.php"); err != nil {
		log.Fatalf("Error extracting demo.php: %v", err)
	}

	if err := extractFile(dynamicFS, "pages/dynamic.php", dynamicTempDir, "index.php"); err != nil {
		log.Fatalf("Error extracting dynamic.php: %v", err)
	}

	if err := extractFile(statefulFS, "pages/stateful.php", statefulTempDir, "index.php"); err != nil {
		log.Fatalf("Error extracting stateful.php: %v", err)
	}

	// Get absolute path to the temp directories
	absIndexTempDir, err := filepath.Abs(indexTempDir)
	if err != nil {
		log.Fatalf("Error getting absolute path for index: %v", err)
	}

	absDemoTempDir, err := filepath.Abs(demoTempDir)
	if err != nil {
		log.Fatalf("Error getting absolute path for demo: %v", err)
	}

	absDynamicTempDir, err := filepath.Abs(dynamicTempDir)
	if err != nil {
		log.Fatalf("Error getting absolute path for dynamic: %v", err)
	}

	absStatefulTempDir, err := filepath.Abs(statefulTempDir)
	if err != nil {
		log.Fatalf("Error getting absolute path for stateful: %v", err)
	}

	// Initialize FrankenPHP
	if err := frankenphp.Init(); err != nil {
		log.Fatalf("Error initializing FrankenPHP: %v", err)
	}
	defer frankenphp.Shutdown()

	// Setup handlers for each route

	// Home page route (/)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Path to index.php file
		indexFile := filepath.Join(absIndexTempDir, "index.php")
		log.Printf("Serving home page from: %s", indexFile)

		// Fixed script path for index
		r.URL.Path = "/index.php"

		// Set up environment variables for PHP execution
		env := map[string]string{
			"SCRIPT_FILENAME": indexFile,
			"SCRIPT_NAME":     "/index.php",
			"PHP_SELF":        "/index.php",
			"DOCUMENT_ROOT":   absIndexTempDir,
			"REQUEST_URI":     r.URL.RequestURI(),
			"REQUEST_METHOD":  r.Method,
			"QUERY_STRING":    r.URL.RawQuery,
			"HTTP_HOST":       r.Host,
		}

		// Create the PHP request
		req, err := frankenphp.NewRequestWithContext(
			r.Clone(r.Context()),
			frankenphp.WithRequestDocumentRoot(absIndexTempDir, false),
			frankenphp.WithRequestEnv(env),
		)
		if err != nil {
			log.Printf("ERROR creating PHP request for index: %v", err)
			http.Error(w, "Error creating PHP request", http.StatusInternalServerError)
			return
		}

		// Serve the PHP file
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			log.Printf("ERROR executing PHP for index: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully served index.php")
	})

	// Demo page route (/demo)
	http.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		// Path to demo index.php file
		demoFile := filepath.Join(absDemoTempDir, "index.php")
		log.Printf("Serving demo page from: %s", demoFile)

		// Fixed script path for demo
		r.URL.Path = "/index.php"

		// Set up environment variables for PHP execution
		env := map[string]string{
			"SCRIPT_FILENAME": demoFile,
			"SCRIPT_NAME":     "/index.php",
			"PHP_SELF":        "/index.php",
			"DOCUMENT_ROOT":   absDemoTempDir,
			"REQUEST_URI":     r.URL.RequestURI(),
			"REQUEST_METHOD":  r.Method,
			"QUERY_STRING":    r.URL.RawQuery,
			"HTTP_HOST":       r.Host,
		}

		// Create the PHP request
		req, err := frankenphp.NewRequestWithContext(
			r.Clone(r.Context()),
			frankenphp.WithRequestDocumentRoot(absDemoTempDir, false),
			frankenphp.WithRequestEnv(env),
		)
		if err != nil {
			log.Printf("ERROR creating PHP request for demo: %v", err)
			http.Error(w, "Error creating PHP request", http.StatusInternalServerError)
			return
		}

		// Serve the PHP file
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			log.Printf("ERROR executing PHP for demo: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully served demo.php")
	})

	// Also support /demo.php path
	http.HandleFunc("/demo.php", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/demo", http.StatusMovedPermanently)
	})

	// Dynamic page route (/dynamic)
	http.HandleFunc("/dynamic", func(w http.ResponseWriter, r *http.Request) {
		// Path to dynamic index.php file
		dynamicFile := filepath.Join(absDynamicTempDir, "index.php")
		log.Printf("Serving dynamic page from: %s", dynamicFile)

		// Fixed script path for dynamic
		r.URL.Path = "/index.php"

		// Set up environment variables for PHP execution
		env := map[string]string{
			"SCRIPT_FILENAME": dynamicFile,
			"SCRIPT_NAME":     "/index.php",
			"PHP_SELF":        "/index.php",
			"DOCUMENT_ROOT":   absDynamicTempDir,
			"REQUEST_URI":     r.URL.RequestURI(),
			"REQUEST_METHOD":  r.Method,
			"QUERY_STRING":    r.URL.RawQuery,
			"HTTP_HOST":       r.Host,
		}

		// Create the PHP request
		req, err := frankenphp.NewRequestWithContext(
			r.Clone(r.Context()),
			frankenphp.WithRequestDocumentRoot(absDynamicTempDir, false),
			frankenphp.WithRequestEnv(env),
		)
		if err != nil {
			log.Printf("ERROR creating PHP request for dynamic: %v", err)
			http.Error(w, "Error creating PHP request", http.StatusInternalServerError)
			return
		}

		// Serve the PHP file
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			log.Printf("ERROR executing PHP for dynamic: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully served dynamic.php")
	})

	// Also support /dynamic.php path
	http.HandleFunc("/dynamic.php", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dynamic"+r.URL.RawQuery, http.StatusMovedPermanently)
	})

	// Stateful page route (/stateful)
	// This example shows how to maintain state via Go
	var goVisitCount int
	http.HandleFunc("/stateful", func(w http.ResponseWriter, r *http.Request) {
		// Increment Go-side counter for this page
		goVisitCount++

		// Path to stateful index.php file
		statefulFile := filepath.Join(absStatefulTempDir, "index.php")
		log.Printf("Serving stateful page from: %s", statefulFile)

		// Fixed script path for stateful
		r.URL.Path = "/index.php"

		// Set up environment variables for PHP execution
		// Including custom Go variables that PHP can read
		env := map[string]string{
			"SCRIPT_FILENAME": statefulFile,
			"SCRIPT_NAME":     "/index.php",
			"PHP_SELF":        "/index.php",
			"DOCUMENT_ROOT":   absStatefulTempDir,
			"REQUEST_URI":     r.URL.RequestURI(),
			"REQUEST_METHOD":  r.Method,
			"QUERY_STRING":    r.URL.RawQuery,
			"HTTP_HOST":       r.Host,
			// Custom Go environment variables that PHP can access
			"GO_VISIT_COUNT":       fmt.Sprintf("%d", goVisitCount),
			"GO_SERVER_START_TIME": time.Now().Format(time.RFC3339),
			"GO_APP_VERSION":       "1.0.0",
			"GO_SERVER_PORT":       port,                          // Inject the server port
			"GO_API_COUNTER":       fmt.Sprintf("%d", apiCounter), // Current API counter value
		}

		// Create the PHP request
		req, err := frankenphp.NewRequestWithContext(
			r.Clone(r.Context()),
			frankenphp.WithRequestDocumentRoot(absStatefulTempDir, false),
			frankenphp.WithRequestEnv(env),
		)
		if err != nil {
			log.Printf("ERROR creating PHP request for stateful: %v", err)
			http.Error(w, "Error creating PHP request", http.StatusInternalServerError)
			return
		}

		// Serve the PHP file
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			log.Printf("ERROR executing PHP for stateful: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully served stateful.php (Go visit count: %d)", goVisitCount)
	})

	// Also support /stateful.php path
	http.HandleFunc("/stateful.php", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/stateful"+r.URL.RawQuery, http.StatusMovedPermanently)
	})

	// Add a REST API endpoint for the counter
	http.HandleFunc("/api/counter", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers to allow PHP to access this endpoint
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		counterMu.Lock()
		defer counterMu.Unlock()

		// POST: Increment counter
		if r.Method == "POST" {
			// Check if there's a value to add
			if r.URL.Query().Get("add") != "" {
				addValue, err := strconv.Atoi(r.URL.Query().Get("add"))
				if err == nil {
					apiCounter += addValue
				} else {
					apiCounter++
				}
			} else {
				apiCounter++
			}
		}

		// Return current counter value as JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"counter": apiCounter,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	fmt.Printf("Multi-page PHP server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
