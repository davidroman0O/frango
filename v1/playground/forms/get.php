<?php
/**
 * GET Form Handler
 * 
 * Demonstrates using $_GET with the improved form handling
 */

// Initialize superglobals if they don't exist
if (!isset($_QUERY)) $_QUERY = isset($_GET) ? $_GET : [];

// Check request headers for debugging
$requestURI = $_SERVER['REQUEST_URI'] ?? 'Not available';
$queryString = $_SERVER['QUERY_STRING'] ?? 'Not available';
$requestMethod = $_SERVER['REQUEST_METHOD'] ?? 'Unknown';
?>
<!DOCTYPE html>
<html>
<head>
    <title>GET Form Results</title>
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
        h2 { color: #3498db; }
        pre {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        a { color: #3498db; }
        .result-item {
            padding: 10px;
            margin: 10px 0;
            border-left: 4px solid #3498db;
            background: #eaf2f8;
        }
        .explanation {
            background: #f9f9f9;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
            border-left: 4px solid #2ecc71;
        }
        .warning-banner {
            background: #fef9e7;
            color: #b7950b;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            border: 1px solid #b7950b;
        }
        .debug-info {
            background: #f5eef8;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
            border-left: 4px solid #8e44ad;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>GET Form Results</h1>
        
        <?php if (empty($_GET) && empty($_QUERY)): ?>
            <div class="warning-banner">
                <h3>No GET parameters detected</h3>
                <p>The URL should contain query parameters (e.g., ?name=value)</p>
            </div>
        <?php else: ?>
            <h2>Submitted GET Data</h2>
            
            <div class="result-item">
                <h3>$_GET Superglobal:</h3>
                <pre><?php var_export($_GET); ?></pre>
            </div>
            
            <div class="result-item">
                <h3>Individual GET Parameters:</h3>
                <ul>
                    <?php foreach ($_GET as $key => $value): ?>
                        <li><strong><?= htmlspecialchars($key) ?>:</strong> <?= htmlspecialchars($value) ?></li>
                    <?php endforeach; ?>
                </ul>
            </div>
            
            <div class="result-item">
                <h3>$_QUERY Superglobal (Alias to $_GET):</h3>
                <pre><?php var_export($_QUERY); ?></pre>
            </div>
            
            <div class="result-item">
                <h3>$_REQUEST Superglobal:</h3>
                <pre><?php var_export($_REQUEST); ?></pre>
            </div>
        <?php endif; ?>
        
        <div class="debug-info">
            <h3>Request Debug Information:</h3>
            <ul>
                <li><strong>Request Method:</strong> <?= htmlspecialchars($requestMethod) ?></li>
                <li><strong>Request URI:</strong> <?= htmlspecialchars($requestURI) ?></li>
                <li><strong>Query String:</strong> <?= htmlspecialchars($queryString) ?></li>
            </ul>
            <h4>Request Headers:</h4>
            <pre><?php 
                $headers = [];
                foreach ($_SERVER as $key => $value) {
                    if (str_starts_with($key, 'HTTP_') || str_starts_with($key, 'PHP_HEADER_')) {
                        $headers[$key] = $value;
                    }
                }
                var_export($headers);
            ?></pre>
        </div>
        
        <div class="explanation">
            <p>Notice that with Frango v1's improved form handling:</p>
            <ul>
                <li><code>$_GET</code> automatically contains all the URL query parameters</li>
                <li><code>$_QUERY</code> is an alias to <code>$_GET</code> for convenience</li>
                <li><code>$_REQUEST</code> contains all GET, POST, and COOKIE data, just like standard PHP</li>
            </ul>
        </div>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms/">Back to Forms</a></p>
        <p><a href="/">Back to Home</a></p>
        <p><a href="/forms/post">Try POST form example</a></p>
    </div>
    
    <?php include_once(__DIR__ . '/../debug_panel.php'); ?>
</body>
</html> 