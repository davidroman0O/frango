<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title><?= htmlspecialchars($title) ?></title>
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
        .data-item {
            background: #f6f8fa;
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 10px;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <h1><?= htmlspecialchars($title) ?></h1>
    
    <div class="card">
        <h2>Message from Go</h2>
        <div class="data-item">
            <?= htmlspecialchars($message) ?>
        </div>
        
        <h2>Current Time</h2>
        <div class="data-item">
            <?= htmlspecialchars($current_time) ?>
        </div>
        
        <h2>Items List</h2>
        <ul>
            <?php foreach ($items as $item): ?>
                <li><?= htmlspecialchars($item) ?></li>
            <?php endforeach; ?>
        </ul>
    </div>
    
    <div class="card">
        <h2>Template Data</h2>
        <p>This page demonstrates accessing data passed from Go to PHP. The following data was provided:</p>
        
        <h3>Raw Data (via $_TEMPLATE superglobal)</h3>
        <pre><?php var_export($_TEMPLATE); ?></pre>
        
        <p>This data is automatically extracted into local variables for easier access in your template.</p>
    </div>
    
    <a href="/" class="back-link">Back to home</a>
</body>
</html> 