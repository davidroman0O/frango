<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Upload Files - Frango Example</title>
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
        h1, h2, h3 {
            color: #333;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: bold;
        }
        input, select {
            margin-bottom: 16px;
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        input[type="file"] {
            padding: 8px 0;
        }
        button {
            background: #0366d6;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 1em;
        }
        button:hover {
            background: #0250be;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .info-text {
            font-size: 0.9em;
            color: #666;
            margin-bottom: 8px;
        }
        .upload-rules {
            background: #f5f9ff;
            padding: 12px;
            border-radius: 4px;
            border-left: 4px solid #0366d6;
            margin-bottom: 16px;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <h1>Upload Files</h1>
    
    <div class="card">
        <div class="upload-rules">
            <h3>Upload Rules</h3>
            <ul>
                <li>Maximum file size: 5MB</li>
                <li>Allowed file types: Images (JPG, PNG, GIF), PDF documents</li>
                <li>No executable files (.exe, .bat, etc.)</li>
            </ul>
        </div>
        
        <form action="/process-upload" method="POST" enctype="multipart/form-data">
            <!-- Single file upload -->
            <div>
                <label for="single-file">Upload a File:</label>
                <input type="file" id="single-file" name="single_file" required>
                <p class="info-text">Select any image or PDF file</p>
            </div>
            
            <!-- Multi-file upload -->
            <div>
                <label for="multi-files">Upload Multiple Files:</label>
                <input type="file" id="multi-files" name="multi_files[]" multiple>
                <p class="info-text">Select multiple files (hold Ctrl/Cmd while selecting)</p>
            </div>
            
            <!-- File upload with additional data -->
            <div>
                <label for="description">File Description:</label>
                <input type="text" id="description" name="description" placeholder="Describe your upload...">
            </div>
            
            <div>
                <label for="category">File Category:</label>
                <select id="category" name="category">
                    <option value="document">Document</option>
                    <option value="image">Image</option>
                    <option value="other">Other</option>
                </select>
            </div>
            
            <button type="submit">Upload Files</button>
        </form>
        
        <a href="/" class="back-link">Back to Home</a>
    </div>
</body>
</html> 