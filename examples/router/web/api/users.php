<?php
/**
 * User API handler script
 * 
 * This script forwards requests to the Go backend API for user operations.
 * It reads the request method to determine which operation to perform.
 */

// Set content type to JSON
header('Content-Type: application/json');

// Get the request method to determine the action
$method = $_SERVER['REQUEST_METHOD'];

switch ($method) {
    case 'GET':
        // List all users - call Go API endpoint
        $apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users';
        $response = file_get_contents($apiUrl);
        
        // Pass through the API response
        echo $response;
        break;
        
    case 'POST':
        // Create a new user - read request body
        $requestBody = file_get_contents('php://input');
        
        // Forward to Go API
        $apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users';
        
        // Create stream context for POST request
        $options = [
            'http' => [
                'method' => 'POST',
                'header' => 'Content-Type: application/json',
                'content' => $requestBody,
                'ignore_errors' => true
            ]
        ];
        $context = stream_context_create($options);
        
        // Execute the request
        $response = file_get_contents($apiUrl, false, $context);
        $statusCode = $http_response_header ? intval(substr($http_response_header[0], 9, 3)) : 500;
        
        // Set response code and return response
        http_response_code($statusCode);
        echo $response;
        break;
        
    default:
        // Method not allowed
        http_response_code(405);
        echo json_encode([
            'error' => 'Method not allowed',
            'allowed_methods' => ['GET', 'POST']
        ]);
} 