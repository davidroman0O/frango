package executor

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidroman0O/frango/v2/pkg/vfs"
)

func setupTestEnvironment(t *testing.T) (*vfs.VFS, string, string) {
	// Create a source temporary directory for our test files
	sourceDir, err := ioutil.TempDir("", "frango-test-source-dir")
	if err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create a separate temp dir for PHP execution (where VFS will write files)
	execDir, err := ioutil.TempDir("", "frango-test-exec-dir")
	if err != nil {
		t.Fatalf("Failed to create exec dir: %v", err)
	}

	// Create includes directory in source
	includesDir := filepath.Join(sourceDir, "includes")
	if err := os.MkdirAll(includesDir, 0755); err != nil {
		t.Fatalf("Failed to create includes dir: %v", err)
	}

	// Create helper.php in includes directory
	helperContent := `<?php
function helper_function() {
    echo "Helper function called from " . __FILE__ . "\n";
    return true;
}
?>`
	if err := os.WriteFile(filepath.Join(includesDir, "helper.php"), []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to create helper.php: %v", err)
	}

	// Create main PHP script
	mainContent := `<?php
// Test script for include path resolution
echo "Running script at: " . __FILE__ . "\n";
echo "Current directory: " . getcwd() . "\n";
echo "Script dir: " . __DIR__ . "\n";
echo "Including helper.php...\n";

// Debug directory listing
echo "Directory contents: " . implode(", ", scandir(".")) . "\n";
echo "Includes directory contents: " . (is_dir("includes") ? implode(", ", scandir("includes")) : "directory not found") . "\n";

// Include the helper file (relative path)
include 'includes/helper.php';

// Call the helper function
$result = helper_function();
echo "Helper function result: " . ($result ? "true" : "false") . "\n";
echo "Done!";
?>`
	if err := os.WriteFile(filepath.Join(sourceDir, "main.php"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main.php: %v", err)
	}

	// Create a VFS with the exec directory as temp dir
	fs, err := vfs.NewVFSWithConfig(vfs.VFSConfig{
		TempDir:     execDir,
		Logger:      log.New(os.Stdout, "VFS: ", log.LstdFlags),
		DevelopMode: true,
	})
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}

	// Add the source directory to the VFS
	err = fs.AddSourceDirectory(sourceDir, "/scripts")
	if err != nil {
		t.Fatalf("Failed to add source directory: %v", err)
	}

	t.Logf("Source directory: %s", sourceDir)
	t.Logf("Execution directory: %s", execDir)
	t.Logf("Files in source directory: %v", listFiles(sourceDir))
	t.Logf("Files in includes directory: %v", listFiles(includesDir))

	// Copy helper file to execution directory to ensure PHP can find it
	execIncludesDir := filepath.Join(execDir, "includes")
	if err := os.MkdirAll(execIncludesDir, 0755); err != nil {
		t.Fatalf("Failed to create includes dir in exec dir: %v", err)
	}

	helperSourcePath := filepath.Join(sourceDir, "includes", "helper.php")
	helperDestPath := filepath.Join(execIncludesDir, "helper.php")

	// Copy the helper file
	helperFileBytes, err := os.ReadFile(helperSourcePath)
	if err != nil {
		t.Fatalf("Failed to read helper file: %v", err)
	}

	if err := os.WriteFile(helperDestPath, helperFileBytes, 0644); err != nil {
		t.Fatalf("Failed to write helper file to execution directory: %v", err)
	}

	t.Logf("Files in exec directory: %v", listFiles(execDir))
	t.Logf("Files in exec includes directory: %v", listFiles(execIncludesDir))

	return fs, sourceDir, execDir
}

func listFiles(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return []string{fmt.Sprintf("Error: %v", err)}
	}
	var names []string
	for _, file := range files {
		names = append(names, file.Name())
	}
	return names
}

func TestScriptDirectory(t *testing.T) {
	// Skip if nowatcher tag is not set (FrankenPHP may not be available)
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Setup test environment
	vfs, sourceDir, execDir := setupTestEnvironment(t)
	defer os.RemoveAll(sourceDir)
	defer os.RemoveAll(execDir)

	// List all VFS files
	t.Logf("VFS files: %v", vfs.ListFiles())
	t.Logf("Files in exec directory: %v", listFiles(execDir))
	t.Logf("Files in exec includes directory: %v", listFiles(filepath.Join(execDir, "includes")))

	// Create executor with test config
	executor := NewExecutor(Config{
		Logger:          log.New(os.Stdout, "Executor: ", log.LstdFlags),
		DevelopmentMode: true,
		DisplayErrors:   true,
	}, vfs)

	// Create test request
	req := httptest.NewRequest("GET", "http://example.com/scripts/main.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	executor.Execute(vfs, "/scripts/main.php", nil, w, req)

	// Get the response
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	// Print the output
	output := string(body)
	t.Logf("PHP Output:\n%s", output)

	// Check for the expected response
	expectedStrings := []string{
		"Helper function called",
		"Helper function result: true",
		"Done!",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it did not", expected)
		}
	}
}
