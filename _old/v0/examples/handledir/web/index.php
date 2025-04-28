<?php
// Root index.php file
?>
<!DOCTYPE html>
<html>
<head>
    <title>HandleDir Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #333;
        }
        ul {
            list-style-type: none;
            padding: 0;
        }
        li {
            margin: 10px 0;
            padding: 10px;
            background-color: #f5f5f5;
            border-radius: 5px;
        }
        a {
            color: #0066cc;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <h1>HandleDir Example</h1>
    <p>This example demonstrates the <code>HandleDir</code> function in Go-PHP.</p>
    
    <h2>Pages</h2>
    <ul>
        <li><a href="/pages/about.php">About Page (with .php extension)</a></li>
        <li><a href="/pages/about">About Page (without .php extension)</a></li>
        <li><a href="/pages/contact.php">Contact Page</a></li>
    </ul>
    
    <h2>API</h2>
    <ul>
        <li><a href="/api/users.php">Users API</a></li>
        <li><a href="/api/users">Users API (clean URL)</a></li>
        <li><a href="/api/items.php">Items API</a></li>
    </ul>
</body>
</html> 