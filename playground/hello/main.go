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

//go:embed hello.php
var contentFS embed.FS

func extractFiles(embedFS embed.FS, targetDir string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// Extract the hello.php file
	content, err := fs.ReadFile(embedFS, "hello.php")
	if err != nil {
		return err
	}

	// Write hello.php to index.php to avoid directory inclusion issues
	targetPath := filepath.Join(targetDir, "index.php")
	return os.WriteFile(targetPath, content, 0644)
}

func main() {
	// Create a temporary directory for extracted files
	tempDir, err := os.MkdirTemp("", "php-hello")
	if err != nil {
		log.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract files to the temporary directory
	if err := extractFiles(contentFS, tempDir); err != nil {
		log.Fatalf("Error extracting files: %v", err)
	}

	// Get absolute path to the temp directory
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	// Path to index.php file
	indexFile := filepath.Join(absTempDir, "index.php")
	log.Printf("PHP index file at: %s", indexFile)

	// Initialize FrankenPHP
	if err := frankenphp.Init(); err != nil {
		log.Fatalf("Error initializing FrankenPHP: %v", err)
	}
	defer frankenphp.Shutdown()

	// Handle all requests with the PHP file
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Fixed script path
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
			http.Error(w, "PHP execution error", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully served PHP")
	})

	port := "8082"
	fmt.Printf("Hello PHP embedded server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
