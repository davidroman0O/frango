<?php
// user_update.php - Update a user by ID (PUT /users/{id})
header('Content-Type: application/json');

// Only allow PUT requests
if ($_SERVER['REQUEST_METHOD'] !== 'PUT') {
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

// Get JSON input
$inputJSON = file_get_contents('php://input');
$input = json_decode($inputJSON, true);

// Validate input
if (!$input) {
    http_response_code(400); // Bad Request
    echo json_encode([
        'error' => 'Invalid input', 
        'message' => 'JSON input required',
        'received' => $inputJSON
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

// Update the user with the provided fields
if (isset($input['name'])) {
    $foundUser['name'] = $input['name'];
}
if (isset($input['email'])) {
    $foundUser['email'] = $input['email'];
}
if (isset($input['role'])) {
    $foundUser['role'] = $input['role'];
}

// Add updated timestamp
$foundUser['updated_at'] = date('Y-m-d H:i:s');

// Update the user in shared memory
if ($usingSharedMemory && $userIndex !== null) {
    $sharedMemory['users'][$userIndex] = $foundUser;
}

// Add metadata about the request
$response = [
    'success' => true,
    'message' => 'User updated successfully',
    'user' => $foundUser,
    'method' => $_SERVER['REQUEST_METHOD'],
    'handler' => 'user_update.php',
    'userId' => $userId,
    'timestamp' => date('Y-m-d H:i:s'),
    'mode' => $usingSharedMemory ? 'FrankenPHP Worker (shared memory)' : 'Standard PHP'
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 