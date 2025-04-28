<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - File Upload Example</title>
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
    <h1>Frango - File Upload Example</h1>
    
    <div class="card">
        <h2>Working with File Uploads</h2>
        <p>This example demonstrates how to handle file uploads with Frango, including:</p>
        <ul>
            <li>Creating file upload forms in PHP</li>
            <li>Processing uploaded files with PHP</li>
            <li>Accessing file information through <code>$_FILES</code></li>
            <li>Validating uploads (file type, size, etc.)</li>
            <li>Moving uploads to a permanent location</li>
            <li>Serving uploaded files from Go</li>
        </ul>
        
        <div>
            <a href="/upload" class="button">Upload Files</a>
            <a href="/view-files" class="button">View Uploaded Files</a>
        </div>
    </div>
    
    <div class="card">
        <h2>How It Works</h2>
        <p>Frango supports standard PHP file upload handling with <code>$_FILES</code>:</p>
        <pre><code>&lt;form method="POST" action="/process-upload" enctype="multipart/form-data"&gt;
    &lt;input type="file" name="file"&gt;
    &lt;button type="submit"&gt;Upload&lt;/button&gt;
&lt;/form&gt;

// In process-upload.php
$file = $_FILES['file'];
if ($file['error'] === UPLOAD_ERR_OK) {
    // Handle the uploaded file
}</code></pre>
        
        <p>The key components of this example are:</p>
        <ol>
            <li>Using <code>enctype="multipart/form-data"</code> in the HTML form</li>
            <li>Accessing uploaded files through <code>$_FILES</code> in PHP</li>
            <li>Handling file validation and storage in PHP</li>
            <li>Using Go to serve the uploaded files</li>
            <li>Mounting a directory for PHP to access using <code>WithMount</code></li>
        </ol>
    </div>
</body>
</html> 