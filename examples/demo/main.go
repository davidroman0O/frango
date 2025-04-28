package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davidroman0O/frango"
)

// Go handler for file uploads
func uploadToGoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (e.g., 32MB max memory)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("userfile") // Match the form input name
	if err != nil {
		http.Error(w, "Failed to get file 'userfile': "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	description := r.FormValue("description")

	// In a real app, you'd save or process the file. Here, we just report info.
	log.Printf("Go Handler: Received file '%s' (%d bytes), Description: '%s'", header.Filename, header.Size, description)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "File successfully received by Go handler",
		"filename":    header.Filename,
		"size":        header.Size,
		"contentType": header.Header.Get("Content-Type"),
		"description": description,
	})
}

// Go handler providing data for PHP rendering
func renderDataFunc(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"message":   "Hello from Go!",
		"timestamp": time.Now().Format(time.RFC3339),
		"requestIP": r.RemoteAddr,
	}
}

// Simple Go API handler
func apiTimeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"serverTime": time.Now().Format(time.RFC3339),
		"status":     "ok",
	})
}

func main() {
	// Setup logger
	logger := log.New(os.Stdout, "[frango-demo] ", log.LstdFlags)

	// Initialize Frango middleware
	middleware, err := frango.New(
		frango.WithDevelopmentMode(true), // Enable dev mode (watching, no cache)
		frango.WithAutoReload(true),      // Enable auto-reload (default SSE)
		frango.WithDisplayErrors(true),   // Show PHP errors in dev
		frango.WithLogger(logger),
		// frango.WithDevServer(frango.DevServerConfig{Port: 8081}), // Optional: Run dev server on separate port
	)
	if err != nil {
		logger.Fatalf("Failed to initialize Frango middleware: %v", err)
	}
	defer middleware.Shutdown()

	// Add the PHP source directory to the VFS
	// Assumes main.go is in v2/examples/demo and php files are in v2/examples/demo/php
	err = middleware.AddSourceDirectory("./php", "/")
	if err != nil {
		logger.Fatalf("Failed to add PHP source directory: %v", err)
	}

	// --- Setup HTTP routes ---
	mux := http.NewServeMux()

	// Dashboard
	mux.Handle("/", middleware.For("/index.php"))

	// Checklist Routes
	mux.Handle("/checklist/superglobals", middleware.For("/checklist/01_superglobals.php"))
	mux.Handle("/checklist/server", middleware.For("/checklist/02_server_vars.php"))
	mux.Handle("/checklist/php_input", middleware.For("/checklist/03_php_input.php"))
	mux.Handle("/checklist/sessions", middleware.For("/checklist/04_sessions.php"))
	mux.Handle("/checklist/paths", middleware.For("/checklist/05_paths_includes.php"))
	mux.Handle("/checklist/env", middleware.For("/checklist/06_env_vars.php"))

	// Routing Examples
	mux.Handle("/users/", middleware.For("/routing/user_profile.php"))          // Will match /users/{userID} pattern in PHP script path
	mux.Handle("/products/", middleware.For("/routing/product_detail.php"))     // Will match /products/{productID}
	mux.Handle("/categories/", middleware.For("/routing/category_page.php"))    // Will match /categories/{category}/{subcategory}
	mux.Handle("/search", middleware.For("/routing/search_results.php"))        // Query params only
	mux.Handle("/segments/show/", middleware.For("/routing/path_segments.php")) // Catch-all for segments demo

	// Forms Examples
	mux.Handle("/forms", middleware.For("/forms/index.php"))
	mux.Handle("/forms/get", middleware.For("/forms/get_handler.php"))
	mux.Handle("/forms/post", middleware.For("/forms/post_handler.php"))
	mux.Handle("/forms/json", middleware.For("/forms/json_handler.php"))
	mux.Handle("/forms/upload-php", middleware.For("/forms/upload_handler_php.php"))
	mux.HandleFunc("/forms/upload-go", uploadToGoHandler) // Go handler

	// Integration Examples
	mux.Handle("/render", middleware.Render("/integration/render_from_go.php", renderDataFunc))
	mux.HandleFunc("/api/time", apiTimeHandler) // Go handler

	// Dev Examples
	mux.Handle("/debug-panel", middleware.For("/dev/debug_panel_test.php"))

	// Register auto-reload routes (e.g., /__frango/sse)
	middleware.RegisterReloadRoutes(mux)

	// --- Start Server ---
	port := "8080"
	addr := ":" + port
	logger.Printf("Starting Frango v2 Demo Server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}
