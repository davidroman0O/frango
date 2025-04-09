<?php
/**
 * User Profile Page
 * Demonstrates path parameter extraction from URL patterns like /users/{id}
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
    <title>User Profile - ID: <?= $_PATH['id'] ?? 'unknown' ?></title>
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
    </style>
</head>
<body>
    <div class="card">
        <h1>User Profile</h1>
        <h2>User ID: <?= htmlspecialchars($_PATH['id'] ?? 'unknown') ?></h2>
        
        <h3>Path Parameters Extracted:</h3>
        <pre><?php var_export($_PATH); ?></pre>
        
        <h3>URL Segments:</h3>
        <pre><?php var_export($_PATH_SEGMENTS); ?></pre>
        
        <p>Notice how the ID parameter was automatically extracted from the URL!</p>
    </div>
    
    <div class="card">
        <h3>Path Helper Functions:</h3>
        <p>User ID from path_param(): <strong><?= path_param('id', 'not found') ?></strong></p>
        <p>Does 'username' parameter exist? <strong><?= has_path_param('username') ? 'Yes' : 'No' ?></strong></p>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/">Back to Home</a></p>
        <p><a href="/products/<?= $_PATH['id'] ?? '1' ?>">View User's Products</a></p>
    </div>
    
    <?php include_once(__DIR__ . '/../debug_panel.php'); ?>
</body>
</html> 