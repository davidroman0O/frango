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

// TestHeadersRequest tests a PHP script accessing request headers
func TestHeadersRequest(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-headers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file that processes request headers
	headersPHP := filepath.Join(tempDir, "request_headers.php")
	phpContent := `<?php
		header("Content-Type: text/plain");
		
		// Check for standard request headers
		echo "Request Headers:\n";
		
		// Print entire $_SERVER array to help debug
		echo "SERVER Variables:\n";
		foreach ($_SERVER as $key => $value) {
			echo "$key: $value\n";
		}
		
		// Check for specific headers we're interested in
		echo "\nChecking for test headers:\n";
		
		// Look for X-Custom-Header in various possible formats
		$customHeaderFound = false;
		$testHeaderFound = false;
		$acceptHeaderFound = false;
		
		foreach ($_SERVER as $key => $value) {
			// Try to find various header naming conventions
			if (stripos($key, 'CUSTOM') !== false && $value == 'CustomValue') {
				echo "Found custom header as $key: $value\n";
				$customHeaderFound = true;
			}
			if (stripos($key, 'TEST') !== false && $value == 'TestValue') {
				echo "Found test header as $key: $value\n";
				$testHeaderFound = true;
			}
			if (stripos($key, 'ACCEPT') !== false && stripos($value, 'text/plain') !== false) {
				echo "Found accept header as $key: $value\n";
				$acceptHeaderFound = true;
			}
		}
		
		if ($customHeaderFound) echo "SUCCESS: X-Custom-Header was found\n";
		if ($testHeaderFound) echo "SUCCESS: X-Test-Header was found\n";
		if ($acceptHeaderFound) echo "SUCCESS: Accept header was found\n";
	?>`

	if err := os.WriteFile(headersPHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(headersPHP, "/request_headers.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request with custom headers
	req := httptest.NewRequest("GET", "/request_headers.php", nil)
	req.Header.Set("X-Custom-Header", "CustomValue")
	req.Header.Set("X-Test-Header", "TestValue")
	req.Header.Set("Accept", "text/plain")

	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/request_headers.php", vfs, nil, w, req)

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

	// Print the entire response for debugging
	t.Logf("Full response body: %s", bodyStr)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Look for success messages instead of exact header names
	successMessages := []string{
		"SUCCESS: X-Custom-Header was found",
		"SUCCESS: X-Test-Header was found",
		"SUCCESS: Accept header was found",
	}

	for _, message := range successMessages {
		if !strings.Contains(bodyStr, message) {
			t.Errorf("Missing expected header confirmation: %s", message)
		}
	}
}

// TestHeadersResponse tests a PHP script setting response headers
func TestHeadersResponse(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-headers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file that sets response headers
	headersPHP := filepath.Join(tempDir, "response_headers.php")
	phpContent := `<?php
		// Set custom response headers
		header("Content-Type: text/plain");
		header("X-Custom-Header: CustomValue");
		header("X-Another-Header: AnotherValue");
		echo "Response with custom headers";
	?>`

	if err := os.WriteFile(headersPHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(headersPHP, "/response_headers.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/response_headers.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/response_headers.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type '%s', got '%s'", "text/plain", contentType)
	}

	// Check custom response headers
	expectedHeaders := map[string]string{
		"X-Custom-Header":  "CustomValue",
		"X-Another-Header": "AnotherValue",
	}

	for name, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(name)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s to be '%s', got '%s'", name, expectedValue, actualValue)
		}
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
	expectedContent := "Response with custom headers"
	if !strings.Contains(bodyStr, expectedContent) {
		t.Errorf("Response does not contain expected content: %s", expectedContent)
	}
}
