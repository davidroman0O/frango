package frango

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// validateHTTPMethod checks if a string is a valid HTTP method name.
func validateHTTPMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE":
		return true
	default:
		return false
	}
}

// calculateFileHash calculates the SHA256 hash of a file's content.
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

// extractPathParams extracts path parameters from a URL pattern and actual path.
// Example: extractPathParams("/users/{id}/posts/{postId}", "/users/42/posts/123")
// returns: map[string]string{"id": "42", "postId": "123"}
func extractPathParams(pattern, path string) map[string]string {
	// Extract HTTP method if pattern includes it
	patternPath := pattern
	if parts := strings.SplitN(pattern, " ", 2); len(parts) > 1 {
		patternPath = parts[1]
	}

	// Split pattern and path into segments
	patternSegments := strings.Split(strings.Trim(patternPath, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Check if segment counts don't match
	if len(patternSegments) != len(pathSegments) {
		return nil
	}

	// Extract parameters
	params := make(map[string]string)
	for i, patternSegment := range patternSegments {
		// Check for parameter pattern {name}
		if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
			// Extract parameter name without braces
			paramName := patternSegment[1 : len(patternSegment)-1]
			if paramName != "" && paramName != "$" { // Skip special {$} if it exists
				// Use actual path segment as parameter value
				params[paramName] = pathSegments[i]
			}
		} else if patternSegment != pathSegments[i] {
			// If a non-parameter segment doesn't match exactly, no match
			return nil
		}
	}

	return params
}

// ResolveDirectory resolves a directory path, supporting both absolute and relative paths.
// It tries multiple strategies to find the directory including:
// 1. Absolute path
// 2. Relative to current working directory
// 3. Relative to caller's directory
// 4. Relative to git repository root
func ResolveDirectory(path string) (string, error) {
	return resolveDirectory(path)
}

// resolveDirectory is the internal implementation of ResolveDirectory
// It tries multiple strategies to find the directory.
func resolveDirectory(path string) (string, error) {
	// If the path is absolute or explicitly relative (starts with ./ or ../)
	if filepath.IsAbs(path) || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("error resolving absolute/explicit relative path '%s': %w", path, err)
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath, nil
		} else if err != nil {
			// If the explicit path doesn't exist, try relative to the caller
			_, filename, _, ok := runtime.Caller(2)
			if ok && !filepath.IsAbs(path) {
				callerDir := filepath.Dir(filename)
				callerPath := filepath.Join(callerDir, path)
				absCallerPath, absErr := filepath.Abs(callerPath)
				if absErr == nil {
					if info, statErr := os.Stat(absCallerPath); statErr == nil && info.IsDir() {
						return absCallerPath, nil
					}
				}

				// Try looking in the repo root directory
				repoRoot, rootErr := findRepoRoot(callerDir)
				if rootErr == nil {
					repoPath := filepath.Join(repoRoot, path)
					absRepoPath, absErr := filepath.Abs(repoPath)
					if absErr == nil {
						if info, statErr := os.Stat(absRepoPath); statErr == nil && info.IsDir() {
							return absRepoPath, nil
						}
					}
				}
			}
			return "", fmt.Errorf("error stating explicit path '%s': %w", absPath, err)
		} else {
			return "", fmt.Errorf("explicit path '%s' exists but is not a directory", absPath)
		}
	}

	// For a bare directory name, try multiple locations
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		absPath, absErr := filepath.Abs(path)
		if absErr == nil {
			return absPath, nil
		}
		return "", fmt.Errorf("found path '%s' relative to CWD but failed to get absolute path: %w", path, absErr)
	}

	// Try relative to Caller
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, ".") {
		_, filename, _, ok := runtime.Caller(2)
		if ok {
			callerDir := filepath.Dir(filename)
			callerPath := filepath.Join(callerDir, path)
			absCallerPath, absErr := filepath.Abs(callerPath)
			if absErr == nil {
				if info, statErr := os.Stat(absCallerPath); statErr == nil && info.IsDir() {
					return absCallerPath, nil
				}
			}

			// Try looking in the repo root directory
			repoRoot, rootErr := findRepoRoot(callerDir)
			if rootErr == nil {
				repoPath := filepath.Join(repoRoot, path)
				absRepoPath, absErr := filepath.Abs(repoPath)
				if absErr == nil {
					if info, statErr := os.Stat(absRepoPath); statErr == nil && info.IsDir() {
						return absRepoPath, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("directory '%s' not found relative to CWD or caller", path)
}

// findRepoRoot attempts to find the root directory of the Git repository
// by walking up the directory tree looking for a .git directory
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

// getMapKeys is a helper function to get the keys of a map for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// copyFile utility copies a single file, creating destination directories
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
