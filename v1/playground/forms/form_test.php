<?php
/**
 * Form Data Test Script
 * 
 * This script displays all form data received via POST and GET methods
 * It shows both the PHP_FORM_* environment variables and standard $_POST superglobal
 */

// Set the content type to plain text for easy debugging
header('Content-Type: text/plain');

echo "=== FORM DATA TEST ===\n\n";

// Show the request method
echo "REQUEST METHOD: " . $_SERVER['REQUEST_METHOD'] . "\n\n";

// Display $_POST superglobal contents
echo "=== \$_POST SUPERGLOBAL ===\n";
echo "Is set: " . (isset($_POST) ? "YES" : "NO") . "\n";
echo "Is array: " . (is_array($_POST) ? "YES" : "NO") . "\n";
echo "Count: " . (is_array($_POST) ? count($_POST) : "N/A") . "\n";
echo "Contents:\n";
var_export($_POST);
echo "\n\n";

// Display $_GET superglobal contents
echo "=== \$_GET SUPERGLOBAL ===\n";
echo "Is set: " . (isset($_GET) ? "YES" : "NO") . "\n";
echo "Is array: " . (is_array($_GET) ? "YES" : "NO") . "\n";
echo "Count: " . (is_array($_GET) ? count($_GET) : "N/A") . "\n";
echo "Contents:\n";
var_export($_GET);
echo "\n\n";

// Display $_REQUEST superglobal contents
echo "=== \$_REQUEST SUPERGLOBAL ===\n";
echo "Is set: " . (isset($_REQUEST) ? "YES" : "NO") . "\n";
echo "Is array: " . (is_array($_REQUEST) ? "YES" : "NO") . "\n";
echo "Count: " . (is_array($_REQUEST) ? count($_REQUEST) : "N/A") . "\n";
echo "Contents:\n";
var_export($_REQUEST);
echo "\n\n";

// Display direct PHP_FORM_ environment variables (raw source data)
echo "=== PHP_FORM_* ENVIRONMENT VARIABLES ===\n";
$formVars = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $formVars[$key] = $value;
    }
}
echo "Count: " . count($formVars) . "\n";
echo "Contents:\n";
var_export($formVars);
echo "\n\n";

// Display PHP_QUERY_ environment variables (raw source data)
echo "=== PHP_QUERY_* ENVIRONMENT VARIABLES ===\n";
$queryVars = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $queryVars[$key] = $value;
    }
}
echo "Count: " . count($queryVars) . "\n";
echo "Contents:\n";
var_export($queryVars);
echo "\n\n";

// Display information about php://input
echo "=== PHP INPUT STREAM ===\n";
$rawInput = file_get_contents('php://input');
echo "Raw input length: " . strlen($rawInput) . " bytes\n";
echo "Raw input contents:\n" . ($rawInput ?: "(empty)") . "\n\n";

// Display auto-prepend file information
echo "=== AUTO-PREPEND FILE ===\n";
$autoPrependFile = $_SERVER['PHP_AUTO_PREPEND_FILE'] ?? 'Not set';
echo "PHP_AUTO_PREPEND_FILE: " . $autoPrependFile . "\n";
echo "File exists: " . (file_exists($autoPrependFile) ? "YES" : "NO") . "\n\n";

// Display execution environment information
echo "=== EXECUTION ENVIRONMENT ===\n";
echo "SCRIPT_FILENAME: " . ($_SERVER['SCRIPT_FILENAME'] ?? 'Not set') . "\n";
echo "SCRIPT_NAME: " . ($_SERVER['SCRIPT_NAME'] ?? 'Not set') . "\n";
echo "PHP_SELF: " . ($_SERVER['PHP_SELF'] ?? 'Not set') . "\n";
echo "DOCUMENT_ROOT: " . ($_SERVER['DOCUMENT_ROOT'] ?? 'Not set') . "\n";
echo "REQUEST_URI: " . ($_SERVER['REQUEST_URI'] ?? 'Not set') . "\n";
echo "PHP Version: " . phpversion() . "\n";
echo "Current working directory: " . getcwd() . "\n\n";

echo "=== END OF TEST ==="; 