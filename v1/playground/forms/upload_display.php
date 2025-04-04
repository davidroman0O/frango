<?php
/**
 * File Upload Display
 * 
 * Displays the results of a file upload
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Results</title>
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
        .result-section {
            margin-top: 20px;
            padding-top: 15px;
            border-top: 1px solid #eee;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
        }
        table, th, td {
            border: 1px solid #ddd;
        }
        th, td {
            padding: 10px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        .file-preview {
            border: 1px solid #ddd;
            padding: 15px;
            margin: 15px 0;
            border-radius: 4px;
            background: #fff;
        }
        .error-code {
            display: inline-block;
            padding: 2px 6px;
            background: #e74c3c;
            color: white;
            border-radius: 3px;
            font-family: monospace;
        }
        .success-code {
            display: inline-block;
            padding: 2px 6px;
            background: #2ecc71;
            color: white;
            border-radius: 3px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">UPLOAD</span> File Results</h1>
        
        <?php if ($_SERVER['REQUEST_METHOD'] !== 'POST' || empty($_FILES)): ?>
            <div style="background: #fff4e5; padding: 15px; border-left: 4px solid #f39c12; margin: 20px 0;">
                <strong>No file upload detected.</strong> Please submit the form with a file from the forms page.
            </div>
        <?php else: ?>
            <?php 
            $file = $_FILES['userfile'] ?? null;
            $hasError = !$file || $file['error'] !== 0;
            
            if ($hasError): 
            ?>
                <div style="background: #fdedec; padding: 15px; border-left: 4px solid #e74c3c; margin: 20px 0;">
                    <strong>File upload error.</strong> 
                    Error code: <span class="error-code"><?= $file ? $file['error'] : 'No file received' ?></span>
                </div>
            <?php else: ?>
                <div style="background: #e8f8f5; padding: 15px; border-left: 4px solid #2ecc71; margin: 20px 0;">
                    <strong>File uploaded successfully!</strong>
                    <p>File <strong><?= htmlspecialchars($file['name']) ?></strong> has been received.</p>
                </div>
            <?php endif; ?>
            
            <div class="result-section">
                <h2>File Information</h2>
                
                <table>
                    <tr>
                        <th>Property</th>
                        <th>Value</th>
                    </tr>
                    <?php if ($file): ?>
                        <tr>
                            <td>Name</td>
                            <td><?= htmlspecialchars($file['name']) ?></td>
                        </tr>
                        <tr>
                            <td>Type</td>
                            <td><?= htmlspecialchars($file['type']) ?></td>
                        </tr>
                        <tr>
                            <td>Size</td>
                            <td><?= $file['size'] ?> bytes (<?= round($file['size'] / 1024, 2) ?> KB)</td>
                        </tr>
                        <tr>
                            <td>Temporary Location</td>
                            <td><?= htmlspecialchars($file['tmp_name']) ?></td>
                        </tr>
                        <tr>
                            <td>Error Code</td>
                            <td>
                                <?php if ($file['error'] === 0): ?>
                                    <span class="success-code">0</span> (No error)
                                <?php else: ?>
                                    <span class="error-code"><?= $file['error'] ?></span>
                                    (<?= getFileErrorMessage($file['error']) ?>)
                                <?php endif; ?>
                            </td>
                        </tr>
                    <?php else: ?>
                        <tr>
                            <td colspan="2">No file information available</td>
                        </tr>
                    <?php endif; ?>
                </table>
                
                <?php 
                // Description from form
                $description = htmlspecialchars($_POST['description'] ?? 'No description provided');
                ?>
                <h3>Form Data:</h3>
                <p><strong>Description:</strong> <?= $description ?></p>
            </div>
            
            <?php if ($file && $file['error'] === 0 && file_exists($file['tmp_name'])): ?>
                <div class="result-section">
                    <h2>File Content Preview</h2>
                    
                    <?php
                    // Get file content
                    $content = file_get_contents($file['tmp_name']);
                    
                    // Determine if it's text or binary
                    $isBinary = false;
                    if (function_exists('mb_detect_encoding')) {
                        $isBinary = !mb_detect_encoding($content, 'UTF-8', true);
                    } else {
                        // Simple binary check
                        $isBinary = preg_match('/[\x00-\x08\x0B\x0C\x0E-\x1F]/', $content);
                    }
                    
                    if (!$isBinary):
                    ?>
                        <div class="file-preview">
                            <pre><?= htmlspecialchars(substr($content, 0, 1000)) ?></pre>
                            <?php if (strlen($content) > 1000): ?>
                                <p><em>Content truncated (showing first 1000 characters only)</em></p>
                            <?php endif; ?>
                        </div>
                    <?php else: ?>
                        <p><em>Binary file content - preview not available</em></p>
                    <?php endif; ?>
                </div>
            <?php endif; ?>
            
            <div class="result-section">
                <h2>$_FILES Superglobal</h2>
                <pre><?php var_export($_FILES); ?></pre>
                
                <h2>Access Example</h2>
                <pre>
$file = $_FILES['userfile'];
$name = $file['name'];
$type = $file['type'];
$size = $file['size'];
$tmpName = $file['tmp_name'];
$error = $file['error'];

// Check for errors
if ($error === 0) {
    // Process the file
    move_uploaded_file($tmpName, '/path/to/destination/' . $name);
}</pre>
            </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
</body>
</html>

<?php
// Helper function to get file error messages
function getFileErrorMessage($code) {
    switch ($code) {
        case UPLOAD_ERR_INI_SIZE:
            return 'The uploaded file exceeds the upload_max_filesize directive in php.ini';
        case UPLOAD_ERR_FORM_SIZE:
            return 'The uploaded file exceeds the MAX_FILE_SIZE directive in the HTML form';
        case UPLOAD_ERR_PARTIAL:
            return 'The uploaded file was only partially uploaded';
        case UPLOAD_ERR_NO_FILE:
            return 'No file was uploaded';
        case UPLOAD_ERR_NO_TMP_DIR:
            return 'Missing a temporary folder';
        case UPLOAD_ERR_CANT_WRITE:
            return 'Failed to write file to disk';
        case UPLOAD_ERR_EXTENSION:
            return 'A PHP extension stopped the file upload';
        default:
            return 'Unknown upload error';
    }
} 