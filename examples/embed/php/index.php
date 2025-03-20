<?php
// Index page for the embed example
$timestamp = date('Y-m-d H:i:s');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Go-PHP Embed Example</title>
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
        pre {
            background: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .api-section {
            margin: 20px 0;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 4px;
            border-left: 4px solid #2196F3;
        }
        .btn {
            display: inline-block;
            padding: 8px 16px;
            background-color: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            margin-right: 10px;
            cursor: pointer;
        }
        .btn:hover {
            background-color: #45a049;
        }
        #responseContainer {
            margin-top: 15px;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            background-color: #fff;
            min-height: 100px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Go-PHP Embed Example</h1>
        
        <p>This example demonstrates embedding PHP files in a Go binary.</p>
        <p>Current time: <strong><?php echo $timestamp; ?></strong></p>
        
        <div class="api-section">
            <h2>API Endpoints</h2>
            <p>These endpoints are served from PHP files embedded in the Go binary:</p>
            
            <button class="btn" onclick="fetchEndpoint('/api/user')">Get User</button>
            <button class="btn" onclick="fetchEndpoint('/api/items')">Get Items</button>
            <button class="btn" onclick="fetchEndpoint('/api/time')">Get Time (Go)</button>
            
            <h3>Response:</h3>
            <pre id="responseContainer">Click a button to make a request...</pre>
        </div>
        
        <div class="api-section">
            <h2>Server Info</h2>
            <p>Information about the current PHP environment:</p>
            <ul>
                <li>PHP Version: <?php echo phpversion(); ?></li>
                <li>Document Root: <?php echo $_SERVER['DOCUMENT_ROOT']; ?></li>
                <li>Script File: <?php echo $_SERVER['SCRIPT_FILENAME']; ?></li>
                <li>Source File: <?php echo $_SERVER['GO_PHP_SOURCE_FILE'] ?? 'N/A'; ?></li>
            </ul>
        </div>
    </div>
    
    <script>
        async function fetchEndpoint(url) {
            const responseContainer = document.getElementById('responseContainer');
            responseContainer.textContent = 'Loading...';
            
            try {
                const response = await fetch(url);
                const data = await response.json();
                responseContainer.textContent = JSON.stringify(data, null, 2);
            } catch (error) {
                responseContainer.textContent = 'Error: ' + error.message;
            }
        }
    </script>
</body>
</html> 