package frango

// Enhanced PHP globals initialization script with improvements for form handling
const phpGlobalsScript = `<?php
/**
 * Simplified PHP globals initialization
 * 
 * This file is automatically included in all PHP scripts executed by Frango.
 * It initializes PHP superglobals from pre-computed values passed from Go.
 */

// ----- INITIALIZE PRE-COMPUTED SUPERGLOBALS -----

// Initialize $_PATH from pre-computed JSON
$_PATH = json_decode($_SERVER['_PATH'] ?? '{}', true);
$GLOBALS['_PATH'] = $_PATH;

/**
 * IMPORTANT: Why normalize arrays in PHP instead of in Go?
 * 
 * Go represents HTTP query and form parameters as map[string][]string (always arrays)
 * This results in a different structure than native PHP, where:
 * - Single values like ?id=123 are strings: $_GET['id'] = "123"
 * - Multiple values like ?id[]=1&id[]=2 are arrays: $_GET['id'] = ["1", "2"]
 * 
 * By normalizing in PHP, we ensure compatibility with:
 * 1. Existing PHP code that checks variable types or expects string values
 * 2. PHP frameworks that assume standard $_GET/$_POST behavior
 * 3. Template code that uses echo $_GET['id'] directly (which would output "Array" otherwise)
 * 
 * While this could be done in Go, it's more straightforward to implement in PHP
 * with minimal performance impact.
 */

// Initialize $_GET from pre-computed JSON with array item normalization
$_GET_RAW = json_decode($_SERVER['_GET'] ?? '{}', true);
$_GET = [];
foreach ($_GET_RAW as $key => $value) {
    // Convert array with single item to string (PHP behavior)
    if (is_array($value) && count($value) === 1) {
        $_GET[$key] = $value[0];
    } else {
        $_GET[$key] = $value;
    }
}
$GLOBALS['_GET'] = $_GET;

// Initialize $_POST from pre-computed JSON with array item normalization
$_POST_RAW = json_decode($_SERVER['_POST'] ?? '{}', true);
$_POST = [];
foreach ($_POST_RAW as $key => $value) {
    // Convert array with single item to string (PHP behavior)
    if (is_array($value) && count($value) === 1) {
        $_POST[$key] = $value[0];
    } else {
        $_POST[$key] = $value;
    }
}
$GLOBALS['_POST'] = $_POST;

// Initialize $_FORM (alias for $_POST) 
$_FORM = $_POST;
$GLOBALS['_FORM'] = $_FORM;

// Initialize $_REQUEST (combination of GET, POST, COOKIE)
$_REQUEST = array_merge($_GET, $_POST, $_COOKIE ?? []);
$GLOBALS['_REQUEST'] = $_REQUEST;

// Initialize $_JSON from pre-computed JSON
$_JSON = json_decode($_SERVER['_JSON'] ?? '{}', true);
$GLOBALS['_JSON'] = $_JSON;

// Initialize $_PATH_SEGMENTS from pre-computed JSON
$_PATH_SEGMENTS = json_decode($_SERVER['_PATH_SEGMENTS'] ?? '[]', true);
$GLOBALS['_PATH_SEGMENTS'] = $_PATH_SEGMENTS;
$GLOBALS['_PATH_SEGMENT_COUNT'] = intval($_SERVER['_PATH_SEGMENT_COUNT'] ?? 0);

// Initialize $_FILES from pre-computed JSON (if available)
if (isset($_SERVER['_FILES'])) {
    $_FILES = json_decode($_SERVER['_FILES'], true);
    $GLOBALS['_FILES'] = $_FILES;
}

// Initialize $_TEMPLATE from pre-computed JSON (if available)
if (isset($_SERVER['_TEMPLATE'])) {
    $_TEMPLATE = json_decode($_SERVER['_TEMPLATE'], true);
    $GLOBALS['_TEMPLATE'] = $_TEMPLATE;
    
    // Also set each template variable in the global scope
    foreach ($_TEMPLATE as $key => $value) {
        $GLOBALS[$key] = $value;
    }
}

// Make route pattern available from both $_SERVER and $_TEMPLATE for compatibility
if (isset($_SERVER['PHP_VAR_ROUTE_PATTERN'])) {
    $_SERVER['ROUTE_PATTERN'] = $_SERVER['PHP_VAR_ROUTE_PATTERN'];
}

// ----- HELPER FUNCTIONS -----

// Helper function to get path parameters
if (!function_exists('path_param')) {
    /**
     * Gets a path parameter with optional default value
     * @param string $name Parameter name
     * @param mixed $default Default value if parameter doesn't exist
     * @return mixed Parameter value or default
     */
    function path_param($name, $default = null) {
        global $_PATH;
        return $_PATH[$name] ?? $default;
    }
}

if (!function_exists('has_path_param')) {
    /**
     * Checks if a path parameter exists
     * @param string $name Parameter name
     * @return bool True if parameter exists
     */
    function has_path_param($name) {
        global $_PATH;
        return isset($_PATH[$name]);
    }
}

if (!function_exists('path_segments')) {
    /**
     * Returns the URL path segments as an array
     * @return array URL path segments
     */
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

// ----- UTILITY VARIABLES -----

// Make common request data easily accessible
$_URL = $_SERVER['REQUEST_URI'] ?? '';
$_CURRENT_URL = $_SERVER['REQUEST_URI'] ?? '';

// Make query parameters easily accessible (same as $_GET)
$_QUERY = $_GET;

// Initialize cookies from HTTP headers if not already set
$_COOKIE = [];
if (isset($_SERVER['HTTP_COOKIE'])) {
    $pairs = explode(';', $_SERVER['HTTP_COOKIE']);
    foreach ($pairs as $pair) {
        $pair = trim($pair);
        if (empty($pair)) continue;
        
        list($name, $value) = explode('=', $pair, 2) + array('', '');
        $_COOKIE[trim($name)] = urldecode(trim($value));
    }
}
$GLOBALS['_COOKIE'] = $_COOKIE;

$GLOBALS['_URL'] = $_URL;
$GLOBALS['_CURRENT_URL'] = $_CURRENT_URL;
$GLOBALS['_QUERY'] = $_QUERY;
`

// InstallPHPGlobals installs the PHP globals script into a VFS
func InstallPHPGlobals(vfs *VFS) error {
	// Create the auto-prepend file in the VFS
	return vfs.CreateVirtualFile("/_frango_php_globals.php", []byte(phpGlobalsScript))
}

// UpdateVFS ensures a VFS has the PHP globals script installed
func UpdateVFS(vfs *VFS) error {
	// First check if the globals file already exists
	if vfs.FileExists("/_frango_php_globals.php") {
		return nil // Already installed
	}

	// Install the globals script
	return InstallPHPGlobals(vfs)
}
