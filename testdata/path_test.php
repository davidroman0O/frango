<?php
// Test script for $_PATH superglobal
header('Content-Type: text/plain');

// Define the path globals superglobals here directly to ensure they're available
// For test purposes we're including the initialization code directly
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available
    $pathParamsJson = $_SERVER['FRANGO_PATH_PARAMS_JSON'] ?? '{}';
    
    // Decode JSON parameters
    $decodedParams = json_decode($pathParamsJson, true);
    
    if (is_array($decodedParams)) {
        $_PATH = $decodedParams;
    }
    
    // Also add any FRANGO_PARAM_ variables from $_SERVER for backward compatibility
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'FRANGO_PARAM_') === 0) {
            $paramName = substr($key, strlen('FRANGO_PARAM_'));
            if (!isset($_PATH[$paramName])) {
                $_PATH[$paramName] = $value;
            }
        }
    }
}

// Initialize $_PATH_SEGMENTS superglobal for URL segments
if (!isset($_PATH_SEGMENTS)) {
    $_PATH_SEGMENTS = [];
    
    // Get segment count
    $segmentCount = intval($_SERVER['FRANGO_URL_SEGMENT_COUNT'] ?? 0);
    
    // Add segments to array
    for ($i = 0; $i < $segmentCount; $i++) {
        $segmentKey = "FRANGO_URL_SEGMENT_$i";
        if (isset($_SERVER[$segmentKey])) {
            $_PATH_SEGMENTS[] = $_SERVER[$segmentKey];
        }
    }
}

// Helper function to get path segments
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

// Test script starts here
echo "Path Parameters Test\n";
echo "===================\n\n";

// Show $_PATH contents
echo "Contents of \$_PATH:\n";
foreach ($_PATH as $key => $value) {
    echo "$key: $value\n";
}
echo "\n";

// Show $_PATH_SEGMENTS contents
echo "Contents of \$_PATH_SEGMENTS:\n";
foreach ($_PATH_SEGMENTS as $index => $segment) {
    echo "[$index]: $segment\n";
}
echo "\n";

// Show backward compatibility
echo "Backward Compatibility Test:\n";
echo "userId via \$_SERVER['FRANGO_PARAM_userId']: " . ($_SERVER['FRANGO_PARAM_userId'] ?? 'not set') . "\n";
echo "userId via \$_PATH['userId']: " . ($_PATH['userId'] ?? 'not set') . "\n";
echo "\n";

// Test helper functions
echo "Path Segments via function: ";
$segments = path_segments();
echo implode(', ', $segments);
echo "\n";

// Debug output for troubleshooting
echo "\nDebug Info:\n";
echo "FRANGO_PATH_PARAMS_JSON: " . ($_SERVER['FRANGO_PATH_PARAMS_JSON'] ?? 'not set') . "\n";
echo "PHP_AUTO_PREPEND_FILE: " . ($_SERVER['PHP_AUTO_PREPEND_FILE'] ?? 'not set') . "\n"; 