<?php
/**
 * JSON Data Handler
 * 
 * Processes and responds to JSON data submissions
 */

// Set content type to JSON if this is a direct request
if ($_SERVER['REQUEST_METHOD'] === 'POST' && 
    (strpos($_SERVER['CONTENT_TYPE'] ?? '', 'application/json') !== false ||
     strpos($_SERVER['HTTP_CONTENT_TYPE'] ?? '', 'application/json') !== false)) {
    
    // Return JSON response
    header('Content-Type: application/json');
    
    // Get JSON data from $_JSON superglobal if available, or from raw input
    $jsonData = [];
    if (isset($_JSON) && !empty($_JSON)) {
        $jsonData = $_JSON;
    } else {
        // Try to parse from raw input as fallback
        $rawInput = file_get_contents('php://input');
        if (!empty($rawInput)) {
            $jsonData = json_decode($rawInput, true) ?: [];
        }
    }
    
    // Add timestamp and server info
    $response = [
        'success' => true,
        'message' => 'JSON data received successfully',
        'received_data' => $jsonData,
        'timestamp' => date('c'),
        'server' => 'Frango PHP',
        'php_version' => phpversion(),
        'superglobals' => [
            'json_available' => isset($_JSON),
            'json_count' => isset($_JSON) ? count($_JSON) : 0,
            'form_count' => isset($_FORM) ? count($_FORM) : 0,
            'post_count' => isset($_POST) ? count($_POST) : 0
        ]
    ];
    
    // Also collect environment variables for debugging
    $envVars = [];
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'PHP_JSON_') === 0 || strpos($key, 'PHP_FORM_') === 0) {
            $envVars[$key] = $value;
        }
    }
    $response['env_vars'] = $envVars;
    
    // Send the response
    echo json_encode($response, JSON_PRETTY_PRINT);
    exit;
}

// For non-JSON requests or GET requests, show the info page
?>
<!DOCTYPE html>
<html>
<head>
    <title>JSON Data Handler</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        h2 { color: #3498db; }
        pre {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
            background: #9b59b6;
            color: white;
        }
        .code-block {
            background: #2c3e50;
            color: white;
            padding: 15px;
            border-radius: 4px;
            margin: 15px 0;
            font-family: monospace;
            white-space: pre-wrap;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">JSON</span> Data Handler</h1>
        <p>This endpoint handles JSON data submissions and demonstrates how to access JSON data in PHP.</p>
        
        <h2>How to Use</h2>
        <p>Send a POST request to this endpoint with Content-Type: application/json</p>
        
        <div class="code-block">
fetch('/forms/json', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({
        user: 'johndoe',
        action: 'update',
        data: {
            id: 123,
            status: 'active'
        }
    })
});</div>
        
        <h2>Server-Side Access</h2>
        <p>In PHP, you can access the JSON data using the <code>$_JSON</code> superglobal:</p>
        
        <div class="code-block">
// Access JSON data through $_JSON superglobal
$user = $_JSON['user'];
$action = $_JSON['action'];
$data = $_JSON['data'];</div>
        
        <h2>Return to Forms</h2>
        <p>This page is used as an API endpoint. <a href="/forms">Return to the form examples</a> to try it out.</p>
    </div>
</body>
</html> 