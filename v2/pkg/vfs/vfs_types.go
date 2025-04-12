/*
package vfs provides a virtual file system for PHP scripts.

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

package vfs

import (
	"log"
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

// FileChangeEvent represents a file change event
type FileChangeEvent struct {
	VirtualPath  string    // The virtual path of the changed file
	PhysicalPath string    // The physical path of the changed file
	EventTime    time.Time // When the change was detected
	ChangeType   string    // Type of change: "modified", "added", "deleted"
}

// FileChangeHandler is a function type for handling file change events
type FileChangeHandler func(event FileChangeEvent)

// FileHash stores a hash and timestamp to track file changes
type FileHash struct {
	Hash      string    // SHA-256 hash of the file content
	Timestamp time.Time // When the hash was calculated
}

// GlobalsProvider is an interface for providing PHP globals script content
type GlobalsProvider interface {
	// GetPHPGlobalsScript returns the PHP script used to initialize PHP globals
	GetPHPGlobalsScript() string

	// GetPHPGlobalsPath returns the virtual path for the PHP globals script
	GetPHPGlobalsPath() string
}

// DefaultGlobalsProvider is a basic implementation of GlobalsProvider
type DefaultGlobalsProvider struct {
	Script string
	Path   string
}

// GetPHPGlobalsScript returns the provided PHP globals script
func (p *DefaultGlobalsProvider) GetPHPGlobalsScript() string {
	return p.Script
}

// GetPHPGlobalsPath returns the provided path for the PHP globals script
func (p *DefaultGlobalsProvider) GetPHPGlobalsPath() string {
	return p.Path
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
	logicalPaths   map[string]string     // Physical path -> logical path for FrankenPHP

	// Event system
	changeHandlers []FileChangeHandler // Handlers for file change events
	handlerMutex   sync.RWMutex        // Mutex for change handlers

	// Granular mutex locks for different operations
	mutex      sync.RWMutex // General mutex for structural changes
	pathMutex  sync.RWMutex // For path resolution operations
	fileMutex  sync.RWMutex // For file content operations
	cacheMutex sync.RWMutex // For cache operations
	refMutex   sync.Mutex   // For reference counting

	watchTicker     *time.Ticker      // For file watching
	watchStop       chan bool         // To signal watching to stop
	logger          *log.Logger       // For logging operations
	invalidated     bool              // Whether any files need refreshing
	changedFiles    map[string]bool   // Tracks which files have changed
	inheritedPaths  map[string]bool   // Which paths come from parent VFS
	developMode     bool              // Whether development mode is enabled
	globalLibs      map[string]string // Path -> temp path for global libraries
	phpGlobalsFile  string            // Path to the PHP globals script in this VFS
	refCount        int               // Number of child VFS instances referencing this one
	isCleanedUp     bool              // Whether this VFS has been cleaned up
	globalsProvider GlobalsProvider   // Provider for PHP globals script

	// Caching maps for improved performance
	pathCache    map[string]string // Virtual path -> resolved path cache
	contentCache map[string][]byte // Path -> content cache (for frequently accessed files)
}

// VFSConfig provides configuration options for creating a new VFS instance
type VFSConfig struct {
	// TempDir is the directory where temporary files will be stored
	// If empty, os.TempDir() will be used
	TempDir string

	// Logger for VFS operations
	Logger *log.Logger

	// DevelopMode enables development features like auto-reloading
	DevelopMode bool

	// GlobalsProvider provides global PHP variables
	GlobalsProvider GlobalsProvider

	// EnableAutoReload determines whether files should be automatically reloaded when changes are detected
	// Only has effect when DevelopMode is true
	EnableAutoReload bool

	// Auto-reload options
	AutoReloadInterval time.Duration // How often to check for file changes
	AutoReloadScript   string        // Custom JavaScript to inject for auto-reload
	AutoReloadTrigger  string        // Reload trigger mechanism: "websocket", "sse", "polling"
}
