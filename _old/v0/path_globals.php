<?php
// Initialize $_PATH superglobal for path parameters
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available
    $pathParamsJson = $_SERVER['FRANGO_PATH_PARAMS_JSON'] ?? '{}';
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