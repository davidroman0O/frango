<?php
// About page
?>
<!DOCTYPE html>
<html>
<head>
    <title>About Page</title>
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
    <h1>About Page</h1>
    <p>This is the about page served via <code>HandleDir</code> function.</p>
    <p>The HandleDir function automatically registered all PHP files in the "pages" directory.</p>
    <p>URL: <code><?= $_SERVER['REQUEST_URI'] ?></code></p>
    <p><a href="/">Back to Home</a></p>
</body>
</html> 