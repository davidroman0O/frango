<?php
/**
 * Simple API endpoint that returns a hello message
 */

// Set content type to JSON
header('Content-Type: application/json');

// Get current time
$now = new DateTime();

// Prepare response
$response = [
    'message' => 'Hello from the Virtual FS API!',
    'timestamp' => $now->format('c'),
    'info' => [
        'php_version' => PHP_VERSION,
        'server' => $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown',
    ]
];

// If we have the app config, include it
if (file_exists($_SERVER['DOCUMENT_ROOT'] . '/config/app.php')) {
    include_once($_SERVER['DOCUMENT_ROOT'] . '/config/app.php');
    if (isset($config)) {
        $response['app'] = $config;
    }
}

// Output the response as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 