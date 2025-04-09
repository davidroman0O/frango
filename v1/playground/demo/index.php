<?php
/**
 * Frango Playground - Environment Variables Demo
 * 
 * This script displays all the special PHP superglobals and variables
 * that are injected by the Frango middleware.
 */

// Helper function to display a variable in a readable format
function display_var($name, $value, $description = '') {
    echo "<div class='var-item'>";
    echo "<h4>$name</h4>";
    if ($description) {
        echo "<div class='desc'>$description</div>";
    }
    echo "<pre>";
    if (is_array($value) || is_object($value)) {
        echo htmlspecialchars(var_export($value, true));
    } else {
        echo htmlspecialchars((string)$value);
    }
    echo "</pre>";
    echo "</div>";
}

// Initialize superglobals if they don't exist
if (!isset($_PATH)) $_PATH = [];
if (!isset($_PATH_SEGMENTS)) $_PATH_SEGMENTS = [];
if (!isset($_PATH_SEGMENT_COUNT)) $_PATH_SEGMENT_COUNT = 0;
if (!isset($_JSON)) $_JSON = [];
if (!isset($_FORM)) $_FORM = [];
if (!isset($_URL)) $_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_CURRENT_URL)) $_CURRENT_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_QUERY)) $_QUERY = isset($_GET) ? $_GET : [];

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
    <title>Frango PHP Environment Demo</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 10px;
            background: #f5f5f5;
            color: #333;
            font-size: 14px;
        }
        h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
        h2 { font-size: 1.2rem; margin: 0; padding: 0; }
        h4 { margin: 0; color: #3498db; font-size: 0.9rem; }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
            gap: 10px;
        }
        .main-container {
            display: grid;
            grid-template-columns: 2fr 1fr;
            gap: 10px;
        }
        .left-column {
            margin-right: 10px;
        }
        .right-column {
            position: sticky;
            top: 10px;
            align-self: flex-start;
            max-height: 98vh;
            overflow-y: auto;
        }
        details {
            background: white;
            border-radius: 4px;
            padding: 8px;
            box-shadow: 0 1px 2px rgba(0,0,0,0.1);
            margin-bottom: 10px;
        }
        summary {
            cursor: pointer;
            font-weight: bold;
        }
        .var-item {
            margin-bottom: 8px;
            background: #f8f9fa;
            border-radius: 3px;
            padding: 6px;
            border-left: 3px solid #3498db;
        }
        pre {
            background: #f0f0f0;
            padding: 6px;
            border-radius: 3px;
            overflow: auto;
            max-height: 120px;
            white-space: pre-wrap;
            margin: 4px 0 0 0;
            font-size: 12px;
        }
        .desc {
            color: #666;
            font-style: italic;
            font-size: 0.8rem;
            margin-bottom: 4px;
        }
        #controls {
            position: fixed;
            bottom: 10px;
            right: 10px;
            background: #fff;
            padding: 8px;
            border-radius: 4px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.2);
            z-index: 100;
        }
        button {
            background: #3498db;
            color: white;
            border: none;
            padding: 5px 10px;
            border-radius: 3px;
            cursor: pointer;
        }
        .env-var {
            background: #f8f9fa;
            border-radius: 3px;
            margin-bottom: 4px;
            font-size: 11px;
            line-height: 1.4;
        }
        .env-var-name {
            font-weight: bold;
            color: #2980b9;
        }
        .env-value {
            color: #333;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <h1>Frango PHP Environment Demo</h1>
    <p>This page demonstrates the PHP environment variables and superglobals that are automatically set up by Frango.</p>
    
    <div class="main-container">
        <div class="left-column">
            <div class="grid">
                <details open>
                    <summary>Path Parameters ($_PATH)</summary>
                    <?php display_var('$_PATH', $_PATH, 'Path parameters extracted from URL patterns'); ?>
                </details>
                
                <details open>
                    <summary>Path Segments ($_PATH_SEGMENTS)</summary>
                    <?php display_var('$_PATH_SEGMENTS', $_PATH_SEGMENTS, 'URL path segments split by "/"'); ?>
                    <?php display_var('$_PATH_SEGMENT_COUNT', $_PATH_SEGMENT_COUNT, 'Count'); ?>
                </details>
            </div>
            
            <div class="grid">
                <details open>
                    <summary>JSON Data ($_JSON)</summary>
                    <?php display_var('$_JSON', $_JSON, 'Parsed JSON data from request body'); ?>
                </details>
                
                <details open>
                    <summary>Form Data ($_FORM)</summary>
                    <?php display_var('$_FORM', $_FORM, 'Form data from all request methods'); ?>
                </details>
            </div>
            
            <div class="grid">
                <details open>
                    <summary>URL and Query Data</summary>
                    <?php display_var('$_URL', $_URL, 'Current URL path'); ?>
                    <?php display_var('$_CURRENT_URL', $_CURRENT_URL, 'Full URL with query string'); ?>
                    <?php display_var('$_QUERY', $_QUERY, 'Query parameters'); ?>
                </details>
                
                <details open>
                    <summary>Helper Functions</summary>
                    <?php display_var('path_segments()', path_segments(), 'Get path segments'); ?>
                    <?php 
                    if (isset($_PATH['id'])) {
                        display_var('path_param("id")', path_param('id'), 'Get a path parameter');
                    } else {
                        echo "<p class='desc'>Add an 'id' path parameter to see path_param() in action</p>";
                    }
                    ?>
                    <?php display_var('has_path_param("id")', has_path_param('id') ? 'true' : 'false', 'Check if parameter exists'); ?>
                </details>
            </div>
            
            <details open>
                <summary>Test Routes</summary>
                <div class="grid">
                    <div class="var-item">
                        <h4>Test Path Parameters</h4>
                        <div class="desc">Click these links to test path parameters and nested routes</div>
                        <ul>
                            <li><a href="/users/123">/users/123</a> - User profile with ID parameter</li>
                            <li><a href="/products/456?color=red">/products/456?color=red</a> - Product with ID and query param</li>
                            <li><a href="/nested/deep/path">/nested/deep/path</a> - Deeply nested path</li>
                            <li><a href="/categories/electronics/laptops">/categories/electronics/laptops</a> - Multiple segments</li>
                            <li><a href="/forms">/forms/</a> - <strong>Form handling examples & tests</strong></li>
                        </ul>
                    </div>
                </div>
            </details>
            
            <details open>
                <summary>Standard PHP Superglobals</summary>
                <div class="grid">
                    <?php display_var('$_GET', $_GET, 'Query parameters'); ?>
                    <?php display_var('$_POST', $_POST, 'POST data'); ?>
                    <?php display_var('$_SERVER (filtered)', array_filter($_SERVER, function($k) {
                        return !str_starts_with($k, 'PHP_') && !str_starts_with($k, 'DEBUG_');
                    }, ARRAY_FILTER_USE_KEY), 'Server info'); ?>
                </div>
            </details>
        </div>
        
        <div class="right-column">
            <details open>
                <summary>PHP_ Environment Variables</summary>
                <div class="env-vars">
                    <?php
                    $phpVars = array_filter($_SERVER, function($k) {
                        return str_starts_with($k, 'PHP_');
                    }, ARRAY_FILTER_USE_KEY);
                    
                    ksort($phpVars);
                    
                    foreach ($phpVars as $key => $value):
                    ?>
                    <div class="env-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="env-value"><?= htmlspecialchars($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                </div>
            </details>
            
            <details open>
                <summary>Debug Environment Variables</summary>
                <div class="env-vars">
                    <?php
                    $debugVars = array_filter($_SERVER, function($k) {
                        return str_starts_with($k, 'DEBUG_');
                    }, ARRAY_FILTER_USE_KEY);
                    
                    ksort($debugVars);
                    
                    foreach ($debugVars as $key => $value):
                    ?>
                    <div class="env-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="env-value"><?= htmlspecialchars($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                </div>
            </details>
        </div>
    </div>
    
    <div id="controls">
        <button onclick="toggleSections()">Toggle All</button>
    </div>
    
    <script>
        function toggleSections() {
            document.querySelectorAll('details').forEach(detail => {
                detail.open = !detail.open;
            });
        }
    </script>
</body>
</html>