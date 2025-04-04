/*
Package frango provides a virtual file system for PHP scripts.

Platform-specific notes:
- Path handling: Different platforms handle paths differently, particularly around:
  * Case sensitivity: Windows is case-insensitive, Unix-like systems are case-sensitive
  * Path separators: Windows uses backslashes, Unix uses forward slashes
  * Maximum path length: Windows has stricter limits than Unix systems
  * Special filenames: Windows has reserved names like 'con', 'nul', etc.

- Further testing needed: The VFS implementation has been tested primarily on Unix-like systems.
  Additional testing is needed on Windows systems to ensure compatibility, especially around:
  * Case insensitivity handling
  * Long path support
  * Path traversal normalization
  * Extended character support in filenames
*/

package frango

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileOrigin represents the source type of a file in the VFS
type FileOrigin string

const (
	// OriginSource indicates a file from the filesystem
	OriginSource FileOrigin = "source"
	// OriginEmbed indicates a file from an embed.FS
	OriginEmbed FileOrigin = "embed"
	// OriginVirtual indicates a file created programmatically
	OriginVirtual FileOrigin = "virtual"
	// OriginInherited indicates a file inherited from a parent VFS
	OriginInherited FileOrigin = "inherited"
)

// FileHash stores a hash and timestamp to track file changes
type FileHash struct {
	Hash      string    // SHA-256 hash of the file content
	Timestamp time.Time // When the hash was calculated
}

// VFS represents a virtual filesystem container for PHP files with branching capability
type VFS struct {
	name           string                // Unique identifier for this VFS
	parent         *VFS                  // Parent VFS (if this is a branch)
	sourceMappings map[string]string     // Virtual path -> source path (for files on disk)
	embedMappings  map[string]string     // Virtual path -> embed temp path (for embedded files)
	virtualFiles   map[string][]byte     // Virtual path -> content (for in-memory files)
	fileOrigins    map[string]FileOrigin // Virtual path -> origin type
	fileHashes     map[string]FileHash   // Path -> hash info (for change detection)
	tempDir        string                // Base temp directory for this VFS
	mutex          sync.RWMutex          // For thread safety
	watchTicker    *time.Ticker          // For file watching
	watchStop      chan bool             // To signal watching to stop
	logger         *log.Logger           // For logging operations
	invalidated    bool                  // Whether any files need refreshing
	changedFiles   map[string]bool       // Tracks which files have changed
	inheritedPaths map[string]bool       // Which paths come from parent VFS
	developMode    bool                  // Whether development mode is enabled
	globalLibs     map[string]string     // Path -> temp path for global libraries
	phpGlobalsFile string                // Path to the PHP globals script in this VFS
	refCount       int                   // Number of child VFS instances referencing this one
	refMutex       sync.Mutex            // Separate mutex for reference counting
	isCleanedUp    bool                  // Whether this VFS has been cleaned up
}

// NewVFS creates a new virtual filesystem
func NewVFS(tempDir string, logger *log.Logger, developMode bool) (*VFS, error) {
	// Create unique ID for this VFS
	id := generateVFSID()

	// Create base temp directory for this VFS
	vfsTempDir := filepath.Join(tempDir, "vfs-"+id)
	if err := os.MkdirAll(vfsTempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create VFS temp directory: %w", err)
	}

	v := &VFS{
		name:           id,
		sourceMappings: make(map[string]string),
		embedMappings:  make(map[string]string),
		virtualFiles:   make(map[string][]byte),
		fileOrigins:    make(map[string]FileOrigin),
		fileHashes:     make(map[string]FileHash),
		tempDir:        vfsTempDir,
		watchStop:      make(chan bool),
		logger:         logger,
		changedFiles:   make(map[string]bool),
		inheritedPaths: make(map[string]bool),
		developMode:    developMode,
		globalLibs:     make(map[string]string),
		refCount:       0, // Initialize reference count to 0
		isCleanedUp:    false,
	}

	// Initialize with PHP globals
	if err := v.initializeGlobals(); err != nil {
		// Clean up on failure
		os.RemoveAll(vfsTempDir)
		return nil, err
	}

	// Start file watching if in development mode
	if developMode {
		v.startWatching()
	}

	return v, nil
}

// Branch creates a new VFS that inherits from this one
func (v *VFS) Branch() *VFS {
	v.mutex.RLock()

	// Check if already cleaned up
	if v.isCleanedUp {
		v.mutex.RUnlock()
		v.logger.Printf("Warning: Trying to branch from cleaned up VFS: %s", v.name)
		return nil
	}

	branchVFS := &VFS{
		name:           generateVFSID(),
		parent:         v,
		sourceMappings: make(map[string]string),
		embedMappings:  make(map[string]string),
		virtualFiles:   make(map[string][]byte),
		fileOrigins:    make(map[string]FileOrigin),
		fileHashes:     make(map[string]FileHash),
		tempDir:        filepath.Join(filepath.Dir(v.tempDir), "vfs-branch-"+generateVFSID()),
		watchStop:      make(chan bool),
		logger:         v.logger,
		changedFiles:   make(map[string]bool),
		inheritedPaths: make(map[string]bool),
		developMode:    v.developMode,
		globalLibs:     make(map[string]string),
	}
	v.mutex.RUnlock()

	// Increment parent reference count
	v.refMutex.Lock()
	v.refCount++
	v.logger.Printf("Branched VFS %s from %s (new ref count: %d)", branchVFS.name, v.name, v.refCount)
	v.refMutex.Unlock()

	// Create temp directory for branch
	os.MkdirAll(branchVFS.tempDir, 0755)

	// Initialize with PHP globals
	if err := branchVFS.initializeGlobals(); err != nil {
		branchVFS.logger.Printf("Warning: Failed to initialize globals in branch VFS: %v", err)
	}

	// Start watching if in develop mode
	if branchVFS.developMode {
		branchVFS.startWatching()
	}

	return branchVFS
}

// wouldCreateCircularReference checks if adding 'potential' as a parent would create a circular reference
func (v *VFS) wouldCreateCircularReference(potential *VFS) bool {
	// If potential is nil, there's no reference
	if potential == nil {
		return false
	}

	// If potential is this VFS, it would create a circular reference
	if v == potential {
		return true
	}

	// Check up the parent chain of 'potential' to see if 'v' is in it
	current := potential.parent
	for current != nil {
		if current == v {
			return true
		}
		current = current.parent
	}

	return false
}

// AddSourceFile adds a file from the filesystem to the VFS
func (v *VFS) AddSourceFile(sourcePath, virtualPath string) error {
	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// Check for symlinks
	fileInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("error accessing source file '%s': %w", sourcePath, err)
	}

	// Prevent symlinks for security reasons
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not supported for security reasons: %s", sourcePath)
	}

	// Lock the VFS for writing
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Calculate hash for change detection
	hash, err := calculateFileHash(sourcePath)
	if err != nil {
		return fmt.Errorf("error calculating hash for '%s': %w", sourcePath, err)
	}

	// Store mappings
	v.sourceMappings[virtualPath] = sourcePath
	v.fileOrigins[virtualPath] = OriginSource
	v.fileHashes[virtualPath] = FileHash{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	v.logger.Printf("Added source file: %s -> %s (hash: %s)", sourcePath, virtualPath, truncateHash(hash))

	return nil
}

// AddSourceDirectory adds all PHP files from a directory to the VFS
func (v *VFS) AddSourceDirectory(sourceDir string, virtualBasePath string) error {
	return v.addSourceDirectoryRecursive(sourceDir, virtualBasePath, true)
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
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing embedded file to '%s': %w", tempPath, err)
	}

	// Calculate hash for change detection
	hash := calculateContentHash(content)

	// Store mapping
	v.embedMappings[virtualPath] = tempPath
	v.fileOrigins[virtualPath] = OriginEmbed
	v.fileHashes[virtualPath] = FileHash{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	v.logger.Printf("Added embedded file mapping: %s -> %s (hash: %s)", virtualPath, tempPath, truncateHash(hash))

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
			if err := os.WriteFile(tempPath, content, 0644); err != nil {
				v.logger.Printf("Warning: Could not write embedded file to '%s': %v", tempPath, err)
				continue
			}

			// Calculate hash for change detection
			hash := calculateContentHash(content)

			// Store mapping
			v.embedMappings[virtualEntryPath] = tempPath
			v.fileOrigins[virtualEntryPath] = OriginEmbed
			v.fileHashes[virtualEntryPath] = FileHash{
				Hash:      hash,
				Timestamp: time.Now(),
			}

			v.logger.Printf("Added embedded file from directory: %s -> %s (hash: %s)", virtualEntryPath, tempPath, truncateHash(hash))
		}
	}

	return nil
}

// CreateVirtualFile creates a file directly in the virtual filesystem with provided content
func (v *VFS) CreateVirtualFile(virtualPath string, content []byte) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// Create target directory in VFS temp space
	targetDir := filepath.Dir(filepath.Join(v.tempDir, virtualPath))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("error creating directory for virtual file '%s': %w", targetDir, err)
	}

	// Write to temp path
	tempPath := filepath.Join(v.tempDir, virtualPath)
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing virtual file to '%s': %w", tempPath, err)
	}

	// Calculate hash for change detection
	hash := calculateContentHash(content)

	// Store mapping
	v.virtualFiles[virtualPath] = content
	v.embedMappings[virtualPath] = tempPath // Use embed mappings for write access
	v.fileOrigins[virtualPath] = OriginVirtual
	v.fileHashes[virtualPath] = FileHash{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	v.logger.Printf("Created virtual file: %s (hash: %s)", virtualPath, truncateHash(hash))

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
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = normalizePath(virtualPath)

	// Check if file exists in VFS
	origin, exists := v.fileOrigins[virtualPath]
	if !exists {
		return fmt.Errorf("file not found in VFS: %s", virtualPath)
	}

	// Special handling for inherited paths - we need to shadow them
	if origin == OriginInherited {
		// Instead of deleting, create a virtual "tombstone" file
		v.virtualFiles[virtualPath] = nil // nil content means "deleted/shadowed"
		v.fileOrigins[virtualPath] = OriginVirtual
		v.logger.Printf("Shadowed inherited file: %s", virtualPath)
		return nil
	}

	// Remove mappings based on origin type
	if origin == OriginSource {
		delete(v.sourceMappings, virtualPath)
	} else if origin == OriginEmbed || origin == OriginVirtual {
		if tempPath, ok := v.embedMappings[virtualPath]; ok {
			// Try to remove the temp file but don't error if it fails
			_ = os.Remove(tempPath)
			delete(v.embedMappings, virtualPath)
		}
		delete(v.virtualFiles, virtualPath)
	}

	// Remove all other mappings
	delete(v.fileOrigins, virtualPath)
	delete(v.fileHashes, virtualPath)
	delete(v.changedFiles, virtualPath)

	v.logger.Printf("Deleted file from VFS: %s", virtualPath)
	return nil
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

// checkFileChanges checks a specific file for changes
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

// Cleanup cleans up resources associated with this VFS
func (v *VFS) Cleanup() {
	// Mark as cleaned up to prevent new operations
	v.refMutex.Lock()
	if v.isCleanedUp {
		v.refMutex.Unlock()
		return // Already cleaned up
	}
	v.isCleanedUp = true
	refCount := v.refCount
	v.refMutex.Unlock()

	// Stop file watching (no longer needed once cleanup is called)
	v.stopWatcher()

	// If we have a parent and this is our first call to Cleanup, decrement parent's reference count
	// Do this regardless of whether we're deferring the actual cleanup
	if v.parent != nil {
		v.parent.refMutex.Lock()
		v.parent.refCount--
		parentRefCount := v.parent.refCount
		parentIsCleanedUp := v.parent.isCleanedUp
		parentVFS := v.parent // Store parent locally to avoid race conditions
		v.parent.refMutex.Unlock()

		v.logger.Printf("Removed reference to parent VFS %s (parent ref count now: %d, cleaned up: %v)",
			parentVFS.name, parentRefCount, parentIsCleanedUp)

		// If parent's refCount dropped to 0 and it's marked for cleanup, clean it up
		if parentRefCount == 0 && parentIsCleanedUp {
			v.logger.Printf("Triggering cleanup of parent VFS %s as it's marked for cleanup and ref count is 0",
				parentVFS.name)
			// Avoid recursion by using a goroutine
			go func(parent *VFS) {
				// Brief delay to ensure all operations have completed
				time.Sleep(10 * time.Millisecond)
				parent.completeCleanup()
			}(parentVFS)
		}
	}

	// Don't fully clean up if there are still references to this VFS
	if refCount > 0 {
		v.logger.Printf("Deferring full cleanup of VFS %s - %d children still referencing it",
			v.name, refCount)
		return
	}

	// Complete our own cleanup
	v.completeCleanup()
}

// stopWatcher stops the file watching ticker if it's running
func (v *VFS) stopWatcher() {
	if v.watchTicker != nil {
		select {
		case v.watchStop <- true:
			// Successfully sent stop signal
		case <-time.After(100 * time.Millisecond):
			// Timeout - force cleanup
			v.mutex.Lock()
			if v.watchTicker != nil {
				v.watchTicker.Stop()
				v.watchTicker = nil
			}
			v.mutex.Unlock()
		}
	}
}

// completeCleanup performs the actual cleanup of resources
func (v *VFS) completeCleanup() {
	// Remove temp directory
	v.mutex.Lock()
	tempDir := v.tempDir
	v.mutex.Unlock()

	if tempDir != "" {
		// Ensure directory exists before removing
		if _, err := os.Stat(tempDir); err == nil {
			err := os.RemoveAll(tempDir)
			if err != nil {
				v.logger.Printf("Warning: Failed to remove temp directory %s: %v", tempDir, err)
			} else {
				v.logger.Printf("Removed temp directory for VFS %s: %s", v.name, tempDir)
			}
		}
	}

	v.logger.Printf("VFS fully cleaned up: %s", v.name)
}

// --- Internal Utility Methods ---

// startWatching starts a goroutine to watch for file changes
func (v *VFS) startWatching() {
	// Don't start watching if already running
	v.mutex.Lock()
	if v.watchTicker != nil {
		v.mutex.Unlock()
		return
	}
	v.watchTicker = time.NewTicker(500 * time.Millisecond)
	v.mutex.Unlock()

	// Run the watcher in a separate goroutine
	go func() {
		for {
			select {
			case <-v.watchTicker.C:
				// Use the safe version that doesn't deadlock
				v.checkForChanges()
			case <-v.watchStop:
				v.mutex.Lock()
				if v.watchTicker != nil {
					v.watchTicker.Stop()
					v.watchTicker = nil
				}
				v.mutex.Unlock()
				return
			}
		}
	}()
}

// checkForChanges checks if any source files have changed
func (v *VFS) checkForChanges() {
	// Make a list of paths to check while holding a read lock
	v.mutex.RLock()
	sourcePaths := make([]string, 0, len(v.sourceMappings))
	// Only collect source paths
	for virtualPath, origin := range v.fileOrigins {
		if origin == OriginSource {
			sourcePaths = append(sourcePaths, virtualPath)
		}
	}
	v.mutex.RUnlock()

	// Check each file individually
	for _, virtualPath := range sourcePaths {
		v.checkFileChanges(virtualPath)
	}
}

// initializeGlobals creates the PHP globals file in the VFS
func (v *VFS) initializeGlobals() error {
	// Path globals PHP file - this is the content of the file
	pathGlobalsContent := `<?php
/**
 * Frango v1 path globals initialization
 * 
 * Initializes the following PHP superglobals:
 * - $_PATH: Contains path parameters extracted from URL patterns
 * - $_PATH_SEGMENTS: Contains URL path segments
 */

// Initialize $_PATH superglobal for path parameters
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available
    $pathParamsJson = $_SERVER['PHP_PATH_PARAMS'] ?? '{}';
    $decodedParams = json_decode($pathParamsJson, true);
    if (is_array($decodedParams)) {
        $_PATH = $decodedParams;
    }
    
    // Also add any PHP_PATH_PARAM_ variables from $_SERVER for backward compatibility
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'PHP_PATH_PARAM_') === 0) {
            $paramName = substr($key, strlen('PHP_PATH_PARAM_'));
            if (!isset($_PATH[$paramName])) {
                $_PATH[$paramName] = $value;
            }
        }
    }
    
    // Make sure $_PATH is globally accessible
    $GLOBALS['_PATH'] = $_PATH;
}

// Initialize $_PATH_SEGMENTS superglobal for URL segments
if (!isset($_PATH_SEGMENTS)) {
    $_PATH_SEGMENTS = [];
    
    // Get segment count
    $segmentCount = intval($_SERVER['PHP_PATH_SEGMENT_COUNT'] ?? 0);
    
    // Add segments to array
    for ($i = 0; $i < $segmentCount; $i++) {
        $segmentKey = "PHP_PATH_SEGMENT_$i";
        if (isset($_SERVER[$segmentKey])) {
            $_PATH_SEGMENTS[] = $_SERVER[$segmentKey];
        }
    }
    
    // Make sure $_PATH_SEGMENTS is globally accessible
    $GLOBALS['_PATH_SEGMENTS'] = $_PATH_SEGMENTS;
    $GLOBALS['_PATH_SEGMENT_COUNT'] = $segmentCount;
}

// Initialize $_JSON for parsed JSON request body
if (!isset($_JSON)) {
    $_JSON = [];
    
    // Load from JSON if available
    $jsonBody = $_SERVER['PHP_JSON'] ?? '{}';
    $decodedJson = json_decode($jsonBody, true);
    if (is_array($decodedJson)) {
        $_JSON = $decodedJson;
    }
    
    // Make sure $_JSON is globally accessible
    $GLOBALS['_JSON'] = $_JSON;
}

// Helper function to get path segments
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

// Initialize template variables from PHP_VAR_ environment variables
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_VAR_') === 0) {
        $varName = substr($key, strlen('PHP_VAR_'));
        $varValue = json_decode($value, true);
        $GLOBALS[$varName] = $varValue;
    }
}
`

	// Create the PHP globals file
	globalsPath := "/_frango_php_globals.php"
	if err := v.CreateVirtualFile(globalsPath, []byte(pathGlobalsContent)); err != nil {
		return fmt.Errorf("failed to create PHP globals file: %w", err)
	}

	v.phpGlobalsFile = globalsPath
	return nil
}

// --- Static Utility Functions ---

// generateVFSID generates a unique ID for VFS instances
func generateVFSID() string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash.Sum(nil))[:8]
}

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

// calculateContentHash calculates the SHA-256 hash of a byte slice
func calculateContentHash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// truncateHash truncates a hash string for display purposes
func truncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
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
