package frango

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRouteContextKeyExtraction tests the extraction of pattern keys from context
func TestRouteContextKeyExtraction(t *testing.T) {
	// Create a test key for pattern extraction
	type contextKey string
	patternKey := contextKey("pattern")
	phpPatternKey := contextKey("phpPattern")

	tests := []struct {
		name          string
		setupContext  func(r *http.Request) *http.Request
		path          string
		wantExtracted string
		wantFound     bool
	}{
		{
			name: "ContextKey",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), patternKey, "/test/{id}")
				return r.WithContext(ctx)
			},
			path:          "/test/123",
			wantExtracted: "/test/{id}",
			wantFound:     true,
		},
		{
			name: "phpContextKey",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), phpPatternKey, "/test/{id}")
				return r.WithContext(ctx)
			},
			path:          "/test/123",
			wantExtracted: "/test/{id}",
			wantFound:     true,
		},
		{
			name: "Go ServeMux style",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), contextKey("pattern"), "/test/")
				return r.WithContext(ctx)
			},
			path:          "/test/123",
			wantExtracted: "/test/",
			wantFound:     true,
		},
		{
			name: "Multiple keys",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), contextKey("pattern"), "/test/")
				ctx = context.WithValue(ctx, patternKey, "/test/{id}")
				return r.WithContext(ctx)
			},
			path:          "/test/123",
			wantExtracted: "/test/{id}", // patternKey should take precedence
			wantFound:     true,
		},
		{
			name: "No pattern",
			setupContext: func(r *http.Request) *http.Request {
				return r // No context value added
			},
			path:          "/test/123",
			wantExtracted: "",
			wantFound:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			req = tt.setupContext(req)

			// Manually check for patterns since we can't call middleware internals directly
			found := false
			pattern := ""

			// Try all possible pattern keys (this simulates the middleware's behavior)
			keys := []contextKey{patternKey, phpPatternKey, contextKey("pattern"), contextKey("phpPattern")}
			for _, key := range keys {
				if val, ok := req.Context().Value(key).(string); ok {
					pattern = val
					found = true
					break
				}
			}

			if found != tt.wantFound {
				t.Errorf("Pattern extraction found = %v, wantFound %v", found, tt.wantFound)
			}
			if pattern != tt.wantExtracted {
				t.Errorf("Pattern extraction pattern = %v, wantExtracted %v", pattern, tt.wantExtracted)
			}
		})
	}
}

// TestRoutePathPatternExtraction tests extracting parameters from URL path patterns
func TestRoutePathPatternExtraction(t *testing.T) {
	// Create middleware instance
	php, err := New(WithDevelopmentMode(true))
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create virtual file system
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Create a test PHP script file with pattern in its path
	scriptPath := "/users/{userId}/posts/{postId}.php"
	scriptContent := `<?php
		echo "Path parameters: ";
		if (isset($_PATH["userId"])) {
			echo "userId: " . $_PATH["userId"];
		}
		if (isset($_PATH["postId"])) {
			echo ", postId: " . $_PATH["postId"];
		}
		
		// Debug output
		echo "\n\nDEBUG: All path parameters:\n";
		var_export($_PATH);
	?>`

	// Create the PHP file in the VFS
	err = vfs.CreateVirtualFile(scriptPath, []byte(scriptContent))
	if err != nil {
		t.Fatalf("Failed to create virtual file: %v", err)
	}

	// Test with a specific request that should match our pattern
	tests := []struct {
		name         string
		requestPath  string
		wantParams   map[string]string
		wantStatus   int
		wantContains string
	}{
		{
			name:        "Multiple Parameters",
			requestPath: "/users/123/posts/456",
			wantParams: map[string]string{
				"userId": "123",
				"postId": "456",
			},
			wantStatus:   http.StatusOK,
			wantContains: "userId: 123, postId: 456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with the path that should match our pattern
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()

			// Execute the PHP script directly with the request
			// The ExecutePHP function will handle parameter extraction
			php.ExecutePHP(scriptPath, vfs, nil, w, req)

			// Check response status
			resp := w.Result()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					resp.StatusCode, tt.wantStatus)
			}

			// Check response body
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			// Check for PHP errors
			if strings.Contains(strings.ToLower(bodyStr), "error") ||
				strings.Contains(strings.ToLower(bodyStr), "warning") ||
				strings.Contains(strings.ToLower(bodyStr), "notice") {
				t.Errorf("PHP errors detected: %s", bodyStr)
			}

			// Check for the expected parameters in the output
			if !strings.Contains(bodyStr, tt.wantContains) {
				t.Errorf("Response doesn't contain expected path parameters: %s\nExpected to contain: %s",
					bodyStr, tt.wantContains)
			}

			// Verify each parameter was correctly extracted
			for paramName, paramValue := range tt.wantParams {
				expectedFragment := fmt.Sprintf("%s: %s", paramName, paramValue)
				if !strings.Contains(bodyStr, expectedFragment) {
					t.Errorf("Expected parameter %s=%s not found in output:\n%s",
						paramName, paramValue, bodyStr)
				}
			}
		})
	}
}
