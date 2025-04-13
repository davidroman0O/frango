package executor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	recorder, exitCode, execErr := e.executeScript(r.Context(), resolvedPath, phpEnv, r)

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

// setupPhpEnvironment prepares the PHP execution environment
func (e *Executor) setupPhpEnvironment(requestData *RequestData, scriptPath, resolvedPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) (map[string]string, error) {
	logger := e.config.Logger

	// Prepare Go-specific environment data
	goEnvData := e.prepareEnvironmentData(requestData, scriptPath, resolvedPath, renderFn, w, r)

	// Set up document root and script name
	documentRoot := filepath.Dir(resolvedPath)
	originalScriptName := "/" + filepath.Base(scriptPath)

	if logger != nil {
		logger.Printf("Executor: Setup paths - DocumentRoot='%s', ScriptName='%s'",
			documentRoot, originalScriptName)
	}

	// Build PHP environment variables
	phpEnv := e.buildPhpEnvironment(requestData, goEnvData, resolvedPath, documentRoot, originalScriptName, r)

	// Add additional environment variables
	e.addScriptEnvironment(phpEnv, resolvedPath, scriptPath)

	// Log environment variables in debug mode
	e.logEnvironmentVariables(phpEnv)

	return phpEnv, nil
}

// addScriptEnvironment adds script-specific environment variables
func (e *Executor) addScriptEnvironment(phpEnv map[string]string, resolvedPath, scriptPath string) {
	// Add globals file path
	globalsFile := "/_frango_php_globals.php"
	phpEnv["PHP_GLOBALS_FILE"] = globalsFile

	// Set auto_prepend_file for globals
	globalsFilePath := filepath.Join(e.vfs.GetTempDir(), strings.TrimPrefix(globalsFile, "/"))
	phpEnv["auto_prepend_file"] = globalsFilePath

	// Add script directory for includes
	scriptDir := filepath.Dir(resolvedPath)
	phpEnv["SCRIPT_DIR"] = scriptDir

	// Add logical filename and dirname for __FILE__ and __DIR__
	phpEnv["FRANGO_LOGICAL_FILENAME"] = resolvedPath
	phpEnv["FRANGO_LOGICAL_DIR"] = scriptDir

	// Add environment variable for chdir
	phpEnv["FRANGO_DO_CHDIR"] = "1"
}

// executeScript runs the PHP script using FrankenPHP
func (e *Executor) executeScript(ctx context.Context, scriptPath string, env map[string]string, r *http.Request) (*httptest.ResponseRecorder, int, error) {
	return executePhpWithRecorder(ctx, scriptPath, env, r, e.config.Logger)
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

// executePhpWithRecorder executes a PHP script using FrankenPHP and returns the recorder, exit code, and any error result.
func executePhpWithRecorder(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) (*httptest.ResponseRecorder, int, error) {
	if logger != nil {
		logger.Printf("Executor: Executing PHP script: %s", scriptPath)
		logger.Printf("Executor: Total PHP environment variables: %d", len(env))
	}

	// Prepare PHP environment
	request, err := preparePhpRequest(ctx, scriptPath, env, r, logger)
	if err != nil {
		return nil, 1, err
	}

	// Execute and capture response
	recorder, execErr := executePhpRequest(request, logger)

	// Process PHP output and check for errors
	exitCode, resultErr := processPhpOutput(recorder, execErr, logger)

	return recorder, exitCode, resultErr
}

// preparePhpRequest prepares the PHP request with necessary environment
func preparePhpRequest(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) (*http.Request, error) {
	// Determine document root from environment
	documentRoot := env["DOCUMENT_ROOT"]

	// Add output buffering control
	addOutputBuffering(env)

	// Prepare request path
	reqClone := cloneRequest(ctx, r, env, logger)

	// Create FrankenPHP request with environment
	phpRequest, err := createFrankenPhpRequest(reqClone, documentRoot, env)
	if err != nil {
		if logger != nil {
			logger.Printf("Executor: Error creating PHP request: %v", err)
		}
		return nil, fmt.Errorf("failed to create PHP request: %w", err)
	}

	return phpRequest, nil
}

// addOutputBuffering adds output buffering control to PHP environment
func addOutputBuffering(env map[string]string) {
	// Ensure PHP's output buffer is flushed and reset
	if env["auto_prepend_text"] == "" {
		env["auto_prepend_text"] = "<?php ob_clean(); ?>"
	} else {
		env["auto_prepend_text"] = "<?php ob_clean(); ?>" + env["auto_prepend_text"]
	}
}

// cloneRequest prepares a request clone with modified path
func cloneRequest(ctx context.Context, r *http.Request, env map[string]string, logger *log.Logger) *http.Request {
	reqClone := r.Clone(ctx)
	reqClone.URL.Path = env["SCRIPT_NAME"]

	if logger != nil {
		logger.Printf("Executor: Modified request path for FrankenPHP: %s", reqClone.URL.Path)
	}

	return reqClone
}

// createFrankenPhpRequest creates a FrankenPHP request with proper configuration
func createFrankenPhpRequest(r *http.Request, documentRoot string, env map[string]string) (*http.Request, error) {
	return frankenphp.NewRequestWithContext(
		r,
		frankenphp.WithRequestDocumentRoot(documentRoot, false),
		frankenphp.WithRequestEnv(env),
	)
}

// executePhpRequest executes the PHP script and returns the recorder
func executePhpRequest(request *http.Request, logger *log.Logger) (*httptest.ResponseRecorder, error) {
	// Create a recorder to capture output
	recorder := httptest.NewRecorder()

	// Execute the PHP script
	execErr := frankenphp.ServeHTTP(recorder, request)
	if execErr != nil && logger != nil {
		logger.Printf("Executor: Error executing PHP script: %v", execErr)
	}

	return recorder, execErr
}

// processPhpOutput processes PHP output and checks for errors
func processPhpOutput(recorder *httptest.ResponseRecorder, execErr error, logger *log.Logger) (int, error) {
	// Get response body
	respBody := recorder.Body.Bytes()

	// Check for PHP errors in content
	phpErrorResult := php.CheckErrors(string(respBody))

	// Check for division by zero errors
	divisionByZeroErr := strings.Contains(strings.ToLower(string(respBody)), "division by zero")
	if divisionByZeroErr && phpErrorResult == nil {
		phpErrorResult = &php.ErrorResult{
			Type:      php.ErrorFatal,
			Indicator: "Division by zero error detected",
			Context:   "FrankenPHP detected division by zero in script execution",
		}
	}

	// Determine exit code
	exitCode := 0
	if execErr != nil {
		exitCode = 1
	}

	// Log output in debug mode
	logPhpOutput(logger, respBody)

	// Create combined error result if needed
	var resultErr error
	if execErr != nil || phpErrorResult != nil {
		resultErr = &PHPExecutionError{
			ExecErr:        execErr,
			PHPErrorResult: phpErrorResult,
		}
	}

	return exitCode, resultErr
}

// logPhpOutput logs PHP output for debugging
func logPhpOutput(logger *log.Logger, output []byte) {
	if logger == nil || len(output) == 0 || !strings.Contains(os.Getenv("LOG_LEVEL"), "DEBUG") {
		return
	}

	if len(output) > 200 {
		logger.Printf("Executor: PHP output (truncated): %s...", output[:200])
	} else {
		logger.Printf("Executor: PHP output: %s", output)
	}
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

// resolveScriptPath resolves a script path using the cache if available
func (e *Executor) resolveScriptPath(vfs *vfs.VFS, scriptPath string) (string, bool, error) {
	logger := e.config.Logger

	// Try cache first (if not in development mode)
	if !e.config.DevelopmentMode {
		if cachedPath, ok := e.getPathFromCache(scriptPath); ok {
			if logger != nil {
				logger.Printf("Executor: Using cached path for '%s' -> %s", scriptPath, cachedPath)
			}
			return cachedPath, true, nil
		}
	}

	// Perform regular path resolution
	resolvedPath, isVfsPath, err := e.resolveScriptPathInternal(vfs, scriptPath)

	// Cache the result if successful and not in development mode
	if err == nil && !e.config.DevelopmentMode {
		e.cacheResolvedPath(scriptPath, resolvedPath)
	}

	return resolvedPath, isVfsPath, err
}

// resolveScriptPathInternal performs the actual path resolution
func (e *Executor) resolveScriptPathInternal(vfs *vfs.VFS, scriptPath string) (string, bool, error) {
	logger := e.config.Logger

	if logger != nil {
		logger.Printf("Executor: Resolving script path for '%s'", scriptPath)
	}

	// Ensure VFS is available
	if vfs == nil {
		return "", false, fmt.Errorf("VFS is required to resolve script path for '%s'", scriptPath)
	}

	// Try resolving directly within the VFS
	resolvedPath, err := vfs.ResolvePath(scriptPath)
	if err == nil {
		if logger != nil {
			logger.Printf("Executor: Found '%s' in VFS -> %s", scriptPath, resolvedPath)
		}
		return resolvedPath, true, nil
	} else if logger != nil {
		logger.Printf("Executor: Script '%s' not found directly in VFS: %v", scriptPath, err)
	}

	// Try parameterized paths
	if strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}") {
		return e.resolveParameterizedPath(vfs, scriptPath, logger)
	}

	// If not found anywhere in VFS, return the error
	if logger != nil {
		logger.Printf("Executor: Script '%s' could not be resolved in VFS.", scriptPath)
	}
	return "", false, fmt.Errorf("script '%s' not found in VFS", scriptPath)
}

// resolveParameterizedPath resolves a script path with parameters
func (e *Executor) resolveParameterizedPath(vfs *vfs.VFS, scriptPath string, logger *log.Logger) (string, bool, error) {
	// For parameterized paths, try literal resolution first
	literalPath, err := vfs.ResolvePathLiteral(scriptPath)
	if err == nil {
		if logger != nil {
			logger.Printf("Executor: Found parameterized script '%s' in VFS -> %s", scriptPath, literalPath)
		}
		return literalPath, true, nil
	} else if logger != nil {
		logger.Printf("Executor: Parameterized script '%s' not found with literal resolution: %v", scriptPath, err)
	}

	// Try to find a matching file pattern
	dirPath := filepath.Dir(scriptPath)
	fileName := filepath.Base(scriptPath)

	// List files in that directory
	files, err := vfs.ListFilesIn(dirPath)
	if err == nil && len(files) > 0 {
		// Look for a matching pattern
		for _, f := range files {
			baseF := filepath.Base(f)
			paramPattern := regexp.MustCompile(`\{[^}]+\}`)
			patternRegex := "^" + paramPattern.ReplaceAllString(regexp.QuoteMeta(fileName), ".*") + "$"
			matched, _ := regexp.MatchString(patternRegex, baseF)

			if matched {
				if logger != nil {
					logger.Printf("Executor: Found matching file pattern: %s", f)
				}
				// Try to resolve this file
				resolvedPath, err := vfs.ResolvePath(f)
				if err == nil {
					return resolvedPath, true, nil
				}
			}
		}
	}

	return "", false, fmt.Errorf("parameterized script '%s' not found in VFS", scriptPath)
}

// getPathFromCache retrieves a cached path
func (e *Executor) getPathFromCache(scriptPath string) (string, bool) {
	if val, ok := e.pathCache.Load(scriptPath); ok {
		return val.(string), true
	}
	return "", false
}

// cacheResolvedPath stores a resolved path in the cache
func (e *Executor) cacheResolvedPath(scriptPath, resolvedPath string) {
	e.pathCache.Store(scriptPath, resolvedPath)
}

// ensurePhpFileExists verifies that the PHP file exists and returns the path
func (e *Executor) ensurePhpFileExists(resolvedPath, scriptPath string) string {
	logger := e.config.Logger

	// Check if the file exists
	if _, err := os.Stat(resolvedPath); err == nil {
		if logger != nil {
			logger.Printf("Using resolved PHP file path: %s", resolvedPath)
		}
		return resolvedPath
	}

	if logger != nil {
		logger.Printf("Warning: Could not access PHP file at %s", resolvedPath)
	}

	// Check if the file exists in alternate locations
	alternativePaths := []string{
		filepath.Join(filepath.Dir(resolvedPath), filepath.Base(scriptPath)),
		scriptPath,
	}

	for _, path := range alternativePaths {
		if _, err := os.Stat(path); err == nil {
			if logger != nil {
				logger.Printf("Found PHP file at alternative path: %s", path)
			}
			return path
		}
	}

	if logger != nil {
		logger.Printf("Error: Failed to locate PHP file for script: %s", scriptPath)
	}
	return ""
}

// extractPathParams extracts path parameters from a URL pattern and actual path
func (e *Executor) extractPathParams(pattern, path string) map[string]string {
	// Extract HTTP method if pattern includes it
	patternPath := pattern
	if parts := strings.SplitN(pattern, " ", 2); len(parts) > 1 {
		patternPath = parts[1]
	}

	// Special case for patterns with multiple parameters in the same segment
	if strings.Contains(patternPath, "}-{") {
		return e.extractMultipleParamsPerSegment(patternPath, path)
	}

	// Check if pattern contains a catchall parameter
	if e.containsCatchAllParam(patternPath) {
		return e.extractCatchAllParams(patternPath, path)
	}

	// Split pattern and path into segments
	patternSegments := strings.Split(strings.Trim(patternPath, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Create parameters map
	params := make(map[string]string)

	// Handle empty parameter case
	if len(pathSegments) == len(patternSegments)-1 &&
		len(patternSegments) > 0 &&
		strings.HasPrefix(patternSegments[len(patternSegments)-1], "{") &&
		strings.HasSuffix(patternSegments[len(patternSegments)-1], "}") {
		paramName := patternSegments[len(patternSegments)-1][1 : len(patternSegments[len(patternSegments)-1])-1]
		params[paramName] = ""
		return params
	}

	// Standard case: extract parameters from matching segments
	maxSegments := len(patternSegments)
	if maxSegments > len(pathSegments) {
		maxSegments = len(pathSegments)
	}

	// Extract parameters from matching segments
	for i := 0; i < maxSegments; i++ {
		patternSegment := patternSegments[i]
		pathSegment := pathSegments[i]

		// Check for parameter pattern {name}
		if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
			// Extract parameter name without braces
			paramName := patternSegment[1 : len(patternSegment)-1]

			// Handle special case for catchall params with asterisk
			if strings.HasPrefix(paramName, "*") {
				paramName = paramName[1:] // Remove the asterisk
			}

			if paramName != "" {
				params[paramName] = pathSegment
			}
		} else if patternSegment != pathSegment {
			// Non-parameter segments must match exactly
			return nil
		}
	}

	return params
}

// containsCatchAllParam checks if the pattern contains a catchall parameter
func (e *Executor) containsCatchAllParam(pattern string) bool {
	// Check for catchall syntax with asterisk
	if strings.Contains(pattern, "{*") {
		return true
	}

	// Skip the check if the pattern contains multiple parameters
	if strings.Count(pattern, "{") > 1 || strings.Count(pattern, "}") > 1 {
		return false
	}

	// This is only for patterns like /path/{param}
	// Where the parameter is the last segment
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	for i, segment := range patternSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			// If the parameter is not the last segment, it's not a catchall
			if i < len(patternSegments)-1 {
				return false
			}
			// This is a parameter at the end of the pattern
			return true
		}
	}

	return false
}

// extractCatchAllParams extracts parameters for catchall patterns
func (e *Executor) extractCatchAllParams(pattern, path string) map[string]string {
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Validate segments count
	if len(patternSegments) < 1 || len(pathSegments) < len(patternSegments)-1 {
		return nil
	}

	// Check if all non-parameter segments match
	for i := 0; i < len(patternSegments)-1; i++ {
		if !strings.HasPrefix(patternSegments[i], "{") || !strings.HasSuffix(patternSegments[i], "}") {
			if patternSegments[i] != pathSegments[i] {
				return nil
			}
		}
	}

	// Extract the last parameter
	lastSegment := patternSegments[len(patternSegments)-1]
	if strings.HasPrefix(lastSegment, "{") && strings.HasSuffix(lastSegment, "}") {
		paramName := lastSegment[1 : len(lastSegment)-1]

		// Handle special case for catchall params with asterisk
		if strings.HasPrefix(paramName, "*") {
			paramName = paramName[1:] // Remove the asterisk
		}

		// Capture all remaining path segments
		remainingPath := strings.Join(pathSegments[len(patternSegments)-1:], "/")

		params := make(map[string]string)
		params[paramName] = remainingPath
		return params
	}

	return nil
}

// extractMultipleParamsPerSegment handles extraction of multiple parameters in the same segment
func (e *Executor) extractMultipleParamsPerSegment(pattern, path string) map[string]string {
	// Split pattern and path into segments
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Basic validation - must have same number of segments
	if len(patternSegments) != len(pathSegments) {
		return nil
	}

	params := make(map[string]string)

	// First, check all regular segments (those without multiple params)
	for i, patternSegment := range patternSegments {
		// Skip segments with multiple parameters for now
		if strings.Contains(patternSegment, "}-{") {
			continue
		}

		// Handle regular parameter segments
		if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
			paramName := patternSegment[1 : len(patternSegment)-1]
			if paramName != "" {
				params[paramName] = pathSegments[i]
			}
		} else if patternSegment != pathSegments[i] {
			// Non-parameter segments must match exactly
			return nil
		}
	}

	// Now handle segments with multiple parameters
	for i, patternSegment := range patternSegments {
		if strings.Contains(patternSegment, "}-{") {
			// Process the segment and extract parameter names
			parts := strings.Split(patternSegment, "}-{")

			// Simple handling for segments like {param1}-{param2}-{param3}
			if len(parts) >= 2 && strings.Count(pathSegments[i], "-") == len(parts)-1 {
				values := strings.Split(pathSegments[i], "-")

				for j, part := range parts {
					paramName := ""
					if j == 0 && strings.HasPrefix(part, "{") {
						paramName = part[1:]
					} else if j == len(parts)-1 && strings.HasSuffix(part, "}") {
						paramName = part[:len(part)-1]
					} else {
						paramName = part
					}

					if paramName != "" && j < len(values) {
						params[paramName] = values[j]
					}
				}
			}
		}
	}

	return params
}
