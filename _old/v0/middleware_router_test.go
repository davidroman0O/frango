package frango

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMiddlewareRouter tests the middleware router
func TestMiddlewareRouter(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-middleware-router-test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	createTestFile(t, tempDir, "index.php", "<?php echo 'Root Index'; ?>")
	createTestFile(t, tempDir, "about.php", "<?php echo 'About Page'; ?>")
	createTestFile(t, tempDir, "users/index.php", "<?php echo 'Users Index'; ?>")
	createTestFile(t, tempDir, "users/profile.php", "<?php echo 'User Profile'; ?>")

	// Create a mock handler to use as "next"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found - From Next Handler"))
	})

	// Initialize the PHP middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Error initializing Frango: %v", err)
	}
	defer php.Shutdown()

	// Create the middleware router
	router := NewMiddlewareRouter(php, nextHandler)

	// Add the source directory
	err = router.AddSourceDirectory(tempDir, "/")
	if err != nil {
		t.Fatalf("Error adding source directory: %v", err)
	}

	// Create test cases
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"Root Path", "/", http.StatusOK, "Root Index"},
		{"About Page", "/about", http.StatusOK, "About Page"},
		{"Users Directory", "/users", http.StatusOK, "Users Index"},
		{"User Profile", "/users/profile", http.StatusOK, "User Profile"},
		{"Non-existent Page", "/not-found", http.StatusNotFound, "Not Found - From Next Handler"},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			// Serve the request through the middleware router
			router.ServeHTTP(w, req)

			// Check the response
			resp := w.Result()
			defer resp.Body.Close()

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Error reading response body: %v", err)
			}

			// Check status code
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			// Check response body
			bodyStr := string(body)
			if bodyStr != tc.expectedBody {
				t.Errorf("Expected body: %q, got: %q", tc.expectedBody, bodyStr)
			}
		})
	}
}

// TestMiddlewareRouter_WithPrefix tests the middleware router with a URL prefix
func TestMiddlewareRouter_WithPrefix(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-middleware-router-prefix-test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	createTestFile(t, tempDir, "index.php", "<?php echo 'API Root'; ?>")
	createTestFile(t, tempDir, "users.php", "<?php echo 'API Users'; ?>")
	createTestFile(t, tempDir, "users/index.php", "<?php echo 'API Users Index'; ?>")

	// Create a mock handler to use as "next"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found - From Next Handler"))
	})

	// Initialize the PHP middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Error initializing Frango: %v", err)
	}
	defer php.Shutdown()

	// Create the middleware router
	router := NewMiddlewareRouter(php, nextHandler)

	// Add the source directory with a prefix
	err = router.AddSourceDirectory(tempDir, "/api/v1")
	if err != nil {
		t.Fatalf("Error adding source directory: %v", err)
	}

	// Create test cases
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"API Root", "/api/v1", http.StatusOK, "API Root"},
		{"API Users", "/api/v1/users", http.StatusOK, "API Users"},
		{"API Users Index", "/api/v1/users", http.StatusOK, "API Users"}, // This will match users.php, not users/index.php
		{"Non-existent API Path", "/api/v1/not-found", http.StatusNotFound, "Not Found - From Next Handler"},
		{"Outside API Path", "/not-found", http.StatusNotFound, "Not Found - From Next Handler"},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			// Serve the request through the middleware router
			router.ServeHTTP(w, req)

			// Check the response
			resp := w.Result()
			defer resp.Body.Close()

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Error reading response body: %v", err)
			}

			// Check status code
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			// Check response body
			bodyStr := string(body)
			if bodyStr != tc.expectedBody {
				t.Errorf("Expected body: %q, got: %q", tc.expectedBody, bodyStr)
			}
		})
	}
}

// TestMiddlewareRouter_WithPathParameters tests the middleware router with path parameters
func TestMiddlewareRouter_WithPathParameters(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-middleware-router-params-test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	createTestFile(t, tempDir, "users/profile.php", `<?php 
	echo 'User ID: ' . $_PATH['id'];
	echo "\nAll Path Parameters: " . json_encode($_PATH);
	?>`)

	// Create a mock handler to use as "next"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found - From Next Handler"))
	})

	// Initialize the PHP middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Error initializing Frango: %v", err)
	}
	defer php.Shutdown()

	// Create the middleware router
	router := NewMiddlewareRouter(php, nextHandler)

	// Add the source directory
	err = router.AddSourceDirectory(tempDir, "/")
	if err != nil {
		t.Fatalf("Error adding source directory: %v", err)
	}

	// Add parameterized route
	err = router.AddRoute("/users/{id}", "/users/profile.php")
	if err != nil {
		t.Fatalf("Error adding parameterized route: %v", err)
	}

	// Test with a parameter
	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	// Serve the request through the middleware router
	router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check response body contains the parameter value
	bodyStr := string(body)
	expected := "User ID: 123"
	if !strings.Contains(bodyStr, expected) {
		t.Errorf("Expected body to contain: %q, got: %q", expected, bodyStr)
	}

	// Validate JSON path parameters
	if !strings.Contains(bodyStr, `{"id":"123"}`) {
		t.Errorf("Expected body to contain JSON path parameters, got: %q", bodyStr)
	}
}

// Helper function to create a test file
func createTestFile(t *testing.T, dir, path, content string) {
	fullPath := filepath.Join(dir, path)

	// Create parent directories if needed
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("Error creating directory %s: %v", parentDir, err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Error writing file %s: %v", fullPath, err)
	}
}
