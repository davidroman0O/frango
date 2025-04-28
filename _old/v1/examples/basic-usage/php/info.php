<?php
// Display PHP info in a clean format
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - PHP Info</title>
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
        pre {
            background: #f6f8fa;
            padding: 10px;
            border-radius: 3px;
            overflow: auto;
        }
        .info-container {
            max-height: 500px;
            overflow: auto;
        }
    </style>
</head>
<body>
    <h1>PHP Information</h1>
    
    <div class="card">
        <h2>Basic PHP Info</h2>
        <ul>
            <li>PHP Version: <strong><?= phpversion() ?></strong></li>
            <li>PHP SAPI: <strong><?= php_sapi_name() ?></strong></li>
            <li>Server Software: <strong><?= $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown' ?></strong></li>
            <li>Server Protocol: <strong><?= $_SERVER['SERVER_PROTOCOL'] ?? 'Unknown' ?></strong></li>
            <li>Request Method: <strong><?= $_SERVER['REQUEST_METHOD'] ?? 'Unknown' ?></strong></li>
            <li>Request Time: <strong><?= date('Y-m-d H:i:s', $_SERVER['REQUEST_TIME'] ?? time()) ?></strong></li>
        </ul>
        
        <p><a href="/">Back to Home</a></p>
    </div>
    
    <div class="card">
        <h2>Server Variables</h2>
        <div class="info-container">
            <pre><?php print_r($_SERVER); ?></pre>
        </div>
    </div>
    
    <div class="card">
        <h2>PHP Extensions</h2>
        <div class="info-container">
            <pre><?php print_r(get_loaded_extensions()); ?></pre>
        </div>
    </div>
</body>
</html> 