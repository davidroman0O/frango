package php

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
)

// CalculatePathHash generates a hash for the given path for wrapper files
func CalculatePathHash(scriptPath string) string {
	h := fnv.New32a()
	h.Write([]byte(scriptPath))
	return fmt.Sprintf("%x", h.Sum32())
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
