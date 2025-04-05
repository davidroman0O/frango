package frango

// Enhanced PHP globals initialization script with improvements for form handling
const phpGlobalsScript = `<?php
/**
 * Enhanced PHP globals initialization
 * 
 * This file is automatically included in all PHP scripts executed by Frango.
 * It ensures standard PHP superglobals like $_GET and $_POST are populated correctly.
 * Adapts to FrankenPHP's worker/request cycle by explicitly populating superglobals.
 */

// ----- INITIALIZE FORM DATA ($_GET and $_POST) -----

// Initialize $_GET from PHP_QUERY_ variables
$_GET = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $paramName = substr($key, 10); // Remove 'PHP_QUERY_' prefix
        $_GET[$paramName] = $value;
    }
}
// Make sure $_GET is globally accessible
$GLOBALS['_GET'] = $_GET;

// Initialize $_POST from PHP_FORM_ variables
$_POST = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $paramName = substr($key, 9); // Fix: Changed from 10 to 9 to correctly remove 'PHP_FORM_' prefix
        $_POST[$paramName] = $value;
    }
}
// Make sure $_POST is globally accessible
$GLOBALS['_POST'] = $_POST;

// Initialize $_REQUEST (combination of $_GET, $_POST, $_COOKIE)
$_REQUEST = array_merge($_COOKIE ?? [], $_GET, $_POST);
$GLOBALS['_REQUEST'] = $_REQUEST;

// Create $_FORM (convenience superglobal that contains form data regardless of method)
$_FORM = [];
// Directly initialize $_FORM from PHP_FORM_ variables
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $paramName = substr($key, 9); // Fix: Also update here from 10 to 9
        $_FORM[$paramName] = $value;
    }
}
// Make sure $_FORM is globally accessible
$GLOBALS['_FORM'] = $_FORM;

// ----- INITIALIZE PATH DATA -----

// Initialize $_PATH superglobal for path parameters
$_PATH = [];

// Load from JSON if available (preferred method)
$pathParamsJson = $_SERVER['PHP_PATH_PARAMS'] ?? '{}';
$decodedParams = json_decode($pathParamsJson, true);
if (is_array($decodedParams)) {
    $_PATH = $decodedParams;
}

// Also add any PHP_PATH_PARAM_ variables from $_SERVER for backward compatibility
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_PATH_PARAM_') === 0) {
        $paramName = substr($key, strlen('PHP_PATH_PARAM_'));
        if (!isset($_PATH[$paramName])) {
            $_PATH[$paramName] = $value;
        }
    }
}

// Make sure $_PATH is globally accessible
$GLOBALS['_PATH'] = $_PATH;

// Initialize $_PATH_SEGMENTS superglobal for URL segments
$_PATH_SEGMENTS = [];

// Get segment count
$segmentCount = intval($_SERVER['PHP_PATH_SEGMENT_COUNT'] ?? 0);

// Add segments to array
for ($i = 0; $i < $segmentCount; $i++) {
    $segmentKey = "PHP_PATH_SEGMENT_$i";
    if (isset($_SERVER[$segmentKey])) {
        $_PATH_SEGMENTS[] = $_SERVER[$segmentKey];
    }
}

// Make sure $_PATH_SEGMENTS is globally accessible
$GLOBALS['_PATH_SEGMENTS'] = $_PATH_SEGMENTS;
$GLOBALS['_PATH_SEGMENT_COUNT'] = $segmentCount;

// ----- INITIALIZE JSON DATA -----

// Initialize $_JSON superglobal for parsed JSON request body
$_JSON = [];

// Load from JSON if available
$jsonBody = $_SERVER['PHP_JSON'] ?? '{}';
$decodedJson = json_decode($jsonBody, true);
if (is_array($decodedJson)) {
    $_JSON = $decodedJson;
}

// Make sure $_JSON is globally accessible
$GLOBALS['_JSON'] = $_JSON;

// ----- INITIALIZE FILE UPLOADS -----

// Initialize $_FILES from PHP_FILE_ variables if they exist
$_FILES = [];
$fileFields = [];

// Collect all PHP_FILE_ variables to identify file upload fields
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FILE_') === 0) {
        $parts = explode('_', $key, 3);
        if (count($parts) >= 3) {
            $fieldName = $parts[2];
            // Structure will be filled later
            if (!isset($fileFields[$fieldName])) {
                $fileFields[$fieldName] = [
                    'name' => '',
                    'type' => '',
                    'tmp_name' => '',
                    'error' => UPLOAD_ERR_NO_FILE,
                    'size' => 0
                ];
            }
        }
    }
}

// If we found any file fields, try to populate $_FILES
if (!empty($fileFields)) {
    $_FILES = $fileFields;
    $GLOBALS['_FILES'] = $_FILES;
}

// ----- HELPER FUNCTIONS -----

// Initialize common helper functions
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

// Add getallheaders() if it doesn't exist
if (!function_exists('getallheaders')) {
    function getallheaders() {
        $headers = [];
        foreach ($_SERVER as $key => $value) {
            if (strpos($key, 'PHP_HEADER_') === 0) {
                $headerName = substr($key, 11); // Remove 'PHP_HEADER_' prefix
                $headers[$headerName] = $value;
            }
        }
        return $headers;
    }
}

// ----- TEMPLATE VARIABLES -----

// Initialize template variables from PHP_VAR_ environment variables
// Makes template variables directly accessible as PHP variables
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_VAR_') === 0) {
        $varName = substr($key, strlen('PHP_VAR_'));
        $varValue = json_decode($value, true);
        
        // Set variable in global scope for direct access
        $GLOBALS[$varName] = $varValue;
        
        // Also make available as a superglobal array
        if (!isset($GLOBALS['_TEMPLATE'])) {
            $GLOBALS['_TEMPLATE'] = [];
        }
        $GLOBALS['_TEMPLATE'][$varName] = $varValue;
    }
}

// ----- UTILITY VARIABLES -----

// Make common request data easily accessible
$_URL = $_SERVER['PHP_PATH'] ?? $_SERVER['REQUEST_URI'] ?? '';
$_CURRENT_URL = $_SERVER['REQUEST_URI'] ?? '';
$_QUERY = $_GET;

$GLOBALS['_URL'] = $_URL;
$GLOBALS['_CURRENT_URL'] = $_CURRENT_URL;
$GLOBALS['_QUERY'] = $_QUERY;

// Create a compatibility layer for php://input if it's empty
// FrankenPHP may reset it between requests
if (empty(file_get_contents('php://input'))) {
    // Try to recreate input stream for POST requests with form data
    if ($_SERVER['REQUEST_METHOD'] === 'POST' && !empty($_POST)) {
        // Create a stream wrapper to provide data for php://input
        if (!in_array('frango', stream_get_wrappers())) {
            class FrangoInputStreamWrapper {
                private $position = 0;
                private $data;
                
                public function stream_open($path, $mode, $options, &$opened_path) {
                    global $_POST;
                    // For URL-encoded form data
                    $this->data = http_build_query($_POST);
                    return true;
                }
                
                public function stream_read($count) {
                    $ret = substr($this->data, $this->position, $count);
                    $this->position += strlen($ret);
                    return $ret;
                }
                
                public function stream_eof() {
                    return $this->position >= strlen($this->data);
                }
                
                public function stream_stat() {
                    return [
                        'size' => strlen($this->data),
                    ];
                }
                
                public function stream_tell() {
                    return $this->position;
                }
                
                public function stream_seek($offset, $whence) {
                    switch ($whence) {
                        case SEEK_SET:
                            if ($offset < strlen($this->data) && $offset >= 0) {
                                $this->position = $offset;
                                return true;
                            }
                            return false;
                        case SEEK_CUR:
                            if ($offset >= 0) {
                                $this->position += $offset;
                                return true;
                            }
                            return false;
                        case SEEK_END:
                            if (strlen($this->data) + $offset >= 0) {
                                $this->position = strlen($this->data) + $offset;
                                return true;
                            }
                            return false;
                    }
                    return false;
                }
            }
            
            // Register our stream wrapper
            stream_wrapper_register('frango', 'FrangoInputStreamWrapper');
        }
    }
}
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
