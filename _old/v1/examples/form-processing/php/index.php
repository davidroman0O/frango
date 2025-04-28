<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - Form Processing Example</title>
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
        .button {
            display: inline-block;
            background: #0366d6;
            color: white;
            padding: 8px 16px;
            border-radius: 4px;
            text-decoration: none;
            margin-top: 10px;
        }
        .button:hover {
            background: #0250be;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <h1>Frango - Form Processing Example</h1>
    
    <div class="card">
        <h2>Form Processing with PHP</h2>
        <p>This example demonstrates how to handle form submissions with Frango, showing how PHP scripts can access form data from various HTTP methods.</p>
        <p>In this example, you'll see:</p>
        <ul>
            <li>A contact form with field validation</li>
            <li>Processing POST data using PHP</li>
            <li>Redirecting users after form submission</li>
            <li>Accessing form data using PHP superglobals</li>
        </ul>
        <a href="/contact" class="button">Go to Contact Form</a>
    </div>
    
    <div class="card">
        <h2>Key Concepts</h2>
        <p>When working with forms in Frango, you need to understand:</p>
        <ul>
            <li><strong>$_POST</strong> - For accessing POST form data</li>
            <li><strong>$_GET</strong> - For accessing query parameters</li>
            <li><strong>$_FORM</strong> - Frango's unified form data superglobal</li>
            <li><strong>Form Validation</strong> - Performed on the PHP side</li>
            <li><strong>HTTP Method Constraints</strong> - Specifying allowed HTTP methods in route patterns</li>
        </ul>
    </div>
    
    <div class="card">
        <h2>How It Works</h2>
        <p>The form is submitted to a specific endpoint that's configured to only accept POST requests in the Go code:</p>
        <pre><code>mux.Handle("POST /contact/submit", php.For("process-form.php"))</code></pre>
        <p>This ensures that the form processing script only runs for POST requests, not GET requests.</p>
    </div>
</body>
</html> 