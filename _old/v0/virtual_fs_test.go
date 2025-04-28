package frango

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/embed/test.php
var testEmbedFiles embed.FS

func setupTestEnvironment(t *testing.T) (string, func()) {
	// Create temp dir for test files
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	require.NoError(t, err, "Failed to create temp directory")

	// Create test file structure
	testFiles := map[string]string{
		"source.php":         "<?php echo 'source file'; ?>",
		"subdir/nested.php":  "<?php echo 'nested file'; ?>",
		"subdir/index.php":   "<?php echo 'index file'; ?>",
		"api/users.get.php":  "<?php echo 'users GET'; ?>",
		"api/users.post.php": "<?php echo 'users POST'; ?>",
		"api/users/{id}.php": "<?php echo 'user by ID'; ?>",
		"static/styles.css":  "body { color: red; }",
		"static/script.js":   "console.log('test');",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err, "Failed to create directory for test file")

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file")
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestVirtualFS_AddSourceFile(t *testing.T) {
	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	sourcePath := filepath.Join(tempDir, "source.php")

	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Add source file
	err = vfs.AddSourceFile(sourcePath, "/test.php")
	assert.NoError(t, err, "Failed to add source file")

	// Verify file is in VFS
	paths := vfs.ListFiles()
	assert.Contains(t, paths, "/test.php", "Source file should be in VFS")

	// Check content access
	content, err := vfs.GetFileContent("/test.php")
	assert.NoError(t, err, "Failed to get file content")
	assert.Equal(t, "<?php echo 'source file'; ?>", string(content), "Content mismatch")
}

func TestVirtualFS_AddSourceDirectory(t *testing.T) {
	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Add source directory
	err = vfs.AddSourceDirectory(tempDir, "/")
	assert.NoError(t, err, "Failed to add source directory")

	// Verify files are in VFS
	paths := vfs.ListFiles()
	assert.Contains(t, paths, "/source.php", "Root file should be in VFS")
	assert.Contains(t, paths, "/subdir/nested.php", "Nested file should be in VFS")
	assert.Contains(t, paths, "/api/users.get.php", "Method file should be in VFS")

	// Check content access for a nested file
	content, err := vfs.GetFileContent("/subdir/nested.php")
	assert.NoError(t, err, "Failed to get file content")
	assert.Equal(t, "<?php echo 'nested file'; ?>", string(content), "Content mismatch")
}

func TestVirtualFS_CreateVirtualFile(t *testing.T) {
	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Create virtual file
	vfContent := []byte("<?php echo 'virtual file'; ?>")
	err = vfs.CreateVirtualFile("/virtual.php", vfContent)
	assert.NoError(t, err, "Failed to create virtual file")

	// Verify file is in VFS
	paths := vfs.ListFiles()
	assert.Contains(t, paths, "/virtual.php", "Virtual file should be in VFS")

	// Check content access
	content, err := vfs.GetFileContent("/virtual.php")
	assert.NoError(t, err, "Failed to get file content")
	assert.Equal(t, vfContent, content, "Content mismatch")
}

func TestVirtualFS_CopyMoveDelete(t *testing.T) {
	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Create a virtual file to work with
	originalContent := []byte("<?php echo 'test file'; ?>")
	err = vfs.CreateVirtualFile("/original.php", originalContent)
	assert.NoError(t, err, "Failed to create virtual file")

	// Test copying the file
	err = vfs.CopyFile("/original.php", "/copy.php")
	assert.NoError(t, err, "Failed to copy file")

	// Check the copied file
	copiedContent, err := vfs.GetFileContent("/copy.php")
	assert.NoError(t, err, "Failed to get copied file content")
	assert.Equal(t, originalContent, copiedContent, "Copied content mismatch")

	// Test moving the file
	err = vfs.MoveFile("/copy.php", "/moved.php")
	assert.NoError(t, err, "Failed to move file")

	// Check the moved file exists
	movedContent, err := vfs.GetFileContent("/moved.php")
	assert.NoError(t, err, "Failed to get moved file content")
	assert.Equal(t, originalContent, movedContent, "Moved content mismatch")

	// Verify the copy no longer exists
	_, err = vfs.GetFileContent("/copy.php")
	assert.Error(t, err, "Copy should no longer exist")

	// Test deleting the file
	err = vfs.DeleteFile("/moved.php")
	assert.NoError(t, err, "Failed to delete file")

	// Verify the moved file no longer exists
	_, err = vfs.GetFileContent("/moved.php")
	assert.Error(t, err, "Moved file should no longer exist")
}

func TestVirtualFS_AddEmbeddedFiles(t *testing.T) {
	// Create test dir for embed files testing
	testDir, err := os.MkdirTemp("", "frango-embed-test-")
	require.NoError(t, err, "Failed to create temp dir for embed tests")
	defer os.RemoveAll(testDir)

	// Ensure testdata directory exists
	err = os.MkdirAll("testdata/embed", 0755)
	require.NoError(t, err, "Failed to create testdata directory")

	// Create a test PHP file in the embed directory if it doesn't exist
	testFile := "testdata/embed/test.php"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		err = os.WriteFile(testFile, []byte("<?php echo 'embedded file'; ?>"), 0644)
		require.NoError(t, err, "Failed to create test embed file")
	}

	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Add embedded file
	err = vfs.AddEmbeddedFile(testEmbedFiles, "testdata/embed/test.php", "/embedded.php")
	assert.NoError(t, err, "Failed to add embedded file")

	// Verify file is in VFS
	paths := vfs.ListFiles()
	assert.Contains(t, paths, "/embedded.php", "Embedded file should be in VFS")

	// Check content access
	content, err := vfs.GetFileContent("/embedded.php")
	assert.NoError(t, err, "Failed to get file content")

	// Read expected content directly and trim whitespace for comparison
	expectedContent, err := os.ReadFile("testdata/embed/test.php")
	require.NoError(t, err, "Failed to read test.php file")

	// Compare trimmed content
	assert.Equal(t, strings.TrimSpace(string(expectedContent)), strings.TrimSpace(string(content)), "Content mismatch after trimming whitespace")
}

func TestVirtualFS_FileOriginTracking(t *testing.T) {
	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Add files of different origins
	sourcePath := filepath.Join(tempDir, "source.php")
	err = vfs.AddSourceFile(sourcePath, "/source.php")
	assert.NoError(t, err)

	err = vfs.CreateVirtualFile("/virtual.php", []byte("<?php echo 'virtual'; ?>"))
	assert.NoError(t, err)

	// Ensure testdata directory exists
	err = os.MkdirAll("testdata/embed", 0755)
	require.NoError(t, err, "Failed to create testdata directory")

	// Create a test PHP file in the embed directory if it doesn't exist
	testFile := "testdata/embed/test.php"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		err = os.WriteFile(testFile, []byte("<?php echo 'embedded file'; ?>"), 0644)
		require.NoError(t, err, "Failed to create test embed file")
	}

	err = vfs.AddEmbeddedFile(testEmbedFiles, "testdata/embed/test.php", "/embedded.php")
	assert.NoError(t, err)

	// Resolve paths to check origin tracking
	sourceFSPath := vfs.resolvePath("/source.php")
	virtualFSPath := vfs.resolvePath("/virtual.php")
	embeddedFSPath := vfs.resolvePath("/embedded.php")

	// Source should resolve to the original file path
	assert.Equal(t, sourcePath, sourceFSPath, "Source path should match original")

	// Virtual and embedded should resolve to temp paths
	assert.NotEqual(t, "", virtualFSPath, "Virtual path should resolve to temp file")
	assert.NotEqual(t, "", embeddedFSPath, "Embedded path should resolve to temp file")
	assert.NotEqual(t, sourceFSPath, virtualFSPath, "Source and virtual paths should be different")
	assert.NotEqual(t, sourceFSPath, embeddedFSPath, "Source and embedded paths should be different")
}

func TestVirtualFS_FileChangeTracking(t *testing.T) {
	// Skip in CI since file watching can be flaky
	if os.Getenv("CI") != "" {
		t.Skip("Skipping file watching test in CI environment")
	}

	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	sourcePath := filepath.Join(tempDir, "source.php")

	// Create middleware with dev mode enabled
	m, err := New(WithDevelopmentMode(true))
	require.NoError(t, err)
	defer m.Shutdown()

	vfs := m.NewFS()

	// Add source file
	err = vfs.AddSourceFile(sourcePath, "/watched.php")
	assert.NoError(t, err)

	// Get initial content
	origContent, err := vfs.GetFileContent("/watched.php")
	assert.NoError(t, err)
	assert.Equal(t, "<?php echo 'source file'; ?>", string(origContent))

	// Modify the file
	newContent := []byte("<?php echo 'modified file'; ?>")
	err = os.WriteFile(sourcePath, newContent, 0644)
	assert.NoError(t, err)

	// Wait for file watcher (this is a bit fragile but necessary for the test)
	time.Sleep(1 * time.Second)

	// Check that VFS detected the change
	assert.True(t, vfs.invalidated, "VFS should be marked as invalidated")
	assert.True(t, vfs.invalidatedPaths["/watched.php"], "File should be marked as invalidated")

	// Force refresh and check updated content
	vfs.refreshIfNeeded("/watched.php")

	// Get updated content
	updatedContent, err := vfs.GetFileContent("/watched.php")
	assert.NoError(t, err)
	assert.Equal(t, string(newContent), string(updatedContent), "Content should be updated")
}

// Test creating a handler from VFS
func TestVirtualFS_For(t *testing.T) {
	// Create middleware and VFS
	m, err := New()
	require.NoError(t, err, "Failed to create middleware")
	defer m.Shutdown()

	vfs := m.NewFS()

	// Create a virtual PHP file
	phpContent := []byte(`<?php 
	header("Content-Type: text/plain");
	echo "Test successful";
	?>`)

	err = vfs.CreateVirtualFile("/test.php", phpContent)
	assert.NoError(t, err)

	// Create handler
	handler := vfs.For("/test.php")
	assert.NotNil(t, handler, "Handler should not be nil")

	// Note: We can't easily test the handler execution as it requires FrankenPHP
	// This would typically be integration tested
}
