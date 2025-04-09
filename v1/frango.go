package frango

import (
	"embed"
	"fmt"
	"html"
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
	errorHandlerPath   string      // Path to custom PHP error handler script
	displayErrors      bool        // Whether to display PHP errors in the output
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

	// Set default values that depend on other options
	if m.displayErrors == false && m.developmentMode {
		// In development mode, display errors by default
		m.displayErrors = true
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

func (m *Middleware) getErrorReportingHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the error
		m.logger.Printf("Error handling request for %s: %v", r.URL.Path, err)

		// Determine status code based on error type
		statusCode := http.StatusInternalServerError
		if os.IsNotExist(err) || strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "access") {
			statusCode = http.StatusForbidden
		}

		// Create different error responses based on development mode
		if m.developmentMode {
			// In development mode: provide rich error details with HTML formatting
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(statusCode)

			// HTML error template
			errHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Frango PHP Error</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; line-height: 1.5; padding: 2rem; max-width: 900px; margin: 0 auto; }
        .error-container { background: #fff0f0; border-left: 4px solid #ff3333; padding: 1rem 1.5rem; border-radius: 4px; }
        .error-type { font-weight: bold; font-size: 1.2rem; color: #cc0000; margin-bottom: 0.5rem; }
        .error-message { font-family: monospace; padding: 0.5rem; background: #f8f8f8; border-radius: 3px; overflow-x: auto; }
        .error-info { margin-top: 1rem; }
        .error-path { font-family: monospace; }
        .stack { margin-top: 1rem; border-top: 1px solid #ddd; padding-top: 1rem; }
        .stack-trace { font-size: 0.9rem; font-family: monospace; white-space: pre-wrap; background: #f8f8f8; padding: 0.8rem; overflow-x: auto; }
        .details { margin-top: 1rem; }
        .details code { font-family: monospace; background: #f0f0f0; padding: 0.1rem 0.3rem; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Frango PHP Error</h1>
    <div class="error-container">
        <div class="error-type">%s Error (%d)</div>
        <div class="error-message">%s</div>
        <div class="error-info">
            <p>Request URL: <span class="error-path">%s</span></p>
            <p>Script Path: <span class="error-path">%s</span></p>
        </div>
        <div class="details">
            <p>This error occurred while trying to serve a PHP script through the Frango middleware. Check that:</p>
            <ul>
                <li>The PHP file exists at the specified path</li>
                <li>The PHP file contains valid PHP code</li>
                <li>Permissions are set correctly on the file and directories</li>
            </ul>
        </div>
    </div>
</body>
</html>
`
			// Get error type string based on status code
			errorType := "Server"
			if statusCode == http.StatusNotFound {
				errorType = "Not Found"
			} else if statusCode == http.StatusForbidden {
				errorType = "Access Denied"
			}

			// Fill in the HTML template
			scriptPath := r.URL.Path
			if r.URL.Path == "" {
				scriptPath = "[not specified]"
			}

			html := fmt.Sprintf(errHTML,
				errorType, statusCode,
				html.EscapeString(err.Error()),
				html.EscapeString(r.URL.String()),
				html.EscapeString(scriptPath))

			fmt.Fprint(w, html)
		} else {
			// In production mode: provide minimal, secure error information
			// Don't expose internal details
			message := "Internal server error"
			if statusCode == http.StatusNotFound {
				message = "The requested resource was not found"
			} else if statusCode == http.StatusForbidden {
				message = "Access denied"
			}

			http.Error(w, message, statusCode)
		}
	})
}

// For returns an http.Handler for a specific PHP script
func (m *Middleware) For(scriptPath string) http.Handler {
	// Use rootVFS or create one if needed
	var vfs *VFS
	if m.rootVFS != nil {
		vfs = m.rootVFS
	} else {
		var err error
		vfs, err = NewVFS(m.tempDir, m.logger, m.developmentMode)
		if err != nil {
			return m.getErrorReportingHandler(err)
		}
	}

	// Check if script has parameters
	hasParameters := strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}")

	// Resolve script path upfront
	resolvedPath := scriptPath
	fileExists := vfs.FileExists(scriptPath)
	var resolveErr error

	if !fileExists {
		// Try to resolve it from sourceDir
		absPath := m.resolveScriptPath(scriptPath)
		if absPath != "" && vfs.FileExists(absPath) {
			resolvedPath = absPath
			fileExists = true
		} else if m.sourceDir != "" {
			// Try to add from source directory
			sourcePath := filepath.Join(m.sourceDir, filepath.FromSlash(strings.TrimPrefix(scriptPath, "/")))
			if _, err := os.Stat(sourcePath); err == nil {
				if err := vfs.AddSourceFile(sourcePath, scriptPath); err == nil {
					fileExists = true
				} else {
					resolveErr = fmt.Errorf("error adding source file '%s' to VFS: %w", sourcePath, err)
				}
			} else {
				resolveErr = fmt.Errorf("file not found: %s", scriptPath)
			}
		} else {
			resolveErr = fmt.Errorf("file not found and no source directory configured: %s", scriptPath)
		}
	}

	// For all paths that don't exist, return proper error handler - no special handling for parameters
	if !fileExists {
		if resolveErr == nil {
			resolveErr = fmt.Errorf("file not found: %s", scriptPath)
		}
		return m.getErrorReportingHandler(resolveErr)
	}

	// Return handler function for files that exist
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block direct access to .php URLs if configured
		if m.blockDirectPHPURLs && strings.HasSuffix(r.URL.Path, ".php") {
			// Check if this is explicitly registered for this path pattern
			if r.Pattern == "" || !strings.HasSuffix(r.Pattern, ".php") {
				http.NotFound(w, r)
				return
			}
		}

		// For parameterized paths, set the pattern on the request
		if hasParameters && r.Pattern == "" {
			r.Pattern = scriptPath
		}

		// Execute the PHP script
		m.ExecutePHP(resolvedPath, vfs, nil, w, r)
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
			if r.Pattern == "" || !strings.HasSuffix(r.Pattern, ".php") {
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

// phpContextKey is a custom type for context keys
type phpContextKey string
