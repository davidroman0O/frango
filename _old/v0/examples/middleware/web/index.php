<?php
// Current timestamp for debugging
$timestamp = date('Y-m-d H:i:s');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Go-PHP Middleware Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }
        h1 {
            color: #333;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
        }
        .section {
            margin: 20px 0;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 4px;
            border-left: 4px solid #2196F3;
        }
        .links {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-top: 15px;
        }
        a {
            display: inline-block;
            padding: 8px 16px;
            background-color: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            margin: 3px;
        }
        a:hover {
            background-color: #45a049;
        }
        .category {
            margin-top: 15px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Go-PHP Middleware Example</h1>
        <p>This example demonstrates how to use Go-PHP library as middleware in an existing Go application.</p>
        <p>Current time: <strong><?php echo $timestamp; ?></strong></p>
        
        <div class="section">
            <h2>Available Routes</h2>
            
            <div class="category">PHP Routes (handled by Go-PHP):</div>
            <div class="links">
                <a href="/">PHP Home</a>
                <a href="/info">PHP Info</a>
                <a href="/api/user">API: User</a>
                <a href="/api/items">API: Items</a>
            </div>
            
            <div class="category">Go Routes (handled by Go directly):</div>
            <div class="links">
                <a href="/go/hello">Go: Hello</a>
                <a href="/go/time">Go: Time</a>
                <a href="/api/nonexistent">API: Missing Endpoint (Go fallback)</a>
            </div>
            
            <div class="category">Static Files:</div>
            <div class="links">
                <a href="/static/style.css">Static: CSS</a>
                <a href="/static/image.jpg">Static: Image</a>
            </div>
        </div>
        
        <div class="section">
            <h2>How It Works</h2>
            <p>The PHP server is used as middleware in a Go HTTP mux:</p>
            <ul>
                <li><code>/php/*</code> - Direct mapping to PHP files</li>
                <li><code>/api/*</code> - PHP first, falling back to Go</li>
                <li><code>/go/*</code> - Go handlers only</li>
                <li><code>/static/*</code> - Static file server</li>
            </ul>
            <p>This allows you to gradually migrate from PHP to Go or to mix both for different parts of your application.</p>
        </div>
    </div>
</body>
</html> 