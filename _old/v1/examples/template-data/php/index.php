<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - Template Data Example</title>
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
            margin-right: 10px;
        }
        .button:hover {
            background: #0250be;
            text-decoration: none;
        }
        code {
            background: #f6f8fa;
            padding: 2px 4px;
            border-radius: 3px;
            font-family: monospace;
        }
        pre {
            background: #f6f8fa;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
    </style>
</head>
<body>
    <h1>Frango - Template Data Example</h1>
    
    <div class="card">
        <h2>Template Data Injection</h2>
        <p>This example demonstrates how to pass data from Go to PHP templates using Frango's <code>Render()</code> method.</p>
        <p>This allows you to:</p>
        <ul>
            <li>Prepare data in Go (from databases, APIs, etc.)</li>
            <li>Pass complex structured data to PHP</li>
            <li>Access that data directly in PHP as variables</li>
            <li>Combine the power of Go's data processing with PHP's templating</li>
        </ul>
        
        <div>
            <a href="/basic" class="button">Basic Example</a>
            <a href="/dashboard" class="button">Dashboard Example</a>
        </div>
    </div>
    
    <div class="card">
        <h2>How It Works</h2>
        <p>In your Go code, you use the <code>Render()</code> method instead of <code>For()</code>, and provide a function that returns the data:</p>
        <pre><code>mux.Handle("/basic", php.Render("basic.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title":   "Basic Template Example",
        "message": "This data was passed from Go to PHP!",
        "items":   []string{"Item 1", "Item 2", "Item 3"},
    }
}))</code></pre>
        
        <p>In your PHP file, you can access the data directly as variables:</p>
        <pre><code>&lt;h1&gt;&lt;?= htmlspecialchars($title) ?&gt;&lt;/h1&gt;
&lt;p&gt;&lt;?= htmlspecialchars($message) ?&gt;&lt;/p&gt;

&lt;ul&gt;
    &lt;?php foreach ($items as $item): ?&gt;
        &lt;li&gt;&lt;?= htmlspecialchars($item) ?&gt;&lt;/li&gt;
    &lt;?php endforeach; ?&gt;
&lt;/ul&gt;</code></pre>
        
        <p>You can also access the data through the <code>$_TEMPLATE</code> superglobal if needed.</p>
    </div>
    
    <div class="card">
        <h2>Template Data Types</h2>
        <p>You can pass a variety of data types from Go to PHP:</p>
        <ul>
            <li><strong>Scalar Values:</strong> strings, numbers, booleans</li>
            <li><strong>Arrays/Slices:</strong> converted to PHP indexed arrays</li>
            <li><strong>Maps:</strong> converted to PHP associative arrays</li>
            <li><strong>Structs:</strong> converted to PHP associative arrays</li>
            <li><strong>Nested Structures:</strong> any combination of the above</li>
        </ul>
    </div>
</body>
</html> 