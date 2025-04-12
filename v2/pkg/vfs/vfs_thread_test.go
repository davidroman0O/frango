package vfs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

// TestVFS_ThreadSafety tests the thread-safety of the VFS
func TestVFS_ThreadSafety(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "vfs-thread-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		virtualPath string
		content     string
	}{
		{"/test1.php", "<?php echo 'test1'; ?>"},
		{"/test2.php", "<?php echo 'test2'; ?>"},
		{"/test3.php", "<?php echo 'test3'; ?>"},
		{"/test4.php", "<?php echo 'test4'; ?>"},
		{"/test5.php", "<?php echo 'test5'; ?>"},
	}

	// Create a logger
	logger := log.New(os.Stdout, "THREAD TEST: ", log.LstdFlags)

	// Create a VFS
	vfs, err := NewVFSWithConfig(VFSConfig{
		TempDir:     tempDir,
		Logger:      logger,
		DevelopMode: false,
	})
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Add the test files
	for _, tf := range testFiles {
		if err := vfs.CreateVirtualFile(tf.virtualPath, []byte(tf.content)); err != nil {
			t.Fatalf("Failed to create virtual file %s: %v", tf.virtualPath, err)
		}
	}

	// Test concurrent ResolvePath
	t.Run("ConcurrentResolvePath", func(t *testing.T) {
		const goroutines = 100
		const iterations = 100

		// Create a new branch for testing to avoid affecting the parent
		branchVFS := vfs.Branch()
		if branchVFS == nil {
			t.Fatalf("Failed to create branch VFS")
		}
		defer branchVFS.Cleanup()

		var wg sync.WaitGroup
		errChan := make(chan error, goroutines*iterations)

		// Launch multiple goroutines to simulate concurrent access
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					// Choose a file based on the iteration
					fileIndex := (routineID + j) % len(testFiles)
					virtualPath := testFiles[fileIndex].virtualPath

					// Resolve the path
					resolvedPath, err := branchVFS.ResolvePath(virtualPath)
					if err != nil {
						errChan <- err
						return
					}

					// Verify we got a valid path
					if resolvedPath == "" {
						errChan <- fmt.Errorf("resolved path is empty for %s", virtualPath)
						return
					}

					// Also test getting file content which should use the path cache
					_, err = branchVFS.GetFileContent(virtualPath)
					if err != nil {
						errChan <- err
						return
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Error in concurrent operations: %v", err)
		}
	})

	// Test concurrent file operations
	t.Run("ConcurrentFileOperations", func(t *testing.T) {
		const goroutines = 50
		const operationsPerGoroutine = 20

		// Create a new branch for testing to avoid affecting the parent
		branchVFS := vfs.Branch()
		if branchVFS == nil {
			t.Fatalf("Failed to create branch VFS")
		}
		defer branchVFS.Cleanup()

		var wg sync.WaitGroup
		errChan := make(chan error, goroutines*operationsPerGoroutine)

		// Launch multiple goroutines to perform different file operations
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				// Base path for this goroutine's operations
				basePath := "/concurrent" + fmt.Sprintf("%d", routineID)

				for j := 0; j < operationsPerGoroutine; j++ {
					// Different operations based on the iteration
					virtualPath := basePath + fmt.Sprintf("/file%d.php", j)
					content := []byte("<?php echo 'concurrent" + fmt.Sprintf("%d-%d", routineID, j) + "'; ?>")

					// Create file
					if err := branchVFS.CreateVirtualFile(virtualPath, content); err != nil {
						errChan <- fmt.Errorf("failed to create file: %w", err)
						continue
					}

					// Read file
					readContent, err := branchVFS.GetFileContent(virtualPath)
					if err != nil {
						errChan <- fmt.Errorf("failed to read file: %w", err)
						continue
					}

					// Verify content
					if string(readContent) != string(content) {
						errChan <- fmt.Errorf("file content mismatch, expected %s, got %s", string(content), string(readContent))
						continue
					}

					// Copy file
					copyPath := virtualPath + ".copy"
					if err := branchVFS.CopyFile(virtualPath, copyPath); err != nil {
						errChan <- fmt.Errorf("failed to copy file: %w", err)
						continue
					}

					// Delete original file
					if err := branchVFS.DeleteFile(virtualPath); err != nil {
						errChan <- fmt.Errorf("failed to delete file: %w", err)
						continue
					}

					// Verify the original file is gone
					if branchVFS.FileExists(virtualPath) {
						errChan <- fmt.Errorf("file should be deleted but still exists: %s", virtualPath)
						continue
					}

					// Verify the copy still exists
					if !branchVFS.FileExists(copyPath) {
						errChan <- fmt.Errorf("copy file should exist but doesn't: %s", copyPath)
						continue
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Error in concurrent operations: %v", err)
		}
	})

	// Test mixed operations with branches
	t.Run("MixedOperationsWithBranches", func(t *testing.T) {
		const branches = 10
		const goroutinesPerBranch = 10
		const operationsPerGoroutine = 10

		var branchVFSs []*VFS
		defer func() {
			for _, bvfs := range branchVFSs {
				bvfs.Cleanup()
			}
		}()

		// Create branches
		for i := 0; i < branches; i++ {
			branch := vfs.Branch()
			if branch == nil {
				t.Fatalf("Failed to create branch VFS")
			}
			branchVFSs = append(branchVFSs, branch)
		}

		var wg sync.WaitGroup
		errChan := make(chan error, branches*goroutinesPerBranch*operationsPerGoroutine)

		// Launch goroutines for each branch
		for branchIdx, branch := range branchVFSs {
			for i := 0; i < goroutinesPerBranch; i++ {
				wg.Add(1)
				go func(branchID, routineID int, branchVFS *VFS) {
					defer wg.Done()

					// Base path for this goroutine's operations
					basePath := fmt.Sprintf("/branch%d/routine%d", branchID, routineID)

					for j := 0; j < operationsPerGoroutine; j++ {
						// Simulate various operations
						switch j % 4 {
						case 0:
							// Create file
							virtualPath := basePath + fmt.Sprintf("/file%d.php", j)
							content := []byte("<?php echo 'branch" + fmt.Sprintf("%d-%d-%d", branchID, routineID, j) + "'; ?>")
							if err := branchVFS.CreateVirtualFile(virtualPath, content); err != nil {
								errChan <- fmt.Errorf("failed to create file in branch %d: %w", branchID, err)
							}
						case 1:
							// Read shared files from parent
							fileIndex := (branchID + routineID + j) % len(testFiles)
							virtualPath := testFiles[fileIndex].virtualPath
							if _, err := branchVFS.GetFileContent(virtualPath); err != nil {
								errChan <- fmt.Errorf("failed to read shared file in branch %d: %w", branchID, err)
							}
						case 2:
							// Add a small sleep to increase chance of concurrency issues
							time.Sleep(time.Millisecond)
						case 3:
							// File exists check
							fileIndex := (branchID + routineID + j) % len(testFiles)
							virtualPath := testFiles[fileIndex].virtualPath
							if !branchVFS.FileExists(virtualPath) {
								errChan <- fmt.Errorf("shared file should exist in branch %d but doesn't: %s", branchID, virtualPath)
							}
						}
					}
				}(branchIdx, i, branch)
			}
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Error in mixed operations: %v", err)
		}
	})
}
