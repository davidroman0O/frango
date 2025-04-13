package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Add script directory to environment for proper includes
	scriptDir := filepath.Dir(resolvedPath)
	phpEnv["SCRIPT_DIR"] = scriptDir

	// Add logical filename for emulating __FILE__ and __DIR__
	phpEnv["LOGICAL_FILENAME"] = resolvedPath

	// Log environment variables in debug mode
	e.logEnvironmentVariables(phpEnv)

	// 7. Execute the PHP script directly (no wrapper)
	phpOutput, exitCode, execErr := executePHP(r.Context(), resolvedPath, phpEnv, r, logger)

	// 8. Check for PHP execution errors
	err = e.CheckPHPErrors(w, r, execErr, exitCode, phpOutput, scriptPath, resolvedPath)
	if err != nil {
		// Error was handled by CheckPHPErrors (either custom handler or default)
		duration := time.Since(startTime)
		if logger != nil {
			logger.Printf("Executor: Finished execution for script: %s (Duration: %s, With Error)", scriptPath, duration)
		}
		return // Stop processing, error response already sent
	}

	// 9. If no errors, write the captured PHP output to the original ResponseWriter
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

// executePHP executes a PHP script using the PHP CLI and returns the output, exit code, and any error result.
func executePHP(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) ([]byte, int, error) {
	if logger != nil {
		logger.Printf("Executor: Executing PHP script directly: %s", scriptPath)
		logger.Printf("Executor: Total PHP environment variables: %d", len(env))
	}

	// Create a temporary PHP file that includes the globals and executes the target script
	tempDir, err := os.MkdirTemp("", "php-exec-")
	if err != nil {
		return nil, 1, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Get the script directory for proper include resolution
	scriptDir := filepath.Dir(scriptPath)
	if dir, ok := env["SCRIPT_DIR"]; ok && dir != "" {
		scriptDir = dir
	}

	// Create the wrapper script
	wrapperPath := filepath.Join(tempDir, "wrapper.php")
	wrapperContent := `<?php
// Direct PHP executor wrapper
error_reporting(E_ALL);
ini_set('display_errors', 1);

// Set include path to include the script directory
set_include_path(get_include_path() . PATH_SEPARATOR . '` + scriptDir + `');

// Change to script directory if it exists
if (is_dir('` + scriptDir + `')) {
    chdir('` + scriptDir + `');
}

// Set up server variables
foreach ($_SERVER as $key => $value) {
    $_SERVER[$key] = $value;
}

// Setup path parameters
if (isset($_SERVER['_PATH']) && $_SERVER['_PATH'] !== '{}') {
    $pathParams = json_decode($_SERVER['_PATH'], true);
    if (is_array($pathParams)) {
        $_GET = array_merge($_GET, $pathParams);
    }
}

// Make additional environment data available
if (isset($_SERVER['_JSON']) && $_SERVER['_JSON'] !== '{}') {
    $_JSON = json_decode($_SERVER['_JSON'], true);
}

// Include the target script
include '` + scriptPath + `';
`

	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		return nil, 1, fmt.Errorf("failed to write wrapper script: %w", err)
	}

	// Create PHP command
	cmd := exec.CommandContext(ctx, "php", wrapperPath)

	// Set environment variables
	cmdEnv := os.Environ()
	for k, v := range env {
		cmdEnv = append(cmdEnv, k+"="+v)
	}
	cmd.Env = cmdEnv

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Combine stdout and stderr for the response
	output := stdout.Bytes()
	stderrOutput := stderr.Bytes()

	// Log stderr if not empty
	if len(stderrOutput) > 0 && logger != nil {
		logger.Printf("Executor: PHP stderr: %s", string(stderrOutput))
	}

	// Check for PHP errors in the content
	phpErrorResult := php.CheckErrors(string(output))

	// Look for division by zero errors
	divisionByZeroErr := strings.Contains(strings.ToLower(string(output))+strings.ToLower(string(stderrOutput)), "division by zero")
	if divisionByZeroErr && phpErrorResult == nil {
		phpErrorResult = &php.ErrorResult{
			Type:      php.ErrorFatal,
			Indicator: "Division by zero error detected",
			Context:   "PHP detected division by zero in script execution",
		}
	}

	// Log output in debug mode
	if logger != nil && len(output) > 0 && strings.Contains(os.Getenv("LOG_LEVEL"), "DEBUG") {
		if len(output) > 200 {
			logger.Printf("Executor: PHP output (truncated): %s...", string(output[:200]))
		} else {
			logger.Printf("Executor: PHP output: %s", string(output))
		}
	}

	// Return the combined error if we have PHP errors
	var resultErr error
	if err != nil || phpErrorResult != nil {
		resultErr = &PHPExecutionError{
			ExecErr:        err,
			PHPErrorResult: phpErrorResult,
		}
	}

	return output, exitCode, resultErr
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

// If SCRIPT_DIR is set, change to that directory to ensure includes work properly
if (isset($_SERVER['SCRIPT_DIR']) && $_SERVER['SCRIPT_DIR'] !== '') {
    if (is_dir($_SERVER['SCRIPT_DIR'])) {
        // Change working directory to script directory for proper include resolution
        chdir($_SERVER['SCRIPT_DIR']);
    }
}

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
    $pathParams = json_decode($_SERVER['_PATH'], true);
    if (is_array($pathParams)) {
        $_GET = array_merge($_GET, $pathParams);
    }
}

// Make additional environment data available to the script
if (isset($_SERVER['_JSON']) && $_SERVER['_JSON'] !== '{}') {
    $_JSON = json_decode($_SERVER['_JSON'], true);
}

// Load globals from the PHP package
` + provider.GetScript()

	// Install the globals script
	return v.CreateVirtualFile(globalsPath, []byte(globalsScript))
}

// handleExecutionError processes PHP execution errors and serves error responses
func (e *Executor) handleExecutionError(w http.ResponseWriter, r *http.Request, err error, statusCode int, scriptPath, originalScriptPath string) {
	logger := e.config.Logger
	if logger != nil {
		logger.Printf("Executor: Handling execution error for '%s': %v (HTTP %d)", scriptPath, err, statusCode)
	}

	// Extract any structured PHP error if available
	var phpErr *PHPExecutionError
	if errors.As(err, &phpErr) && phpErr.PHPErrorResult != nil {
		// Use the PHP error details for the error response
		errorDetails := phpErr.PHPErrorResult.Context
		errorType := phpErr.PHPErrorResult.Type

		// Custom handler logic starts here
		if e.config.ErrorHandlerPath != "" && errorType != php.ErrorNotice {
			// Define internal PHP error handling parameters
			errorHandlingParams := map[string]string{
				"error_type":      string(errorType),
				"error_message":   phpErr.PHPErrorResult.Indicator,
				"error_context":   errorDetails,
				"script_path":     scriptPath,
				"original_script": originalScriptPath,
			}

			// Call error handler PHP script
			errHandlerReq := r.Clone(r.Context())

			// Format original query for error handler
			errURL, _ := url.Parse(e.config.ErrorHandlerPath)
			q := errURL.Query()
			for k, v := range errorHandlingParams {
				q.Set(k, v)
			}
			errURL.RawQuery = q.Encode()

			// Set the error handler URL
			errHandlerReq.URL = errURL

			// Execute the error handler script
			e.Execute(e.vfs, e.config.ErrorHandlerPath, nil, w, errHandlerReq)
			return
		}

		// No custom handler or notice-level error - generate a default error response
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)

		// Get any PHP stderr output if available
		phpStderr := ""
		if phpErr.ExecErr != nil {
			// Try to extract stderr from the error if it's an exec.ExitError
			if exitErr, ok := phpErr.ExecErr.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
				phpStderr = string(exitErr.Stderr)
			}
		}

		// Format the error in a user-friendly way
		if e.config.DisplayErrors {
			// Display detailed error information if enabled
			errorHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>PHP Error</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .error-container { border: 1px solid #f44336; padding: 15px; border-radius: 4px; }
        .error-type { color: #f44336; font-weight: bold; }
        .error-message { margin: 10px 0; }
        .error-context { font-family: monospace; background: #f1f1f1; padding: 10px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="error-container">
        <h2 class="error-type">PHP %s Error</h2>
        <div class="error-message">%s</div>
        <pre class="error-context">%s</pre>
    </div>
</body>
</html>`, errorType, htmlEscape(phpErr.PHPErrorResult.Indicator),
				htmlEscape(phpStderr))
			w.Write([]byte(errorHTML))
		} else {
			// Simplified error for production
			w.Write([]byte("PHP Execution Error. Please check the server logs for details."))
		}
		return
	}

	// For direct executor errors, especially from PHP CLI
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)

	// For direct executor errors, check if the error is from PHP command execution
	if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
		// Direct output from PHP is most useful for debugging
		w.Write(exitErr.Stderr)
	} else {
		w.Write([]byte("PHP Execution Error: " + err.Error()))
	}
}

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	return html.EscapeString(s)
}
