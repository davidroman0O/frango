<?php
// Set content type to JSON
header('Content-Type: application/json');

// Mock data - in a real app this would come from a database
$products = [
    [
        'id' => 1,
        'name' => 'Laptop', 
        'price' => 999.99,
        'description' => 'High-performance laptop',
        'created_at' => date('c')
    ],
    [
        'id' => 2,
        'name' => 'Smartphone',
        'price' => 599.99,
        'description' => 'Latest smartphone model',
        'created_at' => date('c')
    ],
    [
        'id' => 3,
        'name' => 'Tablet',
        'price' => 399.99,
        'description' => 'Portable tablet device',
        'created_at' => date('c')
    ]
];

// Demo endpoint to call Go API from PHP
$callGoEndpoint = false;
if (isset($_GET['call_go']) && $_GET['call_go'] === 'true') {
    $callGoEndpoint = true;
}

// Option to call the Go API and include its results
if ($callGoEndpoint) {
    // Initialize cURL session to call the Go API
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, 'http://localhost:8080/api/go/products');
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    $goApiResponse = curl_exec($ch);
    $goApiError = curl_error($ch);
    curl_close($ch);
    
    // Parse response
    $goApiData = json_decode($goApiResponse, true);
    
    // Add the Go API data to our response
    echo json_encode([
        'success' => true,
        'message' => 'Products retrieved from PHP endpoint',
        'data' => [
            'php_products' => $products,
            'go_products' => $goApiData ?? ['error' => $goApiError],
            'php_count' => count($products),
            'source' => 'PHP API with Go integration'
        ]
    ]);
} else {
    // Just return the PHP data
    echo json_encode([
        'success' => true,
        'message' => 'Products retrieved from PHP endpoint',
        'data' => [
            'products' => $products,
            'count' => count($products),
            'source' => 'PHP API'
        ]
    ]);
} 