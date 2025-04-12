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

	"github.com/davidroman0O/frango/v2/php"
	"github.com/davidroman0O/frango/v2/vfs"
	"github.com/dunglas/frankenphp"
)

// Config holds configuration for the executor
type Config struct {
	Logger           *log.Logger
	DevelopmentMode  bool
	DisplayErrors    bool
	ErrorHandlerPath string
	SourceDir        string // Optional: Base directory for resolving source files
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

	// 4. Create the PHP wrapper script
	// Use the *resolved* physical path for the wrapper content, but pass the original path for context
	wrapperPath, err := e.createPhpWrapper(resolvedPath, scriptPath)
	if err != nil {
		e.handleExecutionError(w, r, fmt.Errorf("failed to create PHP wrapper: %w", err), http.StatusInternalServerError, scriptPath, resolvedPath)
		return
	}
	defer func() {
		if err := os.Remove(wrapperPath); err != nil && logger != nil {
			logger.Printf("Executor: Warning - Failed to remove temporary wrapper script %s: %v", wrapperPath, err)
		} else if logger != nil {
			logger.Printf("Executor: Removed temporary wrapper script %s", wrapperPath)
		}
	}() // Ensure cleanup

	if logger != nil {
		logger.Printf("Executor: Created temporary wrapper script at %s", wrapperPath)
	}

	// 5. Prepare Go-specific environment data
	goEnvData := e.prepareEnvironmentData(requestData, scriptPath, resolvedPath, renderFn, w, r)

	// 6. Prepare document root and script name for FrankenPHP
	documentRoot := filepath.Dir(wrapperPath)
	scriptName := "/" + filepath.Base(wrapperPath)
	originalScriptName := "/" + filepath.Base(scriptPath) // The original script name for PHP_SELF, etc.

	if logger != nil {
		logger.Printf("Executor: Setup paths - DocumentRoot='%s', ScriptName='%s', OriginalScriptName='%s'",
			documentRoot, scriptName, originalScriptName)
	}

	// 7. Build the final PHP environment variables
	// Pass original scriptPath for SCRIPT_NAME, resolvedPath for DOCUMENT_ROOT base
	phpEnv := e.buildPhpEnvironment(requestData, goEnvData, wrapperPath, documentRoot, originalScriptName, r)

	// Add globals file path to environment
	phpEnv["PHP_GLOBALS_FILE"] = globalsFile

	// Log environment variables in debug mode
	e.logEnvironmentVariables(phpEnv)

	// 8. Execute the PHP script
	phpOutput, exitCode, execErr := executePHP(r.Context(), wrapperPath, phpEnv, r, logger)

	// 9. Check for PHP execution errors
	err = e.CheckPHPErrors(w, r, execErr, exitCode, phpOutput, scriptPath, resolvedPath)
	if err != nil {
		// Error was handled by CheckPHPErrors (either custom handler or default)
		duration := time.Since(startTime)
		if logger != nil {
			logger.Printf("Executor: Finished execution for script: %s (Duration: %s, With Error)", scriptPath, duration)
		}
		return // Stop processing, error response already sent
	}

	// 10. If no errors, write the captured PHP output to the original ResponseWriter
	if logger != nil {
		// Log output size instead of content for brevity
		logger.Printf("Executor: PHP script executed successfully (Exit Code: %d). Writing %d bytes of output.", exitCode, len(phpOutput))
	}
	_, writeErr := w.Write(phpOutput)
	if writeErr != nil && logger != nil {
		// Log error if writing the successful response fails
		logger.Printf("Executor: Error writing PHP output to response writer for %s: %v", scriptPath, writeErr)
	}

	duration := time.Since(startTime)
	if logger != nil {
		logger.Printf("Executor: Finished execution for script: %s (Duration: %s, Success)", scriptPath, duration)
	}
}

// executePHP executes a PHP script using FrankenPHP and returns the output, exit code, and any error result.
func executePHP(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) ([]byte, int, error) {
	if logger != nil {
		logger.Printf("Executor: Executing PHP script: %s", scriptPath)
		logger.Printf("Executor: Total PHP environment variables: %d", len(env))
	}

	// Determine document root from environment (must be the parent directory of the script)
	documentRoot := env["DOCUMENT_ROOT"]

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
	respBody := recorder.Body.String()

	// Check for PHP errors in the content
	phpErrorResult := php.CheckErrors(respBody)

	// Division by zero is a critical error that should always trigger the error handler
	divisionByZeroErr := strings.Contains(strings.ToLower(respBody), "division by zero")
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

	return []byte(respBody), exitCode, resultErr
}

// ensurePhpGlobals ensures that the PHP globals script is installed in the VFS
func (e *Executor) ensurePhpGlobals(v *vfs.VFS, globalsPath string) error {
	// Check if the globals file already exists
	if v.FileExists(globalsPath) {
		return nil // Already installed
	}

	// Get the PHP globals script from the php package
	provider := &php.StandardGlobalsProvider{}

	// Install the globals script
	return v.CreateVirtualFile(globalsPath, []byte(provider.GetScript()))
}

// checkPHPErrors examines response body for PHP error conditions
func (e *Executor) checkPHPErrors(body string) *php.ErrorResult {
	// Use the php package's CheckErrors function directly
	// instead of reimplementing error checking logic
	return php.CheckErrors(body)
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createPhpWrapper creates a wrapper PHP script that includes the actual target script
// It handles setting up the include path, working directory, and globals
func (e *Executor) createPhpWrapper(resolvedScriptPath, originalScriptPath string) (string, error) {
	logger := e.config.Logger
	vfs := e.vfs

	if logger != nil {
		logger.Printf("Creating PHP wrapper for script: %s (resolved to: %s)", originalScriptPath, resolvedScriptPath)
	}

	// 1. Generate a unique wrapper filename
	targetFilename := filepath.Base(resolvedScriptPath)
	wrapperFilename := fmt.Sprintf("_wrapper_%s_%s", calculateScriptPathHash(originalScriptPath), targetFilename)
	wrapperPath := filepath.Join(vfs.GetTempDir(), wrapperFilename)

	// 2. Copy the original PHP file to VFS temp directory
	targetPath := filepath.Join(vfs.GetTempDir(), targetFilename)

	// Read the original file content
	originalContent, err := os.ReadFile(resolvedScriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read PHP file: %w", err)
	}

	// Write it to the target location
	if err := os.WriteFile(targetPath, originalContent, 0644); err != nil {
		return "", fmt.Errorf("failed to prepare PHP file: %w", err)
	}

	// 3. Path to the globals file (defined in PHP globals provider)
	globalsFile := "/_frango_php_globals.php"
	globalsFilePath := filepath.Join(vfs.GetTempDir(), strings.TrimPrefix(globalsFile, "/"))

	// 4. Get the directory of the original script - critical for include path resolution
	// Remove any path parameters from the path to ensure it's a valid directory
	cleanOrigPath := e.sanitizePathForChdir(resolvedScriptPath)
	originalScriptDir := filepath.Dir(cleanOrigPath)

	// Ensure the directory exists before using it
	if _, err := os.Stat(originalScriptDir); os.IsNotExist(err) {
		// If directory doesn't exist, use a fallback that we know exists
		originalScriptDir = vfs.GetTempDir()
	}

	// 5. Create a wrapper that properly sets up include paths before including the target script
	wrapperContent := fmt.Sprintf(`<?php
// Auto-generated wrapper for %s
// This wrapper provides isolation between different script executions
// Script hash: %s

// Load common globals defined by frango
require_once '%s';

// Save the current working directory and include path
$original_dir = getcwd();
$original_include_path = get_include_path();

// Set up the working directory to the original script's directory
// This will make all relative includes in the target script work correctly
chdir('%s');
set_include_path(get_include_path() . PATH_SEPARATOR . '%s');

// Define helper functions for resolving paths relative to the original script
if (!function_exists('script_path')) {
    function script_path($path) {
        return '%s' . DIRECTORY_SEPARATOR . $path;
    }
}

// Include the target PHP script
try {
    include './%s';
} finally {
    // Restore original working directory and include path
    chdir($original_dir);
    set_include_path($original_include_path);
}
?>`, resolvedScriptPath, calculateScriptPathHash(originalScriptPath), globalsFilePath,
		originalScriptDir, originalScriptDir, originalScriptDir, targetFilename)

	// Create the wrapper script
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create wrapper: %w", err)
	}

	if logger != nil {
		logger.Printf("Created PHP wrapper at: %s", wrapperPath)
	}

	return wrapperPath, nil
}
