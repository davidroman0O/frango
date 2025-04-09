<?php
/**
 * Product Detail Page
 * Demonstrates path parameter extraction and query string parameters
 */

// Initialize superglobals if they don't exist
if (!isset($_PATH)) $_PATH = [];
if (!isset($_PATH_SEGMENTS)) $_PATH_SEGMENTS = [];
if (!isset($_PATH_SEGMENT_COUNT)) $_PATH_SEGMENT_COUNT = 0;

// Define helper functions if they don't exist
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

if (!function_exists('path_param')) {
    function path_param($name, $default = null) {
        global $_PATH;
        return isset($_PATH[$name]) ? $_PATH[$name] : $default;
    }
}

if (!function_exists('has_path_param')) {
    function has_path_param($name) {
        global $_PATH;
        return isset($_PATH[$name]);
    }
}
?>
<!DOCTYPE html>
<html>
<head>
    <title>Product #<?= $_PATH['id'] ?? 'unknown' ?></title>
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
        .product-card {
            border-left: 5px solid 
                <?= isset($_GET['color']) ? htmlspecialchars($_GET['color']) : '#3498db' ?>;
        }
        h1 { color: #2c3e50; }
        pre {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        a { color: #3498db; }
        .label {
            display: inline-block;
            padding: 2px 6px;
            background: #e0e0e0;
            border-radius: 4px;
            margin-right: 5px;
        }
    </style>
</head>
<body>
    <div class="card product-card">
        <h1>Product Details</h1>
        <h2>Product #<?= htmlspecialchars($_PATH['id'] ?? 'unknown') ?></h2>
        
        <?php if (isset($_GET['color'])): ?>
            <p>Selected color: <span class="label" style="background-color: <?= htmlspecialchars($_GET['color']) ?>">
                <?= htmlspecialchars($_GET['color']) ?>
            </span></p>
        <?php endif; ?>
        
        <h3>Path Parameters:</h3>
        <pre><?php var_export($_PATH); ?></pre>
        
        <h3>Query Parameters:</h3>
        <pre><?php var_export($_GET); ?></pre>
        
        <p>This page demonstrates how $_PATH captures URL parameters while $_GET captures query string parameters.</p>
    </div>
    
    <div class="card">
        <h3>Try Different Colors:</h3>
        <p>
            <a href="/products/<?= $_PATH['id'] ?? '1' ?>?color=red">Red</a> |
            <a href="/products/<?= $_PATH['id'] ?? '1' ?>?color=blue">Blue</a> |
            <a href="/products/<?= $_PATH['id'] ?? '1' ?>?color=green">Green</a> |
            <a href="/products/<?= $_PATH['id'] ?? '1' ?>?color=purple">Purple</a> |
            <a href="/products/<?= $_PATH['id'] ?? '1' ?>?color=orange">Orange</a>
        </p>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/">Back to Home</a></p>
        <p><a href="/nested/deep/path?ref=product&id=<?= $_PATH['id'] ?? '1' ?>">Go to Deep Nested Path</a></p>
        <p><a href="/forms/">Try Form Handling Examples</a></p>
    </div>
    
    <?php include_once(__DIR__ . '/../debug_panel.php'); ?>
</body>
</html> 