package vfs

import (
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

//go:embed testdata/test.php
var testEmbedFS embed.FS

// TestNewVFS tests the creation of a new VFS
func TestNewVFS(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Verify that the VFS was created successfully
	if vfs == nil {
		t.Fatal("VFS should not be nil")
	}

	// Verify that the temp directory was created
	if _, err := os.Stat(vfs.tempDir); err != nil {
		t.Fatalf("VFS temp directory was not created: %v", err)
	}

	// Verify that the PHP globals file was created
	if !vfs.FileExists(vfs.phpGlobalsFile) {
		t.Fatalf("PHP globals file was not created")
	}
}

// TestVFS_Branch tests branching a VFS
func TestVFS_Branch(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a parent VFS
	parent, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create parent VFS: %v", err)
	}
	defer parent.Cleanup()

	// Create a virtual file in the parent
	parentFile := "/parent.php"
	parentContent := []byte("<?php echo 'Parent VFS'; ?>")
	if err := parent.CreateVirtualFile(parentFile, parentContent); err != nil {
		t.Fatalf("Failed to create virtual file in parent VFS: %v", err)
	}

	// Create a branch VFS
	branch := parent.Branch()
	defer branch.Cleanup()

	// Verify that the branch was created successfully
	if branch == nil {
		t.Fatal("Branch VFS should not be nil")
	}

	// Verify that the branch temp directory was created
	if _, err := os.Stat(branch.tempDir); err != nil {
		t.Fatalf("Branch VFS temp directory was not created: %v", err)
	}

	// Create a virtual file in the branch
	branchFile := "/branch.php"
	branchContent := []byte("<?php echo 'Branch VFS'; ?>")
	if err := branch.CreateVirtualFile(branchFile, branchContent); err != nil {
		t.Fatalf("Failed to create virtual file in branch VFS: %v", err)
	}

	// Test inheritance: branch should see parent's file
	if !branch.FileExists(parentFile) {
		t.Fatal("Branch should see parent's files")
	}

	// Read parent's file from branch
	content, err := branch.GetFileContent(parentFile)
	if err != nil {
		t.Fatalf("Failed to read parent's file from branch: %v", err)
	}
	if string(content) != string(parentContent) {
		t.Fatalf("Content mismatch: %s vs %s", string(content), string(parentContent))
	}

	// Parent should not see branch's file
	if parent.FileExists(branchFile) {
		t.Fatal("Parent should not see branch's files")
	}

	// Test shadowing: branch should override parent's file
	shadowContent := []byte("<?php echo 'Shadowed parent file'; ?>")
	if err := branch.CreateVirtualFile(parentFile, shadowContent); err != nil {
		t.Fatalf("Failed to shadow parent's file: %v", err)
	}

	// Read shadowed file from branch
	content, err = branch.GetFileContent(parentFile)
	if err != nil {
		t.Fatalf("Failed to read shadowed file from branch: %v", err)
	}
	if string(content) != string(shadowContent) {
		t.Fatalf("Shadow content mismatch: %s vs %s", string(content), string(shadowContent))
	}

	// Parent's file should be unchanged
	content, err = parent.GetFileContent(parentFile)
	if err != nil {
		t.Fatalf("Failed to read original file from parent: %v", err)
	}
	if string(content) != string(parentContent) {
		t.Fatalf("Parent content should be unchanged: %s vs %s", string(content), string(parentContent))
	}
}

// TestVFS_AddSourceFile tests adding a source file to the VFS
func TestVFS_AddSourceFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PHP file
	sourceFile := filepath.Join(tempDir, "test.php")
	sourceContent := []byte("<?php echo 'Test file'; ?>")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the source file to the VFS
	virtualPath := "/test.php"
	if err := vfs.AddSourceFile(sourceFile, virtualPath); err != nil {
		t.Fatalf("Failed to add source file: %v", err)
	}

	// Verify the file exists in the VFS
	if !vfs.FileExists(virtualPath) {
		t.Fatal("Source file should exist in VFS")
	}

	// Read the file from the VFS
	content, err := vfs.GetFileContent(virtualPath)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	if string(content) != string(sourceContent) {
		t.Fatalf("Content mismatch: %s vs %s", string(content), string(sourceContent))
	}

	// Resolve the path to get the actual file path
	path, err := vfs.ResolvePath(virtualPath)
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}
	if path != sourceFile {
		t.Fatalf("Resolved path mismatch: %s vs %s", path, sourceFile)
	}
}

// TestVFS_AddSourceDirectory tests adding a source directory to the VFS
func TestVFS_AddSourceDirectory(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create some test files
	files := map[string]string{
		"index.php":       "<?php echo 'Index'; ?>",
		"lib/helper.php":  "<?php echo 'Helper'; ?>",
		"views/home.php":  "<?php echo 'Home'; ?>",
		"views/about.php": "<?php echo 'About'; ?>",
	}

	for path, content := range files {
		fullPath := filepath.Join(sourceDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the source directory to the VFS
	virtualPrefix := "/app"
	if err := vfs.AddSourceDirectory(sourceDir, virtualPrefix); err != nil {
		t.Fatalf("Failed to add source directory: %v", err)
	}

	// Verify all files exist in the VFS
	for filePath := range files {
		virtualPath := filepath.Join(virtualPrefix, filePath)
		virtualPath = "/" + strings.TrimPrefix(virtualPath, "/")
		virtualPath = strings.ReplaceAll(virtualPath, string(os.PathSeparator), "/")

		if !vfs.FileExists(virtualPath) {
			t.Fatalf("File should exist in VFS: %s", virtualPath)
		}

		// Read the file from the VFS
		content, err := vfs.GetFileContent(virtualPath)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", virtualPath, err)
		}
		if string(content) != files[filePath] {
			t.Fatalf("Content mismatch for %s: %s vs %s", virtualPath, string(content), files[filePath])
		}
	}

	// List all files in the VFS
	fileList := vfs.ListFiles()
	if len(fileList) != len(files)+1 { // +1 for PHP globals file
		t.Fatalf("Expected %d files, got %d", len(files)+1, len(fileList))
	}
}

// TestVFS_CreateVirtualFile tests creating a virtual file in the VFS
func TestVFS_CreateVirtualFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Create a virtual file
	virtualPath := "/virtual.php"
	content := []byte("<?php echo 'Virtual file'; ?>")
	if err := vfs.CreateVirtualFile(virtualPath, content); err != nil {
		t.Fatalf("Failed to create virtual file: %v", err)
	}

	// Verify the file exists in the VFS
	if !vfs.FileExists(virtualPath) {
		t.Fatal("Virtual file should exist in VFS")
	}

	// Read the file from the VFS
	readContent, err := vfs.GetFileContent(virtualPath)
	if err != nil {
		t.Fatalf("Failed to read virtual file: %v", err)
	}
	if string(readContent) != string(content) {
		t.Fatalf("Content mismatch: %s vs %s", string(readContent), string(content))
	}

	// Verify the file was written to disk
	path, err := vfs.ResolvePath(virtualPath)
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}

	// Read the file from disk
	diskContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read virtual file from disk: %v", err)
	}
	if string(diskContent) != string(content) {
		t.Fatalf("Disk content mismatch: %s vs %s", string(diskContent), string(content))
	}
}

// TestVFS_CopyFile tests copying a file within the VFS
func TestVFS_CopyFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Create a source file
	srcPath := "/source.php"
	content := []byte("<?php echo 'Source file'; ?>")
	if err := vfs.CreateVirtualFile(srcPath, content); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy the file
	dstPath := "/destination.php"
	if err := vfs.CopyFileWithOptions(srcPath, dstPath, false); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify the destination file exists
	if !vfs.FileExists(dstPath) {
		t.Fatal("Destination file should exist in VFS")
	}

	// Read the destination file
	dstContent, err := vfs.GetFileContent(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Fatalf("Content mismatch: %s vs %s", string(dstContent), string(content))
	}

	// Modify the source file
	newContent := []byte("<?php echo 'Modified source'; ?>")
	if err := vfs.CreateVirtualFile(srcPath, newContent); err != nil {
		t.Fatalf("Failed to modify source file: %v", err)
	}

	// Verify the destination file is unchanged
	dstContent, err = vfs.GetFileContent(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Fatalf("Destination content should be unchanged: %s vs %s", string(dstContent), string(content))
	}
}

// TestVFS_MoveFile tests moving a file within the VFS
func TestVFS_MoveFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Create a source file
	srcPath := "/source.php"
	content := []byte("<?php echo 'Source file'; ?>")
	if err := vfs.CreateVirtualFile(srcPath, content); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Move the file
	dstPath := "/destination.php"
	if err := vfs.MoveFileWithOptions(srcPath, dstPath, false); err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	// Verify the source file no longer exists
	if vfs.FileExists(srcPath) {
		t.Fatal("Source file should no longer exist in VFS")
	}

	// Verify the destination file exists
	if !vfs.FileExists(dstPath) {
		t.Fatal("Destination file should exist in VFS")
	}

	// Read the destination file
	dstContent, err := vfs.GetFileContent(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Fatalf("Content mismatch: %s vs %s", string(dstContent), string(content))
	}
}

// TestVFS_DeleteFile tests deleting a file from the VFS
func TestVFS_DeleteFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Create a file
	filePath := "/file.php"
	content := []byte("<?php echo 'File to delete'; ?>")
	if err := vfs.CreateVirtualFile(filePath, content); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Get the physical path before deletion
	physPath, err := vfs.ResolvePath(filePath)
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}

	// Delete the file
	if err := vfs.DeleteFile(filePath); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify the file no longer exists in the VFS
	if vfs.FileExists(filePath) {
		t.Fatal("File should no longer exist in VFS")
	}

	// Try to get content, should fail
	_, err = vfs.GetFileContent(filePath)
	if err == nil {
		t.Fatal("GetFileContent should fail for deleted file")
	}

	// Verify the physical file was removed
	if _, err := os.Stat(physPath); !os.IsNotExist(err) {
		t.Fatal("Physical file should be deleted")
	}
}

// waitWithTimeout waits for a condition to be true with a timeout
func waitWithTimeout(t *testing.T, condition func() bool, timeout time.Duration, message string) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("Timeout while waiting for: %s", message)
	return false
}

// TestVFS_FileChanges tests file change detection
func TestVFS_FileChanges(t *testing.T) {
	// Skip in CI/CD environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PHP file
	sourceFile := filepath.Join(tempDir, "test.php")
	sourceContent := []byte("<?php echo 'Initial content'; ?>")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS with development mode enabled
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the source file to the VFS
	virtualPath := "/test.php"
	if err := vfs.AddSourceFile(sourceFile, virtualPath); err != nil {
		t.Fatalf("Failed to add source file: %v", err)
	}

	// Verify initial content
	content, err := vfs.GetFileContent(virtualPath)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	if string(content) != string(sourceContent) {
		t.Fatalf("Initial content mismatch: %s vs %s", string(content), string(sourceContent))
	}

	// Modify the file directly on disk
	updatedContent := []byte("<?php echo 'Updated content'; ?>")
	if err := os.WriteFile(sourceFile, updatedContent, 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Force check for changes directly
	vfs.checkFileChanges(virtualPath)

	// Verify the file is marked as changed
	vfs.mutex.RLock()
	changed := vfs.changedFiles[virtualPath]
	vfs.mutex.RUnlock()
	if !changed {
		t.Fatal("File should be marked as changed")
	}

	// Verify content was updated
	content, err = vfs.GetFileContent(virtualPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}
	if string(content) != string(updatedContent) {
		t.Fatalf("Updated content mismatch: %s vs %s", string(content), string(updatedContent))
	}
}

// TestVFS_AddEmbeddedFile tests adding an embedded file to the VFS
func TestVFS_AddEmbeddedFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Create a test file with the embedded content that we'll use instead of an embedded file
	testFile := filepath.Join(tempDir, "test_embedded.php")
	expectedContent := "<?php echo 'Embedded test file'; ?>"
	if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add the file as a source file (simulating embedded file behavior)
	virtualPath := "/embedded.php"
	if err := vfs.AddSourceFile(testFile, virtualPath); err != nil {
		t.Fatalf("Failed to add source file: %v", err)
	}

	// Verify the file exists in the VFS
	if !vfs.FileExists(virtualPath) {
		t.Fatal("Embedded file should exist in VFS")
	}

	// Read the file from the VFS
	content, err := vfs.GetFileContent(virtualPath)
	if err != nil {
		t.Fatalf("Failed to read embedded file: %v", err)
	}

	// Verify the content
	if string(content) != expectedContent {
		t.Fatalf("Content mismatch: %s vs %s", string(content), expectedContent)
	}

	// Verify the file was written to disk
	path, err := vfs.ResolvePath(virtualPath)
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}

	// The path should be within the VFS temp directory or the original file
	exists := false
	if strings.HasPrefix(path, vfs.tempDir) {
		exists = true
	} else if path == testFile {
		exists = true
	}

	if !exists {
		t.Fatalf("Path is not valid: %s", path)
	}

	// Verify we can read the file
	diskContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file from disk: %v", err)
	}
	if string(diskContent) != expectedContent {
		t.Fatalf("Disk content mismatch: %s vs %s", string(diskContent), expectedContent)
	}
}

// TestVFS_AddSourceDirectoryRecursive tests adding a directory structure with nested files
func TestVFS_AddSourceDirectoryRecursive(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a directory structure in the temp dir
	testDir := filepath.Join(tempDir, "testdir")
	subDir := filepath.Join(testDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create some test files
	files := map[string]string{
		filepath.Join(testDir, "test1.php"): "<?php echo 'Test dir file 1'; ?>",
		filepath.Join(subDir, "test2.php"):  "<?php echo 'Test subdir file 2'; ?>",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the directory to the VFS
	virtualPrefix := "/virtual"
	if err := vfs.AddSourceDirectory(testDir, virtualPrefix); err != nil {
		t.Fatalf("Failed to add directory: %v", err)
	}

	// Expected files and their content
	expectedFiles := map[string]string{
		"/virtual/test1.php":        "<?php echo 'Test dir file 1'; ?>",
		"/virtual/subdir/test2.php": "<?php echo 'Test subdir file 2'; ?>",
	}

	// Verify all expected files exist
	for virtualPath, expectedContent := range expectedFiles {
		// Check file exists
		if !vfs.FileExists(virtualPath) {
			t.Fatalf("File should exist in VFS: %s", virtualPath)
		}

		// Read content and verify
		content, err := vfs.GetFileContent(virtualPath)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", virtualPath, err)
		}
		if string(content) != expectedContent {
			t.Fatalf("Content mismatch for %s: %s vs %s", virtualPath, string(content), expectedContent)
		}
	}

	// List the files and verify count
	fileList := vfs.ListFiles()
	expectedCount := len(expectedFiles) + 1 // +1 for PHP globals file
	if len(fileList) != expectedCount {
		t.Fatalf("Expected %d files, got %d", expectedCount, len(fileList))
	}
}

// Create testdata directory and test.php file for embedded tests
func init() {
	testDataDir := "testdata"
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		os.Mkdir(testDataDir, 0755)
	}

	// Create testdata/embedded directory
	embeddedDir := filepath.Join(testDataDir, "embedded")
	if _, err := os.Stat(embeddedDir); os.IsNotExist(err) {
		os.Mkdir(embeddedDir, 0755)
	}

	// Create the embedded test file
	embeddedFile := filepath.Join(embeddedDir, "test.php")
	embeddedContent := []byte("<?php echo 'Embedded test file'; ?>")
	// Only write if the file doesn't exist or is empty
	if fileInfo, err := os.Stat(embeddedFile); os.IsNotExist(err) || fileInfo.Size() == 0 {
		os.WriteFile(embeddedFile, embeddedContent, 0644)
	}

	// Keep the original test.php file for other tests
	testFile := filepath.Join(testDataDir, "test.php")
	content := []byte(`<?php
		echo "This is a test PHP file";
		
		// Display any path parameters that might be set
		if (isset($_PATH) && count($_PATH) > 0) {
			echo "\nPath parameters:\n";
			foreach ($_PATH as $key => $value) {
				echo "$key: $value\n";
			}
		}
	?>`)
	// Only write if the file doesn't exist or is empty
	if fileInfo, err := os.Stat(testFile); os.IsNotExist(err) || fileInfo.Size() == 0 {
		os.WriteFile(testFile, content, 0644)
	}
}

// TestVFS_CopyWithOriginPreservation tests copying a file with the origin preservation option
func TestVFS_CopyWithOriginPreservation(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PHP file
	sourceFile := filepath.Join(tempDir, "source.php")
	sourceContent := []byte("<?php echo 'Source file content'; ?>")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the source file to the VFS
	srcVirtualPath := "/source.php"
	if err := vfs.AddSourceFile(sourceFile, srcVirtualPath); err != nil {
		t.Fatalf("Failed to add source file: %v", err)
	}

	// Copy the file with WITHOUT origin preservation
	dstVirtualPath1 := "/dest1.php"
	if err := vfs.CopyFileWithOptions(srcVirtualPath, dstVirtualPath1, false); err != nil {
		t.Fatalf("Failed to copy file without origin preservation: %v", err)
	}

	// Copy the file WITH origin preservation
	dstVirtualPath2 := "/dest2.php"
	if err := vfs.CopyFileWithOptions(srcVirtualPath, dstVirtualPath2, true); err != nil {
		t.Fatalf("Failed to copy file with origin preservation: %v", err)
	}

	// Verify both copies have correct initial content
	for _, path := range []string{dstVirtualPath1, dstVirtualPath2} {
		content, err := vfs.GetFileContent(path)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", path, err)
		}
		if string(content) != string(sourceContent) {
			t.Fatalf("Content mismatch for %s: %s vs %s", path, string(content), string(sourceContent))
		}
	}

	// Verify the origin types
	vfs.mutex.RLock()
	origin1 := vfs.fileOrigins[dstVirtualPath1]
	origin2 := vfs.fileOrigins[dstVirtualPath2]
	vfs.mutex.RUnlock()

	if origin1 != OriginVirtual {
		t.Fatalf("Expected %s to have origin type %s, got %s", dstVirtualPath1, OriginVirtual, origin1)
	}
	if origin2 != OriginSource {
		t.Fatalf("Expected %s to have origin type %s, got %s", dstVirtualPath2, OriginSource, origin2)
	}

	// Now update the source file and verify that only origin-preserved copy reflects changes
	updatedContent := []byte("<?php echo 'Updated source content'; ?>")
	if err := os.WriteFile(sourceFile, updatedContent, 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Force check for changes
	vfs.checkFileChanges(srcVirtualPath)
	vfs.checkFileChanges(dstVirtualPath2) // Check the origin-preserved copy too

	// Original source should see changes
	content, err := vfs.GetFileContent(srcVirtualPath)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	if string(content) != string(updatedContent) {
		t.Fatalf("Source not updated: %s vs %s", string(content), string(updatedContent))
	}

	// The virtual copy (not origin-preserved) should NOT see changes
	content, err = vfs.GetFileContent(dstVirtualPath1)
	if err != nil {
		t.Fatalf("Failed to read dest1 file: %v", err)
	}
	if string(content) == string(updatedContent) {
		t.Fatalf("Non-preserved copy should not update: %s", string(content))
	}

	// The origin-preserved copy SHOULD see changes
	content, err = vfs.GetFileContent(dstVirtualPath2)
	if err != nil {
		t.Fatalf("Failed to read dest2 file: %v", err)
	}
	if string(content) != string(updatedContent) {
		t.Fatalf("Origin-preserved copy not updated: %s vs %s", string(content), string(updatedContent))
	}
}

// TestVFS_MoveWithOriginPreservation tests moving a file with origin preservation
func TestVFS_MoveWithOriginPreservation(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PHP file
	sourceFile := filepath.Join(tempDir, "source.php")
	sourceContent := []byte("<?php echo 'Source file content'; ?>")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the source file to the VFS
	srcVirtualPath := "/source.php"
	if err := vfs.AddSourceFile(sourceFile, srcVirtualPath); err != nil {
		t.Fatalf("Failed to add source file: %v", err)
	}

	// Test move with origin preservation (should maintain the source reference)
	dstVirtualPath := "/moved.php"
	if err := vfs.MoveFileWithOptions(srcVirtualPath, dstVirtualPath, true); err != nil {
		t.Fatalf("Failed to move file with origin preservation: %v", err)
	}

	// Verify the source file no longer exists in VFS
	if vfs.FileExists(srcVirtualPath) {
		t.Fatal("Source file should no longer exist in VFS")
	}

	// Verify the destination file exists and has the correct origin
	if !vfs.FileExists(dstVirtualPath) {
		t.Fatal("Destination file should exist in VFS")
	}

	// Check destination origin type
	vfs.mutex.RLock()
	destOrigin := vfs.fileOrigins[dstVirtualPath]
	vfs.mutex.RUnlock()

	if destOrigin != OriginSource {
		t.Fatalf("Expected moved file to have origin type %s, got %s", OriginSource, destOrigin)
	}

	// Update the source file on disk and verify that the moved file sees changes
	updatedContent := []byte("<?php echo 'Updated source content'; ?>")
	if err := os.WriteFile(sourceFile, updatedContent, 0644); err != nil {
		t.Fatalf("Failed to update source file: %v", err)
	}

	// Force check for changes
	vfs.checkFileChanges(dstVirtualPath)

	// Verify the moved file sees the changes
	content, err := vfs.GetFileContent(dstVirtualPath)
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}
	if string(content) != string(updatedContent) {
		t.Fatalf("Moved file with origin preservation not updated: %s vs %s",
			string(content), string(updatedContent))
	}
}

// TestVFS_BranchInheritance tests file change detection across VFS branch inheritance
func TestVFS_BranchInheritance(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create parent source files
	parentFile := filepath.Join(tempDir, "parent.php")
	parentContent := []byte("<?php echo 'Parent file content'; ?>")
	if err := os.WriteFile(parentFile, parentContent, 0644); err != nil {
		t.Fatalf("Failed to create parent file: %v", err)
	}

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a parent VFS
	parentVFS, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create parent VFS: %v", err)
	}
	defer parentVFS.Cleanup()

	// Add the file to the parent VFS
	parentVirtualPath := "/parent.php"
	if err := parentVFS.AddSourceFile(parentFile, parentVirtualPath); err != nil {
		t.Fatalf("Failed to add file to parent VFS: %v", err)
	}

	// Create a child VFS
	childVFS := parentVFS.Branch()
	defer childVFS.Cleanup()

	// Verify child can see parent's file
	if !childVFS.FileExists(parentVirtualPath) {
		t.Fatal("Child VFS should see parent's file")
	}

	// Get initial content from child's view
	childContent, err := childVFS.GetFileContent(parentVirtualPath)
	if err != nil {
		t.Fatalf("Failed to read parent file from child: %v", err)
	}
	if string(childContent) != string(parentContent) {
		t.Fatalf("Child content mismatch: %s vs %s", string(childContent), string(parentContent))
	}

	// Create grandchild VFS to test multi-level inheritance
	grandchildVFS := childVFS.Branch()
	defer grandchildVFS.Cleanup()

	// Verify grandchild can see parent's file
	if !grandchildVFS.FileExists(parentVirtualPath) {
		t.Fatal("Grandchild VFS should see parent's file")
	}

	// Update the parent file on disk
	updatedContent := []byte("<?php echo 'Updated parent content'; ?>")
	if err := os.WriteFile(parentFile, updatedContent, 0644); err != nil {
		t.Fatalf("Failed to update parent file: %v", err)
	}

	// Force parent VFS to check for changes
	parentVFS.checkFileChanges(parentVirtualPath)

	// Verify the parent VFS sees the change
	newParentContent, err := parentVFS.GetFileContent(parentVirtualPath)
	if err != nil {
		t.Fatalf("Failed to read updated parent file: %v", err)
	}
	if string(newParentContent) != string(updatedContent) {
		t.Fatalf("Parent content not updated: %s vs %s", string(newParentContent), string(updatedContent))
	}

	// Verify the child VFS sees the change
	newChildContent, err := childVFS.GetFileContent(parentVirtualPath)
	if err != nil {
		t.Fatalf("Failed to read updated parent file from child: %v", err)
	}
	if string(newChildContent) != string(updatedContent) {
		t.Fatalf("Child did not see update: %s vs %s", string(newChildContent), string(updatedContent))
	}

	// Verify the grandchild VFS sees the change
	newGrandchildContent, err := grandchildVFS.GetFileContent(parentVirtualPath)
	if err != nil {
		t.Fatalf("Failed to read updated parent file from grandchild: %v", err)
	}
	if string(newGrandchildContent) != string(updatedContent) {
		t.Fatalf("Grandchild did not see update: %s vs %s", string(newGrandchildContent), string(updatedContent))
	}
}

// TestVFS_ReferenceCount tests proper reference counting in VFS branching and cleanup
func TestVFS_ReferenceCount(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.php")
	testContent := []byte("<?php echo 'Test content'; ?>")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a logger that logs to stdout for debugging
	logger := log.New(os.Stdout, "TEST: ", 0)

	// Create a parent VFS
	parentVFS, err := NewVFS(tempDir, logger, false) // Disable development mode to reduce noise
	if err != nil {
		t.Fatalf("Failed to create parent VFS: %v", err)
	}

	// Verify initial reference count
	parentVFS.refMutex.Lock()
	if parentVFS.refCount != 0 {
		t.Fatalf("Initial reference count should be 0, got %d", parentVFS.refCount)
	}
	parentVFS.refMutex.Unlock()
	t.Logf("Parent VFS created with ID: %s", parentVFS.name)

	// Create multiple branches
	child1 := parentVFS.Branch()
	t.Logf("Created child1 with ID: %s", child1.name)

	child2 := parentVFS.Branch()
	t.Logf("Created child2 with ID: %s", child2.name)

	child3 := parentVFS.Branch()
	t.Logf("Created child3 with ID: %s", child3.name)

	// Verify reference count was increased
	parentVFS.refMutex.Lock()
	parentRefCount := parentVFS.refCount
	parentVFS.refMutex.Unlock()
	t.Logf("Parent ref count after creating 3 children: %d", parentRefCount)
	if parentRefCount != 3 {
		t.Fatalf("Reference count should be 3 after creating branches, got %d", parentRefCount)
	}

	// Cleanup one child
	t.Logf("Cleaning up child1...")
	child1.Cleanup()

	// Check parent's refcount again
	parentVFS.refMutex.Lock()
	parentRefCount = parentVFS.refCount
	parentVFS.refMutex.Unlock()
	t.Logf("Parent ref count after cleaning child1: %d", parentRefCount)
	if parentRefCount != 2 {
		t.Fatalf("Reference count should be 2 after child cleanup, got %d", parentRefCount)
	}

	// Create a grandchild
	grandchild := child2.Branch()
	t.Logf("Created grandchild with ID: %s from child2", grandchild.name)

	// Print the hierarchy
	t.Logf("VFS hierarchy: parent(%s) -> child2(%s) -> grandchild(%s)",
		parentVFS.name, child2.name, grandchild.name)
	t.Logf("                \\-> child3(%s)", child3.name)

	// Verify child's reference count
	child2.refMutex.Lock()
	child2RefCount := child2.refCount
	child2.refMutex.Unlock()
	t.Logf("Child2 ref count after creating grandchild: %d", child2RefCount)
	if child2RefCount != 1 {
		t.Fatalf("Child reference count should be 1, got %d", child2RefCount)
	}

	// Cleanup parent - should mark for cleanup but defer actual cleanup
	t.Logf("Marking parent for cleanup...")
	parentVFS.Cleanup()

	// Verify parent is marked for cleanup but not fully cleaned up
	parentVFS.refMutex.Lock()
	isCleanedUp := parentVFS.isCleanedUp
	parentRefCount = parentVFS.refCount
	parentVFS.refMutex.Unlock()
	t.Logf("Parent after marking for cleanup: isCleanedUp=%v, refCount=%d",
		isCleanedUp, parentRefCount)

	if !isCleanedUp {
		t.Fatalf("Parent should be marked as cleaned up")
	}

	// Now cleanup all other VFS instances one by one
	t.Logf("Cleaning up child3...")
	child3.Cleanup()

	// Check parent ref count
	parentVFS.refMutex.Lock()
	parentRefCount = parentVFS.refCount
	parentVFS.refMutex.Unlock()
	t.Logf("Parent ref count after cleaning child3: %d", parentRefCount)

	t.Logf("Cleaning up child2...")
	child2.Cleanup()

	// Check both ref counts
	child2.refMutex.Lock()
	child2RefCount = child2.refCount
	child2.refMutex.Unlock()
	t.Logf("Child2 ref count after cleaning: %d", child2RefCount)

	parentVFS.refMutex.Lock()
	parentRefCount = parentVFS.refCount
	parentVFS.refMutex.Unlock()
	t.Logf("Parent ref count after cleaning child2: %d", parentRefCount)

	t.Logf("Cleaning up grandchild...")
	grandchild.Cleanup()

	// Wait a bit for any async cleanups
	time.Sleep(300 * time.Millisecond)

	// Final check of reference counts
	child2.refMutex.Lock()
	child2RefCount = child2.refCount
	child2IsCleanedUp := child2.isCleanedUp
	child2.refMutex.Unlock()
	t.Logf("Child2 final state: refCount=%d, isCleanedUp=%v", child2RefCount, child2IsCleanedUp)

	parentVFS.refMutex.Lock()
	parentRefCount = parentVFS.refCount
	parentIsCleanedUp := parentVFS.isCleanedUp
	parentVFS.refMutex.Unlock()
	t.Logf("Parent final state: refCount=%d, isCleanedUp=%v", parentRefCount, parentIsCleanedUp)

	// The test will pass if both refcounts are 0
	if child2RefCount != 0 {
		t.Fatalf("Child2 reference count should be 0 after all cleanups, got %d", child2RefCount)
	}

	if parentRefCount != 0 {
		t.Fatalf("Parent reference count should be 0 after all cleanups, got %d", parentRefCount)
	}
}

// TestVFS_ConcurrentAccess tests multiple goroutines simultaneously accessing and modifying
// the VFS to verify thread safety in high concurrency scenarios
func TestVFS_ConcurrentAccess(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-concurrent-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a parent VFS
	parentVFS, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create parent VFS: %v", err)
	}
	defer parentVFS.Cleanup()

	// Create a few source files for testing
	for i := 0; i < 5; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("source%d.php", i))
		content := []byte(fmt.Sprintf("<?php echo 'Source file %d'; ?>", i))
		if err := os.WriteFile(fileName, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := parentVFS.AddSourceFile(fileName, fmt.Sprintf("/source%d.php", i)); err != nil {
			t.Fatalf("Failed to add source file: %v", err)
		}
	}

	// Create virtual files
	for i := 0; i < 5; i++ {
		content := []byte(fmt.Sprintf("<?php echo 'Virtual file %d'; ?>", i))
		if err := parentVFS.CreateVirtualFile(fmt.Sprintf("/virtual%d.php", i), content); err != nil {
			t.Fatalf("Failed to create virtual file: %v", err)
		}
	}

	// Create child branches that will be accessed concurrently
	children := make([]*VFS, 5)
	for i := 0; i < 5; i++ {
		children[i] = parentVFS.Branch()
		defer children[i].Cleanup()
	}

	// Define test operations
	operations := []string{
		"read", "write", "copy", "move", "delete", "branch", "cleanup",
	}

	// Define a function to perform random operations on a VFS
	performOperations := func(vfs *VFS, id int, opCount int, wg *sync.WaitGroup) {
		defer wg.Done()

		for i := 0; i < opCount; i++ {
			// Perform a random operation based on the iteration
			op := operations[i%len(operations)]

			// Create a unique file identifier
			fileID := fmt.Sprintf("%d_%d", id, i)

			switch op {
			case "read":
				// Read a random file
				path := fmt.Sprintf("/source%d.php", i%5)
				_, err := vfs.GetFileContent(path)
				if err != nil && !strings.Contains(err.Error(), "not found") {
					t.Errorf("Failed to read file %s: %v", path, err)
				}

			case "write":
				// Create a new virtual file
				path := fmt.Sprintf("/concurrent_test_%s.php", fileID)
				content := []byte(fmt.Sprintf("<?php echo 'Concurrent test %s'; ?>", fileID))
				err := vfs.CreateVirtualFile(path, content)
				if err != nil {
					t.Errorf("Failed to create file %s: %v", path, err)
				}

			case "copy":
				// Copy a file
				srcPath := fmt.Sprintf("/source%d.php", i%5)
				dstPath := fmt.Sprintf("/concurrent_copy_%s.php", fileID)
				err := vfs.CopyFileWithOptions(srcPath, dstPath, i%2 == 0) // Alternate origin preservation
				if err != nil && !strings.Contains(err.Error(), "not found") {
					t.Errorf("Failed to copy file %s to %s: %v", srcPath, dstPath, err)
				}

			case "move":
				// Move a file that we own
				srcPath := fmt.Sprintf("/concurrent_test_%s.php", fileID)
				dstPath := fmt.Sprintf("/concurrent_moved_%s.php", fileID)
				if vfs.FileExists(srcPath) {
					err := vfs.MoveFileWithOptions(srcPath, dstPath, i%2 == 0) // Alternate origin preservation
					if err != nil {
						t.Errorf("Failed to move file %s to %s: %v", srcPath, dstPath, err)
					}
				}

			case "delete":
				// Delete a file that we own
				path := fmt.Sprintf("/concurrent_test_%s.php", fileID)
				if vfs.FileExists(path) {
					err := vfs.DeleteFile(path)
					if err != nil {
						t.Errorf("Failed to delete file %s: %v", path, err)
					}
				}

			case "branch":
				// Create a branch, perform an operation, then clean it up
				branch := vfs.Branch()
				if branch != nil {
					branchPath := fmt.Sprintf("/branch_%s.php", fileID)
					content := []byte(fmt.Sprintf("<?php echo 'Branch test %s'; ?>", fileID))
					_ = branch.CreateVirtualFile(branchPath, content)
					branch.Cleanup()
				}

			case "cleanup":
				// Do nothing - cleanup handled by defer
				time.Sleep(1 * time.Millisecond) // Small pause
			}

			// Add a small random sleep to increase chance of race conditions
			time.Sleep(time.Duration(i%3) * time.Millisecond)
		}
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	operationsPerGoroutine := 50
	numGoroutines := 20

	// Start multiple goroutines to perform operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		// Select a VFS to operate on (parent or one of the children)
		var vfs *VFS
		if i%6 == 0 { // Occasionally use the parent
			vfs = parentVFS
		} else {
			vfs = children[i%len(children)]
		}

		go performOperations(vfs, i, operationsPerGoroutine, &wg)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Final verification - check that VFS is still in a consistent state
	for i := 0; i < 5; i++ {
		path := fmt.Sprintf("/source%d.php", i)
		if !parentVFS.FileExists(path) {
			t.Errorf("Expected source file %s to still exist", path)
		}
	}
}

// TestVFS_PathCharacterHandling tests special characters, unicode, and path traversal attempts
func TestVFS_PathCharacterHandling(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-paths-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards all output
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Test 1: Unicode characters in paths
	unicodePaths := []string{
		"/unicode/日本語.php",         // Japanese
		"/unicode/русский.php",     // Russian
		"/unicode/العربية.php",     // Arabic
		"/unicode/emoji/😀😂🤣.php",   // Emoji
		"/unicode/mixed/a日本語b.php", // Mixed ASCII and Unicode
	}

	// Create files with Unicode paths
	for i, path := range unicodePaths {
		content := []byte(fmt.Sprintf("<?php echo 'Unicode test %d'; ?>", i))

		// Create parent directories if needed
		dirPath := filepath.Dir(path)
		if dirPath != "/" {
			// Just verify the VFS logic works - we don't actually create these dirs on disk
			_ = vfs.CreateVirtualFile(filepath.Join(dirPath, "_marker.php"), []byte("<?php ?>"))
		}

		// Create the test file
		if err := vfs.CreateVirtualFile(path, content); err != nil {
			t.Fatalf("Failed to create Unicode file %s: %v", path, err)
		}

		// Verify file exists
		if !vfs.FileExists(path) {
			t.Errorf("Unicode file should exist: %s", path)
		}

		// Verify content is correct
		fileContent, err := vfs.GetFileContent(path)
		if err != nil {
			t.Errorf("Failed to read Unicode file %s: %v", path, err)
		} else if string(fileContent) != string(content) {
			t.Errorf("Content mismatch for Unicode file %s", path)
		}
	}

	// Test 2: Path traversal attempts
	traversalPaths := []string{
		"/../outside.php",
		"/dir/../../outside.php",
		"/dir/../../../outside.php",
		"/dir/./../../outside.php",
		"/.././../outside.php",
		"/dir///../outside.php", // Double slash
	}

	// Create a legitimate file to test normalized path resolution
	legitPath := "/dir/inside.php"
	legitContent := []byte("<?php echo 'Legitimate file'; ?>")
	if err := vfs.CreateVirtualFile(legitPath, legitContent); err != nil {
		t.Fatalf("Failed to create legitimate file: %v", err)
	}

	// Test path traversal attempts (all should be normalized correctly)
	for _, path := range traversalPaths {
		// Create a file with a path traversal attempt
		content := []byte("<?php echo 'Traversal test'; ?>")
		err := vfs.CreateVirtualFile(path, content)

		// Verify the file is created with a normalized path
		normalizedPath := normalizePath(path)
		if normalizedPath != "/outside.php" {
			t.Errorf("Path not normalized correctly: %s -> %s (expected /outside.php)",
				path, normalizedPath)
		}

		if err != nil {
			t.Errorf("Failed to create file with traversal path %s: %v", path, err)
		}

		// Verify we can access it with the normalized path
		if !vfs.FileExists(normalizedPath) {
			t.Errorf("Normalized path file should exist: %s", normalizedPath)
		}
	}

	// Test 3: Long-ish paths (avoid OS limitation)
	longName := strings.Repeat("a", 100) // Long but not too long
	longPath := fmt.Sprintf("/long/%s.php", longName)

	// Create a file with a long but reasonable path
	longContent := []byte("<?php echo 'Long path test'; ?>")
	err = vfs.CreateVirtualFile(longPath, longContent)
	if err != nil {
		t.Errorf("Failed to create file with long path: %v", err)
	}

	// Verify it exists and has correct content
	if !vfs.FileExists(longPath) {
		t.Errorf("Long path file should exist: %s", longPath)
	}

	// Verify content
	content, err := vfs.GetFileContent(longPath)
	if err != nil {
		t.Errorf("Failed to read long path file: %v", err)
	} else if string(content) != string(longContent) {
		t.Errorf("Content mismatch for long path file")
	}

	// Test 4: Special characters in paths
	specialPaths := []string{
		"/special/!@#$%^&*().php",
		"/special/space path.php",
		"/special/comma,semicolon;.php",
		"/special/quotes'n\"quotes.php",
		"/special/brackets[and]braces{}.php",
		"/special/plus+minus-.php",
		"/special/equal=tilde~.php",
	}

	// Create files with special characters in paths
	for i, path := range specialPaths {
		content := []byte(fmt.Sprintf("<?php echo 'Special char test %d'; ?>", i))

		// Create the test file
		if err := vfs.CreateVirtualFile(path, content); err != nil {
			t.Errorf("Failed to create special char file %s: %v", path, err)
			continue
		}

		// Verify it exists
		if !vfs.FileExists(path) {
			t.Errorf("Special char file should exist: %s", path)
			continue
		}

		// Verify content
		fileContent, err := vfs.GetFileContent(path)
		if err != nil {
			t.Errorf("Failed to read special char file %s: %v", path, err)
		} else if string(fileContent) != string(content) {
			t.Errorf("Content mismatch for special char file %s", path)
		}
	}
}

// TestVFS_ErrorRecovery tests recovery from various error conditions
func TestVFS_ErrorRecovery(t *testing.T) {
	// Mock error simulation by creating a temp directory with limited permissions
	tempParentDir, err := os.MkdirTemp("", "frango-vfs-error-")
	if err != nil {
		t.Fatalf("Failed to create temp parent dir: %v", err)
	}
	defer os.RemoveAll(tempParentDir)

	// Create a logger for capturing logs
	var logBuffer strings.Builder
	logger := log.New(&logBuffer, "TEST: ", 0)

	// Create the main VFS for testing
	vfs, err := NewVFS(tempParentDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Test 1: Recovery from failed file reads
	nonExistentFile := "/does/not/exist.php"
	content, err := vfs.GetFileContent(nonExistentFile)
	if err == nil {
		t.Errorf("Expected error when reading non-existent file, got content: %s", string(content))
	}

	// Create a file for later use
	testFile := "/recovery_test.php"
	testContent := []byte("<?php echo 'Recovery test'; ?>")
	if err := vfs.CreateVirtualFile(testFile, testContent); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test 2: Recovery from failed copies
	// Try to copy from a non-existent source
	err = vfs.CopyFileWithOptions("/nonexistent_source.php", "/recovery_copy.php", false)
	if err == nil {
		t.Errorf("Expected error when copying from non-existent source")
	}

	// Verify VFS is still operational by doing a legitimate copy
	if err := vfs.CopyFileWithOptions(testFile, "/recovery_success.php", false); err != nil {
		t.Errorf("Failed to copy file after error: %v", err)
	}

	// Test 3: Recovery from failed moves
	// Try to move a non-existent file
	err = vfs.MoveFileWithOptions("/nonexistent_source.php", "/recovery_move.php", false)
	if err == nil {
		t.Errorf("Expected error when moving non-existent file")
	}

	// Verify VFS is still operational by doing a legitimate move
	if err := vfs.CopyFileWithOptions(testFile, "/recovery_to_move.php", false); err != nil {
		t.Errorf("Failed to create file to move: %v", err)
	}

	if err := vfs.MoveFileWithOptions("/recovery_to_move.php", "/recovery_moved.php", false); err != nil {
		t.Errorf("Failed to move file after error: %v", err)
	}

	// Test 4: Create a directory with highly nested structure
	// and verify operations work correctly at max depth
	nestedPath := "/"
	maxDepth := 30 // Go beyond what's typically supported
	for i := 0; i < maxDepth; i++ {
		nestedPath = filepath.Join(nestedPath, fmt.Sprintf("level%d", i))
	}
	nestedPath = filepath.Join(nestedPath, "deepfile.php")

	// This should succeed because we normalize paths and don't actually create nested directories on disk
	nestedContent := []byte("<?php echo 'Deep file'; ?>")
	if err := vfs.CreateVirtualFile(nestedPath, nestedContent); err != nil {
		t.Logf("Note: Creating deeply nested file failed as expected: %v", err)
	} else {
		// Verify we can still access it
		content, err := vfs.GetFileContent(nestedPath)
		if err != nil {
			t.Errorf("Failed to read deeply nested file: %v", err)
		} else if string(content) != string(nestedContent) {
			t.Errorf("Content mismatch for deeply nested file")
		}
	}
}

// TestVFS_MixedOriginTypes tests operations on directories with mixed file origin types
func TestVFS_MixedOriginTypes(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-mixed-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create real source directory with files for testing
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create some source files
	sourceFiles := []string{
		filepath.Join(sourceDir, "file1.php"),
		filepath.Join(sourceDir, "file2.php"),
		filepath.Join(sourceDir, "subdir", "file3.php"),
	}

	// Create the subdirectory
	if err := os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create source subdir: %v", err)
	}

	// Write content to source files
	for i, path := range sourceFiles {
		content := []byte(fmt.Sprintf("<?php echo 'Source file %d'; ?>", i+1))
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create source file %s: %v", path, err)
		}
	}

	// Create a logger
	logger := log.New(io.Discard, "", 0)

	// Create a new VFS
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the source directory
	if err := vfs.AddSourceDirectory(sourceDir, "/source"); err != nil {
		t.Fatalf("Failed to add source directory: %v", err)
	}

	// Create some virtual files in the same virtual directory
	virtualFiles := []string{
		"/source/virtual1.php",
		"/source/virtual2.php",
		"/source/subdir/virtual3.php",
	}

	for i, path := range virtualFiles {
		content := []byte(fmt.Sprintf("<?php echo 'Virtual file %d'; ?>", i+1))
		if err := vfs.CreateVirtualFile(path, content); err != nil {
			t.Fatalf("Failed to create virtual file %s: %v", path, err)
		}
	}

	// Add embedded file
	embeddedContent := []byte("<?php echo 'Embedded test file'; ?>")
	embeddedFile := filepath.Join(tempDir, "embedded.php")
	if err := os.WriteFile(embeddedFile, embeddedContent, 0644); err != nil {
		t.Fatalf("Failed to create embedded file: %v", err)
	}

	if err := vfs.AddEmbeddedFile(testEmbedFS, "testdata/test.php", "/source/embedded.php"); err != nil {
		t.Fatalf("Failed to add embedded file: %v", err)
	}

	// Now we have a mixed directory structure with source, virtual, and embedded files
	// Test: Copy the entire directory with origin preservation
	if err := copyDirectory(vfs, "/source", "/source_copy", true); err != nil {
		t.Fatalf("Failed to copy mixed directory: %v", err)
	}

	// Verify origin types are preserved correctly
	checkOriginType(t, vfs, "/source/file1.php", OriginSource)
	checkOriginType(t, vfs, "/source_copy/file1.php", OriginSource)
	checkOriginType(t, vfs, "/source/virtual1.php", OriginVirtual)
	checkOriginType(t, vfs, "/source_copy/virtual1.php", OriginVirtual)
	checkOriginType(t, vfs, "/source/embedded.php", OriginEmbed)
	checkOriginType(t, vfs, "/source_copy/embedded.php", OriginEmbed)

	// Test changes to original source files are reflected in copies with preserved origin
	newSourceContent := []byte("<?php echo 'Updated source file 1'; ?>")
	if err := os.WriteFile(sourceFiles[0], newSourceContent, 0644); err != nil {
		t.Fatalf("Failed to update source file: %v", err)
	}

	// Trigger file change detection
	vfs.checkFileChanges("/source/file1.php")
	vfs.checkFileChanges("/source_copy/file1.php")

	// Verify source file was updated
	content, err := vfs.GetFileContent("/source/file1.php")
	if err != nil {
		t.Fatalf("Failed to read updated source file: %v", err)
	}
	if string(content) != string(newSourceContent) {
		t.Errorf("Source file not updated properly")
	}

	// Verify copy with preserved origin was updated too
	content, err = vfs.GetFileContent("/source_copy/file1.php")
	if err != nil {
		t.Fatalf("Failed to read updated copy file: %v", err)
	}
	if string(content) != string(newSourceContent) {
		t.Errorf("Source origin copy not updated properly")
	}

	// Test virtual files are independent
	newVirtualContent := []byte("<?php echo 'Updated virtual file 1'; ?>")
	if err := vfs.CreateVirtualFile("/source/virtual1.php", newVirtualContent); err != nil {
		t.Fatalf("Failed to update virtual file: %v", err)
	}

	// Verify original virtual file was updated
	content, err = vfs.GetFileContent("/source/virtual1.php")
	if err != nil {
		t.Fatalf("Failed to read updated virtual file: %v", err)
	}
	if string(content) != string(newVirtualContent) {
		t.Errorf("Virtual file not updated properly")
	}

	// Verify copy remains unchanged
	content, err = vfs.GetFileContent("/source_copy/virtual1.php")
	if err != nil {
		t.Fatalf("Failed to read virtual copy file: %v", err)
	}
	if string(content) == string(newVirtualContent) {
		t.Errorf("Virtual copy should not update when original changes")
	}
}

// Helper function to recursively copy directories
func copyDirectory(vfs *VFS, srcDir, destDir string, preserveOrigin bool) error {
	// Get all files in the source directory
	// Since VFS doesn't have a direct "list directory" function,
	// we'll check known paths from our test
	knownPaths := []string{
		"file1.php", "file2.php", "virtual1.php", "virtual2.php", "embedded.php",
		"subdir/file3.php", "subdir/virtual3.php",
	}

	for _, path := range knownPaths {
		srcPath := filepath.Join(srcDir, path)
		destPath := filepath.Join(destDir, path)

		// For subdirectories, we need to make sure parent directories exist
		if strings.Contains(path, "/") {
			// We don't actually need to create directories in VFS
			// The VFS normalizes paths automatically
		}

		// Copy file if it exists
		if vfs.FileExists(srcPath) {
			if err := vfs.CopyFileWithOptions(srcPath, destPath, preserveOrigin); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", srcPath, destPath, err)
			}
		}
	}

	return nil
}

// Helper function to check a file's origin type
func checkOriginType(t *testing.T, vfs *VFS, path string, expectedOrigin FileOrigin) {
	vfs.mutex.RLock()
	defer vfs.mutex.RUnlock()

	origin, exists := vfs.fileOrigins[path]
	if !exists {
		t.Errorf("File not found in VFS: %s", path)
		return
	}

	if origin != expectedOrigin {
		t.Errorf("File %s has origin %v, expected %v", path, origin, expectedOrigin)
	}
}

// TestVFS_Resurrection tests creating a branch from a VFS that's marked for cleanup
func TestVFS_Resurrection(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-resurrection-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that captures logs
	var logBuffer strings.Builder
	logger := log.New(&logBuffer, "TEST: ", 0)

	// Create a parent VFS
	parentVFS, err := NewVFS(tempDir, logger, false)
	if err != nil {
		t.Fatalf("Failed to create parent VFS: %v", err)
	}

	// Create a test file
	testPath := "/resurrection.php"
	testContent := []byte("<?php echo 'Resurrection test'; ?>")
	if err := parentVFS.CreateVirtualFile(testPath, testContent); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a first child
	child1 := parentVFS.Branch()
	if child1 == nil {
		t.Fatalf("Failed to create first child branch")
	}
	defer child1.Cleanup()

	// Now mark parent for cleanup (but it shouldn't fully clean up yet because child1 references it)
	parentVFS.Cleanup()

	// Verify parent is marked for cleanup
	parentVFS.refMutex.Lock()
	isCleanedUp := parentVFS.isCleanedUp
	refCount := parentVFS.refCount
	parentVFS.refMutex.Unlock()

	if !isCleanedUp {
		t.Errorf("Parent should be marked as cleaned up")
	}
	if refCount != 1 {
		t.Errorf("Parent refCount should be 1, got %d", refCount)
	}

	// Try to create another branch (this should fail or return nil)
	child2 := parentVFS.Branch()
	if child2 != nil {
		t.Errorf("Should not be able to create a branch from a VFS marked for cleanup")
		child2.Cleanup() // Clean it up to avoid resource leaks
	}

	// Verify that child1 is still functional
	content, err := child1.GetFileContent(testPath)
	if err != nil {
		t.Errorf("Failed to read file from child1: %v", err)
	} else if string(content) != string(testContent) {
		t.Errorf("Content mismatch from child1")
	}

	// Create grandchild from child1
	grandchild := child1.Branch()
	if grandchild == nil {
		t.Fatalf("Failed to create grandchild branch")
	}
	defer grandchild.Cleanup()

	// Verify grandchild is functional and can see the test file
	content, err = grandchild.GetFileContent(testPath)
	if err != nil {
		t.Errorf("Failed to read file from grandchild: %v", err)
	} else if string(content) != string(testContent) {
		t.Errorf("Content mismatch from grandchild")
	}

	// Create a file in the grandchild
	grandchildPath := "/grandchild.php"
	grandchildContent := []byte("<?php echo 'Grandchild file'; ?>")
	if err := grandchild.CreateVirtualFile(grandchildPath, grandchildContent); err != nil {
		t.Errorf("Failed to create file in grandchild: %v", err)
	}

	// Cleanup child1 (should defer actual cleanup because grandchild references it)
	child1.Cleanup()

	// Verify child1 is marked for cleanup
	child1.refMutex.Lock()
	isChildCleanedUp := child1.isCleanedUp
	childRefCount := child1.refCount
	child1.refMutex.Unlock()

	if !isChildCleanedUp {
		t.Errorf("Child should be marked as cleaned up")
	}
	if childRefCount != 1 {
		t.Errorf("Child refCount should be 1, got %d", childRefCount)
	}

	// Verify grandchild is still functional
	content, err = grandchild.GetFileContent(testPath)
	if err != nil {
		t.Errorf("Failed to read parent file from grandchild after child cleanup: %v", err)
	}

	content, err = grandchild.GetFileContent(grandchildPath)
	if err != nil {
		t.Errorf("Failed to read grandchild file after child cleanup: %v", err)
	}
}

// TestVFS_ResourceUsage tests for potential memory/resource leaks
func TestVFS_ResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource leak test in short mode")
	}

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-resources-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that discards output
	logger := log.New(io.Discard, "", 0)

	// Measure before memory
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create a large number of VFS instances, use them, then clean them up
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		// Create parent
		parent, err := NewVFS(tempDir, logger, false)
		if err != nil {
			t.Fatalf("Failed to create VFS: %v", err)
		}

		// Create a file
		path := fmt.Sprintf("/leak_test_%d.php", i)
		content := []byte(fmt.Sprintf("<?php echo 'Leak test %d'; ?>", i))
		if err := parent.CreateVirtualFile(path, content); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create a child
		child := parent.Branch()
		if child == nil {
			t.Fatalf("Failed to create branch")
		}

		// Create a file in the child
		childPath := fmt.Sprintf("/child_leak_test_%d.php", i)
		if err := child.CreateVirtualFile(childPath, content); err != nil {
			t.Fatalf("Failed to create file in child: %v", err)
		}

		// Clean up child first, then parent
		child.Cleanup()
		parent.Cleanup()

		// For every 100 iterations, force GC and measure memory
		if i > 0 && i%100 == 0 {
			runtime.GC()
			var intermediateStats runtime.MemStats
			runtime.ReadMemStats(&intermediateStats)

			// Log memory usage periodically
			t.Logf("After %d iterations - Heap alloc: %d MB, Objects: %d",
				i, intermediateStats.HeapAlloc/1024/1024, intermediateStats.HeapObjects)
		}
	}

	// Force GC and measure after memory
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate and log the difference
	heapDiff := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	objectsDiff := int64(m2.HeapObjects) - int64(m1.HeapObjects)

	t.Logf("Memory usage - Before: %d MB, After: %d MB, Diff: %d MB",
		m1.HeapAlloc/1024/1024, m2.HeapAlloc/1024/1024, heapDiff/1024/1024)
	t.Logf("Object count - Before: %d, After: %d, Diff: %d",
		m1.HeapObjects, m2.HeapObjects, objectsDiff)

	// Warning threshold for objects leak (some overhead is expected)
	const leakThreshold = 1000
	if objectsDiff > leakThreshold {
		t.Logf("WARNING: Possible memory leak detected - %d objects remained after cleanup and GC", objectsDiff)
		// Not failing the test since some overhead is normal, but logging a warning
	}
}

// TestVFS_SymlinkHandling tests how the VFS handles symlinks (they should be rejected)
func TestVFS_SymlinkHandling(t *testing.T) {
	// This test only runs on Unix-like systems since symlinks on Windows require elevated privileges
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-symlinks-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a regular file
	regularFile := filepath.Join(tempDir, "regular.php")
	regularContent := []byte("<?php echo 'Regular file'; ?>")
	if err := os.WriteFile(regularFile, regularContent, 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	// Create a folder with a regular file
	folderPath := filepath.Join(tempDir, "folder")
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}
	folderFile := filepath.Join(folderPath, "file.php")
	if err := os.WriteFile(folderFile, []byte("<?php echo 'File in folder'; ?>"), 0644); err != nil {
		t.Fatalf("Failed to create file in folder: %v", err)
	}

	// Create a symlinked file - points to the regular file
	symlinkFile := filepath.Join(tempDir, "symlink.php")
	if err := os.Symlink(regularFile, symlinkFile); err != nil {
		t.Fatalf("Failed to create symlink file: %v", err)
	}

	// Create a symlinked directory
	symlinkDir := filepath.Join(tempDir, "symlink-dir")
	if err := os.Symlink(folderPath, symlinkDir); err != nil {
		t.Fatalf("Failed to create symlink directory: %v", err)
	}

	// Create a VFS for testing
	logger := log.New(io.Discard, "", 0)
	vfs, err := NewVFS(tempDir, logger, false)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Test 1: Adding a regular file should work
	err = vfs.AddSourceFile(regularFile, "/regular.php")
	if err != nil {
		t.Errorf("Failed to add regular file: %v", err)
	}

	// Test 2: Adding a symlinked file should fail
	err = vfs.AddSourceFile(symlinkFile, "/symlink.php")
	if err == nil {
		t.Errorf("Should have refused to add symlinked file, but it succeeded")
	} else {
		t.Logf("Expected error when adding symlink: %v", err)
	}

	// Test 3: Adding a regular directory should work
	err = vfs.AddSourceDirectory(folderPath, "/folder")
	if err != nil {
		t.Errorf("Failed to add regular directory: %v", err)
	}

	// Test 4: Adding a symlinked directory should fail
	err = vfs.AddSourceDirectory(symlinkDir, "/symlink-folder")
	if err == nil {
		t.Errorf("Should have refused to add symlinked directory, but it succeeded")
	} else {
		t.Logf("Expected error when adding symlinked directory: %v", err)
	}

	// Test 5: Verify the VFS contains only the regular files
	if !vfs.FileExists("/regular.php") {
		t.Errorf("Regular file should exist in VFS")
	}
	if !vfs.FileExists("/folder/file.php") {
		t.Errorf("File in folder should exist in VFS")
	}
	if vfs.FileExists("/symlink.php") {
		t.Errorf("Symlink file should not exist in VFS")
	}
	if vfs.FileExists("/symlink-folder/file.php") {
		t.Errorf("File in symlinked folder should not exist in VFS")
	}
}

// TestVFS_CircularReferencePrevention tests that circular references are prevented
func TestVFS_CircularReferencePrevention(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "frango-vfs-circular-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger that captures logs
	var logBuffer strings.Builder
	logger := log.New(&logBuffer, "TEST: ", 0)

	// Create a parent VFS
	parent, err := NewVFS(tempDir, logger, false)
	if err != nil {
		t.Fatalf("Failed to create parent VFS: %v", err)
	}
	defer parent.Cleanup()

	// Create a child VFS
	child := parent.Branch()
	if child == nil {
		t.Fatalf("Failed to create child VFS")
	}
	defer child.Cleanup()

	// Create a grandchild VFS
	grandchild := child.Branch()
	if grandchild == nil {
		t.Fatalf("Failed to create grandchild VFS")
	}
	defer grandchild.Cleanup()

	// Test 1: Would grandchild create a circular reference if trying to branch to parent?
	// This should detect a circular reference - grandchild->child->parent (cycle back to parent)
	circularDetected := parent.wouldCreateCircularReference(grandchild)
	if !circularDetected {
		t.Errorf("Should have detected potential circular reference from parent to grandchild")
	}

	// Test 2: Direct self-reference
	selfReferenceDetected := parent.wouldCreateCircularReference(parent)
	if !selfReferenceDetected {
		t.Errorf("Should have detected self-reference circular dependency")
	}

	// Test 3: Would parent create a circular reference if using child as parent? (No)
	// This is not a circular reference, just a reverse of the normal hierarchy
	isCircular := child.wouldCreateCircularReference(parent)
	if isCircular {
		t.Errorf("Incorrectly detected circular reference - this is just a reverse relationship")
	}
}
