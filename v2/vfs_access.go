package frango

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetFileContent reads the content of a file from the VFS
func (v *VFS) GetFileContent(virtualPath string) ([]byte, error) {
	// Normalize path
	virtualPath = normalizePath(virtualPath)

	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Check this VFS first
	origin, exists := v.fileOrigins[virtualPath]
	if exists {
		// Check for virtual "tombstone" files
		if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
			return nil, fmt.Errorf("file not found in VFS: %s (shadowed)", virtualPath)
		}

		// Get content based on origin type
		switch origin {
		case OriginSource:
			sourcePath := v.sourceMappings[virtualPath]
			return os.ReadFile(sourcePath)
		case OriginEmbed:
			tempPath := v.embedMappings[virtualPath]
			return os.ReadFile(tempPath)
		case OriginVirtual:
			// For virtual files, use in-memory content if available
			if content, ok := v.virtualFiles[virtualPath]; ok && len(content) > 0 {
				return content, nil
			}
			// Fallback to temp file
			tempPath := v.embedMappings[virtualPath]
			return os.ReadFile(tempPath)
		}
	}

	// If not found in this VFS, check parent (if exists)
	if v.parent != nil {
		return v.parent.GetFileContent(virtualPath)
	}

	return nil, fmt.Errorf("file not found in VFS: %s", virtualPath)
}

// FileExists checks if a file exists in the VFS
func (v *VFS) FileExists(virtualPath string) bool {
	// Normalize path
	virtualPath = normalizePath(virtualPath)

	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Check this VFS first
	origin, exists := v.fileOrigins[virtualPath]
	if exists {
		// Check for virtual "tombstone" files
		if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
			return false // File is shadowed/deleted
		}
		return true
	}

	// If not found in this VFS, check parent (if exists)
	if v.parent != nil {
		return v.parent.FileExists(virtualPath)
	}

	return false
}

// ResolvePath resolves a virtual path to its actual filesystem path
func (v *VFS) ResolvePath(virtualPath string) (string, error) {
	// Normalize path
	virtualPath = normalizePath(virtualPath)

	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// If in development mode, check for changes first - but don't lock here
	// to avoid deadlocks between checkForChanges and ResolvePath
	if v.developMode {
		// Don't call checkForChanges while holding a lock
		v.mutex.RUnlock()
		v.checkFileChanges(virtualPath) // Use a specialized function just for checking one file
		v.mutex.RLock()
	}

	// Check this VFS first
	origin, exists := v.fileOrigins[virtualPath]
	if exists {
		// Check for virtual "tombstone" files
		if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
			return "", fmt.Errorf("file not found in VFS: %s (shadowed)", virtualPath)
		}

		// Resolve based on origin type
		switch origin {
		case OriginSource:
			return v.sourceMappings[virtualPath], nil
		case OriginEmbed, OriginVirtual:
			return v.embedMappings[virtualPath], nil
		}
	}

	// If not found in this VFS, check parent (if exists)
	if v.parent != nil {
		// Remember this path is inherited
		v.inheritedPaths[virtualPath] = true

		// Release our lock before calling parent
		parentPath := ""
		var parentErr error
		v.mutex.RUnlock()
		parentPath, parentErr = v.parent.ResolvePath(virtualPath)
		v.mutex.RLock()

		return parentPath, parentErr
	}

	return "", fmt.Errorf("file not found in VFS: %s", virtualPath)
}

// ResolvePathLiteral resolves a virtual path to a filesystem path literally,
// without normalizing or cleaning the path. This is useful for paths with
// special characters like curly braces that should be treated literally.
func (vfs *VFS) ResolvePathLiteral(virtualPath string) (string, error) {
	// Try to find a file with the exact name, preserving special characters like {param}
	vfs.mutex.RLock()
	defer vfs.mutex.RUnlock()

	// Since our internal map uses normalized paths, we need to look through all
	// entries to find one that might be equivalent without normalization
	for storedPath, origin := range vfs.fileOrigins {
		if strings.EqualFold(storedPath, virtualPath) ||
			(strings.Contains(storedPath, "{") && strings.Contains(virtualPath, "{")) {
			// Found a potential match - check if it exists on disk
			switch origin {
			case OriginSource:
				// Get the mapped file path from source mappings
				mappedPath, exists := vfs.sourceMappings[storedPath]
				if exists {
					return mappedPath, nil
				}
			case OriginEmbed:
				// Get the mapped file path from embed mappings
				mappedPath, exists := vfs.embedMappings[storedPath]
				if exists {
					return mappedPath, nil
				}
			case OriginVirtual:
				// For virtual files, the path is the temp directory path
				return filepath.Join(vfs.tempDir, filepath.FromSlash(storedPath)), nil
			}
		}
	}

	// Try looking for the literal file on disk as a last resort
	literalPath := filepath.Join(vfs.tempDir, filepath.FromSlash(virtualPath))
	if _, err := os.Stat(literalPath); err == nil {
		return literalPath, nil
	}

	return "", fmt.Errorf("virtual path not found: %s", virtualPath)
}
