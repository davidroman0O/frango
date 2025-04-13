package executor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davidroman0O/frango/v2/pkg/php"
	"github.com/davidroman0O/frango/v2/pkg/vfs"
	"github.com/dunglas/frankenphp"
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
	config Config
	vfs    *vfs.VFS
	// Potentially add frankenphp related config/state here if needed
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
	}
}

// Execute runs a PHP script using the provided VFS and request context.
func (e *Executor) Execute(vfs *vfs.VFS, scriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	config := e.config
	logger := config.Logger
	startTime := time.Now()
	e.vfs = vfs

	if logger != nil {
		logger.Printf("Executor: Starting execution for script: %s", scriptPath)
	}

	// 0. Make sure the VFS has the PHP globals script installed
	globalsFile := "/_frango_php_globals.php" // Path to PHP globals script
	if err := e.ensurePhpGlobals(vfs, globalsFile); err != nil {
		logger.Printf("Warning: Failed to update VFS with PHP globals: %v", err)
	}

	// 1. Extract relevant data from the HTTP request
	requestData := extractRequestData(r)
	if requestData == nil {
		// Use the default error handler for request parsing errors
		e.handleExecutionError(w, r, fmt.Errorf("failed to extract request data"), http.StatusInternalServerError, scriptPath, "")
		return
	}

	// 2. Resolve the logical script path to a physical path (VFS or filesystem)
	resolvedPath, isVfsPath, err := e.resolveScriptPath(vfs, scriptPath)
	if err != nil {
		e.handleExecutionError(w, r, err, http.StatusNotFound, scriptPath, "") // Treat resolution errors as Not Found
		return
	}
	if logger != nil {
		origin := "Filesystem"
		if isVfsPath {
			origin = "VFS"
		}
		logger.Printf("Executor: Resolved script '%s' to physical path '%s' (Source: %s)", scriptPath, resolvedPath, origin)
	}

	// 3. Verify and ensure the PHP file exists
	resolvedPath = e.ensurePhpFileExists(resolvedPath, scriptPath)
	if resolvedPath == "" {
		e.handleExecutionError(w, r, fmt.Errorf("failed to locate PHP file"), http.StatusInternalServerError, scriptPath, "")
		return
	}

	// 4. Prepare Go-specific environment data
	goEnvData := e.prepareEnvironmentData(requestData, scriptPath, resolvedPath, renderFn, w, r)

	// 5. Prepare document root and script name for FrankenPHP
	documentRoot := filepath.Dir(resolvedPath)
	originalScriptName := "/" + filepath.Base(scriptPath) // The original script name for PHP_SELF, etc.

	if logger != nil {
		logger.Printf("Executor: Setup paths - DocumentRoot='%s', ScriptName='%s'",
			documentRoot, originalScriptName)
	}

	// 6. Build the final PHP environment variables
	phpEnv := e.buildPhpEnvironment(requestData, goEnvData, resolvedPath, documentRoot, originalScriptName, r)

	// Add globals file path to environment
	phpEnv["PHP_GLOBALS_FILE"] = globalsFile

	// Set auto_prepend_file for better integration with FrankenPHP
	globalsFilePath := filepath.Join(vfs.GetTempDir(), strings.TrimPrefix(globalsFile, "/"))
	phpEnv["auto_prepend_file"] = globalsFilePath

	// Add script directory to environment for proper includes
	scriptDir := filepath.Dir(resolvedPath)
	phpEnv["SCRIPT_DIR"] = scriptDir

	// Add logical filename and dirname for emulating __FILE__ and __DIR__
	phpEnv["FRANGO_LOGICAL_FILENAME"] = resolvedPath
	phpEnv["FRANGO_LOGICAL_DIR"] = scriptDir

	// Add environment variable to tell our globals script to set the working directory
	phpEnv["FRANGO_DO_CHDIR"] = "1"

	// Log environment variables in debug mode
	e.logEnvironmentVariables(phpEnv)

	// 7. Execute the PHP script directly (no wrapper)
	recorder, exitCode, execErr := executePhpWithRecorder(r.Context(), resolvedPath, phpEnv, r, logger)

	// 8. Check for PHP execution errors
	err = e.CheckPHPErrors(w, r, execErr, exitCode, recorder.Body.Bytes(), scriptPath, resolvedPath)
	if err != nil {
		// Error was handled by CheckPHPErrors (either custom handler or default)
		duration := time.Since(startTime)
		if logger != nil {
			logger.Printf("Executor: Finished execution for script: %s (Duration: %s, With Error)", scriptPath, duration)
		}
		return // Stop processing, error response already sent
	}

	// 9. If no errors, copy the full response (headers and body) to the original ResponseWriter
	if logger != nil {
		// Log response details
		logger.Printf("Executor: PHP script executed successfully (Exit Code: %d). Copying response with %d bytes and HTTP status %d.",
			exitCode, len(recorder.Body.Bytes()), recorder.Code)
	}

	// Copy all headers from the recorder to the original ResponseWriter
	for key, values := range recorder.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the status code from the PHP response
	w.WriteHeader(recorder.Code)

	// Write the body content from the recorder to the original ResponseWriter
	_, writeErr := w.Write(recorder.Body.Bytes())
	if writeErr != nil && logger != nil {
		// Log error if writing the successful response fails
		logger.Printf("Executor: Error writing PHP output to response writer for %s: %v", scriptPath, writeErr)
	}

	duration := time.Since(startTime)
	if logger != nil {
		logger.Printf("Executor: Finished execution for script: %s (Duration: %s, Success)", scriptPath, duration)
	}
}

// executePhpWithRecorder executes a PHP script using FrankenPHP and returns the recorder, exit code, and any error result.
func executePhpWithRecorder(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) (*httptest.ResponseRecorder, int, error) {
	if logger != nil {
		logger.Printf("Executor: Executing PHP script: %s", scriptPath)
		logger.Printf("Executor: Total PHP environment variables: %d", len(env))
	}

	// Determine document root from environment (must be the parent directory of the script)
	documentRoot := env["DOCUMENT_ROOT"]

	// Add output buffering control to ensure clean output
	// This ensures PHP's output buffer is flushed and reset between script executions
	if env["auto_prepend_text"] == "" {
		env["auto_prepend_text"] = "<?php ob_clean(); ?>"
	} else {
		env["auto_prepend_text"] = "<?php ob_clean(); ?>" + env["auto_prepend_text"]
	}

	// CRITICAL: Modify the request clone path to match the script name
	reqClone := r.Clone(ctx)
	reqClone.URL.Path = env["SCRIPT_NAME"]

	if logger != nil {
		logger.Printf("Executor: Modified request path for FrankenPHP: %s", reqClone.URL.Path)
	}

	// Dump environment variables for debugging (in verbose logging mode)
	if logger != nil && strings.Contains(os.Getenv("LOG_LEVEL"), "DEBUG") {
		var keys []string
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := env[k]
			logger.Printf("  %s = %s", k, v)
		}
	}

	// Create FrankenPHP request
	phpRequest, err := frankenphp.NewRequestWithContext(
		reqClone, // Use the modified request with the script path
		frankenphp.WithRequestDocumentRoot(documentRoot, false), // Exact document root
		frankenphp.WithRequestEnv(env),                          // All environment variables
	)
	if err != nil {
		if logger != nil {
			logger.Printf("Executor: Error creating PHP request: %v", err)
		}
		return nil, 1, fmt.Errorf("failed to create PHP request: %w", err)
	}

	// Create a recorder to capture the output
	recorder := httptest.NewRecorder()

	// Execute the PHP script
	execErr := frankenphp.ServeHTTP(recorder, phpRequest)
	if execErr != nil && logger != nil {
		logger.Printf("Executor: Error executing PHP script: %v", execErr)
	}

	// Get the response body to check for PHP errors
	respBody := recorder.Body.Bytes()

	// Check for PHP errors in the content
	phpErrorResult := php.CheckErrors(string(respBody))

	// Division by zero is a critical error that should always trigger the error handler
	divisionByZeroErr := strings.Contains(strings.ToLower(string(respBody)), "division by zero")
	if divisionByZeroErr && phpErrorResult == nil {
		phpErrorResult = &php.ErrorResult{
			Type:      php.ErrorFatal,
			Indicator: "Division by zero error detected",
			Context:   "FrankenPHP detected division by zero in script execution",
		}
	}

	// Get the status code (default to 0 for success)
	exitCode := 0
	if execErr != nil {
		exitCode = 1
	}

	// Log output in debug mode
	if logger != nil && len(respBody) > 0 && strings.Contains(os.Getenv("LOG_LEVEL"), "DEBUG") {
		if len(respBody) > 200 {
			logger.Printf("Executor: PHP output (truncated): %s...", respBody[:200])
		} else {
			logger.Printf("Executor: PHP output: %s", respBody)
		}
	}

	// Return the combined error if we have PHP errors
	var resultErr error
	if execErr != nil || phpErrorResult != nil {
		resultErr = &PHPExecutionError{
			ExecErr:        execErr,
			PHPErrorResult: phpErrorResult,
		}
	}

	return recorder, exitCode, resultErr
}

// ensurePhpGlobals ensures that the PHP globals script is installed in the VFS
func (e *Executor) ensurePhpGlobals(v *vfs.VFS, globalsPath string) error {
	// Check if the globals file already exists
	if v.FileExists(globalsPath) {
		return nil // Already installed
	}

	// Get the PHP globals script from the php package
	provider := &php.StandardGlobalsProvider{}

	// Create a more robust globals script that properly sets up the PHP environment for direct execution
	globalsScript := `<?php
// PHP Globals script for the Go-PHP Executor
// This script sets up the PHP environment for proper script execution

// Reset the output buffer to ensure clean output
if (function_exists('ob_get_level') && ob_get_level() > 0) {
    ob_end_clean();
}
ob_start();

// If SCRIPT_DIR is set, change to that directory to ensure includes work properly
if (isset($_SERVER['SCRIPT_DIR']) && $_SERVER['SCRIPT_DIR'] !== '' && isset($_SERVER['FRANGO_DO_CHDIR']) && $_SERVER['FRANGO_DO_CHDIR'] === '1') {
    if (is_dir($_SERVER['SCRIPT_DIR'])) {
        // Change working directory to script directory for proper include resolution
        chdir($_SERVER['SCRIPT_DIR']);
        
        // For debugging
        // error_log("Changed working directory to: " . $_SERVER['SCRIPT_DIR']);
        // error_log("Current working directory: " . getcwd());
    }
}

// Define a class to enable overriding __FILE__ and __DIR__ magic constants
class FrangoConstants {
    public static function getLogicalFile() {
        return isset($_SERVER['FRANGO_LOGICAL_FILENAME']) ? $_SERVER['FRANGO_LOGICAL_FILENAME'] : __FILE__;
    }
    
    public static function getLogicalDir() {
        return isset($_SERVER['FRANGO_LOGICAL_DIR']) ? $_SERVER['FRANGO_LOGICAL_DIR'] : __DIR__;
    }
}

// Store the logical file and directory values for external access
$GLOBALS['FRANGO_LOGICAL_FILE'] = FrangoConstants::getLogicalFile();
$GLOBALS['FRANGO_LOGICAL_DIR'] = FrangoConstants::getLogicalDir();

// Set up custom error handler to capture fatal errors
set_error_handler(function($errno, $errstr, $errfile, $errline) {
    // Only handle fatal errors that would halt execution
    if ($errno == E_ERROR || $errno == E_PARSE || $errno == E_CORE_ERROR || 
        $errno == E_COMPILE_ERROR || $errno == E_USER_ERROR) {
        
        // Save error information in environment variables
        putenv("PHP_LAST_ERROR=" . $errstr);
        putenv("PHP_ERROR_TYPE=fatal");
        putenv("PHP_ERROR_CONTEXT=Error in " . $errfile . " on line " . $errline);
        
        // Log the error
        error_log("PHP Fatal Error: " . $errstr . " in " . $errfile . " on line " . $errline);
    }
    
    // Return false to allow the standard PHP error handler to run
    return false;
});

// Merge path parameters into $_GET for compatibility
if (isset($_SERVER['_PATH']) && $_SERVER['_PATH'] !== '{}') {
    $_PATH = json_decode($_SERVER['_PATH'], true);
    if (is_array($_PATH)) {
        // Create the $_PATH global for backwards compatibility
        $GLOBALS['_PATH'] = $_PATH;
        
        // Merge path parameters into $_GET (path parameters take precedence)
        $_GET = array_merge($_GET, $_PATH);
        
        // Also update $_REQUEST
        $_REQUEST = array_merge($_REQUEST, $_PATH);
    }
}

// Make additional environment data available to the script
if (isset($_SERVER['_JSON']) && $_SERVER['_JSON'] !== '{}') {
    $_JSON = json_decode($_SERVER['_JSON'], true);
    $GLOBALS['_JSON'] = $_JSON;
}

// Helper function to get current logical file path
if (!function_exists('get_current_script_path')) {
    function get_current_script_path() {
        return $GLOBALS['FRANGO_LOGICAL_FILE'];
    }
}

// Helper function to get current logical directory path
if (!function_exists('get_current_script_dir')) {
    function get_current_script_dir() {
        return $GLOBALS['FRANGO_LOGICAL_DIR'];
    }
}

// Clean up any previous script execution state
if (function_exists('session_status') && session_status() === PHP_SESSION_ACTIVE) {
    session_write_close();
}

// Reset user-defined variables to avoid leaking state between requests
unset($GLOBALS['_frango_include_state']);
unset($GLOBALS['_frango_request_state']);

// Register shutdown function to flush output
register_shutdown_function(function() {
    if (function_exists('ob_get_level') && ob_get_level() > 0) {
        while (ob_get_level() > 0) {
            ob_end_flush();
        }
    }
});

// Load globals from the PHP package
` + provider.GetScript()

	// Install the globals script
	return v.CreateVirtualFile(globalsPath, []byte(globalsScript))
}
