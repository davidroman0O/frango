package frango

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestPHPDebug runs a simple PHP script to diagnose any issues with our setup
func TestPHPDebug(t *testing.T) {
	// Create a very simple PHP script
	debugPHP := `<?php
	// Debug PHP script to check environment
	header("Content-Type: text/plain");
	
	echo "=== PHP DEBUG INFO ===\n\n";
	
	echo "PHP Version: " . phpversion() . "\n\n";
	
	echo "PHP Auto Prepend: " . (getenv("PHP_AUTO_PREPEND_FILE") ?? "Not set") . "\n";
	echo "PHP Include Path: " . (getenv("PHP_INCLUDE_PATH") ?? "Not set") . "\n\n";
	
	echo "Current Working Directory: " . getcwd() . "\n";
	echo "Script Filename: " . $_SERVER['SCRIPT_FILENAME'] . "\n";
	echo "Document Root: " . $_SERVER['DOCUMENT_ROOT'] . "\n\n";
	
	echo "=== FILE ACCESS TEST ===\n\n";
	
	$globals_path = getenv("PHP_AUTO_PREPEND_FILE");
	echo "Auto-prepend file exists: " . (file_exists($globals_path) ? "YES" : "NO") . "\n";
	
	if (file_exists($globals_path)) {
		echo "Auto-prepend file content first 100 chars: \n";
		echo substr(file_get_contents($globals_path), 0, 100) . "...\n\n";
	}
	
	echo "=== SUPERGLOBALS STATUS ===\n\n";
	echo '$_GET set: ' . (isset($_GET) && is_array($_GET) ? 'YES' : 'NO') . "\n";
	echo '$_POST set: ' . (isset($_POST) && is_array($_POST) ? 'YES' : 'NO') . "\n";
	echo '$_REQUEST set: ' . (isset($_REQUEST) && is_array($_REQUEST) ? 'YES' : 'NO') . "\n";
	echo '$_PATH set: ' . (isset($_PATH) && is_array($_PATH) ? 'YES' : 'NO') . "\n";
	echo '$_PATH_SEGMENTS set: ' . (isset($_PATH_SEGMENTS) && is_array($_PATH_SEGMENTS) ? 'YES' : 'NO') . "\n";
	echo '$_TEMPLATE set: ' . (isset($_TEMPLATE) && is_array($_TEMPLATE) ? 'YES' : 'NO') . "\n";
	echo '$_FORM set: ' . (isset($_FORM) && is_array($_FORM) ? 'YES' : 'NO') . "\n";
	echo '$_JSON set: ' . (isset($_JSON) && is_array($_JSON) ? 'YES' : 'NO') . "\n\n";
	
	echo "=== ENVIRONMENT VARIABLES ===\n\n";
	foreach ($_SERVER as $key => $value) {
		if (strpos($key, 'PHP_') === 0) {
			echo "$key: $value\n";
		}
	}
	
	echo "\n=== END DEBUG INFO ===\n";
	?>`

	// Setup a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "frango-debug-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "debug.php")
	if err := os.WriteFile(testFilePath, []byte(debugPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
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

	// Add PHP file to VFS
	err = vfs.AddSourceFile(testFilePath, "/debug.php")
	if err != nil {
		t.Fatalf("Failed to add file to VFS: %v", err)
	}

	// Create globals file separately to see if it works
	if err := InstallPHPGlobals(vfs); err != nil {
		t.Fatalf("Failed to install PHP globals: %v", err)
	}

	// Execute PHP script
	req := httptest.NewRequest("GET", "/debug.php?test=1", nil)
	resp := httptest.NewRecorder()

	// Get the absolute path to the globals script
	globalsPath, err := vfs.ResolvePath("/_frango_php_globals.php")
	if err != nil {
		t.Logf("Warning: Could not resolve globals path: %v", err)
	} else {
		t.Logf("PHP globals script path: %s", globalsPath)
	}

	php.ExecutePHP("/debug.php", vfs, nil, resp, req)

	// Check response
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	// Print response body for debugging
	body, _ := io.ReadAll(resp.Body)
	t.Logf("\n\nPHP Debug Output:\n%s\n", string(body))
}
