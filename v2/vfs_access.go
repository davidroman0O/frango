package frango

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GetFileContent reads the content of a file from the VFS
func (v *VFS) GetFileContent(virtualPath string) ([]byte, error) {
	// Normalize path
	virtualPath = normalizePath(virtualPath)

	// Check content cache first if not in development mode
	if !v.developMode {
		v.cacheMutex.RLock()
		if cachedContent, ok := v.contentCache[virtualPath]; ok {
			v.cacheMutex.RUnlock()
			return cachedContent, nil
		}
		v.cacheMutex.RUnlock()
	}

	// Take file content read lock
	v.fileMutex.RLock()
	defer v.fileMutex.RUnlock()

	// Take struct read lock for origin check
	v.mutex.RLock()

	// Check this VFS first
	origin, exists := v.fileOrigins[virtualPath]
	if exists {
		// Check for virtual "tombstone" files
		if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
			v.mutex.RUnlock()
			return nil, fmt.Errorf("file not found in VFS: %s (shadowed)", virtualPath)
		}

		// Get content based on origin type
		var content []byte
		var err error

		switch origin {
		case OriginSource:
			sourcePath := v.sourceMappings[virtualPath]
			v.mutex.RUnlock() // Release mutex before I/O
			content, err = os.ReadFile(sourcePath)
		case OriginEmbed:
			tempPath := v.embedMappings[virtualPath]
			v.mutex.RUnlock() // Release mutex before I/O
			content, err = os.ReadFile(tempPath)
		case OriginVirtual:
			// For virtual files, use in-memory content if available
			if inMemoryContent, ok := v.virtualFiles[virtualPath]; ok && len(inMemoryContent) > 0 {
				content = inMemoryContent
				v.mutex.RUnlock()
			} else {
				// Fallback to temp file
				tempPath := v.embedMappings[virtualPath]
				v.mutex.RUnlock() // Release mutex before I/O
				content, err = os.ReadFile(tempPath)
			}
		default:
			v.mutex.RUnlock()
		}

		// Update cache if successful and not in development mode
		if err == nil && content != nil && !v.developMode {
			v.cacheMutex.Lock()
			v.contentCache[virtualPath] = content
			v.cacheMutex.Unlock()
		}

		return content, err
	}

	// Not found in this VFS
	v.mutex.RUnlock()

	// If not found in this VFS, check parent (if exists)
	if v.parent != nil {
		// Release our file lock before calling parent
		v.fileMutex.RUnlock()
		content, err := v.parent.GetFileContent(virtualPath)
		v.fileMutex.RLock()

		// Cache parent content as well for faster subsequent access
		if err == nil && content != nil && !v.developMode {
			v.cacheMutex.Lock()
			v.contentCache[virtualPath] = content
			v.cacheMutex.Unlock()
		}

		return content, err
	}

	return nil, fmt.Errorf("file not found in VFS: %s", virtualPath)
}

// FileExists checks if a file exists in the VFS
func (v *VFS) FileExists(virtualPath string) bool {
	// Normalize path
	virtualPath = normalizePath(virtualPath)

	// Check path cache first if not in development mode
	if !v.developMode {
		v.cacheMutex.RLock()
		if _, ok := v.pathCache[virtualPath]; ok {
			v.cacheMutex.RUnlock()
			return true
		}
		v.cacheMutex.RUnlock()
	}

	// Check content cache
	if !v.developMode {
		v.cacheMutex.RLock()
		if _, ok := v.contentCache[virtualPath]; ok {
			v.cacheMutex.RUnlock()
			return true
		}
		v.cacheMutex.RUnlock()
	}

	// Take a read lock on the path mutex
	v.pathMutex.RLock()
	defer v.pathMutex.RUnlock()

	// Check this VFS first with a read lock on the main mutex
	v.mutex.RLock()
	origin, exists := v.fileOrigins[virtualPath]
	if exists {
		// Check for virtual "tombstone" files
		if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
			v.mutex.RUnlock()
			return false // File is shadowed/deleted
		}
		v.mutex.RUnlock()
		return true
	}
	v.mutex.RUnlock()

	// If not found in this VFS, check parent (if exists)
	if v.parent != nil {
		// Release our path lock before calling parent
		v.pathMutex.RUnlock()
		exists := v.parent.FileExists(virtualPath)
		v.pathMutex.RLock()

		// Cache the path if it exists and we're not in development mode
		if exists && !v.developMode {
			// Resolve the path from parent once we know it exists
			v.pathMutex.RUnlock()
			resolvedPath, err := v.parent.ResolvePath(virtualPath)
			v.pathMutex.RLock()

			if err == nil && resolvedPath != "" {
				v.cacheMutex.Lock()
				v.pathCache[virtualPath] = resolvedPath
				v.cacheMutex.Unlock()
			}
		}

		return exists
	}

	return false
}

// ResolvePath resolves a virtual path to its actual filesystem path
func (v *VFS) ResolvePath(virtualPath string) (string, error) {
	// Normalize path
	virtualPath = normalizePath(virtualPath)

	// Check the cache with read lock first if not in development mode
	if !v.developMode {
		v.cacheMutex.RLock()
		if cachedPath, ok := v.pathCache[virtualPath]; ok {
			v.cacheMutex.RUnlock()
			return cachedPath, nil
		}
		v.cacheMutex.RUnlock()
	}

	// Take path resolution read lock
	v.pathMutex.RLock()
	defer v.pathMutex.RUnlock()

	// If in development mode, check for changes first - but don't lock here
	// to avoid deadlocks between checkForChanges and ResolvePath
	if v.developMode {
		// We use the checkFileChanges directly to avoid any locking issues
		originType, exists := v.fileOrigins[virtualPath]
		if exists && originType == OriginSource {
			sourcePath := v.sourceMappings[virtualPath]
			// Release path mutex to avoid deadlock while checking for changes
			v.pathMutex.RUnlock()
			// Release main locks for checking changes
			v.mutex.RLock()
			oldHash := v.fileHashes[virtualPath].Hash
			v.mutex.RUnlock()

			// Check for changes without locking
			if _, err := os.Stat(sourcePath); err == nil {
				// Check hash without locks
				newHash, err := calculateFileHash(sourcePath)
				if err == nil && newHash != oldHash {
					// Take mutex lock to update hash if changed
					v.mutex.Lock()
					v.fileHashes[virtualPath] = FileHash{
						Hash:      newHash,
						Timestamp: time.Now(),
					}
					v.changedFiles[virtualPath] = true
					v.invalidated = true
					v.mutex.Unlock()

					v.logger.Printf("Source file changed: %s (path: %s)", virtualPath, sourcePath)
					v.logger.Printf("  Hash: %s -> %s", truncateHash(oldHash), truncateHash(newHash))
				}
			}
			// Re-acquire path mutex after checks
			v.pathMutex.RLock()
		}
	}

	// We need to use mutex.RLock() for checking fileOrigins and accessing mappings
	v.mutex.RLock()

	// Check this VFS first
	origin, exists := v.fileOrigins[virtualPath]
	var resolvedPath string
	var resolutionErr error

	if exists {
		// Check for virtual "tombstone" files
		if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
			resolutionErr = fmt.Errorf("file not found in VFS: %s (shadowed)", virtualPath)
		} else {
			// Resolve based on origin type
			switch origin {
			case OriginSource:
				resolvedPath = v.sourceMappings[virtualPath]
			case OriginEmbed, OriginVirtual:
				resolvedPath = v.embedMappings[virtualPath]
			}
		}
		v.mutex.RUnlock()
	} else {
		v.mutex.RUnlock()

		// If not found in this VFS, check parent (if exists)
		if v.parent != nil {
			// Remember this path is inherited
			v.mutex.Lock()
			v.inheritedPaths[virtualPath] = true
			v.mutex.Unlock()

			// Release our path lock before calling parent to avoid deadlocks
			v.pathMutex.RUnlock()
			resolvedPath, resolutionErr = v.parent.ResolvePath(virtualPath)
			v.pathMutex.RLock()
		} else {
			resolutionErr = fmt.Errorf("file not found in VFS: %s", virtualPath)
		}
	}

	// Update the cache if we found a path and we're not in development mode
	if resolvedPath != "" && !v.developMode && resolutionErr == nil {
		v.cacheMutex.Lock()
		v.pathCache[virtualPath] = resolvedPath
		v.cacheMutex.Unlock()
	}

	return resolvedPath, resolutionErr
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
