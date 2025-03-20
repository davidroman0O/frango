<?php
// users_list.php - List all users (GET /users)
header('Content-Type: application/json');

// Get users directly from memory store
$users = [];

// Access the Go memory store via Go handler
if (function_exists('frankenphp_handle_request')) {
    // We're in FrankenPHP, so we can access shared memory
    // The users are stored in the shared memory during request handling
    global $sharedMemory;
    if (isset($sharedMemory) && isset($sharedMemory['users'])) {
        $users = $sharedMemory['users'];
    }
} else {
    // Fallback for non-worker mode
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

// Add metadata about the request
$response = [
    'users' => $users,
    'count' => count($users),
    'method' => $_SERVER['REQUEST_METHOD'],
    'handler' => 'users_list.php',
    'timestamp' => date('Y-m-d H:i:s'),
    'mode' => function_exists('frankenphp_handle_request') ? 'FrankenPHP Worker' : 'Standard PHP'
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 