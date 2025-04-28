<?php
/**
 * Frango v2 Demo - PHP File Upload Handler
 * 
 * Demonstrates processing file uploads via $_FILES superglobal.
 */

// Initialize variables
$uploadSuccess = false;
$errorMessage = '';
$fileInfo = [];

// Process file upload if this is a POST request
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    // Check if a file was uploaded
    if (isset($_FILES['userfile']) && !empty($_FILES['userfile']['name'])) {
        $uploadedFile = $_FILES['userfile'];
        
        // Check for upload errors
        if ($uploadedFile['error'] === UPLOAD_ERR_OK) {
            $uploadSuccess = true;
            
            // Collect file information
            $fileInfo = [
                'name' => $uploadedFile['name'],
                'type' => $uploadedFile['type'],
                'size' => $uploadedFile['size'],
                'tmp_name' => $uploadedFile['tmp_name'],
                'formatted_size' => formatFileSize($uploadedFile['size']),
                'description' => $_POST['description'] ?? 'No description provided',
                'upload_time' => date('Y-m-d H:i:s')
            ];
            
            // For demonstration purposes, we're not moving the file anywhere permanent
            // In a real app, you'd use move_uploaded_file() to save it
        } else {
            // Handle upload errors
            $errorMessage = getUploadErrorMessage($uploadedFile['error']);
        }
    } else {
        $errorMessage = 'No file was uploaded';
    }
}

// Helper function to format file size
function formatFileSize($bytes) {
    if ($bytes >= 1073741824) {
        return number_format($bytes / 1073741824, 2) . ' GB';
    } elseif ($bytes >= 1048576) {
        return number_format($bytes / 1048576, 2) . ' MB';
    } elseif ($bytes >= 1024) {
        return number_format($bytes / 1024, 2) . ' KB';
    } else {
        return $bytes . ' bytes';
    }
}

// Helper function to get upload error message
function getUploadErrorMessage($errorCode) {
    switch ($errorCode) {
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
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>File Upload Results - Frango v2</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>File Upload Results (PHP Handler)</h1>
        <p>This page demonstrates how Frango processes file uploads via the <code>$_FILES</code> superglobal.</p>
        
        <?php if ($_SERVER['REQUEST_METHOD'] !== 'POST'): ?>
            <div class="card">
                <div style="background: #f8d7da; padding: 15px; border-left: 4px solid #dc3545; margin-bottom: 20px;">
                    <strong>No upload received.</strong> Please upload a file on the <a href="/forms">Forms Demo</a> page.
                </div>
            </div>
        <?php elseif (!$uploadSuccess): ?>
            <div class="card">
                <div style="background: #f8d7da; padding: 15px; border-left: 4px solid #dc3545; margin-bottom: 20px;">
                    <strong>Upload failed:</strong> <?= htmlspecialchars($errorMessage) ?>
                </div>
                
                <h2>$_FILES Contents</h2>
                <pre><?php var_export($_FILES); ?></pre>
            </div>
        <?php else: ?>
            <div class="card">
                <div style="background: #d4edda; padding: 15px; border-left: 4px solid #28a745; margin-bottom: 20px;">
                    <strong>Success!</strong> File uploaded and processed by PHP.
                </div>
                
                <h2>File Information</h2>
                <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                    <tr style="background: #f8f9fa;">
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Property</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                    </tr>
                    <tr>
                        <td style="padding: 8px; border: 1px solid #dee2e6;">Filename</td>
                        <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($fileInfo['name']) ?></td>
                    </tr>
                    <tr>
                        <td style="padding: 8px; border: 1px solid #dee2e6;">File Type</td>
                        <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($fileInfo['type']) ?></td>
                    </tr>
                    <tr>
                        <td style="padding: 8px; border: 1px solid #dee2e6;">File Size</td>
                        <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($fileInfo['formatted_size']) ?> (<?= htmlspecialchars($fileInfo['size']) ?> bytes)</td>
                    </tr>
                    <tr>
                        <td style="padding: 8px; border: 1px solid #dee2e6;">Temporary File</td>
                        <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($fileInfo['tmp_name']) ?></td>
                    </tr>
                    <tr>
                        <td style="padding: 8px; border: 1px solid #dee2e6;">Description</td>
                        <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($fileInfo['description']) ?></td>
                    </tr>
                    <tr>
                        <td style="padding: 8px; border: 1px solid #dee2e6;">Upload Time</td>
                        <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($fileInfo['upload_time']) ?></td>
                    </tr>
                </table>
                
                <?php if (strpos($fileInfo['type'], 'image/') === 0): ?>
                    <h3>Image Preview</h3>
                    <div style="max-width: 100%; overflow: hidden; margin-bottom: 20px; text-align: center;">
                        <img src="data:<?= $fileInfo['type'] ?>;base64,<?= base64_encode(file_get_contents($fileInfo['tmp_name'])) ?>" 
                             style="max-width: 100%; max-height: 300px; border: 1px solid #dee2e6; border-radius: 4px;">
                    </div>
                <?php elseif (strpos($fileInfo['type'], 'text/') === 0): ?>
                    <h3>Text Content Preview</h3>
                    <pre style="max-height: 300px; overflow: auto;"><?= htmlspecialchars(file_get_contents($fileInfo['tmp_name'])) ?></pre>
                <?php endif; ?>
                
                <h3>Raw $_FILES Superglobal</h3>
                <pre><?php var_export($_FILES); ?></pre>
                
                <h3>Other Form Data ($_POST)</h3>
                <pre><?php var_export($_POST); ?></pre>
            </div>
        <?php endif; ?>
        
        <div class="card">
            <h2>Technical Details</h2>
            <h3>File Upload Flow</h3>
            <ol>
                <li>Client submits form with <code>enctype="multipart/form-data"</code></li>
                <li>Frango middleware processes the multipart request</li>
                <li>The file content is saved to a temporary location</li>
                <li>PHP script accesses file information via <code>$_FILES</code> superglobal</li>
                <li>Script can use <code>move_uploaded_file()</code> to move file to permanent location</li>
            </ol>
            
            <h3>Code Example</h3>
            <p>This is how you process file uploads in PHP:</p>
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// Check if file was uploaded without errors
if (isset($_FILES['userfile']) && $_FILES['userfile']['error'] === UPLOAD_ERR_OK) {
    $tmpName = $_FILES['userfile']['tmp_name'];
    $fileName = $_FILES['userfile']['name'];
    $fileSize = $_FILES['userfile']['size'];
    $fileType = $_FILES['userfile']['type'];
    
    // Move the file to a permanent location (only with actual file uploads)
    $uploadDir = '/path/to/uploads/';
    $destination = $uploadDir . basename($fileName);
    
    if (move_uploaded_file($tmpName, $destination)) {
        echo "File uploaded successfully to: $destination";
    } else {
        echo "Error moving uploaded file.";
    }
}</pre>
        </div>
        
        <div class="card">
            <h2>Navigation</h2>
            <p><a href="/forms">Back to Forms Demo</a></p>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(__DIR__) . '/debug_panel.php'; ?>
</body>
</html> 