<?php
/**
 * Item creation handler
 * 
 * Process form submission for creating a new item
 */

// Handle only POST requests
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: index.php?error=Invalid request method');
    exit;
}

// Validate required fields
if (empty($_POST['name'])) {
    header('Location: index.php?error=Item name is required');
    exit;
}

// Prepare data for API
$itemData = [
    'name' => $_POST['name'],
    'description' => $_POST['description'] ?? 'No description'
];

// Call the API to create item
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/items';

// Create stream context for POST request
$options = [
    'http' => [
        'method' => 'POST',
        'header' => 'Content-Type: application/json',
        'content' => json_encode($itemData),
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
    header('Location: index.php?success=Item created successfully');
} else {
    // Error
    $errorMessage = isset($result['error']) ? $result['error'] : 'Failed to create item';
    header('Location: index.php?error=' . urlencode($errorMessage));
}
exit; 