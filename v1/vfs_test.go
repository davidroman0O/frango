package frango

import (
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	// Add the embedded file to the VFS
	virtualPath := "/embedded.php"
	if err := vfs.AddEmbeddedFile(testEmbedFS, "testdata/test.php", virtualPath); err != nil {
		t.Fatalf("Failed to add embedded file: %v", err)
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
	expectedContent := "<?php echo 'Embedded test file'; ?>"
	if string(content) != expectedContent {
		t.Fatalf("Content mismatch: %s vs %s", string(content), expectedContent)
	}

	// Verify the file was written to disk
	path, err := vfs.ResolvePath(virtualPath)
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}

	// The path should be within the VFS temp directory
	if !strings.HasPrefix(path, vfs.tempDir) {
		t.Fatalf("Path is not within VFS temp directory: %s", path)
	}

	// Read the file from disk
	diskContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read embedded file from disk: %v", err)
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

	testFile := filepath.Join(testDataDir, "test.php")
	content := []byte("<?php echo 'Embedded test file'; ?>")
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
