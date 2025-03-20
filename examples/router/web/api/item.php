<?php
/**
 * Individual Item API handler script
 * 
 * This script forwards requests to the Go backend API for operations on a specific item.
 * It reads the request method to determine which operation to perform.
 */

// Set content type to JSON
header('Content-Type: application/json');

// Get the request method to determine the action
$method = $_SERVER['REQUEST_METHOD'];

// Get item ID from path
$pathParts = explode('/', trim($_SERVER['PATH_INFO'] ?? '', '/'));
$itemId = end($pathParts);

if (!$itemId) {
    http_response_code(400);
    echo json_encode(['error' => 'Item ID is required']);
    exit;
}

// Base API URL
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/items/' . $itemId;

switch ($method) {
    case 'GET':
        // Get item details
        $response = file_get_contents($apiUrl);
        
        // Pass through the API response
        echo $response;
        break;
        
    default:
        // Method not allowed
        http_response_code(405);
        echo json_encode([
            'error' => 'Method not allowed',
            'allowed_methods' => ['GET']
        ]);
} 