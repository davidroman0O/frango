<?php
/**
 * Frango v2 Demo - Category Page
 * 
 * Demonstrates extraction of multiple path parameters from URLs 
 * like /categories/{category}/{subcategory}
 */

// Extract path parameters from the URL
// For a URL like /categories/electronics/smartphones, these parameters should be available in $_GET
$category = $_GET['category'] ?? 'all';
$subcategory = $_GET['subcategory'] ?? 'all';

// Sample category data - in a real app this would come from a database
$categories = [
    'electronics' => [
        'name' => 'Electronics',
        'description' => 'Electronic devices and accessories',
        'subcategories' => [
            'smartphones' => [
                'name' => 'Smartphones',
                'products' => ['iPhone 13', 'Samsung Galaxy S21', 'Google Pixel 6'],
                'filters' => ['Brand', 'Price', 'Screen Size', 'Camera']
            ],
            'laptops' => [
                'name' => 'Laptops',
                'products' => ['MacBook Pro', 'Dell XPS', 'Lenovo ThinkPad'],
                'filters' => ['Brand', 'Price', 'CPU', 'RAM', 'Storage']
            ],
            'tablets' => [
                'name' => 'Tablets',
                'products' => ['iPad Pro', 'Samsung Galaxy Tab', 'Amazon Fire'],
                'filters' => ['Brand', 'Price', 'Screen Size']
            ]
        ]
    ],
    'clothing' => [
        'name' => 'Clothing',
        'description' => 'Apparel and fashion items',
        'subcategories' => [
            'shirts' => [
                'name' => 'Shirts',
                'products' => ['Cotton T-Shirt', 'Button-up Oxford', 'Polo Shirt'],
                'filters' => ['Size', 'Color', 'Material', 'Brand']
            ],
            'pants' => [
                'name' => 'Pants',
                'products' => ['Jeans', 'Khakis', 'Sweatpants'],
                'filters' => ['Size', 'Color', 'Material', 'Style']
            ],
            'shoes' => [
                'name' => 'Shoes',
                'products' => ['Running Shoes', 'Dress Shoes', 'Sandals'],
                'filters' => ['Size', 'Color', 'Brand', 'Type']
            ]
        ]
    ],
    'books' => [
        'name' => 'Books',
        'description' => 'Books in various formats and genres',
        'subcategories' => [
            'fiction' => [
                'name' => 'Fiction',
                'products' => ['The Great Gatsby', '1984', 'To Kill a Mockingbird'],
                'filters' => ['Author', 'Format', 'Price', 'Rating']
            ],
            'non-fiction' => [
                'name' => 'Non-Fiction',
                'products' => ['Sapiens', 'The Power of Habit', 'Atomic Habits'],
                'filters' => ['Author', 'Format', 'Subject', 'Price']
            ],
            'textbooks' => [
                'name' => 'Textbooks',
                'products' => ['Introduction to Algorithms', 'Principles of Economics', 'Psychology 101'],
                'filters' => ['Subject', 'Edition', 'Price', 'Condition']
            ]
        ]
    ]
];

// Get current category data
$categoryData = $categories[$category] ?? [
    'name' => ucfirst($category),
    'description' => 'Category not found',
    'subcategories' => []
];

// Get current subcategory data
$subcategoryData = null;
if ($subcategory !== 'all' && isset($categoryData['subcategories'][$subcategory])) {
    $subcategoryData = $categoryData['subcategories'][$subcategory];
} else {
    $subcategoryData = [
        'name' => $subcategory === 'all' ? 'All Subcategories' : ucfirst($subcategory),
        'products' => [],
        'filters' => []
    ];
}

// Access the raw path segments if needed
$pathSegments = $GLOBALS['_PATH_SEGMENTS'] ?? [];
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Category: <?= htmlspecialchars($categoryData['name']) ?> - <?= htmlspecialchars($subcategoryData['name']) ?></title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>Category Browser</h1>
        <p>Demonstrating multiple path parameter extraction from URLs like <code>/categories/{category}/{subcategory}</code>.</p>
        
        <div class="card">
            <div style="margin-bottom: 20px;">
                <nav style="display: flex; align-items: center; background-color: #f8f9fa; padding: 10px; border-radius: 4px;">
                    <span style="margin-right: 10px;"><strong>You are browsing:</strong></span>
                    <a href="/" style="margin-right: 10px;">Home</a> &raquo;
                    
                    <a href="/categories/<?= htmlspecialchars($category) ?>" style="margin: 0 10px;">
                        <span style="display: inline-block; padding: 5px 10px; background-color: #007bff; color: white; border-radius: 4px;">
                            <?= htmlspecialchars($categoryData['name']) ?>
                        </span>
                    </a> &raquo;
                    
                    <span style="display: inline-block; padding: 5px 10px; background-color: #6c757d; color: white; border-radius: 4px; margin-left: 10px;">
                        <?= htmlspecialchars($subcategoryData['name']) ?>
                    </span>
                </nav>
            </div>
            
            <?php if ($subcategory === 'all'): ?>
                <h2><?= htmlspecialchars($categoryData['name']) ?></h2>
                <p><?= htmlspecialchars($categoryData['description']) ?></p>
                
                <h3>Subcategories</h3>
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 15px; margin-top: 15px;">
                    <?php foreach ($categoryData['subcategories'] as $subcat => $subcatData): ?>
                        <a href="/categories/<?= htmlspecialchars($category) ?>/<?= htmlspecialchars($subcat) ?>" 
                           style="text-decoration: none; color: inherit;">
                            <div style="background-color: #f8f9fa; padding: 15px; border-radius: 4px; height: 100%; border: 1px solid #dee2e6;">
                                <h4 style="margin-top: 0; color: #007bff;"><?= htmlspecialchars($subcatData['name']) ?></h4>
                                <p><?= count($subcatData['products']) ?> products</p>
                            </div>
                        </a>
                    <?php endforeach; ?>
                </div>
            <?php else: ?>
                <h2><?= htmlspecialchars($subcategoryData['name']) ?></h2>
                <p>Browsing <?= htmlspecialchars($subcategoryData['name']) ?> in <?= htmlspecialchars($categoryData['name']) ?></p>
                
                <div style="display: flex; flex-wrap: wrap; gap: 20px; margin-top: 20px;">
                    <div style="flex: 1; min-width: 200px;">
                        <h3>Filters</h3>
                        <?php if (!empty($subcategoryData['filters'])): ?>
                            <div style="background-color: #f8f9fa; padding: 15px; border-radius: 4px;">
                                <?php foreach ($subcategoryData['filters'] as $filter): ?>
                                    <div style="margin-bottom: 15px;">
                                        <h4 style="margin-top: 0; margin-bottom: 10px;"><?= htmlspecialchars($filter) ?></h4>
                                        <div>
                                            <label style="display: block; margin-bottom: 5px;">
                                                <input type="checkbox"> Option 1
                                            </label>
                                            <label style="display: block; margin-bottom: 5px;">
                                                <input type="checkbox"> Option 2
                                            </label>
                                            <label style="display: block;">
                                                <input type="checkbox"> Option 3
                                            </label>
                                        </div>
                                    </div>
                                <?php endforeach; ?>
                            </div>
                        <?php else: ?>
                            <p>No filters available</p>
                        <?php endif; ?>
                    </div>
                    
                    <div style="flex: 3; min-width: 300px;">
                        <h3>Products</h3>
                        <?php if (!empty($subcategoryData['products'])): ?>
                            <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 15px;">
                                <?php foreach ($subcategoryData['products'] as $product): ?>
                                    <div style="background-color: #f8f9fa; padding: 15px; border-radius: 4px; height: 100%; border: 1px solid #dee2e6;">
                                        <h4 style="margin-top: 0; color: #007bff;"><?= htmlspecialchars($product) ?></h4>
                                        <p>$<?= rand(10, 999) ?>.<?= rand(10, 99) ?></p>
                                        <button style="padding: 5px 10px; background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
                                            Add to Cart
                                        </button>
                                    </div>
                                <?php endforeach; ?>
                            </div>
                        <?php else: ?>
                            <p>No products available in this category</p>
                        <?php endif; ?>
                    </div>
                </div>
            <?php endif; ?>
        </div>
        
        <div class="card">
            <h2>Multiple Path Parameters</h2>
            <p>This page demonstrates extracting multiple parameters from a URL path:</p>
            
            <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                <tr style="background: #f8f9fa;">
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Parameter</th>
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>category</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($category) ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>subcategory</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($subcategory) ?></td>
                </tr>
            </table>
            
            <h3>Raw $_GET Superglobal</h3>
            <pre><?php var_export($_GET); ?></pre>
            
            <h3>Path Segments ($GLOBALS['_PATH_SEGMENTS'])</h3>
            <pre><?php var_export($pathSegments); ?></pre>
            
            <h3>How It Works</h3>
            <p>When you navigate to <code>/categories/electronics/smartphones</code>, Frango:</p>
            <ol>
                <li>Matches the URL pattern <code>/categories/{category}/{subcategory}</code></li>
                <li>Extracts <code>electronics</code> and makes it available as <code>$_GET['category']</code></li>
                <li>Extracts <code>smartphones</code> and makes it available as <code>$_GET['subcategory']</code></li>
            </ol>
            
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// In main.go:
mux.Handle("/categories/", middleware.For("/routing/category_page.php"))

// In PHP:
$category = $_GET['category'] ?? 'all';
$subcategory = $_GET['subcategory'] ?? 'all';

// Use these variables in your code
$categoryData = getCategoryData($category, $subcategory);
            </pre>
        </div>
        
        <div class="card">
            <h2>Try Other Categories</h2>
            <ul>
                <?php foreach ($categories as $catKey => $catData): ?>
                    <li>
                        <a href="/categories/<?= htmlspecialchars($catKey) ?>">
                            <strong><?= htmlspecialchars($catData['name']) ?></strong>
                        </a>
                        <ul>
                            <?php foreach ($catData['subcategories'] as $subcatKey => $subcatData): ?>
                                <li>
                                    <a href="/categories/<?= htmlspecialchars($catKey) ?>/<?= htmlspecialchars($subcatKey) ?>">
                                        <?= htmlspecialchars($subcatData['name']) ?>
                                    </a>
                                </li>
                            <?php endforeach; ?>
                        </ul>
                    </li>
                <?php endforeach; ?>
            </ul>
        </div>
        
        <div class="card">
            <h2>Navigation</h2>
            <p><a href="/products/abc">View Product Details</a></p>
            <p><a href="/users/123">View User Profile</a></p>
            <p><a href="/segments/show/one/two/three">Try Path Segments Demo</a></p>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(dirname(__FILE__)) . '/debug_panel.php'; ?>
</body>
</html> 