<?php
/**
 * Form Handling Test Index
 * 
 * Links to various form handling test pages
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>Form Handling Tests</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        ul {
            padding-left: 20px;
        }
        li {
            margin-bottom: 10px;
        }
        a {
            color: #3498db;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .description {
            margin-left: 5px;
            color: #7f8c8d;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>Form Handling Test Pages</h1>
        <p>These test pages verify the form handling functionality in Frango.</p>
        
        <ul>
            <li>
                <a href="form_submit.php">Form Submission Test</a>
                <span class="description">- Submit form data via POST or GET methods</span>
            </li>
            <li>
                <a href="form_test.php">Direct Form Test</a>
                <span class="description">- View the raw form handling debug information</span>
            </li>
            <li>
                <a href="form_test.php?test=query&value=123">Test with Query Parameters</a>
                <span class="description">- Test $_GET handling directly</span>
            </li>
        </ul>
        
        <h2>How Form Handling Works</h2>
        <p>In Frango, form data is processed as follows:</p>
        <ol>
            <li>Go middleware extracts form data from HTTP requests</li>
            <li>Form data is passed to PHP as environment variables with prefix <code>PHP_FORM_*</code></li>
            <li>The PHP globals initialization script populates standard superglobals like <code>$_POST</code> and <code>$_GET</code></li>
            <li>PHP scripts can access form data through the standard PHP superglobals</li>
        </ol>
        
        <p>The test pages above verify that this process works correctly throughout the request lifecycle.</p>
    </div>
    
    <div class="card">
        <h2>Navigation</h2>
        <ul>
            <li><a href="/">Back to Playground Home</a></li>
        </ul>
    </div>
</body>
</html> 