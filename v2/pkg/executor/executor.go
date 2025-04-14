package executor

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davidroman0O/frango/v2/pkg/php"
	"github.com/davidroman0O/frango/v2/pkg/vfs"
)

// Config holds configuration for the executor
type Config struct {
	Logger           *log.Logger
	DevelopmentMode  bool
	DisplayErrors    bool
	ErrorHandlerPath string

	// Auto-reload specific config passed from Middleware
	EnableAutoReload        bool
	AutoReloadScriptContent string
	AutoReloadPagePath      string // The virtual path of the main script being executed
	AutoReloadDevServerPort int    // Port of the separate dev server (0 if not used)
}

// Executor handles the execution of PHP scripts.
type Executor struct {
	config       Config
	vfs          *vfs.VFS
	pathCache    sync.Map  // Cache for resolved script paths
	recorderPool sync.Pool // Pool for response recorders
}

// RenderData defines the function signature for providing template data.
// This needs to be passed in from the calling package (frango).
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}

// PHPExecutionError wraps the execution error and PHP error result
type PHPExecutionError struct {
	ExecErr        error
	PHPErrorResult *php.ErrorResult
}

// Error implements the error interface
func (e *PHPExecutionError) Error() string {
	if e.ExecErr != nil {
		return e.ExecErr.Error()
	}
	if e.PHPErrorResult != nil {
		return fmt.Sprintf("PHP %s: %s", e.PHPErrorResult.Type, e.PHPErrorResult.Indicator)
	}
	return "unknown PHP error"
}

// Unwrap returns the underlying error
func (e *PHPExecutionError) Unwrap() error {
	return e.ExecErr
}

// NewExecutor creates a new PHP executor instance.
func NewExecutor(config Config, vfs *vfs.VFS) *Executor {
	return &Executor{
		config: config,
		vfs:    vfs,
		recorderPool: sync.Pool{
			New: func() interface{} {
				return httptest.NewRecorder()
			},
		},
	}
}

// Execute runs a PHP script using the provided VFS and request context.
// The renderFn should have the signature: func(w http.ResponseWriter, r *http.Request) map[string]interface{}
func (e *Executor) Execute(vfs *vfs.VFS, scriptPath string, renderFn func(w http.ResponseWriter, r *http.Request) map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	logger := e.config.Logger
	e.vfs = vfs

	if logger != nil {
		logger.Printf("Executor: Starting execution for script: %s", scriptPath)
	}

	// Initialize execution environment
	if err := e.initializeExecution(vfs); err != nil {
		e.handleExecutionError(w, r, err, http.StatusInternalServerError, scriptPath, "")
		return
	}

	// Prepare request data
	requestData, resolvedPath, err := e.prepareRequest(vfs, scriptPath, renderFn, w, r)
	if err != nil {
		e.handleExecutionError(w, r, err, http.StatusNotFound, scriptPath, "")
		return
	}

	// Set up PHP environment
	phpEnv, err := e.setupPhpEnvironment(requestData, scriptPath, resolvedPath, renderFn, w, r)
	if err != nil {
		e.handleExecutionError(w, r, err, http.StatusInternalServerError, scriptPath, resolvedPath)
		return
	}

	// Execute PHP script
	recorder, exitCode, execErr := executePhpWithRecorder(r.Context(), resolvedPath, phpEnv, r, e.config.Logger)

	// Handle errors if any
	err = e.CheckPHPErrors(w, r, execErr, exitCode, recorder.Body.Bytes(), scriptPath, resolvedPath)
	if err != nil {
		duration := time.Since(startTime)
		if logger != nil {
			logger.Printf("Executor: Finished execution for script: %s (Duration: %s, With Error)", scriptPath, duration)
		}
		return // Stop processing, error response already sent
	}

	// Send response to client
	e.sendResponse(w, recorder, exitCode, scriptPath)

	duration := time.Since(startTime)
	if logger != nil {
		logger.Printf("Executor: Finished execution for script: %s (Duration: %s, Success)", scriptPath, duration)
	}
}

// initializeExecution prepares the execution environment
func (e *Executor) initializeExecution(vfs *vfs.VFS) error {
	// Ensure PHP globals script is installed
	globalsFile := "/_frango_php_globals.php"
	if err := e.ensurePhpGlobals(vfs, globalsFile); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Printf("Warning: Failed to update VFS with PHP globals: %v", err)
		}
		return err
	}
	return nil
}

// prepareRequest extracts request data and resolves the script path
func (e *Executor) prepareRequest(vfs *vfs.VFS, scriptPath string, renderFn func(w http.ResponseWriter, r *http.Request) map[string]interface{}, w http.ResponseWriter, r *http.Request) (*RequestData, string, error) {
	logger := e.config.Logger

	// Extract relevant data from the HTTP request
	requestData := extractRequestData(r)
	if requestData == nil {
		return nil, "", fmt.Errorf("failed to extract request data")
	}

	// Resolve the script path (use cache if available)
	resolvedPath, isVfsPath, err := e.resolveScriptPath(vfs, scriptPath)
	if err != nil {
		return nil, "", err
	}

	if logger != nil {
		origin := "Filesystem"
		if isVfsPath {
			origin = "VFS"
		}
		logger.Printf("Executor: Resolved script '%s' to physical path '%s' (Source: %s)", scriptPath, resolvedPath, origin)
	}

	// Verify the PHP file exists
	resolvedPath = e.ensurePhpFileExists(resolvedPath, scriptPath)
	if resolvedPath == "" {
		return nil, "", fmt.Errorf("failed to locate PHP file")
	}

	return requestData, resolvedPath, nil
}

// sendResponse sends the PHP script response to the client
func (e *Executor) sendResponse(w http.ResponseWriter, recorder *httptest.ResponseRecorder, exitCode int, scriptPath string) {
	logger := e.config.Logger

	if logger != nil {
		logger.Printf("Executor: PHP script executed successfully (Exit Code: %d). Preparing response (Status: %d, Size: %d bytes).",
			exitCode, recorder.Code, recorder.Body.Len())
	}

	// Get content type and body bytes from recorder
	contentType := recorder.Header().Get("Content-Type")
	bodyBytes := recorder.Body.Bytes()

	// --- Inject Auto-Reload Script if enabled and content is HTML ---
	if e.config.EnableAutoReload && strings.Contains(strings.ToLower(contentType), "text/html") {
		// Use AutoReloadPagePath from config (which is the original scriptPath passed to ExecuteWithExecutor)
		pagePathForInjection := e.config.AutoReloadPagePath
		if e.config.AutoReloadScriptContent != "" && pagePathForInjection != "" {
			originalLen := len(bodyBytes)
			// Pass the page path to inject function
			bodyBytes = e.injectAutoReloadScript(bodyBytes, e.config.AutoReloadScriptContent, pagePathForInjection)
			newLen := len(bodyBytes)
			if newLen != originalLen {
				if logger != nil {
					logger.Printf("Executor: Injected auto-reload script (%d bytes added). Updating Content-Length.", newLen-originalLen)
				}
				// Update Content-Length header
				recorder.Header().Set("Content-Length", strconv.Itoa(newLen))
				// Delete ETag as content has changed
				recorder.Header().Del("ETag")
			} else if logger != nil {
				logger.Println("Executor: Auto-reload script injection did not modify content length.")
			}
		} else if logger != nil {
			logger.Println("Executor: Auto-reload enabled but script content is empty, skipping injection.")
		}
	} else if e.config.EnableAutoReload && logger != nil {
		logger.Printf("Executor: Auto-reload enabled but content-type ('%s') is not HTML, skipping injection.", contentType)
	}
	// --- End Auto-Reload Script Injection ---

	// Copy all headers from the recorder
	for key, values := range recorder.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the status code
	w.WriteHeader(recorder.Code)

	// Write the potentially modified body content
	_, writeErr := w.Write(bodyBytes)
	if writeErr != nil && logger != nil {
		logger.Printf("Executor: Error writing final output to response writer for %s: %v", scriptPath, writeErr)
	}
}

// injectAutoReloadScript injects the auto-reload script into HTML content,
// replacing the path and dev server port placeholders.
func (e *Executor) injectAutoReloadScript(content []byte, scriptTemplate string, pagePath string) []byte {
	if scriptTemplate == "" || pagePath == "" {
		e.config.Logger.Println("Executor: injectAutoReloadScript returning early - empty template or pagePath.")
		return content
	}

	// Define placeholders
	pathPlaceholder := "%%SCRIPT_PATH%%"
	devPortPlaceholder := "%%DEV_SERVER_PORT%%"
	devPortStr := strconv.Itoa(e.config.AutoReloadDevServerPort)

	e.config.Logger.Printf("Executor: injectAutoReloadScript called. PagePath: '%s', DevPort: %s", pagePath, devPortStr)
	if !strings.Contains(scriptTemplate, pathPlaceholder) {
		e.config.Logger.Printf("Executor: WARNING - scriptTemplate does NOT contain placeholder '%s'!", pathPlaceholder)
	}
	if !strings.Contains(scriptTemplate, devPortPlaceholder) {
		e.config.Logger.Printf("Executor: WARNING - scriptTemplate does NOT contain placeholder '%s'!", devPortPlaceholder)
	}

	// Perform replacements
	interimScript := strings.Replace(scriptTemplate, pathPlaceholder, pagePath, 1)
	finalScript := strings.Replace(interimScript, devPortPlaceholder, devPortStr, 1)

	if finalScript == scriptTemplate {
		e.config.Logger.Printf("Executor: WARNING - strings.Replace did not modify the script template. Placeholders found? path=%t, port=%t",
			strings.Contains(scriptTemplate, pathPlaceholder), strings.Contains(scriptTemplate, devPortPlaceholder))
	} else {
		e.config.Logger.Printf("Executor: Placeholders replaced. Path: '%s', Port: '%s'.", pagePath, devPortStr)
	}

	scriptBytes := []byte(finalScript)

	// Try to inject before closing </body> tag
	bodyTagLower := []byte("</body>")
	if idx := bytes.LastIndex(bytes.ToLower(content), bodyTagLower); idx != -1 {
		// Create a new slice with enough capacity
		newContent := make([]byte, 0, len(content)+len(scriptBytes))
		newContent = append(newContent, content[:idx]...)
		newContent = append(newContent, scriptBytes...)
		newContent = append(newContent, content[idx:]...)
		return newContent
	}

	// If </body> not found, try before </head>
	headTagLower := []byte("</head>")
	if idx := bytes.LastIndex(bytes.ToLower(content), headTagLower); idx != -1 {
		newContent := make([]byte, 0, len(content)+len(scriptBytes))
		newContent = append(newContent, content[:idx]...)
		newContent = append(newContent, scriptBytes...)
		newContent = append(newContent, content[idx:]...)
		return newContent
	}

	// If neither found, simply append to the end (less ideal)
	if e.config.Logger != nil {
		e.config.Logger.Println("Executor: Could not find </body> or </head> tag for script injection, appending to end.")
	}
	return append(content, scriptBytes...)
}

// setupPhpEnvironment prepares the PHP environment variables and data.
