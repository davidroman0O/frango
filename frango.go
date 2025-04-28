package frango

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davidroman0O/frango/internal/utils"
	"github.com/davidroman0O/frango/pkg/executor"
	"github.com/davidroman0O/frango/pkg/vfs"
	"github.com/dunglas/frankenphp"
)

// Middleware is the core Frango PHP middleware for Go applications
type Middleware struct {
	tempDir            string      // Base temporary directory
	logger             *log.Logger // Logger for operations
	initialized        bool        // Whether the middleware has been initialized
	initLock           sync.Mutex  // Lock for initialization
	developmentMode    bool        // Whether to enable development mode with file watching
	blockDirectPHPURLs bool        // Whether to block direct .php URLs
	rootVFS            *vfs.VFS    // Root VFS containing shared files
	vfsCreateLock      sync.Mutex  // Lock for creating new VFS instances
	errorHandlerPath   string      // Path to custom PHP error handler script
	displayErrors      bool        // Whether to display PHP errors in the output

	// Auto-reload related fields
	enableAutoReload        bool            // Whether auto-reload is enabled
	autoReloadTrigger       string          // Mechanism: "sse", "polling", "ws" (ws not fully implemented yet)
	autoReloadScript        string          // Custom JS script content (overrides trigger-based)
	autoReloadScriptContent string          // The actual JS script content to inject
	reloadHub               *ReloadEventHub // Hub for managing client connections

	// Dev Server config (used if WithDevServer is called)
	devServerConfig *DevServerConfig // Configuration for the separate dev server
	devServer       *http.Server     // Instance of the running dev server (if configured)
}

// Option is a function that configures the middleware
type Option func(*Middleware)

// RenderData is a function that provides template data to a PHP script
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}

// ContextKey is used for request context values
type ContextKey string

// --- Default Auto-Reload Scripts ---
// (Moved from vfs_autoreload.go)

const defaultSSEReloadScript = "\n<script>\n(function() {\n    let isReloading = false; // Flag to indicate intentional reload\n\n    // Injected by server - MUST match the placeholder in executor.injectAutoReloadScript\n    const frangoScriptPath = '%%SCRIPT_PATH%%';\n    const frangoDevServerPort = parseInt('%%DEV_SERVER_PORT%%', 10); // 0 if not using dev server\n\n    if (!frangoScriptPath || frangoScriptPath === '%%SCRIPT_PATH%%') {\n         console.error('[Frango Reload] Script path placeholder not replaced, cannot connect.');\n         return;\n    }\n\n    const encodedPath = encodeURIComponent(frangoScriptPath);\n    let eventSourceUrl;\n\n    if (frangoDevServerPort > 0) {\n        // Construct URL for separate dev server\n        // Note: Using backticks for JS template literal requires careful escaping in Go string\n        eventSourceUrl = `http://${window.location.hostname}:${frangoDevServerPort}/_/frango/SSE?page=${encodedPath}`;\n        console.log('[Frango Reload] Connecting SSE to Dev Server:', eventSourceUrl);\n    } else {\n        // Construct relative URL for main server\n        eventSourceUrl = '/_/frango/SSE?page=' + encodedPath;\n        console.log('[Frango Reload] Connecting SSE to Main Server:', eventSourceUrl);\n    }\n\n\n    const eventSource = new EventSource(eventSourceUrl);\n\n    eventSource.addEventListener('fileChange', (event) => {\n        // Server already filtered, just check it's a modification event\n        try {\n            const data = JSON.parse(event.data);\n            console.log('[Frango Reload] File change event received:', data);\n            // Reload if the server sent a modification event (server ensures it's relevant)\n            if (data.event === 'modified') {\n                 console.log('[Frango Reload] Reloading page...');\n                 isReloading = true; // Set flag before reloading\n                 window.location.reload();\n            }\n        } catch (e) {\n             console.error('[Frango Reload] Error parsing message:', e, 'Raw data:', event.data);\n        }\n    });\n\n    eventSource.onerror = (error) => {\n        // If we initiated the reload, the connection closing is expected\n        if (isReloading) {\n            console.log('[Frango Reload] Reloading, connection closed as expected.');\n            return; // Don't log error or try to close\n        }\n        console.error('[Frango Reload] EventSource error:', error, 'URL:', eventSourceUrl);\n        eventSource.close(); // Close on unexpected errors\n        console.log('[Frango Reload] SSE connection closed due to error.');\n    };\n\n    eventSource.onopen = () => {\n        console.log('[Frango Reload] SSE Connected for page:', frangoScriptPath, 'to:', eventSourceUrl);\n    };\n\n})();\n</script>\n"

const defaultPollingReloadScript = `
<script>
(function() {
    let lastUpdate = Date.now();
    function checkForUpdates() {
        fetch('/_frango_reload_poll?last=' + lastUpdate)
            .then(response => {
                if (!response.ok) throw new Error('Poll request failed');
                return response.json();
            })
            .then(data => {
                if (data.hasChanges) {
                    console.log('[Frango Reload] Changes detected, reloading page...');
                    window.location.reload();
                } else {
                    lastUpdate = data.timestamp;
                    setTimeout(checkForUpdates, 1000); // Poll every second
                }
            })
            .catch(error => {
                console.error('[Frango Reload] Polling error:', error);
                setTimeout(checkForUpdates, 5000); // Retry after 5 seconds on error
            });
    }
    console.log('[Frango Reload] Polling Started.');
    checkForUpdates();
})();
</script>
`

// TODO: Implement WebSocket auto-reload script if needed
const defaultWebSocketReloadScript = `
<script>
console.warn('[Frango Reload] WebSocket reload not fully implemented yet.');
// WebSocket implementation would go here
</script>
`

// --- End Default Scripts ---

// New creates a new Frango PHP middleware instance
func New(opts ...Option) (*Middleware, error) {
	// Default configuration
	m := &Middleware{
		tempDir:            os.TempDir(),
		logger:             log.New(os.Stderr, "[frango] ", log.LstdFlags),
		blockDirectPHPURLs: true,
		developmentMode:    true, // Default to development mode ON
		// Default auto-reload settings (will be adjusted based on developmentMode later)
		enableAutoReload:  true,
		autoReloadTrigger: "sse", // Default trigger
	}

	// Apply all options provided by the user
	for _, opt := range opts {
		opt(m)
	}

	// --- Adjust defaults based on other settings ---
	// If not in development mode, auto-reload is forced off unless explicitly enabled
	if !m.developmentMode && m.enableAutoReload {
		// If dev mode is off, but user explicitly said WithAutoReload(true), keep it.
		// Otherwise, disable it.
		// NOTE: Checking for explicit options is complex. Current logic:
		// If developmentMode=false, enableAutoReload defaults to false.
		// User must set BOTH WithDevelopmentMode(false) AND WithAutoReload(true)
		// for auto-reload to be active in non-dev mode (uncommon case).
		// We'll simplify and just disable it if dev mode is off, unless the user explicitly enabled it.
		// Let's refine this logic slightly: If dev mode is false, auto-reload is disabled by default.
		// The Option func WithAutoReload(true) can override this.
		// We need to check if WithAutoReload was set AFTER the default was established.

		// Reset based on dev mode first
		if !m.developmentMode {
			m.enableAutoReload = false
		}
		// Re-apply options to allow override
		for _, opt := range opts {
			opt(m)
		}

	} else if m.developmentMode {
		// If dev mode is ON, auto-reload is ON by default.
		// User could have explicitly set it to false via WithAutoReload(false).
		// The apply loop already handled this.
	}

	// Display errors defaults to true only if in development mode
	if m.developmentMode {
		// Check if user explicitly set it using WithDisplayErrors
		displayErrorsSet := false
		// Create a temporary middleware with defaults to check option effects
		checkM := &Middleware{displayErrors: false} // Start with explicit false
		for _, opt := range opts {
			opt(checkM)
			// If the option changed the value from the default false, it was set
			if checkM.displayErrors {
				displayErrorsSet = true
				break
			}
		}
		if !displayErrorsSet {
			m.displayErrors = true // Default to true in dev mode if not explicitly set by user
		}
	} else {
		// Default to false in production mode (unless explicitly set true)
		// The apply loop already handled explicit setting.
	}
	// --- End Adjust defaults ---

	// Create a unique temp dir for this instance
	instanceTempDir := filepath.Join(m.tempDir, "frango-"+utils.GenerateUniqueID())
	if err := os.MkdirAll(instanceTempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	m.tempDir = instanceTempDir
	m.logger.Printf("Using temp directory: %s", m.tempDir)

	// Initialize FrankenPHP
	m.initLock.Lock()
	if !m.initialized {
		m.logger.Println("Initializing FrankenPHP...")
		// TODO: Allow configuring FrankenPHP options (num threads, workers, etc.)
		if err := frankenphp.Init(frankenphp.WithNumThreads(3)); err != nil {
			m.initLock.Unlock()
			return nil, fmt.Errorf("error initializing FrankenPHP: %w", err)
		}
		m.initialized = true
		m.logger.Println("FrankenPHP initialized successfully")
	}
	m.initLock.Unlock()

	// Create initial root VFS
	// Use a separate lock for VFS creation/access
	m.vfsCreateLock.Lock()
	var vfsErr error
	m.rootVFS, vfsErr = vfs.NewVFSWithConfig(vfs.VFSConfig{
		TempDir:     m.tempDir,
		Logger:      m.logger,
		DevelopMode: m.developmentMode, // Pass dev mode to VFS for its own checks if needed
		// Note: AutoReload specific configs are removed from VFSConfig
	})
	m.vfsCreateLock.Unlock() // Unlock after VFS creation
	if vfsErr != nil {
		// Attempt cleanup even if VFS creation failed partially
		m.Shutdown() // Call shutdown to clean up FrankenPHP and temp dir
		return nil, fmt.Errorf("failed to create root VFS: %w", vfsErr)
	}

	// --- Setup Auto-Reload ---
	if m.enableAutoReload {
		m.logger.Printf("Auto-reload enabled (Trigger: %s)", m.getReloadTriggerType())

		// Determine the script content
		if m.autoReloadScript != "" {
			m.autoReloadScriptContent = m.autoReloadScript
			m.logger.Println("Using custom auto-reload script.")
		} else {
			switch m.getReloadTriggerType() {
			case "websocket", "ws":
				m.autoReloadScriptContent = defaultWebSocketReloadScript
			case "polling", "poll":
				m.autoReloadScriptContent = defaultPollingReloadScript
			default: // "sse" or any unknown value
				m.autoReloadScriptContent = defaultSSEReloadScript
			}
		}

		// Initialize the event hub
		m.reloadHub = NewReloadEventHub(m.logger)

		// Register VFS change handler to broadcast events
		// Ensure rootVFS is not nil before adding handler
		if m.rootVFS != nil {
			m.rootVFS.AddChangeHandler(func(event vfs.FileChangeEvent) {
				if m.reloadHub != nil {
					m.reloadHub.BroadcastFileChange(event)
				}
			})
			m.logger.Println("Registered VFS change handler for auto-reload broadcasting.")
		} else {
			m.logger.Println("Warning: Root VFS is nil, cannot register change handler for auto-reload.")
			// Proceed without broadcast functionality, user needs to add files first
		}

		// --- Start Dev Server if configured ---
		if m.devServerConfig != nil {
			m.logger.Printf("Starting internal auto-reload dev server on port %d", m.devServerConfig.Port)
			devMux := http.NewServeMux()
			// Ensure reloadHub is available before registering handler
			if m.reloadHub != nil {
				devMux.HandleFunc("/_/frango/SSE", m.SSEReloadHandler) // Use the new path
			} else {
				m.logger.Println("Error: Reload hub not initialized, cannot register SSE handler for dev server.")
				// Maybe return an error here?
			}

			// Create and store the http.Server instance
			devAddr := fmt.Sprintf(":%d", m.devServerConfig.Port)
			m.devServer = &http.Server{
				Addr:    devAddr,
				Handler: enableCORS(devMux),
				// TODO: Add timeouts (ReadTimeout, WriteTimeout, IdleTimeout) for robustness
			}

			// Start the server in a goroutine
			go func() {
				m.logger.Printf("Auto-reload dev server listening on %s", devAddr)
				if err := m.devServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					m.logger.Printf("Error starting or running auto-reload dev server: %v", err)
				}
			}()
		} else if m.enableAutoReload && m.getReloadTriggerType() == "sse" {
			// If auto-reload is enabled with SSE but no dev server, log a reminder
			// Note: Path updated here too for consistency in messaging, though registration is external.
			m.logger.Println("Auto-reload with SSE enabled, but no dev server. Ensure '/_/frango/SSE' route is registered on the main application mux.")
		}
		// --- End Start Dev Server ---

	} else {
		m.logger.Println("Auto-reload disabled.")
	}
	// --- End Setup Auto-Reload ---

	return m, nil
}

// getReloadTriggerType returns the normalized reload trigger type
func (m *Middleware) getReloadTriggerType() string {
	if m.autoReloadTrigger == "" {
		return "sse" // Default to SSE
	}
	return strings.ToLower(m.autoReloadTrigger)
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

	// Shut down the auto-reload dev server if it's running
	if m.devServer != nil {
		m.logger.Println("Shutting down auto-reload dev server...")
		// Create a context with a timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5-second timeout
		defer cancel()

		if err := m.devServer.Shutdown(ctx); err != nil {
			m.logger.Printf("Error during auto-reload dev server shutdown: %v", err)
		} else {
			m.logger.Println("Auto-reload dev server shut down gracefully.")
		}
		m.devServer = nil // Clear the reference
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

// NewVFS creates a new VFS instance
func NewVFS(tempDir string, logger *log.Logger, developMode bool) (*vfs.VFS, error) {
	return vfs.NewVFSWithConfig(vfs.VFSConfig{
		TempDir:     tempDir,
		Logger:      logger,
		DevelopMode: developMode,
	})
}

// NewVFS creates a new virtual filesystem instance for the middleware
// If the middleware has a root VFS, the new VFS will branch from it
func (m *Middleware) NewVFS() *vfs.VFS {
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
func (m *Middleware) getRootVFS() (*vfs.VFS, error) {
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

// For creates a handler that will serve the given PHP script.
//
// All handlers created by For share the same main VFS instance, which ensures
// consistency across different routes. File modifications made in one route
// will be visible to all other routes.
//
// If you need isolated environments for different scripts, use ForVFS with
// a branched VFS instance.
func (m *Middleware) For(phpScriptPath string) http.Handler {
	// Use the main VFS instance for all routes to maintain consistency
	return m.ForVFS(m.rootVFS, phpScriptPath)
}

// Render returns an http.Handler that renders a PHP script with data from renderFn
func (m *Middleware) Render(scriptPath string, renderFn RenderData) http.Handler {
	// Use rootVFS or create one if needed
	var vfs *vfs.VFS
	if m.rootVFS != nil {
		vfs = m.rootVFS
	} else {
		var err error
		vfs, err = NewVFS(m.tempDir, m.logger, m.developmentMode)
		defer vfs.Cleanup()
		if err != nil {
			// http.Error(w, "Failed to initialize VFS", http.StatusInternalServerError)
			return m.getErrorReportingHandler(err)
		}
	}

	// Check if file exists in VFS
	if !vfs.FileExists(scriptPath) {
		// Try to normalize the path according to VFS conventions
		absPath := m.resolveScriptPath(scriptPath)
		if absPath != "" && vfs.FileExists(absPath) {
			scriptPath = absPath
		} else {
			// http.NotFound(w, r)
			return m.getErrorReportingHandler(fmt.Errorf("file not found: %s", scriptPath))
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Execute the PHP script with render data
		m.ExecuteWithExecutor(vfs, scriptPath, renderFn, w, r)
	})
}

// ForVFS returns an http.Handler for a specific PHP script in a specific VFS
func (m *Middleware) ForVFS(vfs *vfs.VFS, scriptPath string) http.Handler {
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
			// Try to normalize the path according to VFS conventions
			absPath := m.resolveScriptPath(scriptPath)
			if absPath != "" && vfs.FileExists(absPath) {
				scriptPath = absPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Execute the PHP script
		m.ExecuteWithExecutor(vfs, scriptPath, nil, w, r)
	})
}

// RenderVFS returns an http.Handler that renders a PHP script in a specific VFS with data from renderFn
func (m *Middleware) RenderVFS(vfs *vfs.VFS, scriptPath string, renderFn RenderData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if file exists in VFS
		if !vfs.FileExists(scriptPath) {
			// Try to normalize the path according to VFS conventions
			absPath := m.resolveScriptPath(scriptPath)
			if absPath != "" && vfs.FileExists(absPath) {
				scriptPath = absPath
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Execute the PHP script with render data
		m.ExecuteWithExecutor(vfs, scriptPath, renderFn, w, r)
	})
}

// resolveScriptPath resolves a script path to a properly formatted VFS path
func (m *Middleware) resolveScriptPath(scriptPath string) string {
	// If it's already absolute filesystem path, we can't use it directly with VFS
	// Return empty to indicate resolution failure
	if filepath.IsAbs(scriptPath) {
		return ""
	}

	// If it's a virtual path (starts with /), ensure it's properly formatted
	if strings.HasPrefix(scriptPath, "/") {
		return scriptPath
	}

	// Convert relative path to virtual path with leading slash
	return "/" + scriptPath
}

// ExecuteWithExecutor handles execution of a PHP script through the VFS using the executor module.
func (m *Middleware) ExecuteWithExecutor(vfs *vfs.VFS, scriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	// Ensure scriptPath is normalized if needed before passing
	normalizedScriptPath := normalizePath(scriptPath)

	// Determine the dev server port (0 if not configured)
	devServerPort := 0
	if m.devServerConfig != nil {
		devServerPort = m.devServerConfig.Port
	}

	// Create an executor instance with the current middleware configuration
	execConfig := executor.Config{
		Logger:           m.logger,
		DevelopmentMode:  m.developmentMode,
		DisplayErrors:    m.displayErrors,
		ErrorHandlerPath: m.errorHandlerPath,
		// Pass auto-reload config
		EnableAutoReload:        m.enableAutoReload,
		AutoReloadScriptContent: m.autoReloadScriptContent, // The JS template
		AutoReloadPagePath:      normalizedScriptPath,      // Pass the normalized script path
		AutoReloadDevServerPort: devServerPort,             // Pass the dev server port (0 if not used)
	}

	exec := executor.NewExecutor(execConfig, vfs)

	// Use the executor to run the script
	exec.Execute(vfs, normalizedScriptPath, renderFn, w, r)
}

// --- Reload Event Hub (SSE Server-Side Filter Version) ---

// HubClientSSE stores information about a connected SSE client
type HubClientSSE struct {
	channel  chan string // Channel for sending messages to this client
	pagePath string      // The script path this client initially loaded
}

// ReloadEventHub manages connected clients for auto-reload functionality
type ReloadEventHub struct {
	// Use map from channel to the client struct for easy lookup/removal
	sseClients      map[chan string]*HubClientSSE
	sseClientsMutex sync.RWMutex

	register   chan *HubClientSSE       // Channel for new clients
	unregister chan *HubClientSSE       // Channel for clients leaving
	broadcast  chan vfs.FileChangeEvent // Channel for file change events from VFS

	// Polling related fields (kept for consistency)
	lastChangeTime int64
	changesMutex   sync.RWMutex

	logger *log.Logger
}

// NewReloadEventHub creates a new event hub
func NewReloadEventHub(logger *log.Logger) *ReloadEventHub {
	hub := &ReloadEventHub{
		sseClients:     make(map[chan string]*HubClientSSE),
		register:       make(chan *HubClientSSE),
		unregister:     make(chan *HubClientSSE),
		broadcast:      make(chan vfs.FileChangeEvent, 10), // Buffered broadcast channel
		lastChangeTime: time.Now().UnixNano() / int64(time.Millisecond),
		logger:         logger,
	}
	go hub.runSSE() // Start the hub's processing loop
	return hub
}

// runSSE starts the hub's main loop for handling SSE client registration,
// unregistration, and filtered broadcasts.
func (h *ReloadEventHub) runSSE() {
	for {
		select {
		case client := <-h.register:
			h.sseClientsMutex.Lock()
			h.sseClients[client.channel] = client
			h.logger.Printf("Reload Hub (SSE): Client registered for page %s. Total: %d", client.pagePath, len(h.sseClients))
			h.sseClientsMutex.Unlock()

		case client := <-h.unregister:
			h.sseClientsMutex.Lock()
			if currentClient, ok := h.sseClients[client.channel]; ok {
				// Check if it's the same client instance trying to unregister
				if currentClient == client { // Avoid race condition if channel was reused somehow
					pagePath := client.pagePath // Get page path before deleting
					delete(h.sseClients, client.channel)
					// Don't close the channel here, let SSEReloadHandler's defer handle it
					// close(client.channel)
					h.logger.Printf("Reload Hub (SSE): Client unregistered for page %s. Total: %d", pagePath, len(h.sseClients))
				}
			}
			h.sseClientsMutex.Unlock()

		case event := <-h.broadcast:
			// Update last change time for potential polling clients
			h.changesMutex.Lock()
			h.lastChangeTime = time.Now().UnixNano() / int64(time.Millisecond)
			h.changesMutex.Unlock()

			// Format event data once
			eventData := map[string]interface{}{
				"event": "modified",   // Keep event name consistent for client
				"type":  "fileChange", // Keep type consistent for client
				"path":  event.VirtualPath,
				"time":  event.EventTime.Format(time.RFC3339),
			}
			jsonData, err := json.Marshal(eventData)
			if err != nil {
				h.logger.Printf("Reload Hub (SSE): Error marshalling event data: %v", err)
				continue // Skip broadcast if marshalling fails
			}

			// Format SSE message once
			sseMessage := fmt.Sprintf("event: fileChange\ndata: %s\n\n", jsonData)

			// Send filtered messages (read lock allows concurrent reads)
			h.sseClientsMutex.RLock()
			h.logger.Printf("Reload Hub (SSE): Processing broadcast for event on %s (%s)", event.VirtualPath, event.ChangeType)

			clientsToRemove := []chan string{} // Collect slow/closed clients
			for channel, client := range h.sseClients {
				// *** FILTERING LOGIC ***
				// Reload if the changed file matches the page the client loaded
				// TODO: Add more sophisticated filtering (e.g., for global CSS/JS, includes?)
				if event.VirtualPath == client.pagePath {
					h.logger.Printf("Reload Hub (SSE): Match found! Sending update for %s to client for page %s", event.VirtualPath, client.pagePath)
					// Use non-blocking send with timeout
					select {
					case channel <- sseMessage:
						// Successfully sent
					case <-time.After(250 * time.Millisecond): // Increased timeout slightly
						h.logger.Printf("Reload Hub (SSE): Client send timeout for page %s, queueing for removal.", client.pagePath)
						clientsToRemove = append(clientsToRemove, channel)
					}
				}
			}
			h.sseClientsMutex.RUnlock()

			// Remove slow/closed clients (needs write lock)
			if len(clientsToRemove) > 0 {
				h.sseClientsMutex.Lock()
				for _, channel := range clientsToRemove {
					if client, ok := h.sseClients[channel]; ok {
						pagePath := client.pagePath // Get path for logging
						delete(h.sseClients, channel)
						close(channel) // Close the channel to signal the SSEReloadHandler's loop
						h.logger.Printf("Reload Hub (SSE): Removed slow/closed client for page %s.", pagePath)
					}
				}
				h.logger.Printf("Reload Hub (SSE): Clients after cleanup: %d", len(h.sseClients))
				h.sseClientsMutex.Unlock()
			}
		}
	}
}

// BroadcastFileChange sends a file change event to the hub's broadcast channel
func (h *ReloadEventHub) BroadcastFileChange(event vfs.FileChangeEvent) {
	select {
	case h.broadcast <- event:
		h.logger.Printf("Reload Hub: Queued event for %s (%s)", event.VirtualPath, event.ChangeType)
	default:
		h.logger.Printf("Reload Hub: Broadcast channel full, dropping event for %s (%s)", event.VirtualPath, event.ChangeType)
	}
}

// SSEReloadHandler handles Server-Sent Events connections for auto-reload
func (m *Middleware) SSEReloadHandler(w http.ResponseWriter, r *http.Request) {
	if m.reloadHub == nil || !m.enableAutoReload {
		http.Error(w, "Auto-reload not enabled", http.StatusServiceUnavailable)
		return
	}

	// --- Get Page Path from Query Param ---
	pageQuery := r.URL.Query().Get("page")
	if pageQuery == "" {
		m.logger.Println("Reload Hub (SSE): Connection rejected. Missing 'page' query parameter.")
		http.Error(w, "Missing 'page' query parameter", http.StatusBadRequest)
		return
	}
	// Use the normalization function
	pagePath := normalizePath(pageQuery)
	m.logger.Printf("Reload Hub (SSE): Connection attempt for page: %s", pagePath)
	// --- End Get Page Path ---

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Consider restricting

	// Create client channel and struct
	clientChan := make(chan string, 10) // Buffered channel
	client := &HubClientSSE{
		channel:  clientChan,
		pagePath: pagePath,
	}

	// Register client with the hub
	m.reloadHub.register <- client

	// Unregister when handler exits
	defer func() {
		m.reloadHub.unregister <- client
		// The channel is closed by the hub.runSSE() when unregistering or on timeout
	}()

	// Send initial connected message
	fmt.Fprintf(w, "event: connected\ndata: {\"time\": \"%s\"}\n\n", time.Now().Format(time.RFC3339))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done(): // Client disconnected
			m.logger.Printf("Reload Hub (SSE): Client disconnected for page %s.", client.pagePath)
			return
		case msg, ok := <-clientChan:
			if !ok { // Channel closed by hub (likely due to send timeout/cleanup or unregister)
				m.logger.Printf("Reload Hub (SSE): Hub closed channel for page %s.", client.pagePath)
				return
			}
			_, err := fmt.Fprint(w, msg)
			if err != nil {
				m.logger.Printf("Reload Hub (SSE): Error writing to client for page %s: %v", client.pagePath, err)
				return // Error writing, likely client disconnected
			}
			flusher.Flush() // Flush the message to the client
		}
	}
}

// PollingReloadHandler handles polling requests for file changes
func (m *Middleware) PollingReloadHandler(w http.ResponseWriter, r *http.Request) {
	if m.reloadHub == nil || !m.enableAutoReload {
		http.Error(w, "Auto-reload not enabled", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Consider restricting

	lastParam := r.URL.Query().Get("last")
	var lastClientCheck int64
	if lastParam != "" {
		lastClientCheck, _ = strconv.ParseInt(lastParam, 10, 64) // Ignore error, defaults to 0
	}

	m.reloadHub.changesMutex.RLock()
	lastChangeTime := m.reloadHub.lastChangeTime
	m.reloadHub.changesMutex.RUnlock()

	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	hasChanges := lastClientCheck > 0 && lastChangeTime > lastClientCheck

	response := map[string]interface{}{
		"timestamp":  currentTime,
		"hasChanges": hasChanges,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Printf("Reload Hub: Error encoding polling response: %v", err)
	}
}

// --- End Reload Event Hub ---

// --- Auto-Reload Route Registration ---

// RegisterReloadRoutes registers the necessary HTTP endpoints for auto-reload
// with the provided ServeMux or router.
func (m *Middleware) RegisterReloadRoutes(mux *http.ServeMux) {
	if !m.enableAutoReload {
		m.logger.Println("Auto-reload disabled, skipping route registration.")
		return
	}

	if mux == nil {
		m.logger.Println("Error: Cannot register reload routes, provided mux is nil.")
		return
	}

	trigger := m.getReloadTriggerType()
	m.logger.Printf("Registering auto-reload routes (Trigger: %s)", trigger)

	if trigger == "sse" || trigger == "websocket" { // Assuming WS might use SSE endpoint initially
		mux.HandleFunc("/_/frango/SSE", m.SSEReloadHandler)
		m.logger.Println("Registered SSE handler at /_/frango/SSE")
	}
	// TODO: Add WebSocket handler registration when implemented
	// if trigger == "websocket" {
	//  mux.HandleFunc("/_frango_reload_ws", m.WebSocketReloadHandler)
	// }
	if trigger == "polling" {
		mux.HandleFunc("/_frango_reload_poll", m.PollingReloadHandler)
		m.logger.Println("Registered Polling handler at /_frango_reload_poll")
	}
}

// Helper function (ensure it's defined)
func normalizePath(path string) string {
	if path == "" || path == "." {
		return "/"
	}
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Replace backslashes with forward slashes
	path = strings.ReplaceAll(path, "\\", "/")
	// Clean the path (removes ., .., merges //)
	// Use path package for consistent forward slashes
	cleanedPath := filepath.Clean(path)
	// Important: filepath.Clean might return "." if input is just "/", handle this
	if cleanedPath == "." {
		return "/"
	}
	// Ensure it still starts with / after cleaning
	if !strings.HasPrefix(cleanedPath, "/") {
		cleanedPath = "/" + cleanedPath
	}
	return cleanedPath
}

// phpContextKey is a custom type for context keys
type phpContextKey string

// enableCORS wraps a handler with basic CORS headers for development.
// Allows requests from any origin.
func enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}
