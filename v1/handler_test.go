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

// TestHttpBasicHandlers tests that HTTP handlers work for different PHP scripts
func TestHttpBasicHandlers(t *testing.T) {
	// Create test files
	sourceDir, cleanupFiles := createTestFiles(t)
	defer cleanupFiles()

	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, sourceDir, WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Create test handlers
	indexHandler := php.For("index.php")
	aboutHandler := php.For("about.php")
	usersHandler := php.For("users/index.php")

	// Test cases
	tests := []struct {
		name         string
		handler      http.Handler
		method       string
		path         string
		wantStatus   int
		wantContains string // Partial content match rather than exact match
	}{
		{
			name:         "Index page",
			handler:      indexHandler,
			method:       "GET",
			path:         "/",
			wantStatus:   http.StatusOK,
			wantContains: "Hello from index.php",
		},
		{
			name:         "About page",
			handler:      aboutHandler,
			method:       "GET",
			path:         "/about",
			wantStatus:   http.StatusOK,
			wantContains: "About page",
		},
		{
			name:         "Users index page",
			handler:      usersHandler,
			method:       "GET",
			path:         "/users",
			wantStatus:   http.StatusOK,
			wantContains: "Users index page",
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			// Execute request
			tt.handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatus)
			}

			// Check response body
			body, _ := io.ReadAll(rr.Body)
			bodyStr := string(body)

			// Check for PHP errors
			AssertNoPHPErrors(t, bodyStr)

			if !strings.Contains(bodyStr, tt.wantContains) {
				t.Errorf("Handler returned unexpected body: got %v, which doesn't contain %v",
					bodyStr, tt.wantContains)
			}
		})
	}
}

// TestHttpRenderHandler tests the template variable rendering functionality
func TestHttpRenderHandler(t *testing.T) {
	// Create test files
	sourceDir, cleanupFiles := createTestFiles(t)
	defer cleanupFiles()

	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, sourceDir, WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Update the index.php file to display render variables
	indexPath := filepath.Join(sourceDir, "index.php")
	indexContent := `<?php 
		echo "Hello from index.php\n";
		// Check for title variable
		if (isset($title)) {
			echo "Template Variable: title = " . $title . "\n";
		}
		// Check for user variable
		if (isset($user) && is_array($user)) {
			echo "Template Variable: user = " . json_encode($user) . "\n";
		}
		// Check if variables are available in $_TEMPLATE
		if (isset($_TEMPLATE) && is_array($_TEMPLATE)) {
			echo "Template Variables via $_TEMPLATE:\n";
			foreach ($_TEMPLATE as $key => $value) {
				echo "  $_TEMPLATE[$key] = " . json_encode($value) . "\n";
			}
		}
	?>`
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to update index.php: %v", err)
	}

	// Create render function
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"title": "Test Title",
			"user": map[string]interface{}{
				"name": "John Doe",
				"role": "Admin",
			},
		}
	}

	// Create render handler
	renderHandler := php.Render("index.php", renderFn)

	// Test request
	req := httptest.NewRequest("GET", "/render", nil)
	rr := httptest.NewRecorder()

	// Execute request
	renderHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check response body
	body, _ := io.ReadAll(rr.Body)
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Look for render variables
	expectedPhrases := []string{
		"Template Variable: title",
		"Template Variable: user",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(bodyStr, phrase) {
			t.Errorf("Handler did not include template variables: %s", bodyStr)
			break
		}
	}
}

// TestHttpPathParameters tests path parameter extraction in handlers
func TestHttpPathParameters(t *testing.T) {
	// Create test files
	sourceDir, cleanupFiles := createTestFiles(t)
	defer cleanupFiles()

	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, sourceDir, WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Update the users/index.php file to specifically check for userId
	usersPath := filepath.Join(sourceDir, "users", "index.php")
	usersContent := `<?php 
		echo "Users index page";
		if (isset($_PATH['userId'])) {
			echo " - User ID: " . $_PATH['userId'];
		}
	?>`
	// Create directory if needed
	os.MkdirAll(filepath.Dir(usersPath), 0755)
	if err := os.WriteFile(usersPath, []byte(usersContent), 0644); err != nil {
		t.Fatalf("Failed to update users/index.php: %v", err)
	}

	// Create a user profile script with the parameter in the filename
	userProfilePath := filepath.Join(sourceDir, "users", "{userId}.php")
	userProfileContent := `<?php 
		echo "User profile page";
		if (isset($_PATH['userId'])) {
			echo " - User ID: " . $_PATH['userId'];
		}
	?>`
	if err := os.WriteFile(userProfilePath, []byte(userProfileContent), 0644); err != nil {
		t.Fatalf("Failed to create users/{userId}.php: %v", err)
	}

	// Create handler for the new file that has the parameter in its path
	userHandler := php.For("users/{userId}.php")

	// Test request with a specific user ID
	req := httptest.NewRequest("GET", "/users/42", nil)
	rr := httptest.NewRecorder()

	// Execute request
	userHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check response body
	body, _ := io.ReadAll(rr.Body)
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Look for path parameter in the output
	expectedText := "User profile page - User ID: 42"
	if !strings.Contains(bodyStr, expectedText) {
		t.Errorf("Handler did not extract path parameter: %s", bodyStr)
	}
}
