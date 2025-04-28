<?php
/**
 * PHP-to-PHP File Upload Demo
 * 
 * Demonstrates uploading a file to another PHP script
 */

// ./v1/playground/demo/forms/upload_to_php.php
?>
<!DOCTYPE html>
<html>
<head>
    <title>PHP-to-PHP File Upload</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        h2 { color: #3498db; }
        pre {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
            background: #e74c3c;
            color: white;
        }
        .code-block {
            background: #2c3e50;
            color: white;
            padding: 15px;
            border-radius: 4px;
            margin: 15px 0;
            font-family: monospace;
            white-space: pre-wrap;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input[type="file"], input[type="text"] {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-bottom: 10px;
        }
        button {
            background: #3498db;
            color: white;
            border: none;
            padding: 8px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        #result {
            margin-top: 20px;
            padding: 15px;
            border-radius: 4px;
            display: none;
        }
        .success {
            background-color: #d4edda;
            border-color: #c3e6cb;
            color: #155724;
        }
        .error {
            background-color: #f8d7da;
            border-color: #f5c6cb;
            color: #721c24;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">UPLOAD</span> PHP-to-PHP File Upload</h1>
        <p>This example demonstrates uploading a file from one PHP script to another PHP script.</p>
        
        <form id="uploadForm" action="/forms/php_receiver.php" method="POST" enctype="multipart/form-data">
            <div class="form-group">
                <label for="userfile">Select File:</label>
                <input type="file" id="userfile" name="userfile">
            </div>
            
            <div class="form-group">
                <label for="description">File Description:</label>
                <input type="text" id="description" name="description" value="File uploaded via PHP form">
            </div>
            
            <div class="form-group">
                <button type="submit">Upload to PHP Endpoint</button>
            </div>
        </form>
        
        <div id="result"></div>
        
        <h2>How It Works</h2>
        <p>This form submits the file to another PHP script that processes the upload. The receiving PHP script:</p>
        <ul>
            <li>Validates the uploaded file</li>
            <li>Saves the file metadata</li>
            <li>Shows the upload result and file information</li>
        </ul>
        
        <div class="code-block">
# Server-side code (php_receiver.php)
$uploadedFile = $_FILES['userfile'];
$description = $_POST['description'];

// Validate upload
if ($uploadedFile['error'] === 0) {
    // Process the file
    $fileTmpPath = $uploadedFile['tmp_name'];
    $fileName = $uploadedFile['name'];
    $fileSize = $uploadedFile['size'];
    $fileType = $uploadedFile['type'];
    
    // File was successfully uploaded
    echo "Upload successful!";
} else {
    // Handle upload error
    echo "Upload failed: " . $uploadedFile['error'];
}</div>
        
        <h2>Navigation:</h2>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
    
    <script>
        // Optional: Add AJAX-based submission to demonstrate both techniques
        document.getElementById('uploadForm').addEventListener('submit', function(e) {
            // Comment out to use traditional form submission
            // e.preventDefault();
            
            // For AJAX submission, uncomment the following:
            /*
            const formData = new FormData(this);
            
            fetch('/forms/php_receiver.php', {
                method: 'POST',
                body: formData
            })
            .then(response => response.text())
            .then(data => {
                const resultDiv = document.getElementById('result');
                resultDiv.innerHTML = data;
                resultDiv.style.display = 'block';
                resultDiv.className = data.includes('successful') ? 'success' : 'error';
            })
            .catch(error => {
                document.getElementById('result').innerHTML = 'Error: ' + error.message;
                document.getElementById('result').style.display = 'block';
                document.getElementById('result').className = 'error';
            });
            */
        });
    </script>
</body>
</html> 