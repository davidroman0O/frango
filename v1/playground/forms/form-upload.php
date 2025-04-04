<?php
/**
 * File Upload Test
 * 
 * Simple test focused solely on file upload handling
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Test</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 600px;
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
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input, textarea {
            width: 100%;
            padding: 8px;
            margin-bottom: 15px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        input[type="file"] {
            padding: 10px;
            background: #f8f9fa;
            border: 1px dashed #ccc;
        }
        button {
            background: #e67e22;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #d35400;
        }
        pre {
            background: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .result {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #eee;
        }
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
            background: #e67e22;
            color: white;
        }
        .info {
            margin: 10px 0;
            padding: 10px;
            background: #f1f9ff;
            border-left: 4px solid #3498db;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">UPLOAD</span> File Test</h1>
        <p>This form demonstrates file upload handling in PHP. Files are submitted using multipart/form-data encoding.</p>
        
        <div class="info">
            <p><strong>Note:</strong> File uploads may be limited by PHP settings such as:</p>
            <ul>
                <li>upload_max_filesize (typically 2MB by default)</li>
                <li>post_max_size</li>
                <li>max_execution_time</li>
            </ul>
        </div>
        
        <form action="/forms/form-upload" method="POST" enctype="multipart/form-data">
            <label for="userfile">Select File:</label>
            <input type="file" id="userfile" name="userfile">
            
            <label for="description">File Description:</label>
            <input type="text" id="description" name="description" value="Test file upload">
            
            <button type="submit">Upload File</button>
        </form>
        
        <?php if ($_SERVER['REQUEST_METHOD'] === 'POST'): ?>
        <div class="result">
            <h2>Upload Results</h2>
            
            <h3>$_FILES Superglobal:</h3>
            <pre><?php var_export($_FILES); ?></pre>
            
            <h3>$_POST Data:</h3>
            <pre><?php var_export($_POST); ?></pre>
            
            <?php if (!empty($_FILES) && isset($_FILES['userfile'])): ?>
                <h3>File Details:</h3>
                <ul>
                    <li><strong>Name:</strong> <?= htmlspecialchars($_FILES['userfile']['name']) ?></li>
                    <li><strong>Type:</strong> <?= htmlspecialchars($_FILES['userfile']['type']) ?></li>
                    <li><strong>Size:</strong> <?= htmlspecialchars($_FILES['userfile']['size']) ?> bytes</li>
                    <li><strong>Temp Name:</strong> <?= htmlspecialchars($_FILES['userfile']['tmp_name']) ?></li>
                    <li><strong>Error Code:</strong> <?= htmlspecialchars($_FILES['userfile']['error']) ?></li>
                </ul>
                
                <?php if ($_FILES['userfile']['error'] === 0): ?>
                    <h3>File Content Preview:</h3>
                    <?php
                    $tempFile = $_FILES['userfile']['tmp_name'];
                    $fileContent = file_exists($tempFile) ? file_get_contents($tempFile) : 'Unable to read file';
                    
                    // Determine if this is text or binary content
                    $isBinary = false;
                    if (function_exists('mb_detect_encoding')) {
                        $isBinary = !mb_detect_encoding($fileContent, 'UTF-8', true);
                    } else {
                        // Simple binary check
                        $isBinary = preg_match('/[\x00-\x08\x0B\x0C\x0E-\x1F]/', $fileContent);
                    }
                    
                    if (!$isBinary) {
                        // For text files, show content
                        echo '<pre>' . htmlspecialchars(substr($fileContent, 0, 1024)) . '</pre>';
                        if (strlen($fileContent) > 1024) {
                            echo '<p><em>Content truncated (showing first 1KB only)</em></p>';
                        }
                    } else {
                        echo '<p><em>Binary content - preview not available</em></p>';
                    }
                    ?>
                <?php endif; ?>
            <?php endif; ?>
            
            <h3>PHP_FILE_* Variables:</h3>
            <pre><?php 
                $fileVars = [];
                foreach ($_SERVER as $key => $value) {
                    if (strpos($key, 'PHP_FILE_') === 0) {
                        $fileVars[$key] = $value;
                    }
                }
                var_export($fileVars);
            ?></pre>
        </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms/form-index">Back to Form Tests</a></p>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
</body>
</html> 