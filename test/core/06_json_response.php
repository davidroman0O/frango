<?php
// JSON response test
header('Content-Type: application/json');

// Create a structured data array
$data = [
    'success' => true,
    'message' => 'This is a JSON response from PHP',
    'timestamp' => time(),
    'data' => [
        'items' => [
            ['id' => 1, 'name' => 'Item 1', 'price' => 19.99],
            ['id' => 2, 'name' => 'Item 2', 'price' => 29.99],
            ['id' => 3, 'name' => 'Item 3', 'price' => 39.99]
        ],
        'count' => 3,
        'page' => 1,
        'totalPages' => 1
    ],
    'meta' => [
        'apiVersion' => '1.0',
        'serverTime' => date('c')
    ]
];

// Output the JSON
echo json_encode($data, JSON_PRETTY_PRINT); 

?>
