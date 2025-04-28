<?php
// Set JSON content type
header('Content-Type: application/json');

// Simulate a user data response
$userData = [
    'id' => 1001,
    'username' => 'johndoe',
    'name' => 'John Doe',
    'email' => 'john@example.com',
    'role' => 'admin',
    'created' => date('Y-m-d'),
    'lastLogin' => date('Y-m-d H:i:s'),
    'metadata' => [
        'route' => 'PHP API endpoint via middleware',
        'phpVersion' => phpversion(),
        'timestamp' => time(),
        'source' => $_SERVER['GO_PHP_SOURCE_FILE'] ?? 'unknown'
    ]
];

// Output as JSON
echo json_encode($userData, JSON_PRETTY_PRINT); 