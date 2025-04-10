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

// TestRelativeIncludePaths tests that relative include paths work correctly
// This test verifies that PHP scripts can include files using relative paths like:
// 1. Including files from parent directories (../includes/header.php)
// 2. Including files from grandparent directories (../../includes/header.php)
//
// This is critical because PHP scripts often use relative paths to include shared files,
// and the wrapper mechanism needs to properly maintain the original script's directory
// context to ensure these relative paths resolve correctly.
func TestRelativeIncludePaths(t *testing.T) {
	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-relative-include-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	categoryDir := filepath.Join(tempDir, "categories")
	subCategoryDir := filepath.Join(categoryDir, "electronics")
	includeDir := filepath.Join(tempDir, "includes")

	// Create directories
	for _, dir := range []string{categoryDir, subCategoryDir, includeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create a shared include file
	includeFile := `<?php
function getHeader() {
	return "Header from include file";
}
?>
`
	err = os.WriteFile(filepath.Join(includeDir, "header.php"), []byte(includeFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create include file: %v", err)
	}

	// Create a main PHP file that uses relative path to include
	mainFile := `<?php
header("Content-Type: application/json");

// Include file using a relative path
include_once("../includes/header.php");

$response = [
	"success" => function_exists("getHeader"),
	"header" => getHeader(),
	"file" => __FILE__,
	"dir" => __DIR__,
];

echo json_encode($response, JSON_PRETTY_PRINT);
?>
`
	err = os.WriteFile(filepath.Join(categoryDir, "index.php"), []byte(mainFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Create a subcategory PHP file that uses relative path to include
	subcategoryFile := `<?php
header("Content-Type: application/json");

// Include file using a relative path (going up two levels)
include_once("../../includes/header.php");

$response = [
	"success" => function_exists("getHeader"),
	"header" => getHeader(),
	"file" => __FILE__,
	"dir" => __DIR__,
];

echo json_encode($response, JSON_PRETTY_PRINT);
?>
`
	err = os.WriteFile(filepath.Join(subCategoryDir, "smartphones.php"), []byte(subcategoryFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create subcategory file: %v", err)
	}

	// Create Frango middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create handlers for each PHP file
	mainHandler := php.For("/categories/index.php")
	subcategoryHandler := php.For("/categories/electronics/smartphones.php")

	// Test main file with a relative include
	t.Run("Main file with relative include", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/categories", nil)
		w := httptest.NewRecorder()

		mainHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Response: %s", body)
			return
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Verify no PHP errors occurred
		if strings.Contains(bodyStr, "Warning") || strings.Contains(bodyStr, "Fatal error") {
			t.Errorf("PHP error occurred: %s", bodyStr)
		}

		// Check if the include worked
		if !strings.Contains(bodyStr, "Header from include file") {
			t.Errorf("Include file content not found in response: %s", bodyStr)
		}

		// Check if success is true
		if !strings.Contains(bodyStr, `"success": true`) {
			t.Errorf("Expected 'success: true' in response, got: %s", bodyStr)
		}
	})

	// Test subcategory file with a relative include going up two levels
	t.Run("Subcategory file with relative include", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/categories/electronics/smartphones", nil)
		w := httptest.NewRecorder()

		subcategoryHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Response: %s", body)
			return
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Verify no PHP errors occurred
		if strings.Contains(bodyStr, "Warning") || strings.Contains(bodyStr, "Fatal error") {
			t.Errorf("PHP error occurred: %s", bodyStr)
		}

		// Check if the include worked
		if !strings.Contains(bodyStr, "Header from include file") {
			t.Errorf("Include file content not found in response: %s", bodyStr)
		}

		// Check if success is true
		if !strings.Contains(bodyStr, `"success": true`) {
			t.Errorf("Expected 'success: true' in response, got: %s", bodyStr)
		}
	})
}
