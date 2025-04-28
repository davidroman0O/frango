<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - Basic Example</title>
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
        code {
            background: #f6f8fa;
            padding: 2px 4px;
            border-radius: 3px;
        }
    </style>
</head>
<body>
    <h1>Frango - Basic Example</h1>
    
    <div class="card">
        <h2>Welcome to Frango!</h2>
        <p>This is a simple example demonstrating how to serve PHP files from Go.</p>
        <p>Current time: <strong><?= date('Y-m-d H:i:s') ?></strong></p>
        <p>PHP Version: <strong><?= phpversion() ?></strong></p>
    </div>
    
    <div class="card">
        <h2>Available Pages</h2>
        <ul>
            <li><a href="/">Home</a> - This page</li>
            <li><a href="/info">PHP Info</a> - View PHP configuration</li>
        </ul>
    </div>
    
    <div class="card">
        <h2>How It Works</h2>
        <p>This example uses the following Go code to serve PHP files:</p>
        <pre><code>php, err := frango.New(
    frango.WithSourceDir("./php"),
    frango.WithDevelopmentMode(true),
)

mux := http.NewServeMux()
mux.Handle("/", php.For("index.php"))
mux.Handle("/info", php.For("info.php"))</code></pre>
    </div>
</body>
</html> 