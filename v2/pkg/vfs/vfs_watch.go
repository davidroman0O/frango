package vfs

import (
	"os"
	"time"

	"github.com/davidroman0O/frango/v2/internal/utils"
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
	newHash, err := utils.CalculateFileHash(sourcePath)
	if err != nil {
		v.logger.Printf("Warning: Could not calculate hash for '%s': %v", sourcePath, err)
		return
	}

	// Check if hash changed
	if newHash != oldHash {
		v.logger.Printf("Source file changed: %s (path: %s)", virtualPath, sourcePath)
		v.logger.Printf("  Hash: %s -> %s", utils.TruncateHash(oldHash, 8), utils.TruncateHash(newHash, 8))

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
