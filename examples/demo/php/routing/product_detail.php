<?php
/**
 * Frango v2 Demo - Product Detail Page
 * 
 * Demonstrates path parameter extraction from URLs like /products/{productID}
 * and optional query parameters like ?color=red&size=large
 */

// Extract path parameters from the URL
// For a URL like /products/abc123, the {productID} parameter should be available in $_GET
$productID = $_GET['productID'] ?? 'unknown';

// Get query parameters (traditional part of $_GET)
$color = $_GET['color'] ?? null;
$size = $_GET['size'] ?? null;
$ownerView = isset($_GET['owner']) && $_GET['owner'] === 'true';

// Sample product data - in a real app this would come from a database
$products = [
    'abc' => [
        'id' => 'abc',
        'name' => 'Wireless Headphones',
        'price' => 129.99,
        'category' => 'Electronics',
        'image' => 'headphones.jpg',
        'colors' => ['Black', 'White', 'Blue'],
        'sizes' => ['One Size'],
        'description' => 'High-quality wireless headphones with noise cancellation and long battery life.'
    ],
    'def' => [
        'id' => 'def',
        'name' => 'Running Shoes',
        'price' => 89.95,
        'category' => 'Footwear',
        'image' => 'shoes.jpg',
        'colors' => ['Red', 'Black', 'Gray'],
        'sizes' => ['S', 'M', 'L', 'XL'],
        'description' => 'Lightweight, breathable running shoes with enhanced cushioning for comfort.'
    ],
    'ghi' => [
        'id' => 'ghi',
        'name' => 'Coffee Maker',
        'price' => 49.99,
        'category' => 'Kitchen',
        'image' => 'coffee.jpg',
        'colors' => ['Silver', 'Black'],
        'sizes' => ['One Size'],
        'description' => 'Programmable coffee maker with a 12-cup capacity and auto-shutoff feature.'
    ]
];

// Get the current product data or use placeholder data
$productData = $products[$productID] ?? [
    'id' => $productID,
    'name' => 'Unknown Product',
    'price' => 0.00,
    'category' => 'Uncategorized',
    'image' => 'placeholder.jpg',
    'colors' => [],
    'sizes' => [],
    'description' => 'This product does not exist in our catalog.'
];

// Function to format price
function formatPrice($price) {
    return '$' . number_format($price, 2);
}

// Function to get a random product ID for demo navigation
function getRandomProductID($exclude = null) {
    global $products;
    $ids = array_keys($products);
    if ($exclude !== null) {
        $ids = array_filter($ids, function($id) use ($exclude) {
            return $id != $exclude;
        });
    }
    return $ids[array_rand($ids)] ?? 'abc';
}
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Product: <?= htmlspecialchars($productData['name']) ?></title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>Product Details</h1>
        <p>Demonstrating path parameter extraction from URLs like <code>/products/{productID}</code> with optional query parameters.</p>
        
        <div class="card">
            <div style="display: flex; flex-wrap: wrap;">
                <div style="flex: 1; min-width: 260px; margin-right: 20px;">
                    <!-- Product Image (placeholder) -->
                    <div style="background-color: #eee; width: 100%; height: 200px; display: flex; justify-content: center; align-items: center; margin-bottom: 15px; border-radius: 4px;">
                        <div style="font-size: 18px; color: #777;">
                            [<?= htmlspecialchars($productData['image']) ?>]
                        </div>
                    </div>
                    
                    <!-- Color and Size selection -->
                    <?php if (!empty($productData['colors'])): ?>
                    <div style="margin-bottom: 15px;">
                        <label style="display: block; margin-bottom: 5px; font-weight: bold;">Color:</label>
                        <div>
                            <?php foreach ($productData['colors'] as $productColor): ?>
                                <a href="/products/<?= htmlspecialchars($productID) ?>?color=<?= strtolower(htmlspecialchars($productColor)) ?><?= $size ? '&size=' . htmlspecialchars($size) : '' ?>" 
                                    style="display: inline-block; padding: 5px 10px; margin-right: 5px; margin-bottom: 5px; background-color: <?= strtolower($productColor) === strtolower($color ?? '') ? '#007bff' : '#f8f9fa' ?>; color: <?= strtolower($productColor) === strtolower($color ?? '') ? 'white' : '#333' ?>; border-radius: 4px; text-decoration: none;">
                                    <?= htmlspecialchars($productColor) ?>
                                </a>
                            <?php endforeach; ?>
                        </div>
                    </div>
                    <?php endif; ?>
                    
                    <?php if (!empty($productData['sizes']) && $productData['sizes'][0] !== 'One Size'): ?>
                    <div>
                        <label style="display: block; margin-bottom: 5px; font-weight: bold;">Size:</label>
                        <div>
                            <?php foreach ($productData['sizes'] as $productSize): ?>
                                <a href="/products/<?= htmlspecialchars($productID) ?>?size=<?= strtolower(htmlspecialchars($productSize)) ?><?= $color ? '&color=' . htmlspecialchars($color) : '' ?>" 
                                    style="display: inline-block; padding: 5px 10px; margin-right: 5px; margin-bottom: 5px; background-color: <?= strtolower($productSize) === strtolower($size ?? '') ? '#007bff' : '#f8f9fa' ?>; color: <?= strtolower($productSize) === strtolower($size ?? '') ? 'white' : '#333' ?>; border-radius: 4px; text-decoration: none;">
                                    <?= htmlspecialchars($productSize) ?>
                                </a>
                            <?php endforeach; ?>
                        </div>
                    </div>
                    <?php endif; ?>
                </div>
                
                <div style="flex: 2; min-width: 300px;">
                    <h2><?= htmlspecialchars($productData['name']) ?></h2>
                    
                    <div style="margin-bottom: 15px; font-size: 24px; color: #007bff; font-weight: bold;">
                        <?= formatPrice($productData['price']) ?>
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <span style="display: inline-block; padding: 5px 10px; background-color: #e9ecef; border-radius: 4px; font-size: 0.9em;">
                            <?= htmlspecialchars($productData['category']) ?>
                        </span>
                        
                        <?php if ($ownerView): ?>
                        <span style="display: inline-block; padding: 5px 10px; background-color: #28a745; border-radius: 4px; font-size: 0.9em; color: white; margin-left: 5px;">
                            Owner View
                        </span>
                        <?php endif; ?>
                    </div>
                    
                    <p style="line-height: 1.6;"><?= htmlspecialchars($productData['description']) ?></p>
                    
                    <div style="margin-top: 20px;">
                        <button style="padding: 10px 20px; background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; margin-right: 10px;">
                            Add to Cart
                        </button>
                        <button style="padding: 10px 20px; background-color: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px;">
                            Save for Later
                        </button>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="card">
            <h2>URL Parameters</h2>
            <p>This page demonstrates both path parameters and query parameters:</p>
            
            <h3>Path Parameter</h3>
            <ul>
                <li><strong>productID:</strong> <?= htmlspecialchars($productID) ?></li>
            </ul>
            
            <h3>Query Parameters</h3>
            <ul>
                <li><strong>color:</strong> <?= $color ? htmlspecialchars($color) : '<em>not set</em>' ?></li>
                <li><strong>size:</strong> <?= $size ? htmlspecialchars($size) : '<em>not set</em>' ?></li>
                <li><strong>owner:</strong> <?= $ownerView ? 'true' : '<em>not set or false</em>' ?></li>
            </ul>
            
            <h3>How It Works</h3>
            <p>When you navigate to <code>/products/abc?color=red&size=large</code>, Frango:</p>
            <ol>
                <li>Extracts <code>abc</code> from the path and makes it available as <code>$_GET['productID']</code></li>
                <li>Extracts query parameters and populates <code>$_GET['color']</code> and <code>$_GET['size']</code></li>
            </ol>
            
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// Access path and query parameters from $_GET:
$productID = $_GET['productID'] ?? 'default';
$color = $_GET['color'] ?? null;
$size = $_GET['size'] ?? null;

// Combine them in a database query:
$query = "SELECT * FROM products WHERE id = :id";
$params = ['id' => $productID];

if ($color) {
    $query .= " AND colors LIKE :color";
    $params['color'] = "%$color%";
}

if ($size) {
    $query .= " AND sizes LIKE :size";
    $params['size'] = "%$size%";
}
            </pre>
        </div>
        
        <div class="card">
            <h2>URL Components</h2>
            <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                <tr style="background: #f8f9fa;">
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Component</th>
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>REQUEST_URI</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['REQUEST_URI']) ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>QUERY_STRING</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['QUERY_STRING'] ?? '') ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>$_GET</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><pre><?php var_export($_GET); ?></pre></td>
                </tr>
            </table>
        </div>
        
        <div class="card">
            <h2>Try Different Products</h2>
            <div style="display: flex; flex-wrap: wrap; gap: 10px;">
                <?php foreach ($products as $id => $product): ?>
                    <a href="/products/<?= htmlspecialchars($id) ?>" style="display: inline-block; text-decoration: none; color: inherit; width: 200px;">
                        <div style="background-color: #f8f9fa; padding: 15px; border-radius: 4px; height: 100%;">
                            <div style="background-color: #eee; height: 100px; display: flex; justify-content: center; align-items: center; margin-bottom: 10px;">
                                [<?= htmlspecialchars($product['image']) ?>]
                            </div>
                            <div style="font-weight: bold;"><?= htmlspecialchars($product['name']) ?></div>
                            <div style="color: #007bff;"><?= formatPrice($product['price']) ?></div>
                        </div>
                    </a>
                <?php endforeach; ?>
            </div>
        </div>
        
        <div class="card">
            <h2>Navigation</h2>
            <p><a href="/users/123?view=products">Back to User Profile</a></p>
            <p><a href="/categories/electronics/headphones">Browse Related Categories</a></p>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(dirname(__FILE__)) . '/debug_panel.php'; ?>
</body>
</html> 