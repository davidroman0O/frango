<?php
/**
 * RESTful API endpoint for retrieving products
 * This file handles GET requests to /api/products
 */

// Set content type to JSON
header('Content-Type: application/json');

// Get query parameters for filtering
$category = $_GET['category'] ?? null;
$limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 10;
$limit = min(max($limit, 1), 50); // Constrain between 1 and 50

// Simulate a products database
$products = [
    [
        'id' => 1,
        'name' => 'Frango T-Shirt',
        'price' => 24.99,
        'category' => 'clothing',
        'description' => 'Show your love for Frango with this comfortable cotton t-shirt',
        'image' => 'https://via.placeholder.com/300x300?text=Frango+Tshirt',
    ],
    [
        'id' => 2,
        'name' => 'Go Programming Book',
        'price' => 39.99,
        'category' => 'books',
        'description' => 'Learn Go programming from scratch',
        'image' => 'https://via.placeholder.com/300x300?text=Go+Book',
    ],
    [
        'id' => 3,
        'name' => 'PHP Advanced Techniques',
        'price' => 42.99,
        'category' => 'books',
        'description' => 'Master advanced PHP techniques',
        'image' => 'https://via.placeholder.com/300x300?text=PHP+Book',
    ],
    [
        'id' => 4,
        'name' => 'Frango Coffee Mug',
        'price' => 14.99,
        'category' => 'accessories',
        'description' => 'Enjoy your coffee while coding with Frango',
        'image' => 'https://via.placeholder.com/300x300?text=Coffee+Mug',
    ],
    [
        'id' => 5,
        'name' => 'Web Dev Laptop Sticker Pack',
        'price' => 8.99,
        'category' => 'accessories',
        'description' => 'Decorate your laptop with cool tech stickers',
        'image' => 'https://via.placeholder.com/300x300?text=Sticker+Pack',
    ],
    [
        'id' => 6,
        'name' => 'Mechanical Keyboard',
        'price' => 129.99,
        'category' => 'electronics',
        'description' => 'Enhance your coding experience with this mechanical keyboard',
        'image' => 'https://via.placeholder.com/300x300?text=Mechanical+Keyboard',
    ],
    [
        'id' => 7,
        'name' => 'Frango Hoodie',
        'price' => 49.99,
        'category' => 'clothing',
        'description' => 'Stay warm while coding with this Frango hoodie',
        'image' => 'https://via.placeholder.com/300x300?text=Frango+Hoodie',
    ],
];

// Apply category filter if provided
if ($category) {
    $products = array_filter($products, function($product) use ($category) {
        return $product['category'] === $category;
    });
    $products = array_values($products); // Re-index array
}

// Apply limit
$products = array_slice($products, 0, $limit);

// Prepare the response
$response = [
    'success' => true,
    'products' => $products,
    'metadata' => [
        'total' => count($products),
        'filtered' => $category ? true : false,
        'filter_category' => $category,
        'limit' => $limit,
    ],
    'api_info' => [
        'version' => '1.0',
        'endpoint' => '/api/products',
        'method' => 'GET',
        'timestamp' => date('c'),
    ],
];

// Output the response
echo json_encode($response, JSON_PRETTY_PRINT); 