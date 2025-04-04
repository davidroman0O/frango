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
        h2 { color: #3498db; margin-top: 20px; }
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
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
            color: white;
        }
        .get { background: #3498db; }
        .post { background: #2ecc71; }
        .json { background: #9b59b6; }
        .upload { background: #e67e22; }
        .debug { background: #7f8c8d; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Form Handling Test Pages</h1>
        <p>These test pages verify the form handling functionality in Frango.</p>
        
        <div style="background: #ffedcd; padding: 15px; border-left: 4px solid #e67e22; margin-bottom: 20px;">
            <strong>Note:</strong> These are technical test forms to verify the PHP superglobals functionality. 
            For standard form examples, see the <a href="/forms">main forms page</a>.
        </div>
        
        <h2>Single-Feature Form Tests</h2>
        <ul>
            <li>
                <a href="/forms/form-get"><span class="method get">GET</span> GET Form Test</a>
                <span class="description">- Test submitting data via URL parameters</span>
            </li>
            <li>
                <a href="/forms/form-post"><span class="method post">POST</span> POST Form Test</a>
                <span class="description">- Test submitting data via request body</span>
            </li>
            <li>
                <a href="/forms/form-json"><span class="method json">JSON</span> JSON Request Test</a>
                <span class="description">- Test sending and receiving JSON data</span>
            </li>
            <li>
                <a href="/forms/form-upload"><span class="method upload">UPLOAD</span> File Upload Test</a>
                <span class="description">- Test file upload handling</span>
            </li>
            <li>
                <a href="/forms/form-test"><span class="method debug">DEBUG</span> Raw Form Data Test</a>
                <span class="description">- View detailed form data debug information</span>
            </li>
        </ul>

        <h2>Quick Test Links</h2>
        <ul>
            <li>
                <a href="/forms/form-test?test=query&value=123">Test Query Parameters</a>
                <span class="description">- Test $_GET handling directly</span>
            </li>
            <li>
                <a href="/forms/form-get?name=Example+User&category=test&limit=50">Pre-filled GET Form</a>
                <span class="description">- GET form with predefined parameters</span>
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
            <li><a href="/">Back to Home</a></li>
            <li><a href="/forms">Back to Forms</a></li>
        </ul>
    </div>
</body>
</html> 