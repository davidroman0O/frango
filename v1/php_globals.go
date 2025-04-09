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

// Initialize $_GET from pre-computed JSON
$_GET = json_decode($_SERVER['_GET'] ?? '{}', true);
$GLOBALS['_GET'] = $_GET;

// Initialize $_POST from pre-computed JSON
$_POST = json_decode($_SERVER['_POST'] ?? '{}', true);
$GLOBALS['_POST'] = $_POST;

// Initialize $_FORM from pre-computed JSON
$_FORM = json_decode($_SERVER['_FORM'] ?? '{}', true);
$GLOBALS['_FORM'] = $_FORM;

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

// ----- INITIALIZE REQUEST/SERVER DATA -----

// Initialize $_REQUEST (combination of $_GET, $_POST, $_COOKIE)
$_REQUEST = array_merge($_COOKIE ?? [], $_GET, $_POST);
$GLOBALS['_REQUEST'] = $_REQUEST;

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

// Unwrap arrays in $_GET for backward compatibility
$_QUERY = [];
foreach ($_GET as $key => $value) {
    if (is_array($value) && count($value) === 1) {
        $_QUERY[$key] = $value[0]; 
    } else {
        $_QUERY[$key] = $value;
    }
}

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
