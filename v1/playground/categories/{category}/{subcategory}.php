<?php
/**
 * Categories Page
 * Demonstrates multiple path parameters and segment extraction
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>Category: <?= $_PATH['category'] ?? 'All' ?> > <?= $_PATH['subcategory'] ?? 'All' ?></title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        pre {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        a { color: #3498db; }
        .breadcrumb {
            background: #eaf2f8;
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 15px;
        }
        .path-param {
            display: inline-block;
            padding: 3px 8px;
            background: #3498db;
            color: white;
            border-radius: 4px;
            margin: 0 5px;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>Categories Browser</h1>
        
        <div class="breadcrumb">
            <strong>You are browsing:</strong>
            <a href="/">Home</a> &raquo; 
            <a href="/categories/<?= $_PATH['category'] ?>">
                <span class="path-param"><?= htmlspecialchars($_PATH['category'] ?? 'all') ?></span>
            </a> &raquo; 
            <span class="path-param"><?= htmlspecialchars($_PATH['subcategory'] ?? 'all') ?></span>
        </div>
        
        <h3>Multiple Path Parameters:</h3>
        <pre><?php var_export($_PATH); ?></pre>
        
        <p>
            Notice how multiple path parameters have been extracted:
            <ul>
                <li>Category: <strong><?= htmlspecialchars($_PATH['category'] ?? 'not set') ?></strong></li>
                <li>Subcategory: <strong><?= htmlspecialchars($_PATH['subcategory'] ?? 'not set') ?></strong></li>
            </ul>
        </p>
        
        <h3>Path Segments:</h3>
        <pre><?php var_export($_PATH_SEGMENTS); ?></pre>
    </div>
    
    <div class="card">
        <h3>Try Other Categories:</h3>
        <ul>
            <li><a href="/categories/electronics/smartphones">Electronics > Smartphones</a></li>
            <li><a href="/categories/electronics/tablets">Electronics > Tablets</a></li>
            <li><a href="/categories/clothing/shirts">Clothing > Shirts</a></li>
            <li><a href="/categories/books/fiction">Books > Fiction</a></li>
        </ul>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/">Back to Home</a></p>
        <p><a href="/nested/deep/path">Go to Nested Path Example</a></p>
    </div>
</body>
</html> 