<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - API Example</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .card {
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 15px;
            margin-bottom: 20px;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        code {
            background: #f6f8fa;
            padding: 2px 4px;
            border-radius: 3px;
        }
        .result {
            background: #f6f8fa;
            padding: 10px;
            border-radius: 4px;
            max-height: 300px;
            overflow: auto;
            white-space: pre-wrap;
        }
        button {
            background: #0366d6;
            color: white;
            border: none;
            padding: 8px 12px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #0259c4;
        }
    </style>
</head>
<body>
    <h1>Frango - API Example</h1>
    
    <div class="card">
        <h2>Go-PHP API Integration</h2>
        <p>This example demonstrates how to create API endpoints in both Go and PHP, and how to call between them.</p>
    </div>

    <div class="card">
        <h2>Available API Endpoints</h2>
        <h3>PHP Endpoints:</h3>
        <ul>
            <li><a href="/api/php/products" target="_blank">/api/php/products</a> - List all products (PHP)</li>
            <li><a href="/api/php/products/1" target="_blank">/api/php/products/1</a> - Get product with ID 1 (PHP)</li>
        </ul>
        
        <h3>Go Endpoints:</h3>
        <ul>
            <li><a href="/api/go/products" target="_blank">/api/go/products</a> - List all products (Go)</li>
            <li><a href="/api/go/products/1" target="_blank">/api/go/products/1</a> - Get product with ID 1 (Go)</li>
        </ul>
    </div>

    <div class="card">
        <h2>Test Go-to-PHP Communication</h2>
        <p>Click the button to make a request from this PHP script to the Go API endpoint:</p>
        <button onclick="fetchGoApi()">Fetch from Go API</button>
        <div id="go-result" class="result" style="margin-top: 10px; display: none;"></div>
    </div>

    <div class="card">
        <h2>Test PHP-to-PHP Communication</h2>
        <p>Click the button to make a request from this PHP script to the PHP API endpoint:</p>
        <button onclick="fetchPhpApi()">Fetch from PHP API</button>
        <div id="php-result" class="result" style="margin-top: 10px; display: none;"></div>
    </div>

    <script>
        async function fetchGoApi() {
            const resultElement = document.getElementById('go-result');
            resultElement.style.display = 'block';
            resultElement.textContent = 'Loading...';
            
            try {
                const response = await fetch('/api/go/products');
                const data = await response.json();
                resultElement.textContent = JSON.stringify(data, null, 2);
            } catch (error) {
                resultElement.textContent = 'Error: ' + error.message;
            }
        }
        
        async function fetchPhpApi() {
            const resultElement = document.getElementById('php-result');
            resultElement.style.display = 'block';
            resultElement.textContent = 'Loading...';
            
            try {
                const response = await fetch('/api/php/products');
                const data = await response.json();
                resultElement.textContent = JSON.stringify(data, null, 2);
            } catch (error) {
                resultElement.textContent = 'Error: ' + error.message;
            }
        }
    </script>
</body>
</html> 