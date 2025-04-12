package php

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
)

// CalculatePathHash generates a hash for the given path for wrapper files
func CalculatePathHash(scriptPath string) string {
	h := fnv.New32a()
	h.Write([]byte(scriptPath))
	return fmt.Sprintf("%x", h.Sum32())
}

// CalculateFileHash calculates the SHA256 hash of a file's content
func CalculateFileHash(filePath string) (string, error) {
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

// CalculateContentHash calculates the SHA256 hash of content
func CalculateContentHash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// GetGlobalsPath returns the standard path for PHP globals script
func GetGlobalsPath() string {
	return "/_frango_php_globals.php"
}

// SanitizePhpPath sanitizes a filepath for PHP by converting backslashes to forward slashes
func SanitizePhpPath(path string) string {
	// PHP always uses forward slashes, even on Windows
	return filepath.ToSlash(path)
}

// TruncateHash truncates a hash to 8 characters for display purposes
func TruncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8] + "..."
	}
	return hash
}
