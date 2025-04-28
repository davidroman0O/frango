<?php
/**
 * Frango v2 Demo - JSON Request Handler
 * 
 * Demonstrates processing JSON data sent in request body.
 */

// Set content type to JSON for responses
header('Content-Type: application/json');

// Tracking method used to get JSON data
$jsonAccessMethod = 'none';
$jsonData = null;

// Method 1: Using Frango's $_JSON superglobal (preferred)
if (isset($GLOBALS['_JSON'])) {
    $jsonData = $GLOBALS['_JSON'];
    $jsonAccessMethod = '$_JSON';
}
// Method 2: Fallback to php://input + json_decode
else {
    $rawInput = file_get_contents('php://input');
    if (!empty($rawInput)) {
        $jsonData = json_decode($rawInput, true);
        $jsonAccessMethod = 'php://input';
    }
}

// Check request method
$isPost = $_SERVER['REQUEST_METHOD'] === 'POST';

// Prepare response
$response = [
    'success' => true,
    'message' => $isPost ? 'JSON data processed successfully' : 'Use POST method to submit JSON data',
    'timestamp' => date('c'),
    'request' => [
        'method' => $_SERVER['REQUEST_METHOD'],
        'content_type' => $_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'Not available',
        'json_access_method' => $jsonAccessMethod,
    ]
];

if ($jsonData) {
    // Add processed data to response
    $response['data'] = $jsonData;
    
    // Process specific JSON properties if needed
    if (isset($jsonData['user'])) {
        $response['user_info'] = [
            'provided' => true,
            'name' => $jsonData['user']['name'] ?? 'Not provided',
            'email' => $jsonData['user']['email'] ?? 'Not provided'
        ];
    }
    
    // Add items count if present
    if (isset($jsonData['items']) && is_array($jsonData['items'])) {
        $response['items_count'] = count($jsonData['items']);
    }
    
    // Echo action if present
    if (isset($jsonData['action'])) {
        $response['action_requested'] = $jsonData['action'];
    }
} else if ($isPost) {
    // No JSON data received in a POST request
    $response['success'] = false;
    $response['message'] = 'No JSON data received. Make sure to send with Content-Type: application/json';
}

// Debug information in response
$response['debug'] = [
    'post_data' => $_POST, // Will be empty for proper JSON requests
    'php_input_length' => strlen(file_get_contents('php://input')),
    'php_input_sample' => substr(file_get_contents('php://input'), 0, 50) . (strlen(file_get_contents('php://input')) > 50 ? '...' : ''),
    'server_vars' => [
        'REQUEST_METHOD' => $_SERVER['REQUEST_METHOD'],
        'CONTENT_TYPE' => $_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'Not available',
        'CONTENT_LENGTH' => $_SERVER['CONTENT_LENGTH'] ?? $_SERVER['HTTP_CONTENT_LENGTH'] ?? 'Not available',
    ]
];

// Return response as JSON
echo json_encode($response, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES); 