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
	"sync"
	"time"

	"github.com/dunglas/frankenphp"
)

// FileCache tracks information about cached files
type FileCache struct {
	SourcePath   string    // Original file path
	TempPath     string    // Path in temp directory
	LastModified time.Time // Last modified time
	LastSize     int64     // Last file size
	LastChecked  time.Time // Last time we checked for changes
	mutex        sync.Mutex
}

func main() {
	// Development mode flag - set to false for production/caching
	devMode := true
	if os.Getenv("PHP_PRODUCTION") == "1" {
		devMode = false
	}

	if devMode {
		log.Println("Running in DEVELOPMENT mode (file changes detected immediately, caching disabled)")
	} else {
		log.Println("Running in PRODUCTION mode (with caching enabled)")
	}

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

	// Create a temporary directory for PHP files (persistent during server lifetime)
	tempDir, err := os.MkdirTemp("", "php-mirror")
	if err != nil {
		log.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("Created mirror directory: %s", tempDir)

	// Cache to track file modifications
	fileCache := make(map[string]*FileCache)
	var cacheMutex sync.Mutex

	// Initialize FrankenPHP
	if err := frankenphp.Init(); err != nil {
		log.Fatalf("Error initializing FrankenPHP: %v", err)
	}
	defer frankenphp.Shutdown()

	// Handle all requests by mirroring files from www to temp and serving them
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

		// For PHP files, check the cache and update if needed
		cacheMutex.Lock()
		cacheKey := sourcePath
		fileEntry, exists := fileCache[cacheKey]

		if !exists {
			// Create a new cache entry for this file
			// Create a unique subdirectory for this file
			relativePath := strings.TrimSuffix(strings.TrimPrefix(requestPath, "/"), ".php")
			if relativePath == "" {
				relativePath = "index" // For the root path
			}

			// Create a dedicated directory for this PHP file
			tempDirPath := filepath.Join(tempDir, relativePath)
			if err := os.MkdirAll(tempDirPath, 0755); err != nil {
				log.Printf("Error creating directory structure: %v", err)
				cacheMutex.Unlock()
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}

			// Always use index.php in that directory
			tempFilePath := filepath.Join(tempDirPath, "index.php")

			fileEntry = &FileCache{
				SourcePath: sourcePath,
				TempPath:   tempFilePath,
			}
			fileCache[cacheKey] = fileEntry

			log.Printf("Created new cache entry for %s at %s", sourcePath, tempFilePath)
		}
		cacheMutex.Unlock()

		// Lock just this file's entry
		fileEntry.mutex.Lock()
		defer fileEntry.mutex.Unlock()

		// Check if file has been modified - use size and mod time
		currentModTime := sourceInfo.ModTime()
		currentSize := sourceInfo.Size()

		// Check both modification time and file size for changes
		var needsUpdate bool
		if devMode {
			// In dev mode, always check for changes
			needsUpdate = !exists ||
				currentModTime.After(fileEntry.LastModified) ||
				fileEntry.LastSize != currentSize
		} else {
			// In production mode, only check for changes every 5 seconds
			needsUpdate = !exists ||
				time.Since(fileEntry.LastChecked) > 5*time.Second && (currentModTime.After(fileEntry.LastModified) ||
					fileEntry.LastSize != currentSize)
		}

		// Update the LastChecked time
		fileEntry.LastChecked = time.Now()

		if needsUpdate {
			log.Printf("File changed, updating mirror: %s (Size: %d→%d, Mod: %s→%s)",
				sourcePath,
				fileEntry.LastSize,
				currentSize,
				fileEntry.LastModified.Format("15:04:05.000"),
				currentModTime.Format("15:04:05.000"))

			// Read the updated file content
			content, err := ioutil.ReadFile(sourcePath)
			if err != nil {
				log.Printf("Error reading file %s: %v", sourcePath, err)
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}

			// Write to the mirrored location
			if err := ioutil.WriteFile(fileEntry.TempPath, content, 0644); err != nil {
				log.Printf("Error writing to mirrored file: %v", err)
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}

			// Update last modified time and size
			fileEntry.LastModified = currentModTime
			fileEntry.LastSize = currentSize
			log.Printf("Updated mirrored file: %s -> %s", sourcePath, fileEntry.TempPath)
		} else {
			log.Printf("Serving from mirror (unchanged): %s", fileEntry.TempPath)
		}

		// Get absolute paths for PHP execution
		absFilePath := fileEntry.TempPath
		absTempDir := filepath.Dir(absFilePath)

		// Always set r.URL.Path to /index.php
		r.URL.Path = "/index.php"

		// Set cache control headers based on mode
		if devMode {
			// Development mode: no caching
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else {
			// Production mode: allow some caching
			w.Header().Set("Cache-Control", "public, max-age=60") // Cache for 60 seconds
		}

		// Environment variables based on mode
		cacheEnv := map[string]string{}
		if devMode {
			cacheEnv = map[string]string{
				"PHP_FCGI_MAX_REQUESTS": "1", // Force a new process for each request
				"PHP_OPCACHE_ENABLE":    "0", // Disable opcode cache
			}
		} else {
			cacheEnv = map[string]string{
				"PHP_OPCACHE_ENABLE": "1", // Enable opcode cache
			}
		}

		// Set up environment variables for PHP execution - using the hello approach
		env := map[string]string{
			"SCRIPT_FILENAME": absFilePath,
			"SCRIPT_NAME":     "/" + filepath.Base(absFilePath),
			"PHP_SELF":        "/" + filepath.Base(absFilePath),
			"DOCUMENT_ROOT":   absTempDir,
			"REQUEST_URI":     r.URL.RequestURI(),
			"REQUEST_METHOD":  r.Method,
			"QUERY_STRING":    r.URL.RawQuery,
			"HTTP_HOST":       r.Host,
			"ORIGINAL_PATH":   requestPath,
			"SOURCE_FILE":     sourcePath,
		}

		// Merge cache env with regular env
		for k, v := range cacheEnv {
			env[k] = v
		}

		// Create the PHP request with exactly the same parameters as the hello example
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

		// Set no-cache headers to prevent browser caching
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

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
