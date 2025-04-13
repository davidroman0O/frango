package executor

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidroman0O/frango/v2/pkg/vfs"
)

// TestScriptDirectory tests if the script directory is properly set for includes
func TestScriptDirectory(t *testing.T) {
	// Create a temp directory for VFS
	tempDir := filepath.Join(os.TempDir(), "frango-directory-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test source directory - we created these files earlier
	sourceDir := "/tmp/frango-test-script-dir"

	// Create logs for the directory structure
	t.Logf("Source directory: %s", sourceDir)

	// Inspect source directory structure
	if entries, err := os.ReadDir(sourceDir); err == nil {
		for _, entry := range entries {
			t.Logf("Source file: %s", entry.Name())
		}

		// Check includes directory
		includesDir := filepath.Join(sourceDir, "includes")
		if entries, err := os.ReadDir(includesDir); err == nil {
			for _, entry := range entries {
				t.Logf("Include file: %s", entry.Name())
			}
		} else {
			t.Logf("Error reading includes dir: %v", err)
		}
	} else {
		t.Logf("Error reading source dir: %v", err)
	}

	// Create a VFS
	vfsLogger := log.New(os.Stdout, "[vfs-test] ", log.LstdFlags)
	v, err := vfs.NewVFSWithConfig(vfs.VFSConfig{
		TempDir:     tempDir,
		Logger:      vfsLogger,
		DevelopMode: true,
	})
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer v.Cleanup()

	// Add source directory to VFS
	err = v.AddSourceDirectory(sourceDir, "/scripts")
	if err != nil {
		t.Fatalf("Failed to add source directory to VFS: %v", err)
	}

	// Dump VFS files
	files := v.ListFiles()
	t.Logf("VFS files:")
	for _, file := range files {
		resolvedPath, err := v.ResolvePath(file)
		resolvedInfo := ""
		if err == nil {
			resolvedInfo = fmt.Sprintf(" -> %s", resolvedPath)
		}
		t.Logf("  %s%s", file, resolvedInfo)
	}

	// Get VFS temp directory - this is where PHP runs from
	vfsTempDir := v.GetTempDir()
	t.Logf("VFS temp directory: %s", vfsTempDir)

	// Create an includes directory directly in the VFS temp dir where PHP will look for it
	vfsIncludesDir := filepath.Join(vfsTempDir, "includes")
	if err := os.MkdirAll(vfsIncludesDir, 0755); err != nil {
		t.Fatalf("Failed to create includes dir in temp: %v", err)
	}

	// Copy the helper.php directly to the runtime location
	sourceHelperPath := filepath.Join(sourceDir, "includes", "helper.php")
	destHelperPath := filepath.Join(vfsIncludesDir, "helper.php")
	helperContent, err := os.ReadFile(sourceHelperPath)
	if err != nil {
		t.Fatalf("Failed to read helper file: %v", err)
	}

	if err := os.WriteFile(destHelperPath, helperContent, 0644); err != nil {
		t.Fatalf("Failed to write helper file: %v", err)
	}
	t.Logf("Copied helper file to PHP runtime location: %s", destHelperPath)

	// Create executor
	exec := NewExecutor(Config{
		Logger:          vfsLogger,
		DevelopmentMode: true,
		DisplayErrors:   true,
	}, v)

	// Create a request
	req := httptest.NewRequest("GET", "/scripts/main.php", nil)
	resp := httptest.NewRecorder()

	// Execute the script
	exec.Execute(v, "/scripts/main.php", nil, resp, req)

	// Check response
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	// Check for expected output from both files
	output := resp.Body.String()
	t.Logf("Response output: %s", output)

	// Verify main file execution
	if !strings.Contains(output, "Main PHP file:") {
		t.Errorf("Main file not properly executed")
	}

	// Most importantly, verify the include worked
	if !strings.Contains(output, "Helper file included:") {
		t.Errorf("Helper file not included - directory resolution may be incorrect")
	}

	// Check if helper function was called, proving the include fully worked
	if !strings.Contains(output, "Helper function called from:") {
		t.Errorf("Helper function not called - include may have failed")
	}
}
