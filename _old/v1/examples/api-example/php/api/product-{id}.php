<?php
// Set content type to JSON
// header('Content-Type: application/json');

// Get product ID from URL path parameter
$productId = $_PATH["id"] ?? null;

if (!$productId) {
    http_response_code(400);
    echo json_encode([
        'success' => false,
        'message' => 'Product ID is required',
    ]);
    exit;
}

// Mock data - in a real app this would come from a database
$products = [
    1 => [
        'id' => 1,
        'name' => 'Laptop', 
        'price' => 999.99,
        'description' => 'High-performance laptop',
        'specs' => [
            'processor' => '3.2 GHz Quad Core',
            'memory' => '16 GB RAM',
            'storage' => '512 GB SSD'
        ],
        'created_at' => date('c')
    ],
    2 => [
        'id' => 2,
        'name' => 'Smartphone',
        'price' => 599.99,
        'description' => 'Latest smartphone model',
        'specs' => [
            'processor' => '2.8 GHz Octa Core',
            'memory' => '8 GB RAM',
            'storage' => '256 GB Flash'
        ],
        'created_at' => date('c')
    ],
    3 => [
        'id' => 3,
        'name' => 'Tablet',
        'price' => 399.99,
        'description' => 'Portable tablet device',
        'specs' => [
            'processor' => '2.4 GHz Quad Core',
            'memory' => '4 GB RAM',
            'storage' => '128 GB Flash'
        ],
        'created_at' => date('c')
    ]
];

// Check if the product exists
if (isset($products[$productId])) {
    // Option to call the Go API for this product
    if (isset($_GET['call_go']) && $_GET['call_go'] === 'true') {
        // Initialize cURL session
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, "http://localhost:8080/api/go/products/{$productId}");
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        $goApiResponse = curl_exec($ch);
        $goApiError = curl_error($ch);
        curl_close($ch);
        
        // Parse response
        $goApiData = json_decode($goApiResponse, true);
        
        // Return both the PHP data and Go data
        echo json_encode([
            'success' => true,
            'message' => 'Product retrieved from PHP endpoint',
            'data' => [
                'php_product' => $products[$productId],
                'go_product' => $goApiData ?? ['error' => $goApiError],
                'source' => 'PHP API with Go integration'
            ]
        ]);
    } else {
        // Just return the PHP data
        echo json_encode([
            'success' => true,
            'message' => 'Product retrieved from PHP endpoint',
            'data' => [
                'product' => $products[$productId],
                'source' => 'PHP API'
            ]
        ]);
    }
} else {
    // Product not found
    http_response_code(404);
    echo json_encode([
        'success' => false,
        'message' => 'Product not found',
    ]);
} 