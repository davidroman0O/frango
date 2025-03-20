<?php
// Check if we're in development mode
$devMode = empty($_SERVER['PHP_PRODUCTION']) || $_SERVER['PHP_PRODUCTION'] !== '1';

// Only disable caching in development mode
if ($devMode) {
    header("Cache-Control: no-store, no-cache, must-revalidate, max-age=0");
    header("Pragma: no-cache");
    header("Expires: 0");
}

// Current timestamp for debugging
$timestamp = date('Y-m-d H:i:s');
$mode = $devMode ? 'DEVELOPMENT' : 'PRODUCTION';

// Test require_once functionality
$testResult = require_once './test.php';
$testMessage = getTestMessage();

?>
<!DOCTYPE html>
<html>
<head>
    <title>Go-PHP Library Example</title>
    <?php if ($devMode): ?>
    <meta http-equiv="Cache-Control" content="no-cache, no-store, must-revalidate">
    <meta http-equiv="Pragma" content="no-cache">
    <meta http-equiv="Expires" content="0">
    <?php endif; ?>
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
        .success-banner {
            margin: 20px 0;
            padding: 15px;
            background-color: #dff0d8;
            border-radius: 4px;
            border-left: 4px solid #3c763d;
            color: #3c763d;
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
        <h1>Go-PHP Basic Example</h1>
        
        <p>Mode: <strong><?php echo $mode; ?></strong></p>
        <p>Current time: <strong><?php echo $timestamp; ?></strong></p>
        
        <?php if (isset($INCLUDED_TEST_FILE) && $INCLUDED_TEST_FILE === true): ?>
        <div class="success-banner">
            <h2>require_once Test: SUCCESS</h2>
            <p><?php echo $testMessage; ?></p>
            <p>Unique Marker: <?php echo $testInclusionMarker; ?></p>
            <p>Included at: <?php echo date('Y-m-d H:i:s', $testResult['timestamp']); ?></p>
        </div>
        <?php else: ?>
        <div class="error-banner">
            <h2>require_once Test: FAILED</h2>
            <p>The test.php file was not properly included!</p>
        </div>
        <?php endif; ?>
        
        <div class="api-section">
            <h2>API Endpoints</h2>
            <p>This example demonstrates calling multiple PHP endpoints registered with the Go-PHP library:</p>
            
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
                <li>Debug Doc Root: <?php echo $_SERVER['DEBUG_DOCUMENT_ROOT'] ?? 'N/A'; ?></li>
                <li>Debug Script Name: <?php echo $_SERVER['DEBUG_SCRIPT_NAME'] ?? 'N/A'; ?></li>
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