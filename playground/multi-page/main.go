//go:build nowatcher
// +build nowatcher

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dunglas/frankenphp"
)

//go:embed pages/index.php
var indexFS embed.FS

//go:embed pages/demo.php
var demoFS embed.FS

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

	// Extract files to their respective temporary directories
	if err := extractFile(indexFS, "pages/index.php", indexTempDir, "index.php"); err != nil {
		log.Fatalf("Error extracting index.php: %v", err)
	}

	if err := extractFile(demoFS, "pages/demo.php", demoTempDir, "index.php"); err != nil {
		log.Fatalf("Error extracting demo.php: %v", err)
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

	port := "8082"
	fmt.Printf("Multi-page PHP server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
