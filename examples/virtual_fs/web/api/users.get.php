<?php
// API endpoint for users (GET)
header('Content-Type: application/json');

// Sample user data
$users = [
    ['id' => 1, 'name' => 'Alice', 'email' => 'alice@example.com'],
    ['id' => 2, 'name' => 'Bob', 'email' => 'bob@example.com'],
    ['id' => 3, 'name' => 'Carol', 'email' => 'carol@example.com'],
];

// Return JSON response
echo json_encode([
    'success' => true,
    'data' => $users,
    'count' => count($users),
    'message' => 'Users retrieved successfully'
]);
?> 