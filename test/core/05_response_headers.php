<?php
// Response headers test

// Set various response headers
header('Content-Type: text/html; charset=UTF-8');
header('X-Custom-Header: Custom Value');
header('X-Frango-Test: Testing Custom Headers');
header('Cache-Control: no-cache, no-store, must-revalidate');
header('Pragma: no-cache');
header('Expires: 0');

// Get all headers that have been set
$headersList = headers_list();
?>
<!DOCTYPE html>
<html>
<head>
    <title>Response Headers Test</title>
    <style>
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>Response Headers Test</h1>
    
    <p>This page has set the following headers:</p>
    <ul>
        <li>Content-Type: text/html; charset=UTF-8</li>
        <li>X-Custom-Header: Custom Value</li>
        <li>X-Frango-Test: Testing Custom Headers</li>
        <li>Cache-Control: no-cache, no-store, must-revalidate</li>
        <li>Pragma: no-cache</li>
        <li>Expires: 0</li>
    </ul>
    
    <p>To see these headers, check your browser's developer tools network tab.</p>
    
    <h2>Current Headers According to headers_list():</h2>
    <?php if (empty($headersList)): ?>
        <p>No headers reported by headers_list()</p>
    <?php else: ?>
        <ul>
            <?php foreach ($headersList as $header): ?>
                <li><?= htmlspecialchars($header) ?></li>
            <?php endforeach; ?>
        </ul>
    <?php endif; ?>
    
    <script>
    // JavaScript can't directly access response headers for security reasons
    // But we can show how to verify using fetch API for educational purposes
    document.addEventListener('DOMContentLoaded', function() {
        const url = window.location.href;
        
        // Output to show that JS executed
        const jsOutput = document.createElement('div');
        jsOutput.innerHTML = '<h3>JavaScript Executed</h3><p>Headers can be verified in network tab or by using fetch API in your code.</p>';
        document.body.appendChild(jsOutput);
    });
    </script>
</body>
</html> 