package frango

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBasicPOSTForm tests a minimal POST form to diagnose issues
func TestBasicPOSTForm(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "frango-basic-post-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP script for POST debugging
	postPHP := filepath.Join(tempDir, "post_debug.php")
	phpContent := `<?php
		header("Content-Type: text/plain");
		echo "POST debug:\n";
		echo "Method: " . $_SERVER["REQUEST_METHOD"] . "\n";
		echo "Content-Type: " . $_SERVER["CONTENT_TYPE"] . "\n";
		
		// Print raw input
		$raw = file_get_contents("php://input");
		echo "Raw input length: " . strlen($raw) . "\n";
		echo "Raw input: " . $raw . "\n";
		
		// Print POST vars 
		echo "POST vars:\n";
		foreach ($_POST as $key => $value) {
			echo "$key: $value\n";
		}
		
		// Print form data from $_SERVER
		echo "Form data from \$_SERVER variables:\n";
		$formCount = 0;
		foreach ($_SERVER as $key => $value) {
			if (strpos($key, 'PHP_FORM_') === 0) {
				$formKey = substr($key, 9); // Remove PHP_FORM_ prefix
				echo "  $formKey: $value\n";
				$formCount++;
			}
		}
		if ($formCount === 0) {
			echo "  <no form data found>\n";
		}
	?>`

	if err := os.WriteFile(postPHP, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to create PHP file: %v", err)
	}

	// Setup frango
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Add to VFS
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	err = vfs.AddSourceFile(postPHP, "/post_debug.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Test with direct POST body (no url.Values)
	postBody := "name=John&age=30"

	// Create request
	req := httptest.NewRequest("POST", "/post_debug.php", strings.NewReader(postBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(postBody))

	w := httptest.NewRecorder()

	// Execute PHP
	php.ExecutePHP("/post_debug.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)
	t.Logf("Response: %s", bodyStr)

	// Basic checks
	AssertNoPHPErrors(t, bodyStr)

	// Check that we can see the form data in $_SERVER vars
	// Note: raw input is intentionally empty in the current FrankenPHP implementation
	expectedServerVars := []string{
		"Form data from $_SERVER variables:",
		"  name: John",
		"  age: 30",
	}

	for _, expected := range expectedServerVars {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}
