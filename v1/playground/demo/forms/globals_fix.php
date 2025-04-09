<?php
/**
 * Manual Superglobals Initialization Fix
 * 
 * This file manually initializes PHP superglobals like $_POST, $_GET, and $_FORM
 * from their corresponding environment variables.
 * 
 * Include this at the top of PHP scripts where form data is needed.
 */

// Initialize $_GET from PHP_QUERY_ variables (if not already set)
if (empty($_GET)) {
    $_GET = [];
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'PHP_QUERY_') === 0) {
            $paramName = substr($key, 10); // Remove 'PHP_QUERY_' prefix
            $_GET[$paramName] = $value;
        }
    }
    // Make sure $_GET is globally accessible
    $GLOBALS['_GET'] = $_GET;
}

// Initialize $_POST from PHP_FORM_ variables (if not already set)
if (empty($_POST)) {
    $_POST = [];
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'PHP_FORM_') === 0) {
            $paramName = substr($key, 10); // Remove 'PHP_FORM_' prefix
            $_POST[$paramName] = $value;
        }
    }
    // Make sure $_POST is globally accessible
    $GLOBALS['_POST'] = $_POST;
}

// Initialize $_REQUEST (combination of $_GET, $_POST, $_COOKIE)
$_REQUEST = array_merge($_COOKIE ?? [], $_GET, $_POST);
$GLOBALS['_REQUEST'] = $_REQUEST;

// Create $_FORM (convenience superglobal that contains form data regardless of method)
$_FORM = $_POST;
$GLOBALS['_FORM'] = $_FORM;

// Debug information (uncomment if needed)
/*
$_SERVER['PHP_FORM_DEBUG'] = 'Superglobals were manually initialized by globals_fix.php';
$_SERVER['PHP_FORM_DEBUG_TIME'] = date('Y-m-d H:i:s');
*/
?> 