package php

import (
	"github.com/davidroman0O/frango/v2/vfs"
)

// GlobalsProvider is an interface for providing PHP globals script content
type GlobalsProvider interface {
	// GetScript returns the PHP script used to initialize PHP globals
	GetScript() string

	// GetPath returns the virtual path for the PHP globals script
	GetPath() string
}

// StandardGlobalsProvider is the default implementation of GlobalsProvider
type StandardGlobalsProvider struct{}

// GetScript returns the PHP globals script
func (p *StandardGlobalsProvider) GetScript() string {
	return `<?php
// Frango PHP Globals Script
// This script initializes PHP superglobals and provides helpers for accessing request data.

//=====================================
// INITIALIZE PHP SUPERGLOBALS
//=====================================

// Initialize path parameters from pre-computed JSON 
$_PATH = json_decode($_SERVER['_PATH'] ?? '{}', true);
unset($_SERVER['_PATH']); // Clean up server var to match standard PHP

// Initialize $_GET from pre-computed JSON
$_GET = json_decode($_SERVER['_GET'] ?? '{}', true);
unset($_SERVER['_GET']); // Clean up server var to match standard PHP

// CRITICAL: In FrankenPHP, path parameters should be merged into $_GET
// Path parameters take precedence over query string parameters with the same name
if (!empty($_PATH) && is_array($_PATH)) {
    $_GET = array_merge($_GET, $_PATH); // Merge path params into $_GET
}

// Initialize $_POST from pre-computed JSON
$_POST = json_decode($_SERVER['_POST'] ?? '{}', true);
unset($_SERVER['_POST']); // Clean up server var

// Initialize $_FILES from pre-computed JSON (ensure correct structure for file uploads)
$_FILES = json_decode($_SERVER['_FILES'] ?? '{}', true);
unset($_SERVER['_FILES']); // Clean up server var

// Initialize $_COOKIE from the HTTP_COOKIE header
if (isset($_SERVER['HTTP_COOKIE'])) {
    $cookies = explode('; ', $_SERVER['HTTP_COOKIE']);
    foreach ($cookies as $cookie) {
        if (strpos($cookie, '=') !== false) {
            list($name, $value) = explode('=', $cookie, 2);
            $_COOKIE[urldecode($name)] = urldecode($value);
        }
    }
} else {
    $_COOKIE = array();
}

// Set $_REQUEST according to PHP standards (merge GET, POST, COOKIE)
$_REQUEST = array_merge($_GET, $_POST, $_COOKIE);

// Initialize template variables from PHP_VAR_* variables
$_TEMPLATE = json_decode($_SERVER['_TEMPLATE'] ?? '{}', true);
unset($_SERVER['_TEMPLATE']); // Clean up server var

// Initialize JSON body if provided (non-standard but useful)
$_JSON = json_decode($_SERVER['_JSON'] ?? '{}', true);
unset($_SERVER['_JSON']); // Clean up server var

// Initialize path segments for URL manipulation
$_PATH_SEGMENTS = json_decode($_SERVER['_PATH_SEGMENTS'] ?? '[]', true);
unset($_SERVER['_PATH_SEGMENTS']); // Clean up server var

// Create additional custom superglobals that are expected by tests
$_FORM = $_POST; // $_FORM is an alias for $_POST for convenience
$_URL = $_SERVER['REQUEST_URI']; // Full URL
$_CURRENT_URL = $_SERVER['REQUEST_URI'] ?? ''; // May be different in some setups
$_QUERY = $_GET; // $_QUERY is an alias for $_GET for convenience

// Add additional expected servers
if (!isset($_SERVER['DOCUMENT_ROOT'])) {
    $_SERVER['DOCUMENT_ROOT'] = dirname($_SERVER['SCRIPT_FILENAME']);
}

if (!isset($_SERVER['PHP_SELF']) && isset($_SERVER['SCRIPT_NAME'])) {
    $_SERVER['PHP_SELF'] = $_SERVER['SCRIPT_NAME'];
}

// Extract PHP_VAR_* variables and create them as global variables
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_VAR_') === 0) {
        $var_name = substr($key, 8); // Remove PHP_VAR_ prefix
        $var_value = json_decode($value, true);
        
        // Create the variable in the global scope
        $GLOBALS[$var_name] = $var_value;
        
        // Remove from $_SERVER to avoid namespace pollution
        unset($_SERVER[$key]);
    }
}

//=====================================
// ERROR HANDLING
//=====================================

// Register enhanced error handling for better division by zero and other error detection
function frangoErrorHandler($errno, $errstr, $errfile, $errline) {
    // Create detailed context for error processing
    $context = json_encode([
        'error' => $errstr,
        'type' => $errno,
        'file' => $errfile,
        'line' => $errline,
        'script' => $_SERVER['SCRIPT_FILENAME'] ?? ''
    ]);

    // Store error details in PHP_ERROR_* environment variables
    // Important: Using putenv instead of $_ENV to avoid interfering with superglobals
    putenv("PHP_LAST_ERROR=$errstr");
    putenv("PHP_ERROR_TYPE=$errno");
    putenv("PHP_ERROR_FILE=$errfile");
    putenv("PHP_ERROR_LINE=$errline");
    putenv("PHP_ERROR_SCRIPT=" . ($_SERVER['SCRIPT_FILENAME'] ?? ''));
    putenv("PHP_ERROR_CONTEXT=$context");

    // Handle specific error types
    // Division by zero is specifically checked per test requirements
    if (strpos($errstr, 'division by zero') !== false) {
        putenv("PHP_LAST_ERROR=Division by zero.");
        $specificContext = json_encode([
            'error' => 'Division by zero.',
            'type' => $errno,
            'file' => $errfile,
            'line' => $errline,
            'script' => $_SERVER['SCRIPT_FILENAME'] ?? ''
        ]);
        putenv("PHP_ERROR_CONTEXT=$specificContext");
    }

    // Return false to let PHP handle the error as well
    return false;
}

// Set the custom error handler
set_error_handler('frangoErrorHandler');

// Register a shutdown function to catch fatal errors
register_shutdown_function(function() {
    $error = error_get_last();
    if ($error !== null && in_array($error['type'], [E_ERROR, E_PARSE, E_CORE_ERROR, E_COMPILE_ERROR])) {
        // Set environment variables with error details
        putenv("PHP_LAST_ERROR={$error['message']}");
        putenv("PHP_ERROR_TYPE={$error['type']}");
        putenv("PHP_ERROR_FILE={$error['file']}");
        putenv("PHP_ERROR_LINE={$error['line']}");
        putenv("PHP_ERROR_SCRIPT=" . ($_SERVER['SCRIPT_FILENAME'] ?? ''));
        
        // Create detailed context for error processing
        $context = json_encode([
            'error' => $error['message'],
            'type' => $error['type'],
            'file' => $error['file'],
            'line' => $error['line'],
            'script' => $_SERVER['SCRIPT_FILENAME'] ?? ''
        ]);
        putenv("PHP_ERROR_CONTEXT=$context");
    }
});

//=====================================
// HELPER FUNCTIONS
//=====================================

// Store global helpers in $_FRANGO array
$_FRANGO = array(
    'path_params' => $_PATH,
    'path_segments' => $_PATH_SEGMENTS,
    'json_body' => $_JSON,
    'template' => $_TEMPLATE,
);

/**
 * Check if a path parameter exists
 * 
 * @param string $name The parameter name
 * @return bool True if the parameter exists
 */
function has_path_param($name) {
    global $_PATH;
    return isset($_PATH[$name]);
}

/**
 * Get a path parameter value
 * 
 * @param string $name The parameter name
 * @param mixed $default Default value if parameter doesn't exist
 * @return mixed The parameter value or default if not found
 */
function path_param($name, $default = null) {
    global $_PATH;
    return $_PATH[$name] ?? $default;
}

/**
 * Get all path parameters
 * 
 * @return array All path parameters
 */
function path_params() {
    global $_PATH;
    return $_PATH;
}

/**
 * Get a path segment by index
 * 
 * @param int $index The segment index (0-based)
 * @param mixed $default Default value if segment doesn't exist
 * @return string The path segment or default if not found
 */
function path_segment($index, $default = '') {
    global $_PATH_SEGMENTS;
    return $_PATH_SEGMENTS[$index] ?? $default;
}

/**
 * Get a JSON body value
 * 
 * @param string $key The JSON key
 * @param mixed $default Default value if key doesn't exist
 * @return mixed The value or default if not found
 */
function json_body($key = null, $default = null) {
    global $_JSON;
    if ($key === null) {
        return $_JSON;
    }
    return $_JSON[$key] ?? $default;
}

/**
 * Get a template variable
 * 
 * @param string $key The template variable key
 * @param mixed $default Default value if key doesn't exist
 * @return mixed The value or default if not found
 */
function template_var($key, $default = null) {
    global $_TEMPLATE;
    return $_TEMPLATE[$key] ?? $default;
}
`
}

// GetPath returns the standard path for the PHP globals script
func (p *StandardGlobalsProvider) GetPath() string {
	return "/_frango_php_globals.php"
}

// InstallGlobals installs the PHP globals script into a VFS
func InstallGlobals(v *vfs.VFS) error {
	provider := &StandardGlobalsProvider{}
	// Create the auto-prepend file in the VFS
	return v.CreateVirtualFile(provider.GetPath(), []byte(provider.GetScript()))
}

// UpdateVFS ensures a VFS has the PHP globals script installed
func UpdateVFS(v *vfs.VFS) error {
	provider := &StandardGlobalsProvider{}
	globalsPath := provider.GetPath()

	// First check if the globals file already exists
	if v.FileExists(globalsPath) {
		return nil // Already installed
	}

	// Install the globals script
	return v.CreateVirtualFile(globalsPath, []byte(provider.GetScript()))
}

// phpGlobalsScript is the PHP script that initializes all PHP globals.
// This script is included by the PHP wrapper script to set up the PHP environment.
const phpGlobalsScript = `<?php
/**
 * Frango PHP Globals - Sets up PHP globals and helper functions for Go integration
 * This file is automatically included by the Frango PHP middleware.
 */

//=====================================================
// 1. GUARD AGAINST DIRECT ACCESS 
//=====================================================

// Prevent direct access to this file
if (count(get_included_files()) <= 1) {
    header("HTTP/1.1 403 Forbidden");
    echo "<h1>403 Forbidden</h1><p>Direct access to this file is not allowed.</p>";
    exit;
}

//=====================================================
// 2. SET UP GLOBAL VARIABLES
//=====================================================

// Initialize all PHP standard superglobals
$_GET = json_decode(getenv('_GET') ?: '{}', true);
$_POST = json_decode(getenv('_POST') ?: '{}', true);
$_FILES = json_decode(getenv('_FILES') ?: '{}', true);
$_SERVER = array_merge($_SERVER, [
    'REQUEST_METHOD' => getenv('REQUEST_METHOD'),
    'REQUEST_URI' => getenv('REQUEST_URI'),
    'QUERY_STRING' => getenv('QUERY_STRING'),
    'HTTP_HOST' => getenv('HTTP_HOST'),
    'REMOTE_ADDR' => getenv('REMOTE_ADDR'),
    'DOCUMENT_ROOT' => getenv('DOCUMENT_ROOT'),
    'SCRIPT_FILENAME' => getenv('SCRIPT_FILENAME'),
    'SCRIPT_NAME' => getenv('SCRIPT_NAME'),
    'PHP_SELF' => getenv('PHP_SELF'),
]);

// Create custom globals from Frango
$_PATH = json_decode(getenv('_PATH') ?: '{}', true);
$_PATH_SEGMENTS = json_decode(getenv('_PATH_SEGMENTS') ?: '[]', true);
$_FORM = json_decode(getenv('_FORM') ?: '{}', true);
$_JSON = json_decode(getenv('_JSON') ?: '{}', true);
$_TEMPLATE = json_decode(getenv('_TEMPLATE') ?: '{}', true);

// Create convenient utility variables
$_URL = $_SERVER['REQUEST_URI'];
$_CURRENT_PATH = rtrim(strtok($_URL, '?'), '/');
$_QUERY = $_GET;

//=====================================================
// 3. HELPER FUNCTIONS
//=====================================================

/**
 * Gets a path parameter by name
 * 
 * @param string $name The name of the path parameter
 * @param mixed $default The default value if parameter doesn't exist
 * @return mixed The parameter value or default
 */
function path_param($name, $default = null) {
    global $_PATH;
    return $_PATH[$name] ?? $default;
}

/**
 * Checks if a path parameter exists
 * 
 * @param string $name The name of the path parameter
 * @return bool True if parameter exists
 */
function has_path_param($name) {
    global $_PATH;
    return isset($_PATH[$name]);
}

/**
 * Gets a specific path segment by index (0-based)
 * 
 * @param int $index The index of the path segment
 * @param mixed $default The default value if segment doesn't exist
 * @return mixed The segment value or default
 */
function path_segment($index, $default = null) {
    global $_PATH_SEGMENTS;
    return $_PATH_SEGMENTS[$index] ?? $default;
}

/**
 * Gets all path segments
 * 
 * @return array The path segments
 */
function path_segments() {
    global $_PATH_SEGMENTS;
    return $_PATH_SEGMENTS;
}

/**
 * Gets a query parameter
 * 
 * @param string $name The name of the query parameter
 * @param mixed $default The default value if parameter doesn't exist
 * @return mixed The parameter value or default
 */
function query_param($name, $default = null) {
    global $_GET;
    return $_GET[$name] ?? $default;
}

/**
 * Gets a template variable
 * 
 * @param string $name The name of the template variable
 * @param mixed $default The default value if variable doesn't exist
 * @return mixed The variable value or default
 */
function template_var($name, $default = null) {
    global $_TEMPLATE;
    return $_TEMPLATE[$name] ?? $default;
}

//=====================================================
// 4. ERROR HANDLING
//=====================================================

// Set default error reporting based on PHP INI settings
// These can be overridden in the script itself
if (getenv('PHP_INI_DISPLAY_ERRORS') === '1') {
    ini_set('display_errors', '1');
    error_reporting(E_ALL);
}

// Override PHP's default error handler to capture errors
// This helps prevent blank pages when errors occur
set_error_handler(function($errno, $errstr, $errfile, $errline) {
    if (!(error_reporting() & $errno)) {
        // This error code is not included in error_reporting
        return false;
    }
    
    // Log error if logging is enabled
    if (ini_get('log_errors')) {
        error_log("PHP Error [$errno]: $errstr in $errfile on line $errline");
    }
    
    // Continue with PHP's internal error handler
    return false;
}, E_ALL);

// Custom exception handler to ensure exceptions are displayed properly
set_exception_handler(function($exception) {
    // Format the error message
    $message = "Uncaught Exception: " . $exception->getMessage();
    $file = $exception->getFile();
    $line = $exception->getLine();
    $trace = $exception->getTraceAsString();
    
    // Output error based on display_errors setting
    if (ini_get('display_errors')) {
        http_response_code(500);
        echo "<h1>PHP Exception</h1>";
        echo "<p><strong>$message</strong></p>";
        echo "<p>in <strong>$file</strong> on line <strong>$line</strong></p>";
        echo "<pre>$trace</pre>";
    } else {
        http_response_code(500);
        echo "An error occurred while processing your request.";
    }
    
    // Log error if logging is enabled
    if (ini_get('log_errors')) {
        error_log("PHP Exception: $message in $file on line $line\n$trace");
    }
    
    exit(1);
});

// Register shutdown function to catch fatal errors
register_shutdown_function(function() {
    $error = error_get_last();
    if ($error !== null && in_array($error['type'], [E_ERROR, E_PARSE, E_CORE_ERROR, E_COMPILE_ERROR])) {
        // Format the error message
        $message = $error['message'];
        $file = $error['file'];
        $line = $error['line'];
        
        // Output error based on display_errors setting
        if (ini_get('display_errors')) {
            http_response_code(500);
            echo "<h1>PHP Fatal Error</h1>";
            echo "<p><strong>$message</strong></p>";
            echo "<p>in <strong>$file</strong> on line <strong>$line</strong></p>";
        } else {
            http_response_code(500);
            echo "An error occurred while processing your request.";
        }
        
        // Log error if logging is enabled
        if (ini_get('log_errors')) {
            error_log("PHP Fatal Error: $message in $file on line $line");
        }
    }
});
`
