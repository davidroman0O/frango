package frango

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// generateUniqueID generates a unique identifier for VFS instances or middleware
func generateUniqueID() string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash.Sum(nil))[:8]
}

// calculateFileHash calculates the SHA256 hash of a file's content
func calculateFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file '%s' for hashing: %w", filePath, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copies a file from src to dst, creating parent directories as needed
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory '%s': %w", filepath.Dir(dst), err)
	}
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

// resolveDirectory resolves a directory path, supporting both absolute and relative paths
func resolveDirectory(path string) (string, error) {
	// If the path is absolute, just return it
	if filepath.IsAbs(path) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		} else if err != nil {
			return "", fmt.Errorf("error accessing directory '%s': %w", path, err)
		} else {
			return "", fmt.Errorf("path '%s' is not a directory", path)
		}
	}

	// Try relative to current working directory
	absPath, err := filepath.Abs(path)
	if err == nil {
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath, nil
		}
	}

	// Try relative to caller's directory
	_, filename, _, ok := runtime.Caller(1)
	if ok {
		callerDir := filepath.Dir(filename)
		callerRelPath := filepath.Join(callerDir, path)
		if info, err := os.Stat(callerRelPath); err == nil && info.IsDir() {
			absPath, err := filepath.Abs(callerRelPath)
			if err == nil {
				return absPath, nil
			}
		}

		// Try to find repository root
		repoRoot, err := findRepoRoot(callerDir)
		if err == nil {
			repoRelPath := filepath.Join(repoRoot, path)
			if info, err := os.Stat(repoRelPath); err == nil && info.IsDir() {
				absPath, err := filepath.Abs(repoRelPath)
				if err == nil {
					return absPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("directory '%s' not found", path)
}

// findRepoRoot attempts to find the root directory of a Git repository
func findRepoRoot(startDir string) (string, error) {
	dir := startDir
	for {
		// Check if .git directory exists
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		// Go up one directory level
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the filesystem root without finding .git
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no git repository found in ancestry of %s", startDir)
}
