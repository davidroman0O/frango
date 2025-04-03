<?php
// Test script for $_PATH superglobal
header('Content-Type: text/plain');

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