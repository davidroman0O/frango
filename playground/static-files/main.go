//go:build nowatcher
// +build nowatcher

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunglas/frankenphp"
)

func main() {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}

	// Try to find the www directory
	var wwwDir string
	possiblePaths := []string{
		filepath.Join(cwd, "www"),                               // ./www
		filepath.Join(cwd, "playground", "static-files", "www"), // ./playground/static-files/www
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			wwwDir = path
			break
		}
	}

	if wwwDir == "" {
		log.Fatalf("Cannot find www directory. Tried: %v", possiblePaths)
	}

	// Get absolute path to make sure PHP has the full path
	wwwDir, err = filepath.Abs(wwwDir)
	if err != nil {
		log.Fatalf("Error getting absolute path for www directory: %v", err)
	}

	log.Printf("Finding PHP files from: %s", wwwDir)

	// Create a temporary directory for PHP files
	tempDir, err := os.MkdirTemp("", "php-temp")
	if err != nil {
		log.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("Created temporary directory: %s", tempDir)

	// Initialize FrankenPHP
	if err := frankenphp.Init(); err != nil {
		log.Fatalf("Error initializing FrankenPHP: %v", err)
	}
	defer frankenphp.Shutdown()

	// Handle all requests by copying files from www to temp and serving them
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get the requested path
		requestPath := r.URL.Path

		// Default to index.php if root path
		if requestPath == "/" {
			requestPath = "/index.php"
		}

		// Get the physical file path from www directory
		sourcePath := filepath.Join(wwwDir, strings.TrimPrefix(requestPath, "/"))
		log.Printf("Requested file: %s", sourcePath)

		// Handle directory requests
		sourceInfo, err := os.Stat(sourcePath)
		if err == nil && sourceInfo.IsDir() {
			sourcePath = filepath.Join(sourcePath, "index.php")
			log.Printf("Directory detected, using index.php: %s", sourcePath)
		}

		// Handle paths without .php extension
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) && filepath.Ext(sourcePath) == "" {
			phpPath := sourcePath + ".php"
			if _, err := os.Stat(phpPath); err == nil {
				sourcePath = phpPath
				log.Printf("Adding .php extension: %s", sourcePath)
			}
		}

		// Check if the file exists and is not a directory
		sourceInfo, err = os.Stat(sourcePath)
		if os.IsNotExist(err) {
			log.Printf("File not found: %s", sourcePath)
			http.NotFound(w, r)
			return
		}
		if err == nil && sourceInfo.IsDir() {
			log.Printf("Cannot serve a directory: %s", sourcePath)
			http.NotFound(w, r)
			return
		}

		// Non-PHP files get served directly
		if !strings.HasSuffix(sourcePath, ".php") {
			log.Printf("Serving static file: %s", sourcePath)
			http.ServeFile(w, r, sourcePath)
			return
		}

		// For PHP files, copy to temp directory and serve (like the embed example)
		content, err := ioutil.ReadFile(sourcePath)
		if err != nil {
			log.Printf("Error reading file %s: %v", sourcePath, err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Write to temp as index.php (just like embed example)
		tempFilePath := filepath.Join(tempDir, "index.php")
		if err := ioutil.WriteFile(tempFilePath, content, 0644); err != nil {
			log.Printf("Error writing to temp file: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Copied %s to %s", sourcePath, tempFilePath)

		// Get absolute path for PHP
		absTempDir, err := filepath.Abs(tempDir)
		if err != nil {
			log.Printf("Error getting absolute temp dir path: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Using same approach as the working hello example
		indexFile := filepath.Join(absTempDir, "index.php")
		log.Printf("Serving PHP file: %s", indexFile)

		// Fixed script path for this request
		r.URL.Path = "/index.php"

		// Set up environment variables for PHP execution
		env := map[string]string{
			"SCRIPT_FILENAME": indexFile,
			"SCRIPT_NAME":     "/index.php",
			"PHP_SELF":        "/index.php",
			"DOCUMENT_ROOT":   absTempDir,
			"REQUEST_URI":     r.URL.RequestURI(),
			"REQUEST_METHOD":  r.Method,
			"QUERY_STRING":    r.URL.RawQuery,
			"HTTP_HOST":       r.Host,
			// Add original path for reference
			"ORIGINAL_PATH": requestPath,
		}

		// Create the PHP request
		req, err := frankenphp.NewRequestWithContext(
			r.Clone(r.Context()),
			frankenphp.WithRequestDocumentRoot(absTempDir, false),
			frankenphp.WithRequestEnv(env),
		)
		if err != nil {
			log.Printf("ERROR creating PHP request: %v", err)
			http.Error(w, "Error creating PHP request", http.StatusInternalServerError)
			return
		}

		// Serve the PHP file
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			log.Printf("ERROR executing PHP: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully served: %s", requestPath)
	})

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Default port
	}

	fmt.Printf("Static Files PHP server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
