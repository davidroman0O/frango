<?php
// Home page
?>
<!DOCTYPE html>
<html>
<head>
    <title>PHP Multi-Page Example</title>
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
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to the Multi-Page PHP Example</h1>
        <p>This is a simple demonstration of a multi-page PHP application embedded in Go using FrankenPHP.</p>
        <p>You can navigate between different pages using the links below:</p>
        
        <div class="links">
            <a href="/demo">Go to Demo Page</a>
            <a href="/dynamic">Go to Dynamic Page</a>
            <a href="/dynamic?name=PHP+User&color=blue&count=3">Go to Dynamic Page with Parameters</a>
            <a href="/stateful">Go to Stateful Page</a>
        </div>
    </div>
</body>
</html> 