<?php
/**
 * Form Debug Script
 * 
 * Specifically checks the status of $_FORM initialization
 */
header("Content-Type: text/plain");

echo "=== FORM INITIALIZATION DEBUG ===\n\n";

// Check if request has form data
echo "Request Method: " . $_SERVER['REQUEST_METHOD'] . "\n\n";

// Check PHP_FORM_* variables
$form_vars = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $paramName = substr($key, 10); // Remove prefix
        $form_vars[$paramName] = $value;
    }
}

echo "PHP_FORM_* Variables (Environment):\n";
if (!empty($form_vars)) {
    foreach ($form_vars as $key => $value) {
        echo "$key: $value\n";
    }
} else {
    echo "<none>\n";
}
echo "\n";

echo "POST Variables (\$_POST):\n";
if (!empty($_POST)) {
    foreach ($_POST as $key => $value) {
        echo "$key: $value\n";
    }
} else {
    echo "<none>\n";
}
echo "\n";

echo "FORM Variables (\$_FORM):\n";
if (isset($_FORM) && !empty($_FORM)) {
    foreach ($_FORM as $key => $value) {
        echo "$key: $value\n";
    }
} else {
    echo "<none or not initialized>\n";
}
echo "\n";

// Check for initialization problems
echo "Initialization Analysis:\n";
if (!empty($form_vars) && empty($_FORM)) {
    echo "ERROR: PHP_FORM_* variables exist but $_FORM is empty\n";
    echo "This indicates the globals initialization script is not running properly.\n";
} elseif (!empty($_POST) && empty($_FORM)) {
    echo "ERROR: $_POST is populated but $_FORM is empty\n";
    echo "This indicates the globals initialization is setting $_FORM incorrectly.\n";
} elseif (!empty($_FORM) && empty($_POST) && !empty($form_vars)) {
    echo "UNUSUAL: $_FORM is populated but $_POST is empty\n";
    echo "This might indicate a problem with POST variable initialization.\n";
} elseif (!empty($_FORM) && !empty($_POST)) {
    $differences = [];
    foreach ($_FORM as $key => $value) {
        if (!isset($_POST[$key]) || $_POST[$key] !== $value) {
            $differences[] = "Key '$key' differs: FORM='$value', POST='" . ($_POST[$key] ?? "not set") . "'";
        }
    }
    foreach ($_POST as $key => $value) {
        if (!isset($_FORM[$key])) {
            $differences[] = "Key '$key' in POST but not in FORM";
        }
    }
    
    if (!empty($differences)) {
        echo "WARNING: $_FORM and $_POST have different values:\n";
        foreach ($differences as $diff) {
            echo "- $diff\n";
        }
        echo "\n";
    } else {
        echo "SUCCESS: $_FORM and $_POST are both populated and have the same values.\n";
    }
} elseif (empty($form_vars) && empty($_POST) && empty($_FORM)) {
    echo "No form data found in request. Try submitting a form to test.\n";
} else {
    echo "Unusual state detected. Form initialization may be irregular.\n";
}

// Check globals file
$auto_prepend = ini_get('auto_prepend_file') ?: getenv('PHP_AUTO_PREPEND_FILE') ?: 'Not set';
echo "\nAuto-prepend file: $auto_prepend\n";
if ($auto_prepend !== 'Not set') {
    echo "File exists: " . (file_exists($auto_prepend) ? "YES" : "NO") . "\n";
    if (file_exists($auto_prepend)) {
        $content = file_get_contents($auto_prepend);
        echo "File size: " . strlen($content) . " bytes\n";
        echo "File excerpt:\n" . substr($content, 0, 200) . "...\n";
    }
}

// Check wrapper detection
$script_name = $_SERVER['SCRIPT_FILENAME'] ?? '';
echo "\nScript filename: $script_name\n";
if (strpos($script_name, '_wrapper_') !== false) {
    echo "Running through wrapper script (correct)\n";
} else {
    echo "WARNING: Not running through wrapper script\n";
}

echo "\n=== END DEBUG INFO ===\n"; 