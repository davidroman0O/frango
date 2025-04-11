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

// DefaultGlobalsProvider uses the global phpGlobalsScript variable
type DefaultGlobalsProvider struct{}

// GetPHPGlobalsScript returns the default PHP globals script
func (p *DefaultGlobalsProvider) GetPHPGlobalsScript() string {
	return phpGlobalsScript
}

// GetPHPGlobalsPath returns the default path for the PHP globals script
func (p *DefaultGlobalsProvider) GetPHPGlobalsPath() string {
	return "/_frango_php_globals.php"
}

// VFS represents a virtual filesystem container for PHP files with branching capability
type VFS struct {
	name            string                // Unique identifier for this VFS
	parent          *VFS                  // Parent VFS (if this is a branch)
	sourceMappings  map[string]string     // Virtual path -> source path (for files on disk)
	embedMappings   map[string]string     // Virtual path -> embed temp path (for embedded files)
	virtualFiles    map[string][]byte     // Virtual path -> content (for in-memory files)
	fileOrigins     map[string]FileOrigin // Virtual path -> origin type
	fileHashes      map[string]FileHash   // Path -> hash info (for change detection)
	tempDir         string                // Base temp directory for this VFS
	mutex           sync.RWMutex          // For thread safety
	watchTicker     *time.Ticker          // For file watching
	watchStop       chan bool             // To signal watching to stop
	logger          *log.Logger           // For logging operations
	invalidated     bool                  // Whether any files need refreshing
	changedFiles    map[string]bool       // Tracks which files have changed
	inheritedPaths  map[string]bool       // Which paths come from parent VFS
	developMode     bool                  // Whether development mode is enabled
	globalLibs      map[string]string     // Path -> temp path for global libraries
	phpGlobalsFile  string                // Path to the PHP globals script in this VFS
	refCount        int                   // Number of child VFS instances referencing this one
	refMutex        sync.Mutex            // Separate mutex for reference counting
	isCleanedUp     bool                  // Whether this VFS has been cleaned up
	globalsProvider GlobalsProvider       // Provider for PHP globals script
}

// VFSConfig holds configuration options for creating a new VFS
type VFSConfig struct {
	TempDir         string          // Temporary directory for VFS files
	Logger          *log.Logger     // Logger for VFS operations
	DevelopMode     bool            // Whether development mode is enabled
	GlobalsProvider GlobalsProvider // Provider for PHP globals script
}
