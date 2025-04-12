package vfs

import (
	"path"
	"strings"
)

// normalizePath normalizes a virtual path to ensure it's a valid VFS path
func normalizePath(virtualPath string) string {
	// Ensure path starts with /
	if !strings.HasPrefix(virtualPath, "/") {
		virtualPath = "/" + virtualPath
	}

	// Replace backslashes with forward slashes (for Windows compatibility)
	virtualPath = strings.ReplaceAll(virtualPath, "\\", "/")

	// Replace any double slashes with single slashes
	for strings.Contains(virtualPath, "//") {
		virtualPath = strings.ReplaceAll(virtualPath, "//", "/")
	}

	// Use path/filepath's Clean function to normalize the path
	// This handles cases like /./ and /../
	// Since Clean works with OS-specific paths, use path.Clean which
	// always uses forward slashes
	virtualPath = path.Clean(virtualPath)

	// Ensure the path still starts with /
	if !strings.HasPrefix(virtualPath, "/") {
		virtualPath = "/" + virtualPath
	}

	return virtualPath
}
