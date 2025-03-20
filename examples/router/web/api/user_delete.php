<?php
// user_delete.php - Delete a user by ID (DELETE /users/{id})
header('Content-Type: application/json');

// Only allow DELETE requests
if ($_SERVER['REQUEST_METHOD'] !== 'DELETE') {
    http_response_code(405); // Method Not Allowed
    echo json_encode(['error' => 'Method not allowed', 'method' => $_SERVER['REQUEST_METHOD']]);
    exit;
}

// Extract user ID from the URL path parameters
$userId = null;
if (isset($_SERVER['PATH_PARAMS'])) {
    $pathParams = json_decode($_SERVER['PATH_PARAMS'], true);
    if (isset($pathParams['id'])) {
        $userId = $pathParams['id'];
    }
}

// If ID is not found in PATH_PARAMS, try extracting from the URL
if ($userId === null) {
    $path = $_SERVER['REQUEST_URI'];
    $pathParts = explode('/', trim($path, '/'));
    $userId = $pathParts[count($pathParts) - 1]; // Get the last part of the path
}

// Ensure ID is numeric
if (!is_numeric($userId)) {
    http_response_code(400); // Bad Request
    echo json_encode([
        'error' => 'Invalid user ID', 
        'userId' => $userId
    ]);
    exit;
}

// Get user from shared memory if available
$usingSharedMemory = false;
$userIndex = null;
$foundUser = null;

if (function_exists('frankenphp_handle_request')) {
    // We're in FrankenPHP worker mode
    global $sharedMemory;
    if (isset($sharedMemory) && isset($sharedMemory['users'])) {
        $usingSharedMemory = true;
        // Find the user by ID
        foreach ($sharedMemory['users'] as $index => $user) {
            if ($user['id'] == $userId) {
                $userIndex = $index;
                $foundUser = $user;
                break;
            }
        }
    }
}

// If user not found in shared memory
if ($foundUser === null) {
    http_response_code(404); // Not Found
    echo json_encode([
        'error' => 'User not found',
        'userId' => $userId,
        'mode' => $usingSharedMemory ? 'Worker mode (shared memory)' : 'Standard PHP'
    ]);
    exit;
}

// Delete the user from shared memory
$deletedUser = $foundUser;
if ($usingSharedMemory && $userIndex !== null) {
    // Remove the user from the array
    array_splice($sharedMemory['users'], $userIndex, 1);
}

// Add metadata about the request
$response = [
    'success' => true,
    'message' => 'User deleted successfully',
    'deleted_user' => $deletedUser,
    'method' => $_SERVER['REQUEST_METHOD'],
    'handler' => 'user_delete.php',
    'userId' => $userId,
    'timestamp' => date('Y-m-d H:i:s'),
    'mode' => $usingSharedMemory ? 'FrankenPHP Worker (shared memory)' : 'Standard PHP'
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 