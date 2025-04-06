package frango

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitMiddleware tests creating the middleware with different options
func TestInitMiddleware(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "Default options",
			opts:    []Option{},
			wantErr: false,
		},
		{
			name: "With source directory",
			opts: []Option{
				WithSourceDir(tempDir),
			},
			wantErr: false,
		},
		{
			name: "With development mode",
			opts: []Option{
				WithDevelopmentMode(true),
			},
			wantErr: false,
		},
		{
			name: "With custom logger",
			opts: []Option{
				WithLogger(log.New(os.Stdout, "[test] ", log.LstdFlags)),
			},
			wantErr: false,
		},
		{
			name: "With PHP URLs blocking",
			opts: []Option{
				WithDirectPHPURLsBlocking(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			php, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if php == nil && !tt.wantErr {
				t.Errorf("New() returned nil without error")
				return
			}
			if php != nil {
				php.Shutdown()
			}
		})
	}
}

// TestSimpleRequest tests a basic PHP request through the middleware
func TestSimpleRequest(t *testing.T) {
	// Create temp directory with a simple PHP file
	tempDir, err := os.MkdirTemp("", "frango-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP file
	phpFile := filepath.Join(tempDir, "hello.php")
	phpContent := `<?php echo "Hello, World!"; ?>`
	if err := os.WriteFile(phpFile, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Create middleware
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

	// Add file to VFS
	if err := vfs.AddSourceFile(phpFile, "/hello.php"); err != nil {
		t.Fatalf("Failed to add file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/hello.php", nil)
	w := httptest.NewRecorder()

	// Execute PHP
	php.ExecutePHP("/hello.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	// Verify status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Verify response content
	if !strings.Contains(string(body), "Hello, World!") {
		t.Errorf("Expected response to contain 'Hello, World!', got: %s", string(body))
	}
}
