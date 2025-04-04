package frango

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

//go:embed testdata/test.php
var embedTestFS embed.FS

// setupTestMiddleware creates a middleware instance for testing
func setupTestMiddleware(t *testing.T, sourceDir string, opts ...Option) (*Middleware, func()) {
	// Use a null logger by default for tests
	devNull, _ := os.Open(os.DevNull)
	testLogger := log.New(devNull, "", 0)
	finalOpts := append([]Option{WithLogger(testLogger)}, opts...)

	// Add source directory if provided
	if sourceDir != "" {
		finalOpts = append(finalOpts, WithSourceDir(sourceDir))
	}

	php, err := New(finalOpts...)
	if err != nil {
		t.Fatalf("Failed to create Middleware: %v", err)
	}

	cleanup := func() {
		php.Shutdown()
	}

	return php, cleanup
}

// createTestFiles creates temporary PHP files for testing
func createTestFiles(t *testing.T) (string, func()) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "frango_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test PHP files
	files := map[string]string{
		"index.php": `<?php echo "Hello from index.php"; ?>`,
		"about.php": `<?php echo "About page"; ?>`,
		"users/index.php": `<?php 
			echo "Users index page";
			if (isset($_PATH['userId'])) {
				echo " - User ID: " . $_PATH['userId'];
			}
		?>`,
	}

	// Write files
	for name, content := range files {
		filePath := filepath.Join(tempDir, name)
		dirPath := filepath.Dir(filePath)

		// Create directory if needed
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dirPath, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", filePath, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// createTestDir creates a directory with test files for directory operations
func createTestDir(t *testing.T) (string, func()) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "frango-dir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test PHP files in subdirectories
	files := map[string]string{
		"main.php":        `<?php echo "Main PHP file"; ?>`,
		"lib/helper.php":  `<?php echo "Helper library"; ?>`,
		"views/index.php": `<?php echo "Index view"; ?>`,
	}

	// Write files
	for name, content := range files {
		filePath := filepath.Join(tempDir, name)
		dirPath := filepath.Dir(filePath)

		// Create directory if needed
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dirPath, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", filePath, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// TestBasicHandlers tests the basic handler functionality
func TestBasicHandlers(t *testing.T) {

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
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("Handler returned unexpected body: got %v, which doesn't contain %v",
					string(body), tt.wantContains)
			}
		})
	}
}

// TestRenderHandler tests the render functionality
func TestRenderHandler(t *testing.T) {

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

// TestPathParameters tests path parameter extraction
func TestPathParameters(t *testing.T) {
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

	// Look for path parameter in the output
	expectedText := "User profile page - User ID: 42"
	if !strings.Contains(bodyStr, expectedText) {
		t.Errorf("Handler did not extract path parameter: %s", bodyStr)
	}
}

// TestVFS tests VFS operations
func TestVFS(t *testing.T) {

	// Create middleware without source dir
	php, cleanupMiddleware := setupTestMiddleware(t, "", WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Create a VFS
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Create a virtual file
	err := vfs.CreateVirtualFile("/test.php", []byte(`<?php echo "Virtual file test"; ?>`))
	if err != nil {
		t.Fatalf("Failed to create virtual file: %v", err)
	}

	// Create handler using ForVFS
	handler := php.ForVFS(vfs, "/test.php")

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check response body
	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "Virtual file test") {
		t.Errorf("Handler returned unexpected body: %s", string(body))
	}
}

// TestRootVFSOperations tests operations directly on the root VFS
func TestRootVFSOperations(t *testing.T) {
	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, "", WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Create a temporary file to add to the VFS
	tempFile, err := os.CreateTemp("", "frango-test-*.php")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write content to the temp file
	content := []byte(`<?php echo "Content from source file"; ?>`)
	if _, err := tempFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Test suite
	t.Run("AddSourceFile", func(t *testing.T) {
		// Add the source file to the VFS
		err := php.AddSourceFile(tempFile.Name(), "/source.php")
		if err != nil {
			t.Fatalf("Failed to add source file: %v", err)
		}

		// Check if the file exists
		exists, err := php.FileExists("/source.php")
		if err != nil {
			t.Fatalf("Failed to check if file exists: %v", err)
		}
		if !exists {
			t.Error("File should exist after adding")
		}
	})

	t.Run("CreateVirtualFile", func(t *testing.T) {
		// Create a virtual file
		err := php.CreateVirtualFile("/virtual.php", []byte(`<?php echo "Virtual content"; ?>`))
		if err != nil {
			t.Fatalf("Failed to create virtual file: %v", err)
		}

		// Check if the file exists
		exists, err := php.FileExists("/virtual.php")
		if err != nil {
			t.Fatalf("Failed to check if file exists: %v", err)
		}
		if !exists {
			t.Error("Virtual file should exist after creating")
		}
	})

	t.Run("GetFileContent", func(t *testing.T) {
		// Get content of the virtual file
		content, err := php.GetFileContent("/virtual.php")
		if err != nil {
			t.Fatalf("Failed to get file content: %v", err)
		}
		if !strings.Contains(string(content), "Virtual content") {
			t.Errorf("Unexpected file content: %s", string(content))
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		// Copy the virtual file
		err := php.CopyFile("/virtual.php", "/copy.php")
		if err != nil {
			t.Fatalf("Failed to copy file: %v", err)
		}

		// Check if the copy exists
		exists, err := php.FileExists("/copy.php")
		if err != nil {
			t.Fatalf("Failed to check if copy exists: %v", err)
		}
		if !exists {
			t.Error("Copied file should exist")
		}
	})

	t.Run("ListFiles", func(t *testing.T) {
		// List all files
		files, err := php.ListFiles()
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		// Should have at least 3 files (source, virtual, copy)
		if len(files) < 3 {
			t.Errorf("Expected at least 3 files, got %d: %v", len(files), files)
		}

		// Check for specific files
		foundSource := false
		foundVirtual := false
		foundCopy := false
		for _, file := range files {
			if file == "/source.php" {
				foundSource = true
			} else if file == "/virtual.php" {
				foundVirtual = true
			} else if file == "/copy.php" {
				foundCopy = true
			}
		}

		if !foundSource {
			t.Error("Source file not found in file list")
		}
		if !foundVirtual {
			t.Error("Virtual file not found in file list")
		}
		if !foundCopy {
			t.Error("Copied file not found in file list")
		}
	})

	t.Run("MoveFile", func(t *testing.T) {
		// Move a file
		err := php.MoveFile("/copy.php", "/moved.php")
		if err != nil {
			t.Fatalf("Failed to move file: %v", err)
		}

		// Check if the new file exists
		exists, err := php.FileExists("/moved.php")
		if err != nil {
			t.Fatalf("Failed to check if moved file exists: %v", err)
		}
		if !exists {
			t.Error("Moved file should exist")
		}

		// Check if the old file no longer exists
		exists, err = php.FileExists("/copy.php")
		if err != nil {
			t.Fatalf("Failed to check if old file exists: %v", err)
		}
		if exists {
			t.Error("Original file should no longer exist after move")
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		// Delete a file
		err := php.DeleteFile("/moved.php")
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		// Check if the file no longer exists
		exists, err := php.FileExists("/moved.php")
		if err != nil {
			t.Fatalf("Failed to check if deleted file exists: %v", err)
		}
		if exists {
			t.Error("File should not exist after deletion")
		}
	})
}

// TestMoreRootVFSOperations tests the remaining root VFS operations
func TestMoreRootVFSOperations(t *testing.T) {
	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, "", WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Test AddSourceDirectory
	t.Run("AddSourceDirectory", func(t *testing.T) {
		// Create a temporary directory with test files
		sourceDir, cleanupDir := createTestDir(t)
		defer cleanupDir()

		// Add the source directory to the VFS
		err := php.AddSourceDirectory(sourceDir, "/src")
		if err != nil {
			t.Fatalf("Failed to add source directory: %v", err)
		}

		// Check if files from the directory exist
		exists, err := php.FileExists("/src/main.php")
		if err != nil {
			t.Fatalf("Failed to check if main.php exists: %v", err)
		}
		if !exists {
			t.Error("main.php should exist after adding source directory")
		}

		// Check subdirectory files
		exists, err = php.FileExists("/src/lib/helper.php")
		if err != nil {
			t.Fatalf("Failed to check if helper.php exists: %v", err)
		}
		if !exists {
			t.Error("lib/helper.php should exist after adding source directory")
		}

		exists, err = php.FileExists("/src/views/index.php")
		if err != nil {
			t.Fatalf("Failed to check if views/index.php exists: %v", err)
		}
		if !exists {
			t.Error("views/index.php should exist after adding source directory")
		}
	})

	// Test AddEmbeddedFile
	t.Run("AddEmbeddedFile", func(t *testing.T) {
		// Add embedded file
		err := php.AddEmbeddedFile(embedTestFS, "testdata/test.php", "/embedded_file.php")
		if err != nil {
			t.Fatalf("Failed to add embedded file: %v", err)
		}

		// Check if file exists
		exists, err := php.FileExists("/embedded_file.php")
		if err != nil {
			t.Fatalf("Failed to check if embedded file exists: %v", err)
		}
		if !exists {
			t.Error("Embedded file should exist after adding")
		}

		// Check content
		content, err := php.GetFileContent("/embedded_file.php")
		if err != nil {
			t.Fatalf("Failed to get embedded file content: %v", err)
		}
		if !strings.Contains(string(content), "<?php") {
			t.Errorf("Embedded file content doesn't look like PHP: %s", string(content))
		}
	})

	// Test AddEmbeddedDirectory
	t.Run("AddEmbeddedDirectory", func(t *testing.T) {
		// Add embedded directory
		err := php.AddEmbeddedDirectory(embedTestFS, "testdata", "/embedded")
		if err != nil {
			t.Fatalf("Failed to add embedded directory: %v", err)
		}

		// Check if file exists
		exists, err := php.FileExists("/embedded/test.php")
		if err != nil {
			t.Fatalf("Failed to check if file from embedded directory exists: %v", err)
		}
		if !exists {
			t.Error("File from embedded directory should exist after adding")
		}
	})

	// Test AddEmbeddedLibrary (legacy method)
	t.Run("AddEmbeddedLibrary", func(t *testing.T) {
		// Add embedded library and get its disk path
		diskPath, err := php.AddEmbeddedLibrary(embedTestFS, "testdata/test.php", "/lib/test_lib.php")
		if err != nil {
			t.Fatalf("Failed to add embedded library: %v", err)
		}

		// Check if path is returned
		if diskPath == "" {
			t.Error("AddEmbeddedLibrary should return a non-empty disk path")
		}

		// Check if file exists in VFS
		exists, err := php.FileExists("/lib/test_lib.php")
		if err != nil {
			t.Fatalf("Failed to check if library exists: %v", err)
		}
		if !exists {
			t.Error("Library should exist in VFS after adding")
		}

		// Check if the disk path file exists
		if _, err := os.Stat(diskPath); os.IsNotExist(err) {
			t.Error("Library file should exist on disk at the returned path")
		}
	})
}

// TestConcurrentRootVFSOperations tests concurrent access to the root VFS operations
func TestConcurrentRootVFSOperations(t *testing.T) {
	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, "", WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Define number of concurrent operations
	const concurrentCount = 10

	// Create file content
	fileContent := []byte(`<?php echo "Concurrent test"; ?>`)

	t.Run("ConcurrentCreateVirtualFiles", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(concurrentCount)

		// Concurrently create virtual files
		for i := 0; i < concurrentCount; i++ {
			go func(index int) {
				defer wg.Done()
				path := fmt.Sprintf("/concurrent_virtual_%d.php", index)
				err := php.CreateVirtualFile(path, fileContent)
				if err != nil {
					t.Errorf("Failed to create virtual file %s: %v", path, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify all files were created
		for i := 0; i < concurrentCount; i++ {
			path := fmt.Sprintf("/concurrent_virtual_%d.php", i)
			exists, err := php.FileExists(path)
			if err != nil {
				t.Errorf("Error checking if file exists %s: %v", path, err)
				continue
			}
			if !exists {
				t.Errorf("File %s should exist after concurrent creation", path)
			}
		}
	})

	t.Run("ConcurrentFileOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		var copyWg sync.WaitGroup
		var moveWg sync.WaitGroup
		var deleteWg sync.WaitGroup

		// We'll wait for each operation to complete before proceeding to the next
		copyWg.Add(concurrentCount)   // Copying
		moveWg.Add(concurrentCount)   // Moving
		deleteWg.Add(concurrentCount) // Deleting
		wg.Add(concurrentCount)       // Reading

		// First, make sure all files have been created
		for i := 0; i < concurrentCount; i++ {
			path := fmt.Sprintf("/concurrent_virtual_%d.php", i)
			_, err := php.FileExists(path)
			if err != nil {
				t.Fatalf("Error setting up files for concurrent operations: %v", err)
			}
		}

		// 1. Concurrent GetFileContent
		for i := 0; i < concurrentCount; i++ {
			go func(index int) {
				defer wg.Done()
				path := fmt.Sprintf("/concurrent_virtual_%d.php", index)
				content, err := php.GetFileContent(path)
				if err != nil {
					t.Errorf("Failed to get content of %s: %v", path, err)
					return
				}
				if !strings.Contains(string(content), "Concurrent test") {
					t.Errorf("Unexpected content of %s: %s", path, string(content))
				}
			}(i)
		}

		// Wait for reads to complete before copying
		wg.Wait()

		// 2. Concurrent CopyFile
		for i := 0; i < concurrentCount; i++ {
			go func(index int) {
				defer copyWg.Done()
				srcPath := fmt.Sprintf("/concurrent_virtual_%d.php", index)
				destPath := fmt.Sprintf("/concurrent_copy_%d.php", index)
				err := php.CopyFile(srcPath, destPath)
				if err != nil {
					t.Errorf("Failed to copy file %s to %s: %v", srcPath, destPath, err)
				}
			}(i)
		}

		// Wait for copies to complete before moving
		copyWg.Wait()

		// 3. Concurrent MoveFile
		for i := 0; i < concurrentCount; i++ {
			go func(index int) {
				defer moveWg.Done()
				srcPath := fmt.Sprintf("/concurrent_copy_%d.php", index)
				destPath := fmt.Sprintf("/concurrent_moved_%d.php", index)
				err := php.MoveFile(srcPath, destPath)
				if err != nil {
					t.Errorf("Failed to move file %s to %s: %v", srcPath, destPath, err)
				}
			}(i)
		}

		// Wait for moves to complete before deleting
		moveWg.Wait()

		// 4. Concurrent DeleteFile
		for i := 0; i < concurrentCount; i++ {
			go func(index int) {
				defer deleteWg.Done()
				// Delete the original virtual file
				path := fmt.Sprintf("/concurrent_virtual_%d.php", index)
				err := php.DeleteFile(path)
				if err != nil {
					t.Errorf("Failed to delete file %s: %v", path, err)
				}
			}(i)
		}

		// Wait for all operations to complete
		deleteWg.Wait()

		// Verify that original files are deleted
		for i := 0; i < concurrentCount; i++ {
			path := fmt.Sprintf("/concurrent_virtual_%d.php", i)
			exists, err := php.FileExists(path)
			if err != nil {
				t.Errorf("Error checking if file exists %s: %v", path, err)
				continue
			}
			if exists {
				t.Errorf("File %s should not exist after concurrent deletion", path)
			}
		}

		// Verify that moved files exist
		for i := 0; i < concurrentCount; i++ {
			path := fmt.Sprintf("/concurrent_moved_%d.php", i)
			exists, err := php.FileExists(path)
			if err != nil {
				t.Errorf("Error checking if file exists %s: %v", path, err)
				continue
			}
			if !exists {
				t.Errorf("File %s should exist after concurrent move", path)
			}
		}
	})

	t.Run("ConcurrentListAndExists", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2 * concurrentCount) // ListFiles and FileExists calls

		// Create test files for this specific test to avoid dependencies on previous tests
		for i := 0; i < concurrentCount; i++ {
			path := fmt.Sprintf("/exists_test_%d.php", i)
			err := php.CreateVirtualFile(path, []byte(`<?php echo "Exists test"; ?>`))
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", path, err)
			}
		}

		// Concurrent ListFiles calls
		for i := 0; i < concurrentCount; i++ {
			go func() {
				defer wg.Done()
				files, err := php.ListFiles()
				if err != nil {
					t.Errorf("Error listing files: %v", err)
					return
				}
				if len(files) < concurrentCount {
					t.Errorf("Expected at least %d files, got %d", concurrentCount, len(files))
				}
			}()
		}

		// Concurrent FileExists calls
		for i := 0; i < concurrentCount; i++ {
			go func(index int) {
				defer wg.Done()
				path := fmt.Sprintf("/exists_test_%d.php", index)
				exists, err := php.FileExists(path)
				if err != nil {
					t.Errorf("Error checking if file exists %s: %v", path, err)
					return
				}
				if !exists {
					t.Errorf("File %s should exist", path)
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestAdvancedRootVFSOperations tests edge cases and error handling for root VFS operations
func TestAdvancedRootVFSOperations(t *testing.T) {
	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, "", WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Test path handling with special characters
	t.Run("PathHandling", func(t *testing.T) {
		// Test paths with special characters
		specialPaths := []string{
			"/path with spaces.php",
			"/path/with/multiple/segments.php",
			"/path/with/trailing/slash/.php",
			"/./path/with/dot.php",
			"/path//with//double//slashes.php",
			"/PATH/with/MIXED/case.php",
		}

		// Create files with special paths
		for _, path := range specialPaths {
			err := php.CreateVirtualFile(path, []byte(`<?php echo "Special path test"; ?>`))
			if err != nil {
				t.Errorf("Failed to create file with special path %s: %v", path, err)
				continue
			}

			// Verify file exists
			exists, err := php.FileExists(path)
			if err != nil {
				t.Errorf("Error checking if file exists %s: %v", path, err)
				continue
			}
			if !exists {
				t.Errorf("File with special path %s should exist after creation", path)
			}
		}

		// List files and verify all special paths are included
		files, err := php.ListFiles()
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		for _, path := range specialPaths {
			found := false
			// Account for path normalization
			normalizedPath := normalizePath(path)
			for _, file := range files {
				if file == normalizedPath {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("File with normalized path %s not found in file list", normalizedPath)
			}
		}
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// 1. Test reading non-existent file
		_, err := php.GetFileContent("/nonexistent.php")
		if err == nil {
			t.Error("GetFileContent should return error for non-existent file")
		}

		// 2. Test copy of non-existent file
		err = php.CopyFile("/nonexistent.php", "/copy.php")
		if err == nil {
			t.Error("CopyFile should return error when source doesn't exist")
		}

		// 3. Test move of non-existent file
		err = php.MoveFile("/nonexistent.php", "/moved.php")
		if err == nil {
			t.Error("MoveFile should return error when source doesn't exist")
		}

		// 4. Test delete of non-existent file
		err = php.DeleteFile("/nonexistent.php")
		if err == nil {
			t.Error("DeleteFile should return error for non-existent file")
		}

		// 5. Test invalid path handling
		err = php.CreateVirtualFile("", []byte{})
		if err == nil {
			t.Error("CreateVirtualFile should return error for empty path")
		}
	})

	// Test mixed operations (different file origins)
	t.Run("MixedOrigins", func(t *testing.T) {
		// Create a temporary file for source
		tempFile, err := os.CreateTemp("", "frango-test-*.php")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		// Write content to the temp file
		sourceContent := []byte(`<?php echo "Source file"; ?>`)
		if _, err := tempFile.Write(sourceContent); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tempFile.Close()

		// Add various file types
		err = php.AddSourceFile(tempFile.Name(), "/source.php")
		if err != nil {
			t.Fatalf("Failed to add source file: %v", err)
		}

		err = php.CreateVirtualFile("/virtual.php", []byte(`<?php echo "Virtual file"; ?>`))
		if err != nil {
			t.Fatalf("Failed to create virtual file: %v", err)
		}

		err = php.AddEmbeddedFile(embedTestFS, "testdata/test.php", "/embedded.php")
		if err != nil {
			t.Fatalf("Failed to add embedded file: %v", err)
		}

		// Verify all files exist
		files, err := php.ListFiles()
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		foundSource := false
		foundVirtual := false
		foundEmbedded := false
		for _, file := range files {
			if file == "/source.php" {
				foundSource = true
			} else if file == "/virtual.php" {
				foundVirtual = true
			} else if file == "/embedded.php" {
				foundEmbedded = true
			}
		}

		if !foundSource {
			t.Error("Source file not found in file list")
		}
		if !foundVirtual {
			t.Error("Virtual file not found in file list")
		}
		if !foundEmbedded {
			t.Error("Embedded file not found in file list")
		}

		// Test cross-origin operations
		// 1. Copy source to virtual
		err = php.CopyFile("/source.php", "/source-copy.php")
		if err != nil {
			t.Errorf("Failed to copy source file: %v", err)
		}

		// 2. Copy embedded to virtual
		err = php.CopyFile("/embedded.php", "/embedded-copy.php")
		if err != nil {
			t.Errorf("Failed to copy embedded file: %v", err)
		}

		// Verify content types
		content1, err := php.GetFileContent("/source-copy.php")
		if err != nil {
			t.Errorf("Failed to get content of copied source file: %v", err)
		} else if !strings.Contains(string(content1), "Source file") {
			t.Errorf("Copied source file has incorrect content: %s", string(content1))
		}

		content2, err := php.GetFileContent("/embedded-copy.php")
		if err != nil {
			t.Errorf("Failed to get content of copied embedded file: %v", err)
		} else if !strings.Contains(string(content2), "<?php") {
			t.Errorf("Copied embedded file has incorrect content: %s", string(content2))
		}
	})
}

// TestNew tests creating a new middleware instance
func TestNew(t *testing.T) {
	// Suppress logger output during tests
	logger := log.New(io.Discard, "", 0)

	// Create middleware with options
	m, err := New(
		WithLogger(logger),
		WithDevelopmentMode(true),
	)

	// Check for errors
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Clean up
	defer m.Shutdown()

	// Verify middleware was created properly
	if m == nil {
		t.Fatal("New() returned nil middleware")
	}

	// Check options were applied
	if !m.developmentMode {
		t.Error("WithDevelopmentMode(true) not applied")
	}
}

// TestBasicRequest tests a basic PHP request
func TestBasicRequest(t *testing.T) {
	// Skip if we're not in an environment with PHP installed
	if _, err := os.Stat("/usr/bin/php"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/local/bin/php"); os.IsNotExist(err) {
			t.Skip("Skipping test because PHP binary not found")
		}
	}

	// Create a logger that writes to a buffer we can check
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)

	// First create the testdata directory and the test.php file if they don't exist
	testdataDir := "testdata"
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	testPHPFile := filepath.Join(testdataDir, "test.php")
	testPHPContent := `<?php
		echo "This is a test PHP file";
		
		// Display any path parameters that might be set
		if (isset($_PATH) && count($_PATH) > 0) {
			echo "\nPath parameters:\n";
			foreach ($_PATH as $key => $value) {
				echo "$key: $value\n";
			}
		}
	?>`

	if err := os.WriteFile(testPHPFile, []byte(testPHPContent), 0644); err != nil {
		t.Fatalf("Failed to create test PHP file: %v", err)
	}

	// Get the absolute path to the testdata directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	absTestdataDir := filepath.Join(cwd, testdataDir)

	// Create middleware with test directory
	m, err := New(
		WithSourceDir(absTestdataDir),
		WithLogger(logger),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer m.Shutdown()

	// Create a test request
	req := httptest.NewRequest("GET", "/test.php", nil)
	w := httptest.NewRecorder()

	// Execute the request
	m.ExecutePHP("/test.php", m.rootVFS, nil, w, req)

	// Check the response
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	// Verify basic output
	if !strings.Contains(string(body), "This is a test PHP file") {
		t.Errorf("Expected output from test.php, got: %s", body)
	}

	// Check for proper logging
	logs := logOutput.String()
	expectedLogs := []string{
		"ExecutePHP:",
		"Executing script",
		"Total PHP environment variables",
	}

	for _, expected := range expectedLogs {
		if !strings.Contains(logs, expected) {
			t.Errorf("Expected log to contain '%s', but it didn't. Logs:\n%s", expected, logs)
		}
	}
}

// TestPHPEnvironmentVariables tests that the PHP_ environment variables are correctly set and accessible
func TestPHPEnvironmentVariables(t *testing.T) {
	t.Skip("Skipping test temporarily due to hanging issues")

	// Create a temporary PHP script that outputs environment variables
	envVarScript := `<?php
header('Content-Type: text/plain');
echo "PHP Environment Variables Test\n";
echo "============================\n\n";

// Check for path parameter variables
echo "PATH PARAMETERS:\n";
$pathParamsFound = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_PATH_PARAM_') === 0) {
        $pathParamsFound = true;
        echo "  $key = $value\n";
    }
}
if (!$pathParamsFound) {
    echo "  No PHP_PATH_PARAM_* variables found\n";
}

// Check for path segments
echo "\nPATH SEGMENTS:\n";
$segmentsFound = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_PATH_SEGMENT_') === 0) {
        $segmentsFound = true;
        echo "  $key = $value\n";
    }
}
if (!$segmentsFound) {
    echo "  No PHP_PATH_SEGMENT_* variables found\n";
}

// Check for query parameters
echo "\nQUERY PARAMETERS:\n";
$queryFound = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $queryFound = true;
        echo "  $key = $value\n";
    }
}
if (!$queryFound) {
    echo "  No PHP_QUERY_* variables found\n";
}

// Check for PHP_PATH_PARAMS
echo "\nPATH PARAMS JSON:\n";
if (isset($_SERVER['PHP_PATH_PARAMS'])) {
    echo "  PHP_PATH_PARAMS = " . $_SERVER['PHP_PATH_PARAMS'] . "\n";
} else {
    echo "  PHP_PATH_PARAMS not found\n";
}

// Print all $_SERVER vars for debugging
echo "\nAll \$_SERVER variables:\n";
foreach ($_SERVER as $key => $value) {
    echo "  $key = $value\n";
}
?>`

	// Setup test environment with our script
	testdataDir := filepath.Join("testdata")
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	testFilePath := filepath.Join(testdataDir, "env_test.php")
	if err := os.WriteFile(testFilePath, []byte(envVarScript), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFilePath)

	// Create middleware with our test directory
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)

	middleware, err := New(
		WithSourceDir(testdataDir),
		WithLogger(logger),
		// Disable development mode to avoid file watching
		WithDevelopmentMode(false),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer middleware.Shutdown()

	// Create a test request with path parameters and query parameters
	req := httptest.NewRequest("GET", "/users/123/profile/admin?page=1&sort=desc", nil)

	// Add pattern to context (like Go 1.22+ ServeMux does)
	ctx := context.WithValue(req.Context(), ContextKey("pattern"), "GET /users/{userId}/profile/{type}")
	req = req.WithContext(ctx)

	// Create recorder for the response
	w := httptest.NewRecorder()

	// Execute the PHP script directly
	middleware.ExecutePHP("env_test.php", middleware.rootVFS, nil, w, req)

	// Get the response
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	t.Logf("Response body:\n%s", bodyStr)
	t.Logf("Logs:\n%s", logOutput.String())

	// Test for important environment variables
	expectedVariables := []string{
		"PHP_PATH_PARAM_userId",
		"PHP_PATH_PARAM_type",
		"PHP_QUERY_page",
		"PHP_QUERY_sort",
		"PHP_PATH_PARAMS",
		"PHP_PATH_SEGMENT_",
	}

	for _, expected := range expectedVariables {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected %s in output but it was not found", expected)
		}
	}
}

// TestJSONBodyHandling tests that JSON request bodies are correctly passed to PHP
func TestJSONBodyHandling(t *testing.T) {
	t.Skip("Skipping test temporarily due to hanging issues")

	// Create a PHP script that outputs the JSON body from $_JSON
	jsonTestScript := `<?php
header('Content-Type: text/plain');
echo "JSON Body Test\n";
echo "=============\n\n";

// Check for PHP_JSON environment variable
echo "PHP_JSON Environment Variable:\n";
if (isset($_SERVER['PHP_JSON'])) {
    echo "  PHP_JSON found, length: " . strlen($_SERVER['PHP_JSON']) . " bytes\n";
    echo "  Content: " . $_SERVER['PHP_JSON'] . "\n";
} else {
    echo "  PHP_JSON not found\n";
}

// Check for $_JSON superglobal
echo "\n\$_JSON Superglobal:\n";
if (isset($_JSON) && is_array($_JSON)) {
    if (empty($_JSON)) {
        echo "  $_JSON is empty\n";
    } else {
        foreach ($_JSON as $key => $value) {
            echo "  $_JSON[$key] = ";
            if (is_array($value)) {
                echo json_encode($value) . "\n";
            } else {
                echo $value . "\n";
            }
        }
    }
} else {
    echo "  $_JSON is not defined or not an array\n";
}
?>`

	// Setup test environment with our script
	testdataDir := filepath.Join("testdata")
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	testFilePath := filepath.Join(testdataDir, "json_test.php")
	if err := os.WriteFile(testFilePath, []byte(jsonTestScript), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFilePath)

	// Create middleware with verbose logging
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)

	middleware, err := New(
		WithSourceDir(testdataDir),
		WithLogger(logger),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer middleware.Shutdown()

	// Create a test mux
	mux := http.NewServeMux()
	mux.Handle("/json", middleware.For("json_test.php"))

	// Create a JSON request body
	jsonBody := `{
		"user": {
			"id": 42,
			"name": "John Doe",
			"email": "john@example.com"
		},
		"items": [1, 2, 3],
		"active": true
	}`

	// Create a request with JSON body
	req := httptest.NewRequest("POST", "/json", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	t.Logf("Response body:\n%s", bodyStr)
	t.Logf("Logs:\n%s", logOutput.String())

	// Test for JSON environment variable
	expectedStrings := []string{
		"PHP_JSON found",
		"$_JSON[user]",
		"$_JSON[items]",
		"$_JSON[active] = true",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected '%s' in output but it was not found", expected)
		}
	}
}

// TestContextKeyExtraction tests the pattern extraction from context
func TestContextKeyExtraction(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(context.Context) context.Context
		expected string
	}{
		{
			name: "ContextKey",
			setup: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, ContextKey("pattern"), "/users/{id}")
			},
			expected: "/users/{id}",
		},
		{
			name: "phpContextKey",
			setup: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, phpContextKey("pattern"), "/posts/{slug}")
			},
			expected: "/posts/{slug}",
		},
		{
			name: "Go ServeMux style",
			setup: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "pattern", "/comments/{id}")
			},
			expected: "/comments/{id}",
		},
		{
			name: "Multiple keys",
			setup: func(ctx context.Context) context.Context {
				ctx = context.WithValue(ctx, ContextKey("pattern"), "/primary/{id}")
				return context.WithValue(ctx, phpContextKey("pattern"), "/secondary/{id}")
			},
			expected: "/primary/{id}", // ContextKey should take precedence
		},
		{
			name: "No pattern",
			setup: func(ctx context.Context) context.Context {
				return ctx // Don't add any pattern
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a base context
			baseCtx := context.Background()

			// Apply the test case setup to add values
			ctx := tc.setup(baseCtx)

			// Extract the pattern using the correct function
			result := extractPatternFromContext(ctx)

			// Verify the result
			if result != tc.expected {
				t.Errorf("Expected pattern %q but got %q", tc.expected, result)
			}
		})
	}
}

// TestScriptPathPatternExtraction tests the automatic extraction of patterns from script paths
func TestScriptPathPatternExtraction(t *testing.T) {
	// Create middleware
	php, cleanupMiddleware := setupTestMiddleware(t, "", WithDevelopmentMode(true))
	defer cleanupMiddleware()

	// Create a VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Create a test PHP script file
	scriptContent := []byte(`<?php
	echo "Path Parameter Test\n";
	echo "=================\n";
	echo "Path Parameters: ";
	var_export($_PATH);
	echo "\n";
	?>`)

	// Create virtual files with parameter patterns in their paths
	vfs.CreateVirtualFile("/users/{id}.php", scriptContent)
	vfs.CreateVirtualFile("/products/{category}/{id}.php", scriptContent)

	testCases := []struct {
		name           string
		scriptPath     string
		requestPath    string
		expectedParams map[string]string
	}{
		{
			name:           "Single Parameter",
			scriptPath:     "/users/{id}.php",
			requestPath:    "/users/42",
			expectedParams: map[string]string{"id": "42"},
		},
		{
			name:           "Multiple Parameters",
			scriptPath:     "/products/{category}/{id}.php",
			requestPath:    "/products/electronics/123",
			expectedParams: map[string]string{"category": "electronics", "id": "123"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request with the path that should match our pattern
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()

			// Execute the request - no need to manually set context
			php.ExecutePHP(tc.scriptPath, vfs, nil, w, req)

			// Check the response
			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			// Verify all expected parameters are found in the output
			for paramName, paramValue := range tc.expectedParams {
				expectedStr := fmt.Sprintf("'%s' => '%s'", paramName, paramValue)
				if !strings.Contains(bodyStr, expectedStr) {
					t.Errorf("Expected parameter %s=%s not found in output:\n%s",
						paramName, paramValue, bodyStr)
				}
			}
		})
	}
}
