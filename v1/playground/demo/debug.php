<?php
/**
 * Frango Debug Script
 * 
 * This script outputs detailed information about the PHP environment
 * including all superglobals, environment variables, and file paths.
 */
header("Content-Type: text/plain");

echo "=== FRANGO DEBUG INFORMATION ===\n\n";

echo "PHP Version: " . phpversion() . "\n";
echo "Server Software: " . ($_SERVER['SERVER_SOFTWARE'] ?? 'Unknown') . "\n";
echo "Request Method: " . ($_SERVER['REQUEST_METHOD'] ?? 'Unknown') . "\n";
echo "Content Type: " . ($_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'None') . "\n";
echo "Content Length: " . ($_SERVER['CONTENT_LENGTH'] ?? $_SERVER['HTTP_CONTENT_LENGTH'] ?? 'None') . "\n\n";

echo "=== REQUEST URL INFO ===\n";
echo "Request URI: " . ($_SERVER['REQUEST_URI'] ?? 'Unknown') . "\n";
echo "Script Name: " . ($_SERVER['SCRIPT_NAME'] ?? 'Unknown') . "\n";
echo "Script Filename: " . ($_SERVER['SCRIPT_FILENAME'] ?? 'Unknown') . "\n";
echo "Document Root: " . ($_SERVER['DOCUMENT_ROOT'] ?? 'Unknown') . "\n\n";

echo "=== PHP CONFIGURATION ===\n";
echo "Auto Prepend File: " . (ini_get('auto_prepend_file') ?: 'Not set') . "\n";
echo "Auto Prepend Env: " . (getenv('PHP_AUTO_PREPEND_FILE') ?: 'Not set') . "\n";
echo "Include Path: " . (ini_get('include_path') ?: 'Not set') . "\n";
echo "PHP_INCLUDE_PATH Env: " . (getenv('PHP_INCLUDE_PATH') ?: 'Not set') . "\n\n";

// Check if globals file exists
$globalsFile = getenv('PHP_AUTO_PREPEND_FILE');
if ($globalsFile) {
    echo "Globals file exists: " . (file_exists($globalsFile) ? "YES" : "NO") . "\n";
    if (file_exists($globalsFile)) {
        echo "Globals file size: " . filesize($globalsFile) . " bytes\n";
        echo "Globals file first 100 chars:\n";
        echo substr(file_get_contents($globalsFile), 0, 100) . "...\n\n";
    }
} else {
    echo "No globals file set in environment\n\n";
}

echo "=== SUPERGLOBALS STATUS ===\n";
echo '$_GET initialization: ' . (isset($_GET) && is_array($_GET) ? 'YES' : 'NO') . "\n";
echo '$_GET count: ' . (isset($_GET) ? count($_GET) : 'N/A') . "\n";
echo '$_POST initialization: ' . (isset($_POST) && is_array($_POST) ? 'YES' : 'NO') . "\n";
echo '$_POST count: ' . (isset($_POST) ? count($_POST) : 'N/A') . "\n";
echo '$_FORM initialization: ' . (isset($_FORM) && is_array($_FORM) ? 'YES' : 'NO') . "\n";
echo '$_FORM count: ' . (isset($_FORM) ? count($_FORM) : 'N/A') . "\n";
echo '$_PATH initialization: ' . (isset($_PATH) && is_array($_PATH) ? 'YES' : 'NO') . "\n";
echo '$_REQUEST initialization: ' . (isset($_REQUEST) && is_array($_REQUEST) ? 'YES' : 'NO') . "\n";
echo '$_JSON initialization: ' . (isset($_JSON) && is_array($_JSON) ? 'YES' : 'NO') . "\n\n";

echo "=== RAW POST DATA ===\n";
$rawInput = file_get_contents('php://input');
echo "php://input length: " . strlen($rawInput) . " bytes\n";
if (strlen($rawInput) > 0) {
    echo "Content: " . $rawInput . "\n\n";
} else {
    echo "Content: <empty>\n\n";
}

echo "=== FORM DATA ENVIRONMENT VARIABLES ===\n";
$foundFormVars = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        echo "$key: $value\n";
        $foundFormVars = true;
    }
}
if (!$foundFormVars) {
    echo "No PHP_FORM_* variables found\n";
}
echo "\n";

echo "=== QUERY DATA ENVIRONMENT VARIABLES ===\n";
$foundQueryVars = false;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        echo "$key: $value\n";
        $foundQueryVars = true;
    }
}
if (!$foundQueryVars) {
    echo "No PHP_QUERY_* variables found\n";
}
echo "\n";

echo "=== BACKTRACE ===\n";
$trace = debug_backtrace();
foreach ($trace as $i => $frame) {
    $file = $frame['file'] ?? 'unknown';
    $line = $frame['line'] ?? 'unknown';
    $function = $frame['function'] ?? 'unknown';
    echo "#$i $file($line): $function()\n";
}
echo "\n";

echo "=== INCLUDED FILES ===\n";
$includedFiles = get_included_files();
foreach ($includedFiles as $i => $file) {
    echo "#$i $file\n";
}
echo "\n";

echo "=== WRAPPER CHECK ===\n";
$isWrapper = false;
$currentScript = $_SERVER['SCRIPT_FILENAME'] ?? '';
if (strpos($currentScript, '_wrapper_') !== false) {
    echo "This script is being executed through a wrapper\n";
    $isWrapper = true;
} else {
    echo "This script is being executed directly (no wrapper)\n";
}
echo "Script filename: $currentScript\n\n";

echo "=== $_POST CONTENT ===\n";
var_export($_POST);
echo "\n\n";

echo "=== $_FORM CONTENT ===\n";
var_export($_FORM ?? []);
echo "\n\n";

echo "=== END DEBUG INFO ===\n";
?> 