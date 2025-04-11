package frango

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// NewVFS creates a new virtual filesystem
func NewVFS(tempDir string, logger *log.Logger, developMode bool) (*VFS, error) {
	return NewVFSWithConfig(VFSConfig{
		TempDir:         tempDir,
		Logger:          logger,
		DevelopMode:     developMode,
		GlobalsProvider: &DefaultGlobalsProvider{},
	})
}

// NewVFSWithConfig creates a new virtual filesystem with the specified configuration
func NewVFSWithConfig(config VFSConfig) (*VFS, error) {
	// Create unique ID for this VFS
	id := generateVFSID()

	// Create base temp directory for this VFS
	vfsTempDir := filepath.Join(config.TempDir, "vfs-"+id)
	if err := os.MkdirAll(vfsTempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create VFS temp directory: %w", err)
	}

	// Set default globals provider if not specified
	if config.GlobalsProvider == nil {
		config.GlobalsProvider = &DefaultGlobalsProvider{}
	}

	v := &VFS{
		name:            id,
		sourceMappings:  make(map[string]string),
		embedMappings:   make(map[string]string),
		virtualFiles:    make(map[string][]byte),
		fileOrigins:     make(map[string]FileOrigin),
		fileHashes:      make(map[string]FileHash),
		tempDir:         vfsTempDir,
		watchStop:       make(chan bool),
		logger:          config.Logger,
		changedFiles:    make(map[string]bool),
		inheritedPaths:  make(map[string]bool),
		developMode:     config.DevelopMode,
		globalLibs:      make(map[string]string),
		refCount:        0, // Initialize reference count to 0
		isCleanedUp:     false,
		globalsProvider: config.GlobalsProvider,
	}

	// Initialize with PHP globals
	if err := v.initializeGlobals(); err != nil {
		// Clean up on failure
		os.RemoveAll(vfsTempDir)
		return nil, err
	}

	// Start file watching if in development mode
	if config.DevelopMode {
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
		name:            generateVFSID(),
		parent:          v,
		sourceMappings:  make(map[string]string),
		embedMappings:   make(map[string]string),
		virtualFiles:    make(map[string][]byte),
		fileOrigins:     make(map[string]FileOrigin),
		fileHashes:      make(map[string]FileHash),
		tempDir:         filepath.Join(filepath.Dir(v.tempDir), "vfs-branch-"+generateVFSID()),
		watchStop:       make(chan bool),
		logger:          v.logger,
		changedFiles:    make(map[string]bool),
		inheritedPaths:  make(map[string]bool),
		developMode:     v.developMode,
		globalLibs:      make(map[string]string),
		globalsProvider: v.globalsProvider, // Inherit globals provider from parent
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

// initializeGlobals creates the PHP globals file in the VFS
func (v *VFS) initializeGlobals() error {
	// Get globals path from provider
	globalsPath := v.globalsProvider.GetPHPGlobalsPath()

	// Get script content from provider
	script := v.globalsProvider.GetPHPGlobalsScript()

	// Create the globals file in the VFS
	if err := v.CreateVirtualFile(globalsPath, []byte(script)); err != nil {
		return fmt.Errorf("failed to create PHP globals file: %w", err)
	}

	v.phpGlobalsFile = globalsPath
	return nil
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

// startWatching registers this VFS with the global watcher
func (v *VFS) startWatching() {
	GetGlobalWatcher().RegisterVFS(v)

	// Register existing source files
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	for virtualPath, origin := range v.fileOrigins {
		if origin == OriginSource {
			sourcePath := v.sourceMappings[virtualPath]
			GetGlobalWatcher().RegisterFile(v, sourcePath)
		}
	}
}

// stopWatcher stops the file watching
func (v *VFS) stopWatcher() {
	// Unregister this VFS from the global watcher
	GetGlobalWatcher().UnregisterVFS(v)
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
