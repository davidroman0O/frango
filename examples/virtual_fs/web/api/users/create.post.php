<?php
// API endpoint for creating users (POST)
header('Content-Type: application/json');

// Check method
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode([
        'success' => false,
        'message' => 'Method not allowed'
    ]);
    exit;
}

// Get JSON data
$json = file_get_contents('php://input');
$data = json_decode($json, true);

// Validate input
if (!$data || !isset($data['name']) || !isset($data['email'])) {
    http_response_code(400);
    echo json_encode([
        'success' => false,
        'message' => 'Invalid input. Name and email are required.'
    ]);
    exit;
}

// Process the request (just simulate in this example)
$newUser = [
    'id' => rand(100, 999),
    'name' => $data['name'],
    'email' => $data['email'],
    'created_at' => date('Y-m-d H:i:s')
];

// Return JSON response
echo json_encode([
    'success' => true,
    'data' => $newUser,
    'message' => 'User created successfully'
]);
?> 