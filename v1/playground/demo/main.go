package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	frango "github.com/davidroman0O/frango/v1"
)

//go:embed index.php
//go:embed users/*.php
//go:embed products/*.php
//go:embed nested/deep/path/*.php
//go:embed categories/**/*.php
//go:embed forms/*.php
//go:embed debug_panel.php
var phpFiles embed.FS

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "HTTP port to listen on")
	sourceDir := flag.String("source", ".", "Source directory for PHP files")
	flag.Parse()

	// Create the PHP middleware
	php, err := frango.New(
		frango.WithSourceDir(*sourceDir),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a new router for PHP files
	mux := http.NewServeMux()

	// Path parameter examples - explicitly set the pattern
	mux.HandleFunc("/categories/{category}/{subcategory}", func(w http.ResponseWriter, r *http.Request) {
		// Set the pattern on the request for parameter extraction
		r.Pattern = "/categories/{category}/{subcategory}"

		// Serve the PHP file
		handler := php.For("categories/{category}/{subcategory}.php")
		handler.ServeHTTP(w, r)
	})

	// Single category path parameter
	mux.HandleFunc("/categories/{category}", func(w http.ResponseWriter, r *http.Request) {
		r.Pattern = "/categories/{category}"
		handler := php.For("categories/{category}/index.php")
		handler.ServeHTTP(w, r)
	})

	// Add route for the debug panel
	mux.Handle("/debug_panel.php", http.StripPrefix("/", php.For("debug_panel.php")))

	// Regular static routes
	mux.Handle("/", http.StripPrefix("/", php.For("index.php")))
	mux.Handle("/categories", http.StripPrefix("/categories", php.For("categories/index.php")))

	// Static file handler (for CSS/JS assets)
	fileServer := http.FileServer(http.Dir(filepath.Join(*sourceDir, "assets")))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fileServer))

	// Form specific routes - explicit mappings for each form endpoint
	mux.Handle("/forms/form_display", php.For("/forms/form_display.php"))
	mux.Handle("/forms/post_display", php.For("/forms/post_display.php"))
	mux.Handle("/forms/get_display", php.For("/forms/get_display.php"))
	mux.Handle("/forms/json", php.For("/forms/json.php"))
	mux.Handle("/forms/php_receiver", php.For("/forms/upload_receiver.php"))

	// New PHP uploader examples
	mux.Handle("/forms/upload_to_php", php.For("/forms/upload_to_php.php"))
	mux.Handle("/forms/upload_to_go", php.For("/forms/upload_to_go.php"))

	// For form submissions using the hyphenated convention
	mux.Handle("/forms/post_test", php.For("/forms/post_test.php"))
	mux.Handle("/forms/get_test", php.For("/forms/get_test.php"))
	mux.Handle("/forms/upload_test", php.For("/forms/upload_test.php"))
	mux.Handle("/forms/json_test", php.For("/forms/json_test.php"))
	mux.Handle("/forms/test_index", php.For("/forms/test_index.php"))

	// Debug pages
	mux.Handle("/forms/debug", php.For("/forms/debug.php"))

	// Go file upload handler - this handles uploads directly in Go
	mux.HandleFunc("/upload", uploadHandler)

	// Default routes
	mux.Handle("/forms", php.For("/forms/index.php"))
	mux.Handle("/forms/", php.For("/forms/index.php"))
	mux.Handle("/users/", php.For("/users/{id}.php"))
	mux.Handle("/products/", php.For("/products/{id}.php"))
	mux.Handle("/nested", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/path", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/path/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/categories/", php.For("/categories/{category}/{subcategory}.php"))

	// Root route
	mux.Handle("/debug", php.For("/debug.php"))

	// API endpoints to demonstrate Go handlers for form data
	mux.HandleFunc("/api/get", func(w http.ResponseWriter, r *http.Request) {
		// Handle GET request
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"received": true,
			"method":   "GET",
			"params": map[string]string{
				"name":     r.URL.Query().Get("name"),
				"category": r.URL.Query().Get("category"),
				"limit":    r.URL.Query().Get("limit"),
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		// Handle POST request
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Log received values for debugging
		log.Printf("POST handler received: username=%q, email=%q, comment=%q",
			r.FormValue("username"), r.FormValue("email"), r.FormValue("comment"))

		// Log all form values received
		log.Printf("All form values: %v", r.Form)

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"received": true,
			"method":   "POST",
			"data": map[string]string{
				"username": r.FormValue("username"),
				"email":    r.FormValue("email"),
				"comment":  r.FormValue("comment"),
			},
			"all_form_data": r.Form,
			"timestamp":     time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/form", func(w http.ResponseWriter, r *http.Request) {
		// Handle both GET and POST requests
		var data map[string]string

		if r.Method == "GET" {
			// For GET requests, use query parameters
			data = map[string]string{
				"product":  r.URL.Query().Get("product"),
				"quantity": r.URL.Query().Get("quantity"),
				"notes":    r.URL.Query().Get("notes"),
			}
		} else {
			// For POST requests, parse form data
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Log received values for debugging
			log.Printf("FORM handler received: product=%q, quantity=%q, notes=%q",
				r.FormValue("product"), r.FormValue("quantity"), r.FormValue("notes"))

			// Log all form values received
			log.Printf("All form values: %v", r.Form)

			data = map[string]string{
				"product":  r.FormValue("product"),
				"quantity": r.FormValue("quantity"),
				"notes":    r.FormValue("notes"),
			}
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"received":      true,
			"method":        r.Method,
			"data":          data,
			"all_form_data": r.Form,
			"timestamp":     time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		// Handle JSON data
		var data map[string]interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"received":  true,
			"method":    "JSON",
			"data":      data,
			"timestamp": time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	})

	// Start HTTP server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting PHP playground server on http://localhost%s", addr)
	log.Printf("Using PHP source directory: %s", *sourceDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// uploadHandler processes file uploads directly in Go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form, 32 MB max memory
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read form fields
	description := r.FormValue("description")

	// Create temp directory if it doesn't exist
	tempDir := os.TempDir()
	uploadDir := filepath.Join(tempDir, "frango-uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Failed to create upload directory: %v", err)
	}

	// Create a unique filename
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%d_%s", timestamp, header.Filename)
	tempFilePath := filepath.Join(uploadDir, filename)

	// Save the file to disk (optional - we could just process in memory)
	tempFile, err := os.Create(tempFilePath)
	var fileSize int64

	if err == nil {
		defer tempFile.Close()
		fileSize, err = io.Copy(tempFile, file)
		if err != nil {
			log.Printf("Failed to save file: %v", err)
		}
	} else {
		log.Printf("Failed to create temp file: %v", err)
		fileSize = header.Size
	}

	// Get file details
	fileInfo := map[string]interface{}{
		"filename":     header.Filename,
		"size":         fileSize,
		"contentType":  header.Header.Get("Content-Type"),
		"description":  description,
		"uploadTime":   time.Now().Format(time.RFC3339),
		"tempLocation": tempFilePath,
	}

	// Log upload for debugging
	log.Printf("File uploaded: %s (%d bytes, %s)",
		header.Filename, fileSize, header.Header.Get("Content-Type"))

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success":  true,
		"message":  "File uploaded successfully to Go handler",
		"fileInfo": fileInfo,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
