<?php
// Return some sample user data as JSON
header('Content-Type: application/json');

// Create sample data
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

// Add request and server info for debugging
$response = [
    'status' => 'success',
    'message' => 'Users retrieved successfully',
    'users' => $users,
    'request' => [
        'uri' => $_SERVER['REQUEST_URI'],
        'method' => $_SERVER['REQUEST_METHOD'],
        'time' => date('Y-m-d H:i:s')
    ]
];

// Output JSON
echo json_encode($response, JSON_PRETTY_PRINT);
?> 