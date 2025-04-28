# Virtual File System (VFS) API Reference

This document provides a comprehensive reference for Frango's Virtual File System (VFS) API, which enables seamless integration between Go's filesystem and PHP's file operations.

## Table of Contents

- [Overview](#overview)
- [VFS Configuration](#vfs-configuration)
- [Core VFS Types](#core-vfs-types)
- [VFS Operations](#vfs-operations)
- [Working with Paths](#working-with-paths)
- [File Access Patterns](#file-access-patterns)
- [Advanced VFS Usage](#advanced-vfs-usage)

## Overview

Frango's Virtual File System (VFS) provides a bridge between Go's filesystem and PHP's file operations. It allows PHP scripts running in Frango to:

1. Access files and directories from the host filesystem
2. Work with in-memory files that don't exist on disk
3. Use custom file sources like embedded files, database storage, or cloud storage
4. Map virtual paths to different physical locations

The VFS is one of the core components that makes Frango flexible and powerful, enabling sophisticated use cases beyond simple file serving.

## VFS Configuration

When initializing a Frango instance, several VFS-related options are available:

### Basic Source Directory

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
)
```

The `WithSourceDir` option sets the base directory for PHP files. All PHP file paths used in Frango will be relative to this directory.

### Custom VFS Implementation

```go
// Create a custom VFS
customVFS := NewCustomVFS()

php, err := frango.New(
    frango.WithVFS(customVFS),
)
```

The `WithVFS` option allows providing a completely custom VFS implementation that satisfies the VFS interface.

## Core VFS Types

### VFS Structure

The core VFS structure includes:

```go
type VFS struct {
    name           string                // Unique identifier for this VFS
    parent         *VFS                  // Parent VFS (if this is a branch)
    sourceMappings map[string]string     // Virtual path -> source path
    embedMappings  map[string]string     // Virtual path -> embed temp path
    virtualFiles   map[string][]byte     // Virtual path -> content
    fileOrigins    map[string]FileOrigin // Virtual path -> origin type
    fileHashes     map[string]FileHash   // Path -> hash info
    tempDir        string                // Base temp directory
    mutex          sync.RWMutex          // For thread safety
    // Other internal fields...
}
```

### FileOrigin Type

The `FileOrigin` type defines the possible origins of files in the VFS:

```go
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
```

### FileHash Information

For change detection, each file has hash information:

```go
type FileHash struct {
    Hash         string    // Hash of file content
    LastModified time.Time // Last modification time
    Size         int64     // Size in bytes
}
```

## VFS Operations

### Creating a VFS

```go
// Create a new VFS
vfs, err := frango.NewVFS("/tmp", logger, true)
if err != nil {
    // Handle error
}
defer vfs.Cleanup()

// Create a branch from an existing VFS
branchVFS := parentVFS.Branch()
if branchVFS == nil {
    // Handle branching error
}
defer branchVFS.Cleanup()
```

### Adding Files

```go
// Add a file from the filesystem
err := vfs.AddSourceFile("/path/to/local/file.php", "/virtual/path.php")

// Add all files from a directory
err := vfs.AddSourceDirectory("/path/to/local/dir", "/virtual/prefix")

// Add a file from an embed.FS
err := vfs.AddEmbeddedFile(embedFS, "path/in/embed/file.php", "/virtual/path.php")

// Add all files from a directory in an embed.FS
err := vfs.AddEmbeddedDirectory(embedFS, "path/in/embed", "/virtual/prefix")

// Create a virtual file (in-memory)
err := vfs.CreateVirtualFile("/virtual/path.php", []byte("<?php echo 'Hello, World!'; ?>"))
```

### File Operations

```go
// Check if a file exists
exists := vfs.FileExists("/virtual/path.php")

// Get file content
content, err := vfs.GetFileContent("/virtual/path.php")

// Resolve a virtual path to a physical path
physicalPath, err := vfs.ResolvePath("/virtual/path.php")

// Copy a file
err := vfs.CopyFile("/source/path.php", "/dest/path.php")

// Move a file
err := vfs.MoveFile("/source/path.php", "/dest/path.php")

// Delete a file
err := vfs.DeleteFile("/virtual/path.php")

// List all files
files := vfs.ListFiles()
```

### Advanced Operations

```go
// Copy a file with options
err := vfs.CopyFileWithOptions("/source.php", "/copy.php", true) // preserve origin

// Move a file with options
err := vfs.MoveFileWithOptions("/source.php", "/moved.php", true) // preserve origin
```

## Working with Paths

### Path Normalization

All virtual paths are normalized to prevent path traversal attacks:

```go
// Input: "directory/../file.php"
// Normalized: "/file.php"

// Input: "./directory/./file.php"
// Normalized: "/directory/file.php"

// Input: "//directory/file.php"
// Normalized: "/directory/file.php"
```

### Resolving Paths

The `ResolvePath` method converts a virtual path to a physical path:

```go
physicalPath, err := vfs.ResolvePath("/virtual/path.php")
if err != nil {
    // Handle error (file not found)
}

// physicalPath might be "/tmp/vfs-123abc/source/file.php"
```

## File Access Patterns

### Direct Access

For optimal performance, files can be accessed directly:

```go
// Get content
content, err := vfs.GetFileContent("/path/to/file.php")

// Check file existence
if vfs.FileExists("/path/to/file.php") {
    // File exists
}
```

### Modifying Files

To modify files in the VFS:

```go
// Create a new file
vfs.CreateVirtualFile("/path/to/new.php", []byte("<?php echo 'New file'; ?>"))

// Update an existing file
content, err := vfs.GetFileContent("/path/to/file.php")
if err == nil {
    // Modify content
    newContent := append(content, []byte("\n<?php echo 'Added content'; ?>")...)
    vfs.CreateVirtualFile("/path/to/file.php", newContent)
}
```

## Advanced VFS Usage

### VFS Branching

VFS branching creates isolated environments for different requests:

```go
// Create a root VFS
rootVFS, err := frango.NewVFS("/tmp", logger, false)
if err != nil {
    // Handle error
}
defer rootVFS.Cleanup()

// Set up shared files
rootVFS.AddSourceDirectory("./common", "/")

// For each request, create a branch
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Create a branch VFS for this request
    requestVFS := rootVFS.Branch()
    if requestVFS == nil {
        http.Error(w, "Failed to create VFS", http.StatusInternalServerError)
        return
    }
    defer requestVFS.Cleanup()
    
    // Add request-specific files
    requestVFS.CreateVirtualFile("/dynamic.php", generateDynamicPHP(r))
    
    // Use the branch VFS for this request
    php.ForVFS(requestVFS, "/index.php").ServeHTTP(w, r)
}
```

### File Change Detection

In development mode, Frango monitors source files for changes:

```go
// Create a VFS with development mode enabled
vfs, err := frango.NewVFS("/tmp", logger, true)
```

When a file is accessed after being modified on disk, the VFS will detect the change and reload the file.

## Conclusion

Frango's Virtual File System provides a powerful interface for integrating PHP with Go's filesystem capabilities. By understanding and effectively using the VFS API, you can:

1. Create secure, isolated environments for PHP scripts
2. Serve dynamic, generated content without physical files
3. Implement custom storage backends for PHP files
4. Control access patterns to the filesystem
5. Bridge PHP's filesystem functions with Go's powerful I/O libraries

The VFS is a key component that enables Frango to provide a flexible, powerful PHP execution environment within Go applications. 