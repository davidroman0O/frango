<?php
// Return some sample item data as JSON
header('Content-Type: application/json');

// Create sample data
$items = [
    [
        'id' => 1,
        'name' => 'Item 1',
        'description' => 'This is the first item',
        'price' => 19.99
    ],
    [
        'id' => 2,
        'name' => 'Item 2',
        'description' => 'This is the second item',
        'price' => 29.99
    ],
    [
        'id' => 3,
        'name' => 'Item 3',
        'description' => 'This is the third item',
        'price' => 39.99
    ]
];

// Add request and server info for debugging
$response = [
    'status' => 'success',
    'message' => 'Items retrieved successfully',
    'items' => $items,
    'request' => [
        'uri' => $_SERVER['REQUEST_URI'],
        'method' => $_SERVER['REQUEST_METHOD'],
        'time' => date('Y-m-d H:i:s')
    ]
];

// Output JSON
echo json_encode($response, JSON_PRETTY_PRINT);
?> 