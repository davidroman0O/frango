<?php
/**
 * Deeply Nested Path Example
 * Demonstrates how path segments are handled in deeply nested paths
 */

// Initialize superglobals if they don't exist
if (!isset($_PATH)) $_PATH = [];
if (!isset($_PATH_SEGMENTS)) $_PATH_SEGMENTS = [];
if (!isset($_PATH_SEGMENT_COUNT)) $_PATH_SEGMENT_COUNT = 0;
if (!isset($_URL)) $_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';

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
    <title>Deeply Nested Path Example</title>
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
        .segment {
            display: inline-block;
            padding: 4px 8px;
            background: #3498db;
            color: white;
            border-radius: 4px;
            margin: 0 2px;
        }
        .path-viz {
            margin: 15px 0;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>Deeply Nested Path</h1>
        <h3>Current URL: <?= htmlspecialchars($_URL) ?></h3>
        
        <div class="path-viz">
            <div>Path visualization:</div>
            <div>
                <?php foreach ($_PATH_SEGMENTS as $index => $segment): ?>
                    <?php if ($index > 0): ?> / <?php endif; ?>
                    <span class="segment"><?= htmlspecialchars($segment) ?></span>
                <?php endforeach; ?>
            </div>
        </div>
        
        <h3>Path Segments:</h3>
        <pre><?php var_export($_PATH_SEGMENTS); ?></pre>
        
        <p>Segment count: <strong><?= $_PATH_SEGMENT_COUNT ?></strong></p>
        
        <h3>Query Parameters:</h3>
        <pre><?php var_export($_GET); ?></pre>
    </div>
    
    <div class="card">
        <h3>Try These Related Paths:</h3>
        <ul>
            <li><a href="/nested">/nested</a> - One level deep</li>
            <li><a href="/nested/deep">/nested/deep</a> - Two levels deep</li>
            <li><a href="/nested/deep/path">/nested/deep/path</a> - Current page</li>
            <li><a href="/nested/deep/path/extra">/nested/deep/path/extra</a> - Extra segment</li>
        </ul>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/">Back to Home</a></p>
        <?php if (isset($_GET['id'])): ?>
            <p><a href="/products/<?= htmlspecialchars($_GET['id']) ?>">Back to Product #<?= htmlspecialchars($_GET['id']) ?></a></p>
        <?php endif; ?>
        <p><a href="/categories/electronics/laptops">Browse Categories</a></p>
        <p><a href="/forms/">Try Form Handling Examples</a></p>
    </div>
    
    <?php include_once(__DIR__ . '/../../../debug_panel.php'); ?>
</body>
</html> 