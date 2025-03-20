<?php
/**
 * Individual User API handler script
 * 
 * This script forwards requests to the Go backend API for operations on a specific user.
 * It reads the request method to determine which operation to perform.
 */

// Set content type to JSON
header('Content-Type: application/json');

// Get the request method to determine the action
$method = $_SERVER['REQUEST_METHOD'];

// Get user ID from path
$pathParts = explode('/', trim($_SERVER['PATH_INFO'] ?? '', '/'));
$userId = end($pathParts);

if (!$userId) {
    http_response_code(400);
    echo json_encode(['error' => 'User ID is required']);
    exit;
}

// Base API URL
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users/' . $userId;

switch ($method) {
    case 'GET':
        // Get user details
        $response = file_get_contents($apiUrl);
        
        // Pass through the API response
        echo $response;
        break;
        
    case 'PUT':
        // Update user - read request body
        $requestBody = file_get_contents('php://input');
        
        // Create stream context for PUT request
        $options = [
            'http' => [
                'method' => 'PUT',
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
        
    case 'DELETE':
        // Create stream context for DELETE request
        $options = [
            'http' => [
                'method' => 'DELETE',
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
            'allowed_methods' => ['GET', 'PUT', 'DELETE']
        ]);
} 