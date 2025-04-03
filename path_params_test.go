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

// TestPathParametersWithMiddlewareRouter tests that path parameters work correctly with middleware router
func TestPathParametersWithMiddlewareRouter(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-path-params-test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test PHP file
	phpContent := `<?php
// Output the path parameters from environment variables directly
echo "Path parameters from env: ";
$params = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'FRANGO_PARAM_') === 0) {
        $paramName = substr($key, strlen('FRANGO_PARAM_'));
        $params[$paramName] = $value;
    }
}
echo json_encode($params);

// Check for a specific parameter
if (isset($_SERVER['FRANGO_PARAM_id'])) {
    echo "\nID from env: " . $_SERVER['FRANGO_PARAM_id'];
}

// Try JSON params
if (isset($_SERVER['FRANGO_PATH_PARAMS_JSON'])) {
    echo "\nJSON params: " . $_SERVER['FRANGO_PATH_PARAMS_JSON'];
}

// Output $_PATH if available
echo "\n\nPHP $_PATH: ";
echo json_encode($_PATH ?? []);

// Output specific value
if (isset($_PATH['id'])) {
    echo "\nID from $_PATH: " . $_PATH['id'];
}
?>`

	// Create the users directory and profile.php file
	usersDir := filepath.Join(tempDir, "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatalf("Error creating users directory: %v", err)
	}

	profilePath := filepath.Join(usersDir, "profile.php")
	if err := os.WriteFile(profilePath, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Error writing PHP file: %v", err)
	}

	// Initialize the PHP middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Error initializing Frango: %v", err)
	}
	defer php.Shutdown()

	// Create a fallback handler
	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	// Create the middleware router
	router := NewMiddlewareRouter(php, fallbackHandler)

	// Add the source directory
	err = router.AddSourceDirectory(tempDir, "/")
	if err != nil {
		t.Fatalf("Error adding source directory: %v", err)
	}

	// Add the parameterized route
	err = router.AddRoute("/users/{id}", "/users/profile.php")
	if err != nil {
		t.Fatalf("Error adding parameterized route: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read the body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	body := string(bodyBytes)
	t.Logf("Response body: %s", body)

	// Validate JSON parameters
	if !strings.Contains(body, `"id":"123"`) {
		t.Errorf("Expected body to contain JSON path parameters with id=123, got: %q", body)
	}

	// Validate specific parameter
	if !strings.Contains(body, "ID from env: 123") {
		t.Errorf("Expected body to contain 'ID from env: 123', got: %q", body)
	}

	// Look for JSON params
	if !strings.Contains(body, "JSON params:") {
		t.Errorf("Expected body to contain JSON params, got: %q", body)
	}
}
