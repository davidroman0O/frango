package vfs

import (
	"os"
	"sync"
	"time"

	"github.com/davidroman0O/frango/v2/internal/utils"
)

// GlobalWatcher provides a centralized system for watching file changes across all VFS instances
type GlobalWatcher struct {
	watchedFiles     map[string]map[*VFS]bool // Map of file paths to VFS instances watching them
	watchedInstances map[*VFS]bool            // Set of VFS instances being watched
	mutex            sync.RWMutex
	ticker           *time.Ticker
	stopChan         chan bool
	started          bool
}

// global watcher singleton
var (
	globalWatcher     *GlobalWatcher
	globalWatcherOnce sync.Once
)

// GetGlobalWatcher returns the singleton global watcher instance
func GetGlobalWatcher() *GlobalWatcher {
	globalWatcherOnce.Do(func() {
		globalWatcher = &GlobalWatcher{
			watchedFiles:     make(map[string]map[*VFS]bool),
			watchedInstances: make(map[*VFS]bool),
			stopChan:         make(chan bool),
			started:          false,
		}
	})
	return globalWatcher
}

// Start begins the file watching process if it hasn't already started
func (w *GlobalWatcher) Start() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.started {
		return
	}

	w.ticker = time.NewTicker(500 * time.Millisecond)
	w.started = true

	go w.watchLoop()
}

// Stop halts the file watching process
func (w *GlobalWatcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.started {
		return
	}

	w.stopChan <- true
	w.started = false
}

// RegisterVFS adds a VFS instance to the global watcher
func (w *GlobalWatcher) RegisterVFS(vfs *VFS) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.watchedInstances[vfs] = true

	// Make sure the watcher is running
	if !w.started && len(w.watchedInstances) > 0 {
		w.mutex.Unlock()
		w.Start()
		w.mutex.Lock()
	}
}

// UnregisterVFS removes a VFS instance from the global watcher
func (w *GlobalWatcher) UnregisterVFS(vfs *VFS) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Remove VFS from instances map
	delete(w.watchedInstances, vfs)

	// Remove VFS from all file watchers
	for filePath, vfsMap := range w.watchedFiles {
		delete(vfsMap, vfs)
		if len(vfsMap) == 0 {
			delete(w.watchedFiles, filePath)
		}
	}

	// If no more VFS instances are being watched, stop the watcher
	if len(w.watchedInstances) == 0 {
		w.mutex.Unlock()
		w.Stop()
		w.mutex.Lock()
	}
}

// RegisterFile adds a file to be watched for a specific VFS
func (w *GlobalWatcher) RegisterFile(vfs *VFS, filePath string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Add to watched files map
	vfsMap, exists := w.watchedFiles[filePath]
	if !exists {
		vfsMap = make(map[*VFS]bool)
		w.watchedFiles[filePath] = vfsMap
	}
	vfsMap[vfs] = true
}

// watchLoop is the main goroutine that checks for file changes
func (w *GlobalWatcher) watchLoop() {
	for {
		select {
		case <-w.ticker.C:
			w.checkAllFiles()
		case <-w.stopChan:
			w.ticker.Stop()
			return
		}
	}
}

// checkAllFiles checks all registered files for changes
func (w *GlobalWatcher) checkAllFiles() {
	w.mutex.RLock()
	filesToCheck := make(map[string]map[*VFS]bool)

	// Create a copy of the watched files map to avoid holding the lock during I/O
	for filePath, vfsMap := range w.watchedFiles {
		vfsCopy := make(map[*VFS]bool)
		for vfs := range vfsMap {
			vfsCopy[vfs] = true
		}
		filesToCheck[filePath] = vfsCopy
	}
	w.mutex.RUnlock()

	// Check each file for changes
	for filePath, vfsMap := range filesToCheck {
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		// Calculate new hash
		newHash, err := utils.CalculateFileHash(filePath)
		if err != nil {
			continue
		}

		// Notify each VFS that's watching this file
		for vfs := range vfsMap {
			vfs.mutex.RLock()
			oldHashInfo, exists := vfs.fileHashes[filePath]
			oldHash := ""
			if exists {
				oldHash = oldHashInfo.Hash
			}
			vfs.mutex.RUnlock()

			// If hash changed, mark file as changed in this VFS
			if exists && oldHash != newHash {
				var virtualPath string

				vfs.mutex.Lock()
				// Find the virtual path corresponding to this physical path
				for vPath, sourcePath := range vfs.sourceMappings {
					if sourcePath == filePath {
						virtualPath = vPath
						break
					}
				}

				vfs.fileHashes[filePath] = FileHash{
					Hash:      newHash,
					Timestamp: time.Now(),
				}
				vfs.changedFiles[filePath] = true
				vfs.invalidated = true
				vfs.logger.Printf("Source file changed: %s (hash: %s -> %s)",
					filePath, utils.TruncateHash(oldHash, 8), utils.TruncateHash(newHash, 8))
				vfs.mutex.Unlock()

				// If we found a virtual path, notify file change handlers
				if virtualPath != "" {
					// Call the notification outside of the lock
					vfs.logger.Printf("Watcher: Detected hash change for %s, triggering notifyFileChanged", filePath)
					vfs.notifyFileChanged(virtualPath, filePath, "modified")
				}
			}
		}
	}
}
