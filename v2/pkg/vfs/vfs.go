/*
package vfs provides a virtual file system for PHP scripts.

This file is a wrapper around the modular VFS implementation split across multiple files:
- vfs_types.go: Type definitions and interfaces
- vfs_core.go: Core VFS functionality (constructor, branch, cleanup)
- vfs_files.go: File operations (add, create, delete, etc.)
- vfs_utils.go: Utility functions (path handling, hashing, etc.)
- vfs_watch.go: File watching functionality
- php_globals.go: PHP globals script and related functionality
*/

package vfs

// This file intentionally left minimal as the functionality has been split into
// separate files for better maintainability.
