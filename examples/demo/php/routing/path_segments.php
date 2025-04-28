<?php
/**
 * Frango v2 Demo - Path Segments
 * 
 * Demonstrates access to raw path segments from URLs like
 * /segments/show/one/two/three
 */

// Access the raw path segments - these are provided by Frango middleware
$segments = $GLOBALS['_PATH_SEGMENTS'] ?? [];

// CSS classes for styling the segments
$colorClasses = [
    'bg-primary',   // Blue
    'bg-success',   // Green
    'bg-warning',   // Yellow
    'bg-danger',    // Red
    'bg-info',      // Light blue
    'bg-secondary', // Gray
    'bg-dark'       // Dark
];

function getSegmentColor($index) {
    global $colorClasses;
    return $colorClasses[$index % count($colorClasses)];
}

?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Path Segments Demo - Frango v2</title>
    <link rel="stylesheet" href="/assets/style.css">
    <style>
        .path-segment {
            display: inline-block;
            padding: 6px 12px;
            margin: 3px;
            color: white;
            border-radius: 4px;
        }
        .bg-primary { background-color: #007bff; }
        .bg-success { background-color: #28a745; }
        .bg-warning { background-color: #ffc107; color: #212529; }
        .bg-danger { background-color: #dc3545; }
        .bg-info { background-color: #17a2b8; }
        .bg-secondary { background-color: #6c757d; }
        .bg-dark { background-color: #343a40; }
        
        .segment-arrow {
            display: inline-block;
            font-size: 20px;
            margin: 0 5px;
            color: #6c757d;
        }
        
        .segment-visual {
            display: flex;
            align-items: center;
            flex-wrap: wrap;
            margin: 20px 0;
        }
        
        table.code-example {
            width: 100%;
            margin: 20px 0;
            border-collapse: collapse;
        }
        
        table.code-example tr:nth-child(odd) {
            background-color: #f8f9fa;
        }
        
        table.code-example td {
            padding: 8px;
            border: 1px solid #dee2e6;
        }
        
        table.code-example td:first-child {
            width: 50px;
            text-align: center;
            color: #6c757d;
            font-family: monospace;
        }
        
        table.code-example td:last-child {
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Path Segments Demo</h1>
        <p>Demonstrating access to raw URL path segments using <code>$GLOBALS['_PATH_SEGMENTS']</code>.</p>
        
        <div class="card">
            <h2>Your Current URL Path</h2>
            
            <div class="segment-visual">
                <span class="path-segment bg-dark">
                    <i>root</i>
                </span>
                <span class="segment-arrow">→</span>
                
                <?php foreach ($segments as $index => $segment): ?>
                    <span class="path-segment <?= getSegmentColor($index) ?>">
                        <?= htmlspecialchars($segment) ?>
                    </span>
                    
                    <?php if ($index < count($segments) - 1): ?>
                        <span class="segment-arrow">→</span>
                    <?php endif; ?>
                <?php endforeach; ?>
            </div>
            
            <h3>Path Segments Array</h3>
            <p>The raw segments are available in <code>$GLOBALS['_PATH_SEGMENTS']</code>:</p>
            <pre><?php var_export($segments); ?></pre>
            
            <h3>Access Individual Segments</h3>
            <table class="code-example">
                <tr>
                    <td>1</td>
                    <td><code>$segments = $GLOBALS['_PATH_SEGMENTS'] ?? [];</code></td>
                </tr>
                <tr>
                    <td>2</td>
                    <td><code>// Get specific segments by index</code></td>
                </tr>
                <tr>
                    <td>3</td>
                    <td><code>$firstSegment = $segments[0] ?? null;  // The first segment</code></td>
                </tr>
                <tr>
                    <td>4</td>
                    <td><code>$secondSegment = $segments[1] ?? null; // The second segment</code></td>
                </tr>
                <tr>
                    <td>5</td>
                    <td><code>$lastSegment = end($segments);        // The last segment</code></td>
                </tr>
            </table>
        </div>
        
        <div class="card">
            <h2>Path Segment Information</h2>
            
            <?php if (empty($segments)): ?>
                <p>No path segments found in the current URL.</p>
            <?php else: ?>
                <table style="width: 100%; border-collapse: collapse;">
                    <tr style="background-color: #f8f9fa;">
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Index</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Segment</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Access Code</th>
                    </tr>
                    
                    <?php foreach ($segments as $index => $segment): ?>
                        <tr>
                            <td style="padding: 8px; border: 1px solid #dee2e6;">
                                <?= $index ?>
                            </td>
                            <td style="padding: 8px; border: 1px solid #dee2e6;">
                                <span class="path-segment <?= getSegmentColor($index) ?>" style="margin: 0;">
                                    <?= htmlspecialchars($segment) ?>
                                </span>
                            </td>
                            <td style="padding: 8px; border: 1px solid #dee2e6; font-family: monospace;">
                                $segments[<?= $index ?>]
                            </td>
                        </tr>
                    <?php endforeach; ?>
                </table>
            <?php endif; ?>
            
            <h3>How It Works</h3>
            <p>When you navigate to a URL like <code>/segments/show/one/two/three</code>, Frango:</p>
            <ol>
                <li>Processes the URL path</li>
                <li>Splits it into individual segments</li>
                <li>Makes the segments available as an array in <code>$GLOBALS['_PATH_SEGMENTS']</code></li>
            </ol>
            
            <p>This is useful when you need access to the raw path components, especially when working with REST-like APIs or custom routing logic.</p>
            
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// In main.go:
mux.Handle("/segments/", middleware.For("/routing/path_segments.php"))

// In PHP:
$segments = $GLOBALS['_PATH_SEGMENTS'] ?? [];

// Use these variables in your code
$action = $segments[1] ?? 'default';
$resourceId = $segments[2] ?? null;
            </pre>
        </div>
        
        <div class="card">
            <h2>Try Different URLs</h2>
            <p>Test path segment extraction with these example URLs:</p>
            
            <ul>
                <li><a href="/segments/show/users/profile/settings">Example 1: /segments/show/users/profile/settings</a></li>
                <li><a href="/segments/show/products/electronics/smartphones">Example 2: /segments/show/products/electronics/smartphones</a></li>
                <li><a href="/segments/show/blog/2023/12/25/hello-world">Example 3: /segments/show/blog/2023/12/25/hello-world</a></li>
                <li><a href="/segments/show">Example 4: /segments/show (minimal segments)</a></li>
            </ul>
            
            <p>You can also create your own URLs by adding more segments after <code>/segments/show/</code></p>
        </div>
        
        <div class="card">
            <h2>Path Segments vs. Path Parameters</h2>
            <p>Frango v2 offers two ways to access URL path components:</p>
            
            <div style="display: flex; gap: 20px; flex-wrap: wrap; margin: 20px 0;">
                <div style="flex: 1; min-width: 300px; background-color: #f8f9fa; padding: 15px; border-radius: 4px;">
                    <h3>Path Segments (Raw)</h3>
                    <p>Access all segments of the URL path as an array.</p>
                    <pre style="background: #eee; padding: 10px; border-radius: 4px; margin: 10px 0;">
$segments = $GLOBALS['_PATH_SEGMENTS'];
$firstSegment = $segments[0] ?? null;</pre>
                    <p><strong>Best for:</strong> Custom routing logic, REST-like APIs, or when you need access to the entire path.</p>
                </div>
                
                <div style="flex: 1; min-width: 300px; background-color: #f8f9fa; padding: 15px; border-radius: 4px;">
                    <h3>Path Parameters (Named)</h3>
                    <p>Access named parameters extracted from the URL path.</p>
                    <pre style="background: #eee; padding: 10px; border-radius: 4px; margin: 10px 0;">
$userId = $_GET['userID'];
$category = $_GET['category'];</pre>
                    <p><strong>Best for:</strong> When you have specific named parameters in your URL patterns that you want to extract by name.</p>
                </div>
            </div>
            
            <p>See examples:</p>
            <ul>
                <li><a href="/users/123">User Profile (Named Parameters: /users/{userID})</a></li>
                <li><a href="/categories/electronics/smartphones">Categories (Named Parameters: /categories/{category}/{subcategory})</a></li>
            </ul>
        </div>
        
        <div class="card">
            <h2>Navigation</h2>
            <p><a href="/products/abc">View Product Details</a></p>
            <p><a href="/users/123">View User Profile</a></p>
            <p><a href="/categories/electronics/smartphones">Browse Categories</a></p>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(dirname(__FILE__)) . '/debug_panel.php'; ?>
</body>
</html> 