<?php
/**
 * RESTful API endpoint for creating a new product
 * This file handles POST requests to /api/products
 */

// Set content type to JSON
header('Content-Type: application/json');

// Get the raw POST data (assuming JSON)
$rawInput = file_get_contents('php://input');
$data = json_decode($rawInput, true);

// Validate the input
$errors = [];

if (empty($data)) {
    $errors[] = "No data provided or invalid JSON format";
} else {
    // Required fields
    $requiredFields = ['name', 'price', 'category', 'description'];
    
    foreach ($requiredFields as $field) {
        if (!isset($data[$field]) || empty($data[$field])) {
            $errors[] = "Missing required field: $field";
        }
    }
    
    // Price must be numeric and positive
    if (isset($data['price']) && (!is_numeric($data['price']) || $data['price'] <= 0)) {
        $errors[] = "Price must be a positive number";
    }
    
    // Category must be from allowed list
    $allowedCategories = ['clothing', 'books', 'accessories', 'electronics'];
    if (isset($data['category']) && !in_array($data['category'], $allowedCategories)) {
        $errors[] = "Invalid category. Allowed categories: " . implode(', ', $allowedCategories);
    }
}

// If there are validation errors, return them
if (!empty($errors)) {
    http_response_code(400); // Bad Request
    
    echo json_encode([
        'success' => false,
        'message' => 'Validation failed',
        'errors' => $errors,
        'api_info' => [
            'version' => '1.0',
            'endpoint' => '/api/products',
            'method' => 'POST',
            'timestamp' => date('c'),
        ],
    ], JSON_PRETTY_PRINT);
    exit;
}

// In a real app, you would save to a database
// For this demo, we'll just simulate a successful response

// Generate a new product ID (would normally come from database)
$newProductId = rand(100, 999);

// Create the new product object
$newProduct = [
    'id' => $newProductId,
    'name' => $data['name'],
    'price' => (float)$data['price'],
    'category' => $data['category'],
    'description' => $data['description'],
    'image' => $data['image'] ?? 'https://via.placeholder.com/300x300?text=Product+Image',
    'created_at' => date('c'),
];

// Return success response
http_response_code(201); // Created

echo json_encode([
    'success' => true,
    'message' => 'Product created successfully',
    'product' => $newProduct,
    'api_info' => [
        'version' => '1.0',
        'endpoint' => '/api/products',
        'method' => 'POST',
        'timestamp' => date('c'),
    ],
    'demo_note' => 'This is a demo endpoint. In a real application, the product would be saved to a database.',
], JSON_PRETTY_PRINT); 