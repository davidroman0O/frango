<?php
// About page
?>
<!DOCTYPE html>
<html>
<head>
    <title>About - Static Files PHP Example</title>
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
        .content {
            margin-top: 20px;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 4px;
        }
        .nav {
            margin-top: 30px;
        }
        a {
            display: inline-block;
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
        <h1>About This Example</h1>
        
        <div class="content">
            <h2>Direct File Access</h2>
            <p>This example demonstrates serving PHP files directly from a directory structure instead of embedding them using Go's embed feature.</p>
            
            <h2>How It Works</h2>
            <p>The Go application is configured to:</p>
            <ul>
                <li>Find PHP files at a specified directory path</li>
                <li>Serve them directly using FrankenPHP without embedding</li>
                <li>Support the full directory structure without copying files</li>
            </ul>
            
            <h2>Benefits</h2>
            <ul>
                <li>Edit PHP files directly without recompiling the Go application</li>
                <li>No need to extract files to a temporary directory</li>
                <li>More natural development workflow for PHP files</li>
            </ul>
        </div>
        
        <div class="nav">
            <a href="/">Back to Home</a>
        </div>
    </div>
</body>
</html> 