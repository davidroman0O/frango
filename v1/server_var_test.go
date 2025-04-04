package frango

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServerFormVariables tests accessing form data from $_SERVER array
func TestServerFormVariables(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "frango-server-vars-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP script to access form data via $_SERVER
	phpFile := filepath.Join(tempDir, "server_vars.php")
	phpContent := `<?php
		header("Content-Type: text/plain");
		echo "Server Variables Test\n";
		echo "=====================\n";
		
		// Loop through all server vars and find form data
		foreach ($_SERVER as $key => $value) {
			if (strpos($key, 'PHP_FORM_') === 0) {
				$formKey = substr($key, 9); // Remove PHP_FORM_
				echo "Form field '$formKey' = '$value'\n";
			}
		}
	?>`

	if err := os.WriteFile(phpFile, []byte(phpContent), 0644); err != nil {
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

	// Create VFS
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add test file to VFS
	err = vfs.AddSourceFile(phpFile, "/server_vars.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create POST request with form data
	formData := "name=Jane+Smith&email=jane%40example.com&message=Hello+World"
	req := httptest.NewRequest("POST", "/server_vars.php", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Execute PHP script
	php.ExecutePHP("/server_vars.php", vfs, nil, w, req)

	// Get response
	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Log results
	t.Logf("Response: %s", bodyStr)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check that we can see the form data in $_SERVER variables
	expectedFields := []string{
		"Form field 'name' = 'Jane Smith'",
		"Form field 'email' = 'jane@example.com'",
		"Form field 'message' = 'Hello World'",
	}

	for _, expected := range expectedFields {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected form data: '%s' not found in response", expected)
		}
	}
}
