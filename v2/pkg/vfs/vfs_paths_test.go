package vfs

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestVFS_LogicalPathTracking(t *testing.T) {
	// Create temp dir for tests
	tempDir, err := os.MkdirTemp("", "vfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test.php")
	err = os.WriteFile(testFilePath, []byte("<?php echo 'Hello World'; ?>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a logger
	logger := log.New(os.Stderr, "VFS-TEST: ", log.LstdFlags)

	// Create VFS
	vfs, err := NewVFS(tempDir, logger, false)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Test with source file
	err = vfs.AddSourceFile(testFilePath, "/virtual/test.php")
	if err != nil {
		t.Fatalf("Failed to add source file: %v", err)
	}

	// Test GetLogicalPath
	logicalPath := vfs.GetLogicalPath(testFilePath)
	if logicalPath != "/virtual/test.php" {
		t.Errorf("Expected logical path to be '/virtual/test.php', got '%s'", logicalPath)
	}

	// Test with virtual file
	err = vfs.CreateVirtualFile("/virtual/test2.php", []byte("<?php echo 'Virtual File'; ?>"))
	if err != nil {
		t.Fatalf("Failed to create virtual file: %v", err)
	}

	// Resolve path to get physical path
	physicalPath, err := vfs.ResolvePath("/virtual/test2.php")
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}

	// Test GetLogicalPath on virtual file
	logicalPath = vfs.GetLogicalPath(physicalPath)
	if logicalPath != "/virtual/test2.php" {
		t.Errorf("Expected logical path to be '/virtual/test2.php', got '%s'", logicalPath)
	}

	// Test GetLogicalPathFromVirtual
	logicalPath, err = vfs.GetLogicalPathFromVirtual("/virtual/test.php")
	if err != nil {
		t.Fatalf("Failed to get logical path from virtual: %v", err)
	}
	if logicalPath != "/virtual/test.php" {
		t.Errorf("Expected logical path to be '/virtual/test.php', got '%s'", logicalPath)
	}

	// Test with branched VFS
	branchVFS := vfs.Branch()
	defer branchVFS.Cleanup()

	// Verify branch can resolve logical paths from parent
	logicalPath = branchVFS.GetLogicalPath(testFilePath)
	if logicalPath != "/virtual/test.php" {
		t.Errorf("Branched VFS: Expected logical path to be '/virtual/test.php', got '%s'", logicalPath)
	}

	// Add a new source file to branch
	branchTestFile := filepath.Join(tempDir, "branch-test.php")
	err = os.WriteFile(branchTestFile, []byte("<?php echo 'Branch Test'; ?>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create branch test file: %v", err)
	}

	err = branchVFS.AddSourceFile(branchTestFile, "/virtual/branch-test.php")
	if err != nil {
		t.Fatalf("Failed to add source file to branch: %v", err)
	}

	// Test branch-specific logical path
	logicalPath = branchVFS.GetLogicalPath(branchTestFile)
	if logicalPath != "/virtual/branch-test.php" {
		t.Errorf("Expected branch logical path to be '/virtual/branch-test.php', got '%s'", logicalPath)
	}

	// Verify original VFS doesn't have branch's logical path
	logicalPath = vfs.GetLogicalPath(branchTestFile)
	if logicalPath != branchTestFile {
		t.Errorf("Expected original VFS to return physical path, got '%s'", logicalPath)
	}
}
