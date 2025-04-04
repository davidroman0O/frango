<?php
header('Content-Type: text/plain');
echo "PHP Environment Variables Test\n";
echo "============================\n\n";

// Check for path parameter variables
echo "PATH PARAMETERS:\n";
$pathParamsFound = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_PATH_PARAM_') === 0) {
        $pathParamsFound = true;
        echo "  $key = $value\n";
    }
}
if (!$pathParamsFound) {
    echo "  No PHP_PATH_PARAM_* variables found\n";
}

// Check for path segments
echo "\nPATH SEGMENTS:\n";
$segmentsFound = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_PATH_SEGMENT_') === 0) {
        $segmentsFound = true;
        echo "  $key = $value\n";
    }
}
if (!$segmentsFound) {
    echo "  No PHP_PATH_SEGMENT_* variables found\n";
}

// Check for query parameters
echo "\nQUERY PARAMETERS:\n";
$queryFound = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $queryFound = true;
        echo "  $key = $value\n";
    }
}
if (!$queryFound) {
    echo "  No PHP_QUERY_* variables found\n";
}

// Check for PHP_PATH_PARAMS
echo "\nPATH PARAMS JSON:\n";
if (isset($_SERVER['PHP_PATH_PARAMS'])) {
    echo "  PHP_PATH_PARAMS = " . $_SERVER['PHP_PATH_PARAMS'] . "\n";
} else {
    echo "  PHP_PATH_PARAMS not found\n";
}

// Check for $_PATH superglobal
echo "\n\$_PATH SUPERGLOBAL:\n";
if (isset($_PATH) && is_array($_PATH)) {
    if (empty($_PATH)) {
        echo "  $_PATH is empty\n";
    } else {
        foreach ($_PATH as $key => $value) {
            echo "  $_PATH[$key] = $value\n";
        }
    }
} else {
    echo "  $_PATH is not defined or not an array\n";
}

// Check for $_PATH_SEGMENTS superglobal
echo "\n\$_PATH_SEGMENTS SUPERGLOBAL:\n";
if (isset($_PATH_SEGMENTS) && is_array($_PATH_SEGMENTS)) {
    if (empty($_PATH_SEGMENTS)) {
        echo "  $_PATH_SEGMENTS is empty\n";
    } else {
        foreach ($_PATH_SEGMENTS as $index => $value) {
            echo "  $_PATH_SEGMENTS[$index] = $value\n";
        }
    }
} else {
    echo "  $_PATH_SEGMENTS is not defined or not an array\n";
}
?>