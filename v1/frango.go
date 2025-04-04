package frango

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dunglas/frankenphp"
)

// Middleware is the core Frango PHP middleware for Go applications
type Middleware struct {
	sourceDir          string      // Main source directory for PHP files
	tempDir            string      // Base temporary directory
	logger             *log.Logger // Logger for operations
	initialized        bool        // Whether the middleware has been initialized
	initLock           sync.Mutex  // Lock for initialization
	developmentMode    bool        // Whether to enable development mode with file watching
	blockDirectPHPURLs bool        // Whether to block direct .php URLs
	rootVFS            *VFS        // Root VFS containing shared files
	vfsCreateLock      sync.Mutex  // Lock for creating new VFS instances
}

// Option is a function that configures the middleware
type Option func(*Middleware)

// RenderData is a function that provides template data to a PHP script
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}

// ContextKey is used for request context values
type ContextKey string

// New creates a new Frango PHP middleware instance
func New(opts ...Option) (*Middleware, error) {
	// Default configuration
	m := &Middleware{
		sourceDir:          "",
		tempDir:            os.TempDir(),
		logger:             log.New(os.Stderr, "[frango] ", log.LstdFlags),
		blockDirectPHPURLs: true,
		developmentMode:    true, // Default to development mode
	}

	// Apply all options
	for _, opt := range opts {
		opt(m)
	}

	// Create a unique temp dir for this instance
	instanceTempDir := filepath.Join(m.tempDir, "frango-"+generateUniqueID())
	if err := os.MkdirAll(instanceTempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	m.tempDir = instanceTempDir

	// CRITICAL: Initialize FrankenPHP just once at middleware creation
	// This ensures a single PHP process is available for all requests
	m.initLock.Lock()
	defer m.initLock.Unlock()

	if !m.initialized {
		m.logger.Println("Initializing FrankenPHP...")
		if err := frankenphp.Init(); err != nil {
			return nil, fmt.Errorf("error initializing FrankenPHP: %w", err)
		}
		m.initialized = true
		m.logger.Println("FrankenPHP initialized successfully")
	}

	// Create initial root VFS if source dir is specified
	if m.sourceDir != "" {
		m.vfsCreateLock.Lock()
		defer m.vfsCreateLock.Unlock()

		var err error
		m.rootVFS, err = NewVFS(m.tempDir, m.logger, m.developmentMode)
		if err != nil {
			return nil, fmt.Errorf("failed to create root VFS: %w", err)
		}

		// Add the source directory to the root VFS
		if err := m.rootVFS.AddSourceDirectory(m.sourceDir, "/"); err != nil {
			return nil, fmt.Errorf("failed to add source directory to VFS: %w", err)
		}
	}

	return m, nil
}

// SourceDir returns the configured source directory
func (m *Middleware) SourceDir() string {
	return m.sourceDir
}

// TempDir returns the temporary directory used by the middleware
func (m *Middleware) TempDir() string {
	return m.tempDir
}

// Shutdown cleans up resources used by the middleware
func (m *Middleware) Shutdown() {
	m.initLock.Lock()
	defer m.initLock.Unlock()

	// Clean up the root VFS if it exists
	if m.rootVFS != nil {
		m.rootVFS.Cleanup()
		m.rootVFS = nil
	}

	// CRITICAL: Properly shut down FrankenPHP
	// Must be done once after we're done with the middleware
	if m.initialized {
		frankenphp.Shutdown()
		m.initialized = false
	}

	// Clean up the temp directory
	if m.tempDir != "" && m.tempDir != os.TempDir() {
		m.logger.Printf("Cleaning up temp directory: %s", m.tempDir)
		if err := os.RemoveAll(m.tempDir); err != nil {
			m.logger.Printf("Error removing temp directory: %v", err)
		}
	}
}

// NewVFS creates a new virtual filesystem instance
// If the middleware has a root VFS, the new VFS will branch from it
func (m *Middleware) NewVFS() *VFS {
	m.vfsCreateLock.Lock()
	defer m.vfsCreateLock.Unlock()

	// Create or use the root VFS
	if m.rootVFS == nil {
		vfs, err := NewVFS(m.tempDir, m.logger, m.developmentMode)
		if err != nil {
			m.logger.Printf("Error creating new VFS: %v", err)
			return nil
		}
		return vfs
	}

	// Branch from the root VFS
	return m.rootVFS.Branch()
}

// getRootVFS gets or creates the root VFS
func (m *Middleware) getRootVFS() (*VFS, error) {
	m.vfsCreateLock.Lock()
	defer m.vfsCreateLock.Unlock()

	if m.rootVFS == nil {
		var err error
		m.rootVFS, err = NewVFS(m.tempDir, m.logger, m.developmentMode)
		if err != nil {
			return nil, fmt.Errorf("failed to create root VFS: %w", err)
		}
	}

	return m.rootVFS, nil
}

// --- Operations on the root VFS ---

// AddSourceFile adds a file from the filesystem to the root VFS
func (m *Middleware) AddSourceFile(sourcePath string, virtualPath string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.AddSourceFile(sourcePath, virtualPath)
}

// AddSourceDirectory adds all files from a directory to the root VFS
func (m *Middleware) AddSourceDirectory(sourceDir string, virtualPrefix string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.AddSourceDirectory(sourceDir, virtualPrefix)
}

// AddEmbeddedFile adds a single file from an embed.FS to the root VFS
func (m *Middleware) AddEmbeddedFile(embedFS embed.FS, fsPath string, virtualPath string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.AddEmbeddedFile(embedFS, fsPath, virtualPath)
}

// AddEmbeddedDirectory adds a directory from an embed.FS to the root VFS
func (m *Middleware) AddEmbeddedDirectory(embedFS embed.FS, fsPath string, virtualPrefix string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.AddEmbeddedDirectory(embedFS, fsPath, virtualPrefix)
}

// AddEmbeddedLibrary adds an embedded file to the root VFS and returns its disk path
// This is maintained for backward compatibility
func (m *Middleware) AddEmbeddedLibrary(embedFS embed.FS, fsPath string, targetLibraryPath string) (string, error) {
	// Create a VFS if we don't have one yet
	vfs, err := m.getRootVFS()
	if err != nil {
		return "", err
	}

	// Add the embedded file to the root VFS
	if err := vfs.AddEmbeddedFile(embedFS, fsPath, targetLibraryPath); err != nil {
		return "", fmt.Errorf("failed to add embedded file to VFS: %w", err)
	}

	// Resolve the virtual path to a disk path
	diskPath, err := vfs.ResolvePath(targetLibraryPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve library path: %w", err)
	}

	return diskPath, nil
}

// CreateVirtualFile creates a file directly in the root VFS
func (m *Middleware) CreateVirtualFile(virtualPath string, content []byte) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.CreateVirtualFile(virtualPath, content)
}

// CopyFile copies a file within the root VFS
func (m *Middleware) CopyFile(srcVirtualPath, destVirtualPath string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.CopyFile(srcVirtualPath, destVirtualPath)
}

// MoveFile moves a file within the root VFS
func (m *Middleware) MoveFile(srcVirtualPath, destVirtualPath string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.MoveFile(srcVirtualPath, destVirtualPath)
}

// DeleteFile deletes a file from the root VFS
func (m *Middleware) DeleteFile(virtualPath string) error {
	vfs, err := m.getRootVFS()
	if err != nil {
		return err
	}
	return vfs.DeleteFile(virtualPath)
}

// ListFiles lists all files in the root VFS
func (m *Middleware) ListFiles() ([]string, error) {
	vfs, err := m.getRootVFS()
	if err != nil {
		return nil, err
	}
	return vfs.ListFiles(), nil
}

// GetFileContent gets the content of a file from the root VFS
func (m *Middleware) GetFileContent(virtualPath string) ([]byte, error) {
	vfs, err := m.getRootVFS()
	if err != nil {
		return nil, err
	}
	return vfs.GetFileContent(virtualPath)
}

// FileExists checks if a file exists in the root VFS
func (m *Middleware) FileExists(virtualPath string) (bool, error) {
	vfs, err := m.getRootVFS()
	if err != nil {
		return false, err
	}
	return vfs.FileExists(virtualPath), nil
}

// For returns an http.Handler for a specific PHP script
func (m *Middleware) For(scriptPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use rootVFS or create one if needed
		var vfs *VFS
		if m.rootVFS != nil {
			vfs = m.rootVFS
		} else {
			var err error
			vfs, err = NewVFS(m.tempDir, m.logger, m.developmentMode)
			if err != nil {
				http.Error(w, "Failed to initialize VFS", http.StatusInternalServerError)
				return
			}
			defer vfs.Cleanup()
		}

		// Block direct access to .php URLs if configured
		if m.blockDirectPHPURLs && strings.HasSuffix(r.URL.Path, ".php") {
			// Check if this is explicitly registered for this path pattern
			pattern := extractPatternFromContext(r.Context())
			if pattern == "" || !strings.HasSuffix(pattern, ".php") {
				http.NotFound(w, r)
				return
			}
		}

		// Check if script path contains parameters (like /users/{userId}.php)
		hasParameters := strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}")

		// For parameter paths like /users/{userId}.php, we need special handling
		if hasParameters {
			// Use the script path itself to auto-extract parameters from URL
			// For example: if scriptPath = "/users/{userId}.php" and r.URL.Path = "/users/42"
			// this will extract userId=42 from the URL path

			// First, make sure the VFS has this file
			if !vfs.FileExists(scriptPath) {
				// Try to resolve it from sourceDir
				absPath := m.resolveScriptPath(scriptPath)
				if absPath != "" {
					scriptPath = absPath
				}

				// For scripts with parameters, the exact file might not exist in VFS yet
				// but could be available from the source directory
				if m.sourceDir != "" {
					sourcePath := filepath.Join(m.sourceDir, filepath.FromSlash(strings.TrimPrefix(scriptPath, "/")))
					if _, err := os.Stat(sourcePath); err == nil {
						// Add it from the source directory
						if err := vfs.AddSourceFile(sourcePath, scriptPath); err != nil {
							m.logger.Printf("Error adding source file '%s' to VFS: %v", sourcePath, err)
						}
					}
				}
			}

			// Execute the script - the ExecutePHP function will handle parameter extraction
			m.ExecutePHP(scriptPath, vfs, nil, w, r)
			return
		}

		// For non-parameterized scripts, follow the normal path
		if !vfs.FileExists(scriptPath) {
			// Try to resolve as a path relative to sourceDir
			absPath := m.resolveScriptPath(scriptPath)
			if absPath != "" && vfs.FileExists(absPath) {
				scriptPath = absPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Execute the PHP script
		m.ExecutePHP(scriptPath, vfs, nil, w, r)
	})
}

// Render returns an http.Handler that renders a PHP script with data from renderFn
func (m *Middleware) Render(scriptPath string, renderFn RenderData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use rootVFS or create one if needed
		var vfs *VFS
		if m.rootVFS != nil {
			vfs = m.rootVFS
		} else {
			var err error
			vfs, err = NewVFS(m.tempDir, m.logger, m.developmentMode)
			if err != nil {
				http.Error(w, "Failed to initialize VFS", http.StatusInternalServerError)
				return
			}
			defer vfs.Cleanup()
		}

		// Check if file exists in VFS
		if !vfs.FileExists(scriptPath) {
			// Try to resolve as a path relative to sourceDir
			absPath := m.resolveScriptPath(scriptPath)
			if absPath != "" && vfs.FileExists(absPath) {
				scriptPath = absPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Execute the PHP script with render data
		m.ExecutePHP(scriptPath, vfs, renderFn, w, r)
	})
}

// ForVFS returns an http.Handler for a specific PHP script in a specific VFS
func (m *Middleware) ForVFS(vfs *VFS, scriptPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block direct access to .php URLs if configured
		if m.blockDirectPHPURLs && strings.HasSuffix(r.URL.Path, ".php") {
			// Check if this is explicitly registered for this path pattern
			pattern := extractPatternFromContext(r.Context())
			if pattern == "" || !strings.HasSuffix(pattern, ".php") {
				http.NotFound(w, r)
				return
			}
		}

		// Check if file exists in VFS
		if !vfs.FileExists(scriptPath) {
			// Try to resolve as a path relative to sourceDir
			absPath := m.resolveScriptPath(scriptPath)
			if absPath != "" && vfs.FileExists(absPath) {
				scriptPath = absPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Execute the PHP script
		m.ExecutePHP(scriptPath, vfs, nil, w, r)
	})
}

// RenderVFS returns an http.Handler that renders a PHP script in a specific VFS with data from renderFn
func (m *Middleware) RenderVFS(vfs *VFS, scriptPath string, renderFn RenderData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if file exists in VFS
		if !vfs.FileExists(scriptPath) {
			// Try to resolve as a path relative to sourceDir
			absPath := m.resolveScriptPath(scriptPath)
			if absPath != "" && vfs.FileExists(absPath) {
				scriptPath = absPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Execute the PHP script with render data
		m.ExecutePHP(scriptPath, vfs, renderFn, w, r)
	})
}

// resolveScriptPath resolves a script path to an absolute path
func (m *Middleware) resolveScriptPath(scriptPath string) string {
	// If it's already absolute, return as is
	if filepath.IsAbs(scriptPath) {
		return scriptPath
	}

	// If it's a virtual path (starts with /), return as is
	if strings.HasPrefix(scriptPath, "/") {
		return scriptPath
	}

	// Otherwise, join with sourceDir
	if m.sourceDir != "" {
		return filepath.Join("/", scriptPath)
	}

	return ""
}

// extractPatternFromContext extracts the pattern from the request context
// This is the key used by Go 1.22+ ServeMux to store the matched pattern
func extractPatternFromContext(ctx context.Context) string {
	// Get the pattern from the context
	if ctx == nil {
		return ""
	}

	// Try our custom context key first (ContextKey)
	if val := ctx.Value(ContextKey("pattern")); val != nil {
		if pattern, ok := val.(string); ok {
			return pattern
		}
	}

	// For backward compatibility, also try the old phpContextKey
	if val := ctx.Value(phpContextKey("pattern")); val != nil {
		if pattern, ok := val.(string); ok {
			return pattern
		}
	}

	// For future Go 1.22+ compatibility, also try the standard ServeMux pattern key
	// This key may change in future Go versions, so we need to check multiple possibilities
	for _, key := range []any{
		&http.ServeMux{},
		"pattern",
		"http.pattern",
		"net/http.pattern",
	} {
		if val := ctx.Value(key); val != nil {
			if pattern, ok := val.(string); ok {
				return pattern
			}
		}
	}

	return ""
}

// phpContextKey is a custom type for context keys
type phpContextKey string
