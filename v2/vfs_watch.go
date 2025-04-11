package frango

import (
	"os"
	"time"
)

// checkFileChanges checks a specific file for changes
// This is kept for backward compatibility with code that directly calls this method
func (v *VFS) checkFileChanges(virtualPath string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Only check source files - they're the only ones that can change
	origin, exists := v.fileOrigins[virtualPath]
	if !exists || origin != OriginSource {
		return
	}

	sourcePath := v.sourceMappings[virtualPath]
	oldHash := v.fileHashes[virtualPath].Hash

	// Skip if file doesn't exist
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return
	}

	// Calculate new hash
	newHash, err := calculateFileHash(sourcePath)
	if err != nil {
		v.logger.Printf("Warning: Could not calculate hash for '%s': %v", sourcePath, err)
		return
	}

	// Check if hash changed
	if newHash != oldHash {
		v.logger.Printf("Source file changed: %s (path: %s)", virtualPath, sourcePath)
		v.logger.Printf("  Hash: %s -> %s", truncateHash(oldHash), truncateHash(newHash))

		// Update hash
		v.fileHashes[virtualPath] = FileHash{
			Hash:      newHash,
			Timestamp: time.Now(),
		}

		// Mark as changed
		v.changedFiles[virtualPath] = true
		v.invalidated = true
	}
}

// checkForChanges is kept for backward compatibility
// This simply forwards to the more efficient global watcher implementation
func (v *VFS) checkForChanges() {
	// This is now handled by the global watcher
	// Left as a no-op for backward compatibility
}
