package frango

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRequestPlainText tests a simple plain text response from PHP
func TestRequestPlainText(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-basic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP file
	simplePHP := filepath.Join(tempDir, "simple.php")
	phpContent := `<?php
		// Simple plain text response
		header("Content-Type: text/plain");
		echo "This is a plain text response";
	?>`

	if err := os.WriteFile(simplePHP, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to create PHP file: %v", err)
	}

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add test file to VFS
	err = vfs.AddSourceFile(simplePHP, "/simple.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/simple.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/simple.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "text/plain", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check expected content
	if !strings.Contains(bodyStr, "This is a plain text response") {
		t.Errorf("Expected response to contain 'This is a plain text response', got: %s", bodyStr)
	}
}

// TestRequestQueryParameters tests a PHP script handling query parameters
func TestRequestQueryParameters(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-query-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file that processes query parameters
	queryPHP := filepath.Join(tempDir, "query_params.php")
	phpContent := `<?php
		header("Content-Type: text/plain");
		
		echo "Query Parameters:\n";
		
		// Loop through GET parameters
		foreach ($_GET as $key => $value) {
			echo "$key: $value\n";
		}
		
		// Check for specific parameters
		if (isset($_GET['id'])) {
			echo "ID detected: " . $_GET['id'] . "\n";
		}
		
		if (isset($_GET['name'])) {
			echo "Name detected: " . $_GET['name'] . "\n";
		}
		
		// Also check $_REQUEST
		if (count($_REQUEST) > 0) {
			echo "\nRequest Parameters:\n";
			foreach ($_REQUEST as $key => $value) {
				echo "REQUEST[$key]: $value\n";
			}
		}
	?>`

	if err := os.WriteFile(queryPHP, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to create PHP file: %v", err)
	}

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add test file to VFS
	err = vfs.AddSourceFile(queryPHP, "/query_params.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create query parameters
	queryParams := url.Values{}
	queryParams.Set("id", "123")
	queryParams.Set("name", "John Doe")
	queryParams.Set("action", "view")

	// Create request with query parameters
	requestURL := "/query_params.php?" + queryParams.Encode()
	req := httptest.NewRequest("GET", requestURL, nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/query_params.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "text/plain", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check that each query parameter is present in the response
	for key, values := range queryParams {
		value := values[0]
		if !strings.Contains(bodyStr, key+": "+value) {
			t.Errorf("Expected query parameter '%s' with value '%s' not found in response", key, value)
		}
	}

	// Check for special messages for id and name
	if !strings.Contains(bodyStr, "ID detected: 123") {
		t.Errorf("Expected 'ID detected: 123' not found in response")
	}

	if !strings.Contains(bodyStr, "Name detected: John Doe") {
		t.Errorf("Expected 'Name detected: John Doe' not found in response")
	}
}
