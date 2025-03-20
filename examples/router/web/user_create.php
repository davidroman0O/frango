<?php
/**
 * User creation handler
 * 
 * Process form submission for creating a new user
 */

// Handle only POST requests
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: index.php?error=Invalid request method');
    exit;
}

// Validate required fields
if (empty($_POST['name']) || empty($_POST['email'])) {
    header('Location: index.php?error=Name and email are required');
    exit;
}

// Prepare data for API
$userData = [
    'name' => $_POST['name'],
    'email' => $_POST['email'],
    'role' => $_POST['role'] ?? 'user'
];

// Call the API to create user
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users';

// Create stream context for POST request
$options = [
    'http' => [
        'method' => 'POST',
        'header' => 'Content-Type: application/json',
        'content' => json_encode($userData),
        'ignore_errors' => true
    ]
];
$context = stream_context_create($options);

// Execute request
$response = file_get_contents($apiUrl, false, $context);
$statusCode = $http_response_header ? intval(substr($http_response_header[0], 9, 3)) : 500;

// Parse the response
$result = json_decode($response, true);

// Handle the result
if ($statusCode === 201) {
    // Success
    header('Location: index.php?success=User created successfully');
} else {
    // Error
    $errorMessage = isset($result['error']) ? $result['error'] : 'Failed to create user';
    header('Location: index.php?error=' . urlencode($errorMessage));
}
exit; 