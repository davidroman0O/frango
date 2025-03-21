<?php
// user_detail.php - Get user details by ID (GET /users/{id})
header('Content-Type: application/json');

// Only allow GET requests
if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
    http_response_code(405); // Method Not Allowed
    echo json_encode(['error' => 'Method not allowed', 'method' => $_SERVER['REQUEST_METHOD']]);
    exit;
}

// Extract user ID from the URL path parameters
// In frango with method-based routing, path parameters are stored in $_SERVER['PATH_PARAMS']
// Try to get ID from PATH_PARAMS first (newer method)
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
        'userId' => $userId,
        'path' => $path
    ]);
    exit;
}

// Get user from shared memory if available
$foundUser = null;
$usingSharedMemory = false;

if (function_exists('frankenphp_handle_request')) {
    // We're in FrankenPHP worker mode
    global $sharedMemory;
    if (isset($sharedMemory) && isset($sharedMemory['users'])) {
        $usingSharedMemory = true;
        // Find the user by ID
        foreach ($sharedMemory['users'] as $user) {
            if ($user['id'] == $userId) {
                $foundUser = $user;
                break;
            }
        }
    }
}

// If not found in shared memory or not using it, use static data
if ($foundUser === null && !$usingSharedMemory) {
    // Simulate user retrieval (in a real app, this would come from a database)
    $users = [
        1 => [
            'id' => 1,
            'name' => 'John Doe',
            'email' => 'john@example.com',
            'role' => 'admin',
            'details' => [
                'joined' => '2020-01-01',
                'status' => 'active',
                'preferences' => [
                    'theme' => 'dark',
                    'notifications' => true
                ]
            ]
        ],
        2 => [
            'id' => 2,
            'name' => 'Jane Smith',
            'email' => 'jane@example.com',
            'role' => 'user',
            'details' => [
                'joined' => '2021-03-15',
                'status' => 'active',
                'preferences' => [
                    'theme' => 'light',
                    'notifications' => false
                ]
            ]
        ],
        3 => [
            'id' => 3,
            'name' => 'Bob Johnson',
            'email' => 'bob@example.com',
            'role' => 'user',
            'details' => [
                'joined' => '2022-06-30',
                'status' => 'inactive',
                'preferences' => [
                    'theme' => 'system',
                    'notifications' => true
                ]
            ]
        ]
    ];
    
    // Check if user exists in static data
    if (isset($users[$userId])) {
        $foundUser = $users[$userId];
    }
}

// If user not found in either source
if ($foundUser === null) {
    http_response_code(404); // Not Found
    echo json_encode([
        'error' => 'User not found',
        'userId' => $userId
    ]);
    exit;
}

// Add metadata about the request
$response = [
    'user' => $foundUser,
    'method' => $_SERVER['REQUEST_METHOD'],
    'handler' => 'user_detail.php',
    'userId' => $userId,
    'timestamp' => date('Y-m-d H:i:s'),
    'mode' => $usingSharedMemory ? 'FrankenPHP Worker (shared memory)' : 'Standard PHP'
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 