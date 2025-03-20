<?php
// Home page
?>
<!DOCTYPE html>
<html>
<head>
    <title>Static Files PHP Example</title>
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
        }
        .links {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-top: 20px;
        }
        a {
            display: inline-block;
            padding: 10px 15px;
            background-color: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            width: fit-content;
        }
        a:hover {
            background-color: #45a049;
        }
        .info {
            margin-top: 20px;
            padding: 15px;
            background-color: #f5f5f5;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Static Files PHP Example</h1>
        <p>This example serves PHP files directly from a directory structure instead of embedding them.</p>
        
        <div class="links">
            <a href="/about.php">About Page</a>
            <a href="/contact.php">Contact Page</a>
            <a href="/counter.php">Counter Page</a>
        </div>
        
        <div class="info">
            <h3>Server Information</h3>
            <p>Current time: <?php echo date('Y-m-d H:i:s'); ?></p>
            <p>PHP Version: <?php echo phpversion(); ?></p>
            <p>Document Root: <?php echo $_SERVER['DOCUMENT_ROOT']; ?></p>
            <p>Script: <?php echo $_SERVER['SCRIPT_FILENAME']; ?></p>
        </div>
    </div>
</body>
</html> 