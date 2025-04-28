<?php
// users_create.php - Create a new user (POST /users)
header('Content-Type: application/json');

// Only allow POST requests
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405); // Method Not Allowed
    echo json_encode(['error' => 'Method not allowed', 'method' => $_SERVER['REQUEST_METHOD']]);
    exit;
}

// Get JSON input
$inputJSON = file_get_contents('php://input');
$input = json_decode($inputJSON, true);

// Validate input
if (!$input || !isset($input['name'])) {
    http_response_code(400); // Bad Request
    echo json_encode([
        'error' => 'Invalid input', 
        'message' => 'Name is required',
        'received' => $input
    ]);
    exit;
}

// Access the shared memory for users if available
$users = [];
$usingSharedMemory = false;

if (function_exists('frankenphp_handle_request')) {
    // We're in FrankenPHP worker mode, so we can access shared memory
    global $sharedMemory;
    if (isset($sharedMemory) && isset($sharedMemory['users'])) {
        $users = $sharedMemory['users'];
        $usingSharedMemory = true;
    }
} 

// If no shared memory or empty, use fallback data
if (empty($users)) {
    $users = [
        [
            'id' => 1,
            'name' => 'John Doe',
            'email' => 'john@example.com',
            'role' => 'admin'
        ],
        [
            'id' => 2,
            'name' => 'Jane Smith',
            'email' => 'jane@example.com',
            'role' => 'user'
        ],
        [
            'id' => 3,
            'name' => 'Bob Johnson',
            'email' => 'bob@example.com',
            'role' => 'user'
        ]
    ];
}

// Generate a new ID (use max ID + 1)
$maxId = 0;
foreach ($users as $user) {
    if ($user['id'] > $maxId) {
        $maxId = $user['id'];
    }
}
$newId = $maxId + 1;

// Create new user
$newUser = [
    'id' => $newId,
    'name' => $input['name'],
    'email' => $input['email'] ?? 'no-email@example.com',
    'role' => $input['role'] ?? 'user',
    'created_at' => date('Y-m-d H:i:s')
];

// Add to users array
$users[] = $newUser;

// Update shared memory if we're in worker mode
if ($usingSharedMemory && isset($sharedMemory)) {
    $sharedMemory['users'] = $users;
}

// Add metadata about the request
$response = [
    'success' => true,
    'message' => 'User created successfully',
    'user' => $newUser,
    'method' => $_SERVER['REQUEST_METHOD'],
    'handler' => 'users_create.php',
    'timestamp' => date('Y-m-d H:i:s'),
    'mode' => $usingSharedMemory ? 'FrankenPHP Worker (persistent)' : 'Standard PHP (non-persistent)'
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 