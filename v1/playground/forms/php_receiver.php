<?php
/**
 * PHP File Upload Receiver
 * 
 * Receives and processes file uploads from the PHP form
 */

// Initialize variables
$success = false;
$message = "No file uploaded";
$fileInfo = [];

// Check if a file was uploaded
if (isset($_FILES['userfile']) && !empty($_FILES['userfile']['name'])) {
    $uploadedFile = $_FILES['userfile'];
    $description = $_POST['description'] ?? 'No description provided';
    
    // Validate upload
    if ($uploadedFile['error'] === 0) {
        // Process the file
        $fileTmpPath = $uploadedFile['tmp_name'];
        $fileName = $uploadedFile['name'];
        $fileSize = $uploadedFile['size'];
        $fileType = $uploadedFile['type'];
        
        // Format file size for display
        $formattedSize = formatFileSize($fileSize);
        
        // File was successfully uploaded
        $success = true;
        $message = "File uploaded successfully!";
        
        // Store file info for display
        $fileInfo = [
            'name' => $fileName,
            'size' => $formattedSize,
            'type' => $fileType,
            'description' => $description,
            'tmp_path' => $fileTmpPath
        ];
    } else {
        // Handle upload error
        $message = "Upload failed: " . getUploadErrorMessage($uploadedFile['error']);
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
function getUploadErrorMessage($code) {
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
?>
<!DOCTYPE html>
<html>
<head>
    <title>PHP Upload Result</title>
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
        .success-message {
            background-color: #d4edda;
            color: #155724;
            padding: 15px;
            border-radius: 4px;
            margin: 15px 0;
        }
        .error-message {
            background-color: #f8d7da;
            color: #721c24;
            padding: 15px;
            border-radius: 4px;
            margin: 15px 0;
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
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">UPLOAD</span> PHP Upload Result</h1>
        
        <?php if ($success): ?>
            <div class="success-message">
                <strong>Success!</strong> <?php echo $message; ?>
            </div>
            
            <h2>File Information</h2>
            <table>
                <tr>
                    <th>Property</th>
                    <th>Value</th>
                </tr>
                <?php foreach ($fileInfo as $property => $value): ?>
                <tr>
                    <td><?php echo ucfirst(str_replace('_', ' ', $property)); ?></td>
                    <td><?php echo htmlspecialchars($value); ?></td>
                </tr>
                <?php endforeach; ?>
            </table>
            
            <?php if (substr($fileInfo['type'], 0, 5) === 'image/'): ?>
                <h2>Image Preview</h2>
                <div style="max-width: 100%; margin: 15px 0;">
                    <img src="data:<?php echo $fileInfo['type']; ?>;base64,<?php echo base64_encode(file_get_contents($fileInfo['tmp_path'])); ?>" 
                         style="max-width: 100%; max-height: 300px; border: 1px solid #ddd;">
                </div>
            <?php endif; ?>
            
        <?php else: ?>
            <div class="error-message">
                <strong>Error!</strong> <?php echo $message; ?>
            </div>
        <?php endif; ?>
        
        <h2>Debug Information</h2>
        <pre><?php print_r($_FILES); ?></pre>
        
        <h2>Navigation:</h2>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
</body>
</html> 