package executor

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
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
func (e *Executor) Execute(vfs *vfs.VFS, scriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
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
func (e *Executor) prepareRequest(vfs *vfs.VFS, scriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) (*RequestData, string, error) {
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
		logger.Printf("Executor: PHP script executed successfully (Exit Code: %d). Copying response with %d bytes and HTTP status %d.",
			exitCode, len(recorder.Body.Bytes()), recorder.Code)
	}

	// Copy all headers from the recorder
	for key, values := range recorder.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the status code
	w.WriteHeader(recorder.Code)

	// Write the body content
	_, writeErr := w.Write(recorder.Body.Bytes())
	if writeErr != nil && logger != nil {
		logger.Printf("Executor: Error writing PHP output to response writer for %s: %v", scriptPath, writeErr)
	}
}
