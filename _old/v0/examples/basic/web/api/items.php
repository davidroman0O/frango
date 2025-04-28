<?php
// Set JSON content type
header('Content-Type: application/json');

// Simulate an items data response
$items = [
    [
        'id' => 1,
        'name' => 'Laptop',
        'price' => 999.99,
        'inStock' => true
    ],
    [
        'id' => 2,
        'name' => 'Smartphone',
        'price' => 499.99,
        'inStock' => true
    ],
    [
        'id' => 3,
        'name' => 'Headphones',
        'price' => 149.99,
        'inStock' => false
    ],
    [
        'id' => 4,
        'name' => 'Monitor',
        'price' => 299.99,
        'inStock' => true
    ],
    [
        'id' => 5,
        'name' => 'Keyboard',
        'price' => 79.99,
        'inStock' => true
    ]
];

// Add metadata
$response = [
    'count' => count($items),
    'items' => $items,
    'timestamp' => date('Y-m-d H:i:s'),
    'source' => $_SERVER['GO_PHP_SOURCE_FILE'] ?? 'unknown'
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT); 