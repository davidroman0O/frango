package vfs

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidroman0O/frango/internal/utils"
)

// AddSourceFile adds a file from the filesystem to the VFS
func (v *VFS) AddSourceFile(sourcePath, virtualPath string) error {
	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// First check for symlinks without any locks
	fileInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("error accessing source file '%s': %w", sourcePath, err)
	}

	// Prevent symlinks for security reasons
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not supported for security reasons: %s", sourcePath)
	}

	// Calculate hash for change detection without locks
	hash, err := utils.CalculateFileHash(sourcePath)
	if err != nil {
		return fmt.Errorf("error calculating hash for '%s': %w", sourcePath, err)
	}

	// Now lock the VFS for structural changes
	v.mutex.Lock()

	// Store mappings
	v.sourceMappings[virtualPath] = sourcePath
	v.fileOrigins[virtualPath] = OriginSource
	v.fileHashes[sourcePath] = FileHash{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	v.mutex.Unlock()

	// Track logical path (virtual path) for this physical path
	v.TrackLogicalPath(sourcePath, virtualPath)

	v.logger.Printf("Added source file: %s -> %s (hash: %s)", sourcePath, virtualPath, utils.TruncateHash(hash, 8))

	// Register with global watcher if in development mode
	if v.developMode {
		GetGlobalWatcher().RegisterFile(v, sourcePath)

		// Notify file added event
		v.logger.Printf("AddSourceFile: Triggering notifyFileChanged(added) for %s", virtualPath)
		v.notifyFileChanged(virtualPath, sourcePath, "added")
	} else {
		// Update path cache
		v.cacheMutex.Lock()
		v.pathCache[virtualPath] = sourcePath
		v.cacheMutex.Unlock()
	}

	return nil
}

// AddSourceDirectory adds all PHP files from a directory to the VFS
func (v *VFS) AddSourceDirectory(sourceDir string, virtualBasePath string) error {
	return v.addSourceDirectoryRecursive(sourceDir, virtualBasePath, true)
}

// addSourceDirectoryRecursive adds all PHP files from a directory to the VFS
func (v *VFS) addSourceDirectoryRecursive(sourceDir, virtualBasePath string, recursive bool) error {
	// Normalize the virtual base path
	virtualBasePath = normalizePath(virtualBasePath)

	// Verify sourceDir exists and is a directory
	dirInfo, err := os.Lstat(sourceDir)
	if err != nil {
		return fmt.Errorf("error accessing directory '%s': %w", sourceDir, err)
	}

	// Prevent symlinked directories for security reasons
	if dirInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinked directories are not supported for security reasons: %s", sourceDir)
	}

	// Double-check it's a directory
	if !dirInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", sourceDir)
	}

	// Read directory
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("error reading directory '%s': %w", sourceDir, err)
	}

	// Process files and subdirectories
	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())

		// Check for symlinks
		fileInfo, err := os.Lstat(sourcePath)
		if err != nil {
			v.logger.Printf("Warning: Error accessing '%s': %v - skipping", sourcePath, err)
			continue
		}

		// Skip symlinks for security reasons
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			v.logger.Printf("Warning: Skipping symlink for security reasons: %s", sourcePath)
			continue
		}

		// Handle directories
		if entry.IsDir() {
			if recursive {
				// Create virtual subdirectory path
				virtualSubdir := filepath.Join(virtualBasePath, entry.Name())
				if err := v.addSourceDirectoryRecursive(sourcePath, virtualSubdir, recursive); err != nil {
					v.logger.Printf("Warning: Error processing subdirectory '%s': %v", sourcePath, err)
				}
			}
			continue
		}

		// Handle files - only add PHP files
		if filepath.Ext(entry.Name()) == ".php" {
			virtualPath := filepath.Join(virtualBasePath, entry.Name())
			if err := v.AddSourceFile(sourcePath, virtualPath); err != nil {
				v.logger.Printf("Warning: Error adding source file '%s': %v", sourcePath, err)
			}
		}
	}

	return nil
}

// AddEmbeddedFile adds a single file from an embed.FS to the VFS
func (v *VFS) AddEmbeddedFile(embedFS embed.FS, fsPath string, virtualPath string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// Read the content from the embedded filesystem
	content, err := embedFS.ReadFile(fsPath)
	if err != nil {
		return fmt.Errorf("error reading embedded file '%s': %w", fsPath, err)
	}

	// Create target directory in VFS temp space
	targetDir := filepath.Dir(filepath.Join(v.tempDir, virtualPath))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("error creating directory for embedded file '%s': %w", targetDir, err)
	}

	// Write to temp path
	tempPath := filepath.Join(v.tempDir, virtualPath)

	// If the path would be the same as the temp directory itself, append a filename
	// This is a safeguard against trying to write to a directory
	if tempPath == v.tempDir || tempPath+"/" == v.tempDir+"/" {
		if v.logger != nil {
			v.logger.Printf("Warning: Virtual path %s would result in writing to the temp directory itself; appending default filename", virtualPath)
		}
		tempPath = filepath.Join(v.tempDir, "frango_globals.php")
	}

	// Debug log to help identify path issues
	if v.logger != nil {
		v.logger.Printf("Creating embedded file at path: %s (virtual path: %s)", tempPath, virtualPath)
	}

	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing embedded file to '%s': %w", tempPath, err)
	}

	// Calculate hash for change detection
	hash := utils.CalculateContentHash(content)

	// Store mapping
	v.embedMappings[virtualPath] = tempPath
	v.fileOrigins[virtualPath] = OriginEmbed
	v.fileHashes[virtualPath] = FileHash{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	// Track logical path for this embedded file
	v.mutex.Unlock()
	v.TrackLogicalPath(tempPath, virtualPath)
	v.mutex.Lock()

	v.logger.Printf("Added embedded file mapping: %s -> %s (hash: %s)", virtualPath, tempPath, utils.TruncateHash(hash, 8))

	return nil
}

// AddEmbeddedDirectory adds an entire directory from an embed.FS to the VFS
func (v *VFS) AddEmbeddedDirectory(embedFS embed.FS, fsPath string, virtualPrefix string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual prefix
	virtualPrefix = normalizePath(virtualPrefix)

	// List the directory contents
	entries, err := embedFS.ReadDir(fsPath)
	if err != nil {
		return fmt.Errorf("error reading embedded directory '%s': %w", fsPath, err)
	}

	// Process each entry
	for _, entry := range entries {
		entryPath := filepath.Join(fsPath, entry.Name())
		virtualEntryPath := filepath.Join(virtualPrefix, entry.Name())
		virtualEntryPath = strings.ReplaceAll(virtualEntryPath, string(os.PathSeparator), "/")

		if entry.IsDir() {
			// Recursively process subdirectory
			v.mutex.Unlock() // Unlock to allow the recursive call to lock
			if err := v.AddEmbeddedDirectory(embedFS, entryPath, virtualEntryPath); err != nil {
				v.mutex.Lock() // Lock again before returning
				return err
			}
			v.mutex.Lock() // Lock again after recursive call
		} else {
			// Process file
			content, err := embedFS.ReadFile(entryPath)
			if err != nil {
				v.logger.Printf("Warning: Could not read embedded file '%s': %v", entryPath, err)
				continue
			}

			// Create target directory in VFS temp space
			targetDir := filepath.Dir(filepath.Join(v.tempDir, virtualEntryPath))
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				v.logger.Printf("Warning: Could not create directory for embedded file '%s': %v", targetDir, err)
				continue
			}

			// Write to temp path
			tempPath := filepath.Join(v.tempDir, virtualEntryPath)
			if tempPath == v.tempDir || tempPath+"/" == v.tempDir+"/" {
				if v.logger != nil {
					v.logger.Printf("Warning: Virtual path %s would result in writing to the temp directory itself; appending default filename", virtualEntryPath)
				}
				tempPath = filepath.Join(v.tempDir, "frango_globals.php")
			}
			if err := os.WriteFile(tempPath, content, 0644); err != nil {
				v.logger.Printf("Warning: Could not write embedded file to '%s': %v", tempPath, err)
				continue
			}

			// Calculate hash for change detection
			hash := utils.CalculateContentHash(content)

			// Store mapping
			v.embedMappings[virtualEntryPath] = tempPath
			v.fileOrigins[virtualEntryPath] = OriginEmbed
			v.fileHashes[virtualEntryPath] = FileHash{
				Hash:      hash,
				Timestamp: time.Now(),
			}

			v.logger.Printf("Added embedded file from directory: %s -> %s (hash: %s)", virtualEntryPath, tempPath, utils.TruncateHash(hash, 8))
		}
	}

	return nil
}

// CreateVirtualFile creates a file directly in the virtual filesystem with provided content
func (v *VFS) CreateVirtualFile(virtualPath string, content []byte) error {
	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// Use fileMutex for file content operations
	v.fileMutex.Lock()
	defer v.fileMutex.Unlock()

	// Use mutex for structural changes
	v.mutex.Lock()

	// Create target directory in VFS temp space
	targetDir := filepath.Dir(filepath.Join(v.tempDir, virtualPath))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		v.mutex.Unlock()
		return fmt.Errorf("error creating directory for virtual file '%s': %w", targetDir, err)
	}

	// Write to temp path
	tempPath := filepath.Join(v.tempDir, virtualPath)
	if tempPath == v.tempDir || tempPath+"/" == v.tempDir+"/" {
		if v.logger != nil {
			v.logger.Printf("Warning: Virtual path %s would result in writing to the temp directory itself; appending default filename", virtualPath)
		}
		tempPath = filepath.Join(v.tempDir, "frango_globals.php")
	}

	// Debug log to help identify path issues
	if v.logger != nil {
		v.logger.Printf("Creating virtual file at path: %s (virtual path: %s)", tempPath, virtualPath)
	}

	// Release the main lock before I/O
	v.mutex.Unlock()

	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing virtual file to '%s': %w", tempPath, err)
	}

	// Calculate hash for change detection
	hash := utils.CalculateContentHash(content)

	// Re-acquire lock for map updates
	v.mutex.Lock()

	// Store mapping
	v.virtualFiles[virtualPath] = content
	v.embedMappings[virtualPath] = tempPath // Use embed mappings for write access
	v.fileOrigins[virtualPath] = OriginVirtual
	v.fileHashes[virtualPath] = FileHash{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	v.mutex.Unlock()

	// Track logical path for this virtual file
	v.TrackLogicalPath(tempPath, virtualPath)

	v.logger.Printf("Created virtual file: %s (hash: %s)", virtualPath, utils.TruncateHash(hash, 8))

	// Update caches if not in development mode
	if !v.developMode {
		v.cacheMutex.Lock()
		v.pathCache[virtualPath] = tempPath
		v.contentCache[virtualPath] = content
		v.cacheMutex.Unlock()
	}

	return nil
}

// For backward compatibility
func (v *VFS) CopyFileSimple(srcVirtualPath, destVirtualPath string) error {
	return v.CopyFileWithOptions(srcVirtualPath, destVirtualPath, false)
}

// For backward compatibility
func (v *VFS) MoveFileSimple(srcVirtualPath, destVirtualPath string) error {
	return v.MoveFileWithOptions(srcVirtualPath, destVirtualPath, false)
}

// CopyFile is the original function signature, maintained for backward compatibility
func (v *VFS) CopyFile(srcVirtualPath, destVirtualPath string) error {
	return v.CopyFileWithOptions(srcVirtualPath, destVirtualPath, false)
}

// MoveFile is the original function signature, maintained for backward compatibility
func (v *VFS) MoveFile(srcVirtualPath, destVirtualPath string) error {
	return v.MoveFileWithOptions(srcVirtualPath, destVirtualPath, false)
}

// CopyFileWithOptions copies a file with the option to preserve its origin type
func (v *VFS) CopyFileWithOptions(srcVirtualPath, destVirtualPath string, preserveOrigin bool) error {
	// Normalize paths
	srcVirtualPath = normalizePath(srcVirtualPath)
	destVirtualPath = normalizePath(destVirtualPath)

	// Lock for reading source information
	v.mutex.RLock()
	originType, exists := v.fileOrigins[srcVirtualPath]
	var sourcePath string
	var sourceHash FileHash
	var embedPath string // For embedded files

	if exists {
		// Get original path based on origin type
		switch originType {
		case OriginSource:
			sourcePath = v.sourceMappings[srcVirtualPath]
			sourceHash = v.fileHashes[srcVirtualPath]
		case OriginEmbed:
			embedPath = v.embedMappings[srcVirtualPath]
			sourceHash = v.fileHashes[srcVirtualPath]
		}
	} else if v.parent != nil {
		// Check if file exists in parent
		if v.parent.FileExists(srcVirtualPath) {
			// Need to get parent's origin information
			v.mutex.RUnlock()
			return v.copyFromParent(srcVirtualPath, destVirtualPath, preserveOrigin)
		}
	}
	v.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("source file not found in VFS: %s", srcVirtualPath)
	}

	// If preserving origin and it's a source file, create a new source mapping
	if preserveOrigin && originType == OriginSource && sourcePath != "" {
		v.mutex.Lock()
		defer v.mutex.Unlock()

		v.sourceMappings[destVirtualPath] = sourcePath
		v.fileOrigins[destVirtualPath] = OriginSource
		v.fileHashes[destVirtualPath] = sourceHash
		v.logger.Printf("Copied file with preserved source origin: %s -> %s (source: %s)",
			srcVirtualPath, destVirtualPath, sourcePath)
		return nil
	}

	// If preserving origin and it's an embedded file, create a new embed mapping
	if preserveOrigin && originType == OriginEmbed && embedPath != "" {
		v.mutex.Lock()
		defer v.mutex.Unlock()

		v.embedMappings[destVirtualPath] = embedPath
		v.fileOrigins[destVirtualPath] = OriginEmbed
		v.fileHashes[destVirtualPath] = sourceHash
		v.logger.Printf("Copied file with preserved embed origin: %s -> %s (embed: %s)",
			srcVirtualPath, destVirtualPath, embedPath)
		return nil
	}

	// Otherwise, get content and create as a virtual file
	content, err := v.GetFileContent(srcVirtualPath)
	if err != nil {
		return fmt.Errorf("error reading source file '%s': %w", srcVirtualPath, err)
	}

	// Create the destination file as virtual
	return v.CreateVirtualFile(destVirtualPath, content)
}

// MoveFileWithOptions moves a file with the option to preserve its origin type
func (v *VFS) MoveFileWithOptions(srcVirtualPath, destVirtualPath string, preserveOrigin bool) error {
	// First copy the file with origin preservation
	if err := v.CopyFileWithOptions(srcVirtualPath, destVirtualPath, preserveOrigin); err != nil {
		return err
	}

	// Then delete the source
	return v.DeleteFile(srcVirtualPath)
}

// Helper method to copy a file from parent VFS
func (v *VFS) copyFromParent(srcVirtualPath, destVirtualPath string, preserveOrigin bool) error {
	// If preserveOrigin is true, we need to check the origin type in the parent
	if preserveOrigin {
		srcPath, srcOrigin, err := v.getParentPathAndOrigin(srcVirtualPath)
		if err != nil {
			return err
		}

		// If it's a source file in the parent, create a source mapping
		if srcOrigin == OriginSource && srcPath != "" {
			v.mutex.Lock()
			defer v.mutex.Unlock()

			// Get the hash from parent
			var sourceHash FileHash
			v.parent.mutex.RLock()
			if hashInfo, ok := v.parent.fileHashes[srcVirtualPath]; ok {
				sourceHash = hashInfo
			}
			v.parent.mutex.RUnlock()

			v.sourceMappings[destVirtualPath] = srcPath
			v.fileOrigins[destVirtualPath] = OriginSource
			v.fileHashes[destVirtualPath] = sourceHash
			v.logger.Printf("Copied file with preserved source origin from parent: %s -> %s (source: %s)",
				srcVirtualPath, destVirtualPath, srcPath)
			return nil
		}

		// If it's an embedded file in the parent, preserve that too
		if srcOrigin == OriginEmbed {
			v.mutex.Lock()
			defer v.mutex.Unlock()

			// Get embed path and hash from parent
			var embedPath string
			var embedHash FileHash

			v.parent.mutex.RLock()
			if path, ok := v.parent.embedMappings[srcVirtualPath]; ok {
				embedPath = path
			}
			if hashInfo, ok := v.parent.fileHashes[srcVirtualPath]; ok {
				embedHash = hashInfo
			}
			v.parent.mutex.RUnlock()

			if embedPath != "" {
				v.embedMappings[destVirtualPath] = embedPath
				v.fileOrigins[destVirtualPath] = OriginEmbed
				v.fileHashes[destVirtualPath] = embedHash
				v.logger.Printf("Copied file with preserved embed origin from parent: %s -> %s (embed: %s)",
					srcVirtualPath, destVirtualPath, embedPath)
				return nil
			}
		}
	}

	// Otherwise, get content and create as a virtual file
	content, err := v.parent.GetFileContent(srcVirtualPath)
	if err != nil {
		return fmt.Errorf("error reading source file from parent '%s': %w", srcVirtualPath, err)
	}

	return v.CreateVirtualFile(destVirtualPath, content)
}

// Helper method to get path and origin type from parent VFS
func (v *VFS) getParentPathAndOrigin(virtualPath string) (string, FileOrigin, error) {
	if v.parent == nil {
		return "", "", fmt.Errorf("no parent VFS")
	}

	v.parent.mutex.RLock()
	defer v.parent.mutex.RUnlock()

	originType, exists := v.parent.fileOrigins[virtualPath]
	if !exists {
		// Check if parent has a parent recursively
		if v.parent.parent != nil {
			return v.parent.getParentPathAndOrigin(virtualPath)
		}
		return "", "", fmt.Errorf("file not found in parent VFS: %s", virtualPath)
	}

	// Get the actual path based on origin type
	var sourcePath string
	if originType == OriginSource {
		sourcePath = v.parent.sourceMappings[virtualPath]
	} else if originType == OriginEmbed {
		sourcePath = v.parent.embedMappings[virtualPath]
	}

	return sourcePath, originType, nil
}

// DeleteFile removes a file from the VFS
func (v *VFS) DeleteFile(virtualPath string) error {
	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// Lock for structural changes
	v.mutex.Lock()

	// Check if file exists in VFS
	origin, exists := v.fileOrigins[virtualPath]
	if !exists {
		v.mutex.Unlock()
		return fmt.Errorf("file not found in VFS: %s", virtualPath)
	}

	// Get the physical path before removing from maps if applicable
	var physicalPath string
	if origin == OriginSource {
		physicalPath = v.sourceMappings[virtualPath]
	} else if origin == OriginEmbed || origin == OriginVirtual {
		physicalPath = v.embedMappings[virtualPath]
	}

	// Special handling for inherited paths - we need to shadow them
	if origin == OriginInherited {
		// Instead of deleting, create a virtual "tombstone" file
		v.virtualFiles[virtualPath] = nil // nil content means "deleted/shadowed"
		v.fileOrigins[virtualPath] = OriginVirtual
		v.mutex.Unlock()

		// Invalidate caches
		v.invalidateCaches(virtualPath)

		// Notify shadowed file event if in development mode
		if v.developMode && physicalPath != "" {
			v.notifyFileChanged(virtualPath, physicalPath, "shadowed")
		}

		v.logger.Printf("Shadowed inherited file: %s", virtualPath)
		return nil
	}

	// Get temp file path before removing from maps if applicable
	var tempPath string
	if origin == OriginEmbed || origin == OriginVirtual {
		if path, ok := v.embedMappings[virtualPath]; ok {
			tempPath = path
		}
	}

	// Remove mappings based on origin type
	if origin == OriginSource {
		delete(v.sourceMappings, virtualPath)
	} else if origin == OriginEmbed || origin == OriginVirtual {
		delete(v.embedMappings, virtualPath)
		delete(v.virtualFiles, virtualPath)
	}

	// Remove all other mappings
	delete(v.fileOrigins, virtualPath)
	delete(v.fileHashes, virtualPath)
	delete(v.changedFiles, virtualPath)

	v.mutex.Unlock()

	// Try to remove the temp file but don't error if it fails
	if tempPath != "" {
		_ = os.Remove(tempPath)
	}

	// Invalidate caches
	v.invalidateCaches(virtualPath)

	// Notify file deleted event if in development mode
	if v.developMode && physicalPath != "" {
		v.notifyFileChanged(virtualPath, physicalPath, "deleted")
	}

	v.logger.Printf("Deleted file from VFS: %s", virtualPath)
	return nil
}

// invalidateCaches removes a path from all caches
func (v *VFS) invalidateCaches(virtualPath string) {
	v.cacheMutex.Lock()
	delete(v.pathCache, virtualPath)
	delete(v.contentCache, virtualPath)
	v.cacheMutex.Unlock()
}

// ListFiles returns a list of all files in the VFS
func (v *VFS) ListFiles() []string {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Create a map of all files (for deduplication)
	files := make(map[string]bool)

	// Add files from this VFS
	for path, origin := range v.fileOrigins {
		// Skip virtual "tombstone" files (nil content means deleted/shadowed)
		if origin == OriginVirtual && v.virtualFiles[path] == nil {
			continue
		}
		files[path] = true
	}

	// Add files from parent VFS (if any)
	if v.parent != nil {
		parentFiles := v.parent.ListFiles()
		for _, path := range parentFiles {
			// Check if this file is shadowed in current VFS
			if origin, exists := v.fileOrigins[path]; exists && origin == OriginVirtual && v.virtualFiles[path] == nil {
				continue // Skip shadowed files
			}
			files[path] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(files))
	for path := range files {
		result = append(result, path)
	}

	return result
}

// listFilesIn returns a list of files in the specified virtual directory
func (vfs *VFS) listFilesIn(dirPath string) ([]string, error) {
	vfs.mutex.RLock()
	defer vfs.mutex.RUnlock()

	// Normalize the directory path
	dirPath = normalizePath(dirPath)
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	filesList := []string{}

	// Find files that start with the directory path
	for path := range vfs.fileOrigins {
		if strings.HasPrefix(path, dirPath) {
			// Check if it's a direct child of the directory
			// (no further subdirectories in the relative path)
			relPath := strings.TrimPrefix(path, dirPath)
			if !strings.Contains(relPath, "/") {
				filesList = append(filesList, path)
			}
		}
	}

	if len(filesList) == 0 {
		return nil, fmt.Errorf("no files found in directory: %s", dirPath)
	}

	return filesList, nil
}
