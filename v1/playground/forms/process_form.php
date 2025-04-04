<?php
/**
 * Form Processing Bridge
 * 
 * This file acts as a bridge between HTTP form submissions and PHP scripts
 * It manually captures form data and makes it available to other scripts
 */

// Start or resume a session to store the form data
session_start();

// Determine the request method
$method = $_SERVER['REQUEST_METHOD'] ?? 'UNKNOWN';

// Initialize form data storage
if (!isset($_SESSION['form_data'])) {
    $_SESSION['form_data'] = [
        'POST' => [],
        'GET' => [],
        'JSON' => [],
        'last_updated' => 0
    ];
}

// Process based on the method and content type
$contentType = $_SERVER['PHP_HEADER_CONTENT_TYPE'] ?? $_SERVER['CONTENT_TYPE'] ?? '';

// For debugging
$_SESSION['form_data']['debug'] = [
    'request_method' => $method,
    'content_type' => $contentType,
    'request_uri' => $_SERVER['REQUEST_URI'] ?? 'unknown',
    'time' => date('Y-m-d H:i:s')
];

// Extract PHP_FORM_* variables directly from $_SERVER
$formData = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $formField = substr($key, 10); // Remove 'PHP_FORM_' prefix
        $formData[$formField] = $value;
    }
}

// Capture POST data
if ($method === 'POST') {
    // Store the raw input
    $rawInput = file_get_contents('php://input');
    $_SESSION['form_data']['raw_input'] = $rawInput;
    
    // For JSON submissions
    if (strpos($contentType, 'application/json') !== false) {
        $jsonData = json_decode($rawInput, true) ?: [];
        $_SESSION['form_data']['JSON'] = $jsonData;
        $_SESSION['form_data']['content_type'] = $contentType;
        $_SESSION['form_data']['last_updated'] = time();
        
        // If it's a JSON request, return JSON response
        header('Content-Type: application/json');
        echo json_encode([
            'success' => true,
            'message' => 'JSON data received',
            'data' => $jsonData
        ]);
        exit;
    }
    // For standard form submissions
    else {
        // Use the PHP_FORM_* variables we captured above
        if (!empty($formData)) {
            $_SESSION['form_data']['POST'] = $formData;
        } else {
            // Fallback to traditional methods
            $postData = [];
            parse_str($rawInput, $postData);
            
            // Also try to get data from $_POST (might be already parsed by PHP)
            if (empty($postData) && !empty($_POST)) {
                $postData = $_POST;
            }
            
            $_SESSION['form_data']['POST'] = $postData;
        }
        
        $_SESSION['form_data']['content_type'] = $contentType;
        $_SESSION['form_data']['last_updated'] = time();
        
        // Log for debugging
        $_SESSION['form_data']['debug_info'] = [
            'method' => $method,
            'content_type' => $contentType,
            'raw_length' => strlen($rawInput),
            'parsed_count' => count($_SESSION['form_data']['POST']),
            'timestamp' => date('Y-m-d H:i:s'),
            'form_data_direct' => $formData
        ];
        
        // Commit session data before redirect
        session_write_close();
        
        // After processing POST data, redirect to the POST display page
        header("Location: /forms/post_display");
        exit;
    }
}
// Capture GET data
else if ($method === 'GET') {
    // Store GET parameters directly
    $_SESSION['form_data']['GET'] = $_GET;
    $_SESSION['form_data']['last_updated'] = time();
    
    // Log for debugging
    $_SESSION['form_data']['debug_info'] = [
        'method' => $method,
        'query_string' => $_SERVER['QUERY_STRING'] ?? '',
        'get_count' => count($_GET),
        'timestamp' => date('Y-m-d H:i:s')
    ];
    
    // Commit session data before redirect
    session_write_close();
    
    // After processing GET data, redirect to the GET display page
    header("Location: /forms/get_display");
    exit;
}

// If we reach here, something went wrong
$_SESSION['form_data']['error'] = "Unsupported request method: $method";
session_write_close();
header("Location: /forms/");
exit; 