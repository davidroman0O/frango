<?php
// Demo page
$current_time = date('Y-m-d H:i:s');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Demo Page - PHP Multi-Page Example</title>
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
        .info {
            background-color: #f9f9f9;
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
        }
        a {
            display: inline-block;
            margin-top: 20px;
            padding: 10px 15px;
            background-color: #2196F3;
            color: white;
            text-decoration: none;
            border-radius: 4px;
        }
        a:hover {
            background-color: #0b7dda;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Demo Page</h1>
        <p>This is the demo page of our multi-page PHP application.</p>
        
        <div class="info">
            <h2>PHP Information</h2>
            <p>Current server time: <?php echo $current_time; ?></p>
            <p>PHP Version: <?php echo phpversion(); ?></p>
            <p>Server Software: <?php echo $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown'; ?></p>
        </div>
        
        <a href="/">Back to Home Page</a>
    </div>
</body>
</html> 