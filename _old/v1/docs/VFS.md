# Frango Virtual File System (VFS) Documentation

## Architecture Overview

The Frango VFS provides a comprehensive virtual file system that abstracts filesystem operations for PHP script execution. It supports multiple file origins, hierarchical branching, reference counting, and robust security features.

```
┌─────────────────────────────────────────────────────────────┐
│                        Frango VFS                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌───────────────┐    ┌───────────────┐    ┌───────────────┐│
│  │  File Origins │    │ VFS Branching │    │ Memory & Ref  ││
│  │               │    │               │    │   Counting    ││
│  └───────────────┘    └───────────────┘    └───────────────┘│
│                                                             │
│  ┌───────────────┐    ┌───────────────┐    ┌───────────────┐│
│  │File Operations│    │Path Processing│    │ Security &    ││
│  │               │    │& Normalization│    │ Symlink Prot. ││
│  └───────────────┘    └───────────────┘    └───────────────┘│
│                                                             │
│  ┌───────────────┐    ┌───────────────┐    ┌───────────────┐│
│  │ Concurrency   │    │ Change        │    │ Cleanup &     ││
│  │ Protection    │    │ Detection     │    │ Resource Mgmt ││
│  └───────────────┘    └───────────────┘    └───────────────┘│
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 1. Core Data Structures

### 1.1 VFS Structure

The VFS is built around a core structure that maintains all mappings and state:

```go
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
```

### 1.2 File Origins

Files in the VFS can have four different origins:

```go
type FileOrigin int

const (
    OriginSource    FileOrigin = iota  // From local filesystem
    OriginEmbed                        // From embedded filesystem
    OriginVirtual                      // Created programmatically
    OriginInherited                    // Inherited from parent VFS
)
```

Origin types affect how files are processed, copied, and updated:

```
┌─────────────────┬───────────────────────────────────────────────────┐
│ Origin Type     │ Behavior                                          │
├─────────────────┼───────────────────────────────────────────────────┤
│ OriginSource    │ • Points to real filesystem file                  │
│                 │ • Can detect source changes                       │
│                 │ • Real-time updating in development mode          │
├─────────────────┼───────────────────────────────────────────────────┤
│ OriginEmbed     │ • Extracted from embedded FS to temp file         │
│                 │ • Static content (doesn't change at runtime)      │
│                 │ • Cleanup removes temp files                      │
├─────────────────┼───────────────────────────────────────────────────┤
│ OriginVirtual   │ • Exists only in memory                           │
│                 │ • Created/modified programmatically               │
│                 │ • Not affected by external changes                │
├─────────────────┼───────────────────────────────────────────────────┤
│ OriginInherited │ • References file in parent VFS                   │
│                 │ • Deletion creates "tombstone" file               │
│                 │ • Inherits parent's change detection              │
└─────────────────┴───────────────────────────────────────────────────┘
```

### 1.3 File Hash Information

For change detection, each file has associated hash information:

```go
type FileHash struct {
    Hash         string    // MD5 hash of file content
    LastModified time.Time // Last time the file was modified  
    Size         int64     // Size of the file in bytes
}
```

## 2. Lifecycle Management

### 2.1 Initialization Flow

The VFS initialization process follows this sequence:

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  NewVFS()   │────▶│ Generate Unique  │────▶│Create Temp Dir  │
└─────────────┘     │     VFS ID       │     └─────────────────┘
                    └──────────────────┘              │
                                                      ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Return VFS     │◀────│ Start Watcher    │◀────│ Initialize      │
│   Instance      │     │  (if dev mode)   │     │  PHP Globals    │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

Initialization code:

```go
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
```

### 2.2 Branching Architecture

VFS branching creates a hierarchical relationship between VFS instances:

```
                 ┌─────────────┐
                 │  Parent VFS │
                 └─────────────┘
                        │
              ┌─────────┴─────────┐
              │                   │
     ┌─────────────┐     ┌─────────────┐
     │  Child VFS  │     │  Child VFS  │
     └─────────────┘     └─────────────┘
              │
     ┌─────────────┐
     │Grandchild VFS│
     └─────────────┘
```

Branching implementation:

```go
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
```

### 2.3 Reference Counting System

The reference counting system prevents premature cleanup:

```
┌──────────────────────────┐      ┌───────────────────────────┐
│                          │      │                           │
│   Parent VFS (refCount=2)│      │  Parent VFS (refCount=1)  │
│                          │      │                           │
└──────────────────────────┘      └───────────────────────────┘
          /        \                           │
         /          \                          │
┌──────────┐    ┌──────────┐        ┌─────────────────────┐
│ Child1   │    │ Child2   │        │     Child2          │
│          │    │          │        │                     │
└──────────┘    └──────────┘        └─────────────────────┘
                      │                        │
                      │                        │
               ┌────────────┐           ┌────────────┐
               │ Grandchild │           │ Grandchild │
               │            │           │            │
               └────────────┘           └────────────┘
```

Reference counting implementation in cleanup:

```go
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
```

## 3. File Operations in Detail

### 3.1 File Operation Logic Map

```
┌─────────────────────────────────────────────────────────────────────┐
│                          File Operations                            │
├─────────────┬───────────────────────────────────────────────────────┤
│ Operation   │ Process                                               │
├─────────────┼───────────────────────────────────────────────────────┤
│ Add Source  │ 1. Normalize virtual path                             │
│ File        │ 2. Verify source file exists & readable               │
│             │ 3. Record mapping (virtual→source)                    │
│             │ 4. Set origin type to OriginSource                    │
│             │ 5. Compute & store file hash for change detection     │
├─────────────┼───────────────────────────────────────────────────────┤
│ Add Embedded│ 1. Normalize virtual & FS paths                       │
│ File        │ 2. Verify embedded file exists & readable             │
│             │ 3. Extract to temp file                               │
│             │ 4. Record mappings (virtual→temp)                     │
│             │ 5. Set origin type to OriginEmbed                     │
├─────────────┼───────────────────────────────────────────────────────┤
│ Create      │ 1. Normalize virtual path                             │
│ Virtual File│ 2. Store content in memory map                        │
│             │ 3. Set origin type to OriginVirtual                   │
├─────────────┼───────────────────────────────────────────────────────┤
│ Get File    │ 1. Check if inherited (delegate to parent if needed)  │
│ Content     │ 2. Check for changes if OriginSource                  │
│             │ 3. Return content based on origin type:               │
│             │    - Source: Read source file                         │
│             │    - Embed: Read temp file                            │
│             │    - Virtual: Return from memory                      │
├─────────────┼───────────────────────────────────────────────────────┤
│ Delete File │ 1. Normalize virtual path                             │
│             │ 2. For inherited files, create "tombstone" file       │
│             │ 3. For other origins, remove mappings and temp files  │
└─────────────┴───────────────────────────────────────────────────────┘
```

### 3.2 Copy/Move Operations with Origin Preservation

The VFS implements special logic for copy/move operations with origin preservation:

```
┌───────────────────────────────────────────────────────────────────────────┐
│                      Copy/Move with Origin Preservation                   │
├─────────────────┬─────────────────────────────────────────────────────────┤
│ Origin Type     │ Behavior when preserveOrigin=true                       │
├─────────────────┼─────────────────────────────────────────────────────────┤
│ OriginSource    │ • Destination file points to same source file           │
│                 │ • Changes to source file affect both virtual paths      │
├─────────────────┼─────────────────────────────────────────────────────────┤
│ OriginEmbed     │ • Destination shares temp file with source              │
│                 │ • Both paths reference same extracted file              │
├─────────────────┼─────────────────────────────────────────────────────────┤
│ OriginVirtual   │ • References same in-memory content                     │
│                 │ • Changes to one affect both                            │
├─────────────────┼─────────────────────────────────────────────────────────┤
│ OriginInherited │ • Both paths inherit from parent                        │
│                 │ • Changes/deletions in parent affect both               │
└─────────────────┴─────────────────────────────────────────────────────────┘
```

Implementation of CopyFileWithOptions:

```go
func (v *VFS) CopyFileWithOptions(srcVirtualPath, destVirtualPath string, preserveOrigin bool) error {
    v.mutex.Lock()
    defer v.mutex.Unlock()

    // Normalize paths
    srcVirtualPath = normalizePath(srcVirtualPath)
    destVirtualPath = normalizePath(destVirtualPath)

    // Check if source exists
    if !v.fileExistsLocked(srcVirtualPath) {
        return fmt.Errorf("source file not found: %s", srcVirtualPath)
    }

    // Special handling based on file origin and preserveOrigin flag
    srcOrigin, _ := v.fileOrigins[srcVirtualPath]

    // If we're preserving origin, we'll copy the source mappings
    if preserveOrigin {
        // Copy mappings based on the origin type
        switch srcOrigin {
        case OriginSource:
            // Copy source mapping
            if sourcePath, ok := v.sourceMappings[srcVirtualPath]; ok {
                v.sourceMappings[destVirtualPath] = sourcePath
            }
        case OriginEmbed:
            // Copy embed mapping
            if tempPath, ok := v.embedMappings[srcVirtualPath]; ok {
                v.embedMappings[destVirtualPath] = tempPath
            }
        case OriginVirtual:
            // Copy virtual file content reference
            if content, ok := v.virtualFiles[srcVirtualPath]; ok {
                v.virtualFiles[destVirtualPath] = content
            }
        case OriginInherited:
            // Mark destination as inherited
            v.inheritedPaths[destVirtualPath] = true
        }

        // Set the same origin type
        v.fileOrigins[destVirtualPath] = srcOrigin
        
        // Copy file hash if it exists
        if hash, ok := v.fileHashes[srcVirtualPath]; ok {
            v.fileHashes[destVirtualPath] = hash
        }

        v.logger.Printf("Copied file with origin preservation: %s -> %s (origin: %v)",
            srcVirtualPath, destVirtualPath, srcOrigin)
        return nil
    }

    // Default case: Just copy the content (without preserving origin)
    content, err := v.getFileContentLocked(srcVirtualPath)
    if err != nil {
        return fmt.Errorf("failed to read source file: %w", err)
    }

    // Create as virtual file
    v.virtualFiles[destVirtualPath] = content
    v.fileOrigins[destVirtualPath] = OriginVirtual

    v.logger.Printf("Copied file content: %s -> %s", srcVirtualPath, destVirtualPath)
    return nil
}
```

### 3.3 Source File Change Detection

In development mode, the VFS monitors source files for changes:

```
┌──────────────────────┐     ┌─────────────────────┐
│  Watch Ticker        │────▶│  For each source    │
│  (500ms interval)    │     │      file:          │
└──────────────────────┘     └─────────────────────┘
                                       │
                                       ▼
┌──────────────────────┐     ┌─────────────────────┐
│ Mark file as changed │◀────│ Compare current hash│
│ in changedFiles map  │     │ with stored hash    │
└──────────────────────┘     └─────────────────────┘
         │
         ▼
┌──────────────────────┐
│ Update file content  │
│ on next access       │
└──────────────────────┘
```

Implementation:

```go
func (v *VFS) checkForChanges() {
    v.mutex.RLock()
    // Make a copy of sourceMappings to avoid long lock hold
    sourcePaths := make(map[string]string)
    for vPath, sPath := range v.sourceMappings {
        sourcePaths[vPath] = sPath
    }
    // Copy stored hashes
    fileHashes := make(map[string]FileHash)
    for path, hash := range v.fileHashes {
        fileHashes[path] = hash
    }
    v.mutex.RUnlock()

    // Track if we found any changes
    hasChanges := false

    // Check each source file
    for vPath, sPath := range sourcePaths {
        // Skip if file doesn't exist (might have been deleted)
        if _, err := os.Stat(sPath); os.IsNotExist(err) {
            continue
        }

        // Calculate current hash
        currHash, err := calculateFileHash(sPath)
        if err != nil {
            continue
        }

        // Compare with stored hash
        if storedHash, ok := fileHashes[vPath]; ok {
            if currHash.Hash != storedHash.Hash ||
               currHash.Size != storedHash.Size ||
               currHash.LastModified != storedHash.LastModified {
                
                // Acquire write lock to update
                v.mutex.Lock()
                v.fileHashes[vPath] = currHash
                v.changedFiles[vPath] = true
                v.invalidated = true
                v.mutex.Unlock()
                
                hasChanges = true
                v.logger.Printf("Detected change in source file: %s", vPath)
            }
        }
    }

    if hasChanges {
        v.logger.Printf("Source files changed, marked VFS as invalidated")
    }
}
```

## 4. Security Features

### 4.1 Path Normalization and Traversal Prevention

All virtual paths are normalized to prevent path traversal attacks:

```go
func normalizePath(path string) string {
    // Convert backslashes to forward slashes
    path = strings.ReplaceAll(path, "\\", "/")
    
    // Ensure path starts with /
    if !strings.HasPrefix(path, "/") {
        path = "/" + path
    }
    
    // Split path into components
    parts := strings.Split(path, "/")
    var result []string
    
    for _, part := range parts {
        if part == "" || part == "." {
            // Skip empty parts and current directory
            continue
        } else if part == ".." {
            // Remove last part for parent directory
            if len(result) > 0 {
                result = result[:len(result)-1]
            }
        } else {
            // Add valid part
            result = append(result, part)
        }
    }
    
    // Reconstruct path
    normalized := "/" + strings.Join(result, "/")
    return normalized
}
```

### 4.2 Symlink Detection

The VFS prevents symlink attacks by checking if source files are symlinks:

```go
func checkForSymlink(path string) error {
    // Get file info
    info, err := os.Lstat(path)
    if err != nil {
        return err
    }
    
    // Check if it's a symlink
    if info.Mode()&os.ModeSymlink != 0 {
        return fmt.Errorf("security error: symlinks are not allowed: %s", path)
    }
    
    return nil
}
```

### 4.3 Circular Reference Prevention

The VFS prevents infinite recursion in branching by detecting circular references:

```go
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
```

## 5. Concurrency Management

### 5.1 Locking Strategy

The VFS uses a dual-mutex approach for thread safety:

```
┌────────────────────────────────────────────────────────────────────┐
│                          Locking Strategy                          │
├──────────────┬─────────────────────────────────────────────────────┤
│ Mutex        │ Purpose                                             │
├──────────────┼─────────────────────────────────────────────────────┤
│ mutex        │ • Protects all file operations                      │
│ (RWMutex)    │ • Allows concurrent reads with RLock()              │
│              │ • Exclusive access for writes with Lock()           │
├──────────────┼─────────────────────────────────────────────────────┤
│ refMutex     │ • Separate mutex for reference counting             │
│ (Mutex)      │ • Prevents deadlocks during cleanup                 │
│              │ • Protects refCount and isCleanedUp flags           │
└──────────────┴─────────────────────────────────────────────────────┘
```

Locking patterns:

```go
// Read operation with shared lock
func (v *VFS) FileExists(virtualPath string) bool {
    v.mutex.RLock()
    defer v.mutex.RUnlock()
    return v.fileExistsLocked(normalizedPath)
}

// Write operation with exclusive lock
func (v *VFS) CreateVirtualFile(virtualPath string, content []byte) error {
    v.mutex.Lock()
    defer v.mutex.Unlock()
    
    // Implementation...
}

// Reference counting with separate mutex
func (v *VFS) Branch() *VFS {
    // Use main mutex for reading VFS state
    v.mutex.RLock()
    // Check state and create new VFS
    v.mutex.RUnlock()
    
    // Use reference mutex for updating reference count
    v.refMutex.Lock()
    v.refCount++
    v.refMutex.Unlock()
    
    // Rest of implementation...
}
```

### 5.2 Thread Safety Analysis

The VFS is designed for concurrent access with these guarantees:

1. **Multiple Readers**: Multiple goroutines can read simultaneously
2. **Writer Exclusivity**: Write operations block all other access
3. **Deadlock Prevention**: Consistent lock ordering (always mutex before refMutex)
4. **Memory Ordering**: Proper synchronization ensures visibility of changes
5. **Race-Free Cleanup**: Special handling to prevent race conditions during cleanup

## 6. Memory and Resource Management

### 6.1 Memory Lifecycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Memory Lifecycle                             │
├─────────────────┬───────────────────────────────────────────────────┤
│ Phase           │ Resources Managed                                 │
├─────────────────┼───────────────────────────────────────────────────┤
│ Initialization  │ • Create temp directory structure                 │
│                 │ • Initialize maps and internal structures         │
│                 │ • Start file watcher goroutine (if dev mode)      │
├─────────────────┼───────────────────────────────────────────────────┤
│ Operation       │ • Extract embedded files to temp files as needed  │
│                 │ • Create virtual files in memory                  │
│                 │ • Track file origins and mappings                 │
├─────────────────┼───────────────────────────────────────────────────┤
│ Cleanup Mark    │ • Set isCleanedUp flag                           │
│                 │ • Stop file watcher                              │
│                 │ • Decrement parent reference count               │
├─────────────────┼───────────────────────────────────────────────────┤
│ Complete Cleanup│ • Remove temporary directory and all files        │
│                 │ • Release all resources                           │
│                 │ • Only runs when refCount = 0                     │
└─────────────────┴───────────────────────────────────────────────────┘
```

### 6.2 Temporary File Management

The VFS creates a hierarchical temp directory structure:

```
/tmp/
  ├── vfs-${PARENT_ID}/            # Parent VFS temp dir
  │     ├── globals.php            # PHP globals file
  │     ├── embedded_file_1.tmp    # Extracted embedded files
  │     └── embedded_file_2.tmp
  │
  ├── vfs-branch-${CHILD1_ID}/     # Child VFS temp dir
  │     ├── globals.php
  │     └── child_embedded.tmp
  │
  └── vfs-branch-${CHILD2_ID}/     # Another child VFS temp dir
        ├── globals.php
        └── another_embedded.tmp
```

## 7. Integration with PHP Runtime

### 7.1 PHP Globals Initialization

The VFS initializes PHP globals to provide standard PHP environment:

```go
func (v *VFS) initializeGlobals() error {
    // Create globals.php in the VFS temp directory
    globalsPath := filepath.Join(v.tempDir, "globals.php")
    
    // PHP code for globals initialization
    globalsContent := []byte(`<?php
// Standard PHP globals initialization
$_SERVER = array(
    'SERVER_SOFTWARE' => 'Frango PHP Engine',
    'DOCUMENT_ROOT' => '/',
    'SCRIPT_FILENAME' => '',
    'REQUEST_URI' => '',
    'REQUEST_METHOD' => 'GET',
    'QUERY_STRING' => '',
    'SERVER_PROTOCOL' => 'HTTP/1.1',
    'REMOTE_ADDR' => '127.0.0.1',
    'SERVER_NAME' => 'localhost',
    'HTTP_HOST' => 'localhost',
);
$_GET = array();
$_POST = array();
$_COOKIE = array();
$_FILES = array();
$_ENV = array();
$_REQUEST = array();
$_SESSION = array();
?>`)

    // Write globals file
    if err := os.WriteFile(globalsPath, globalsContent, 0644); err != nil {
        return fmt.Errorf("failed to write PHP globals file: %w", err)
    }
    
    v.phpGlobalsFile = globalsPath
    return nil
}
```

## 8. Best Practices

### 8.1 Usage Patterns

```go
// GOOD: Always defer cleanup
vfs, err := frango.NewVFS("/tmp", logger, true)
if err != nil {
    return err
}
defer vfs.Cleanup()

// GOOD: Branch with proper cleanup
branch := vfs.Branch()
if branch == nil {
    return errors.New("failed to create branch")
}
defer branch.Cleanup()

// GOOD: Consider origin preservation needs
if needSourceUpdates {
    err := vfs.CopyFileWithOptions("/source.php", "/copy.php", true)
} else {
    err := vfs.CopyFileWithOptions("/source.php", "/copy.php", false)
}

// BAD: Failing to clean up
vfs, _ := frango.NewVFS("/tmp", logger, true)
// Missing defer vfs.Cleanup() → memory leak!

// BAD: Not checking branch result
branch := vfs.Branch()
// Missing check if branch is nil!
```

### 8.2 Performance Considerations

1. **Disable developMode in Production**: File watching adds overhead
2. **Use RLock for Read-Heavy Operations**: Take advantage of RWMutex
3. **Minimize Branching Depth**: Deep hierarchies can increase lookup time
4. **Prefer Virtual Files for Small Content**: Avoids filesystem overhead
5. **Clean Up Resources Promptly**: Use defer pattern for reliable cleanup

## 9. Implementation Challenges

### 9.1 Edge Cases

```
┌──────────────────────────────────────────────────────────────────────┐
│                     Key Implementation Challenges                     │
├────────────────────┬─────────────────────────────────────────────────┤
│ Challenge          │ Solution                                        │
├────────────────────┼─────────────────────────────────────────────────┤
│ Cross-platform     │ • Normalize all paths to use forward slashes    │
│ path handling      │ • Handle case sensitivity differences           │
│                    │ • Special handling for absolute paths           │
├────────────────────┼─────────────────────────────────────────────────┤
│ Inheriting from    │ • Special marker for inherited files            │
│ parent VFS         │ • "Shadowing" via tombstone files on deletion   │
│                    │ • Recursive lookup for file content             │
├────────────────────┼─────────────────────────────────────────────────┤
│ Memory leaks in    │ • Two-phase cleanup (mark + complete)           │
│ VFS hierarchy      │ • Reference counting for deferred cleanup       │
│                    │ • Cleanup verification in tests                 │
├────────────────────┼─────────────────────────────────────────────────┤
│ Race conditions    │ • Dual mutex strategy                           │
│                    │ • Local copies of state during long operations  │
│                    │ • Avoiding callbacks while holding locks        │
└────────────────────┴─────────────────────────────────────────────────┘
```

## 10. Testing Approach

The VFS includes comprehensive test coverage:

```
┌──────────────────────────────────────────────────────────────────────┐
│                           Test Categories                            │
├────────────────────┬─────────────────────────────────────────────────┤
│ Test Type          │ Coverage Areas                                  │
├────────────────────┼─────────────────────────────────────────────────┤
│ Basic Operations   │ • File creation, reading, deletion              │
│                    │ • Directory operations                          │
│                    │ • Path normalization                            │
├────────────────────┼─────────────────────────────────────────────────┤
│ Origin Preservation│ • Copy/move with preserveOrigin=true/false      │
│                    │ • Proper propagation of changes                 │
│                    │ • Inheritance behavior                          │
├────────────────────┼─────────────────────────────────────────────────┤
│ VFS Branching      │ • Parent-child relationships                    │
│                    │ • Deep hierarchies (parent→child→grandchild)    │
│                    │ • Reference counting correctness                │
├────────────────────┼─────────────────────────────────────────────────┤
│ Security           │ • Path traversal prevention                     │
│                    │ • Symlink attack prevention                     │
│                    │ • Circular reference detection                  │
├────────────────────┼─────────────────────────────────────────────────┤
│ Resource Management│ • Memory usage monitoring                       │
│                    │ • Cleanup verification                          │
│                    │ • Resource leak detection                       │
├────────────────────┼─────────────────────────────────────────────────┤
│ Concurrency        │ • Race condition checks                         │
│                    │ • Parallel access testing                       │
│                    │ • Deadlock detection                            │
└────────────────────┴─────────────────────────────────────────────────┘
```

# Appendix

## A. API Reference

### Initialization and Cleanup

```go
func NewVFS(tempDir string, logger *log.Logger, developMode bool) (*VFS, error)
func (v *VFS) Branch() *VFS
func (v *VFS) Cleanup()
```

### File Operations

```go
func (v *VFS) AddSourceFile(sourcePath, virtualPath string) error
func (v *VFS) AddSourceDirectory(sourceDir string, virtualBasePath string) error
func (v *VFS) AddEmbeddedFile(embedFS embed.FS, fsPath string, virtualPath string) error
func (v *VFS) AddEmbeddedDirectory(embedFS embed.FS, fsPath string, virtualPrefix string) error
func (v *VFS) CreateVirtualFile(virtualPath string, content []byte) error
func (v *VFS) GetFileContent(virtualPath string) ([]byte, error)
func (v *VFS) FileExists(virtualPath string) bool
func (v *VFS) ResolvePath(virtualPath string) (string, error)
func (v *VFS) DeleteFile(virtualPath string) error
func (v *VFS) ListFiles() []string
```

### Copy and Move Operations

```go
func (v *VFS) CopyFileWithOptions(srcVirtualPath, destVirtualPath string, preserveOrigin bool) error
func (v *VFS) MoveFileWithOptions(srcVirtualPath, destVirtualPath string, preserveOrigin bool) error
func (v *VFS) CopyFile(srcVirtualPath, destVirtualPath string) error
func (v *VFS) MoveFile(srcVirtualPath, destVirtualPath string) error
func (v *VFS) CopyFileSimple(srcVirtualPath, destVirtualPath string) error
func (v *VFS) MoveFileSimple(srcVirtualPath, destVirtualPath string) error
```

## B. Error Code Reference

Common error codes returned by VFS operations:

```
┌─────────────────────────┬──────────────────────────────────────────┐
│ Error Pattern           │ Description                              │
├─────────────────────────┼──────────────────────────────────────────┤
│ "file not found: X"     │ Requested file doesn't exist in the VFS  │
├─────────────────────────┼──────────────────────────────────────────┤
│ "source file not found" │ Source file for copy/move doesn't exist  │
├─────────────────────────┼──────────────────────────────────────────┤
│ "security error: X"     │ Potential security violation detected    │
├─────────────────────────┼──────────────────────────────────────────┤
│ "path traversal attempt"│ Detected attempt to escape VFS sandbox   │
├─────────────────────────┼──────────────────────────────────────────┤
│ "file already exists"   │ Destination file already exists          │
├─────────────────────────┼──────────────────────────────────────────┤
│ "VFS is cleaned up"     │ Operation on VFS that's already cleaned  │
└─────────────────────────┴──────────────────────────────────────────┘
```

## C. Version History

```
┌────────────┬───────────────────────────────────────────────────────┐
│ Version    │ Key Changes                                           │
├────────────┼───────────────────────────────────────────────────────┤
│ v1.0       │ • Initial implementation                              │
│            │ • Basic file operations                               │
│            │ • Simple branching                                    │
├────────────┼───────────────────────────────────────────────────────┤
│ v1.1       │ • Added reference counting                            │
│            │ • Improved cleanup process                            │
│            │ • Enhanced security checks                            │
├────────────┼───────────────────────────────────────────────────────┤
│ v1.2       │ • Added origin preservation                           │
│            │ • Enhanced path normalization                         │
│            │ • Added comprehensive testing                         │
└────────────┴───────────────────────────────────────────────────────┘
``` 