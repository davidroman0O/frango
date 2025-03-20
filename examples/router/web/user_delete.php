<?php
/**
 * User deletion handler
 * 
 * Process form submission for deleting a user
 */

// Handle only POST requests
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: index.php?error=Invalid request method');
    exit;
}

// Validate required fields
if (empty($_POST['id'])) {
    header('Location: index.php?error=User ID is required');
    exit;
}

$userId = $_POST['id'];

// Call the API to delete user
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users/' . $userId;

// Create stream context for DELETE request
$options = [
    'http' => [
        'method' => 'DELETE',
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
if ($statusCode === 200) {
    // Success
    header('Location: index.php?success=User deleted successfully');
} else {
    // Error
    $errorMessage = isset($result['error']) ? $result['error'] : 'Failed to delete user';
    header('Location: index.php?error=' . urlencode($errorMessage));
}
exit; 