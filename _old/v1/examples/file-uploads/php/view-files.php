<?php
header('Content-Type: text/html');

// Start session to get upload results
session_start();

// Get upload results if available
$uploadResults = $_SESSION['upload_results'] ?? null;

// Clear the session data after retrieving it
if (isset($_SESSION['upload_results'])) {
    unset($_SESSION['upload_results']);
}

// Use absolute path to the uploads directory (same as in process-upload.php)
$uploadsDir = dirname($_SERVER['DOCUMENT_ROOT']) . '/uploads/';
$allFiles = [];

if (is_dir($uploadsDir)) {
    $files = scandir($uploadsDir);
    foreach ($files as $file) {
        if ($file != '.' && $file != '..') {
            $filePath = $uploadsDir . $file;
            $fileSize = filesize($filePath);
            $fileType = mime_content_type($filePath);
            $fileTime = filemtime($filePath);
            
            $allFiles[] = [
                'name' => $file,
                'size' => $fileSize,
                'type' => $fileType,
                'time' => $fileTime,
                'path' => '/uploads/' . $file
            ];
        }
    }
    
    // Sort files by upload time (newest first)
    usort($allFiles, function($a, $b) {
        return $b['time'] - $a['time'];
    });
}

// Helper function to format file size
function formatBytes($bytes, $precision = 2) {
    $units = ['B', 'KB', 'MB', 'GB', 'TB'];
    $bytes = max($bytes, 0);
    $pow = floor(($bytes ? log($bytes) : 0) / log(1024));
    $pow = min($pow, count($units) - 1);
    return round($bytes / (1024 ** $pow), $precision) . ' ' . $units[$pow];
}

// Helper function to get file type icon
function getFileIcon($fileType) {
    if (strpos($fileType, 'image/') === 0) {
        return '🖼️';
    } elseif ($fileType === 'application/pdf') {
        return '📄';
    } else {
        return '📁';
    }
}

// Get user-friendly file type
function getFileTypeLabel($fileType) {
    if (strpos($fileType, 'image/') === 0) {
        return 'Image (' . str_replace('image/', '', $fileType) . ')';
    } elseif ($fileType === 'application/pdf') {
        return 'PDF Document';
    } else {
        return $fileType;
    }
}

?>
<!DOCTYPE html>
<html>
<head>
    <title>View Uploaded Files - Frango Example</title>
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
        .alert {
            padding: 12px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .alert-success {
            background-color: #d4edda;
            color: #155724;
            border-left: 4px solid #28a745;
        }
        .alert-error {
            background-color: #f8d7da;
            color: #721c24;
            border-left: 4px solid #dc3545;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 20px;
        }
        th, td {
            text-align: left;
            padding: 12px;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f5f5f5;
        }
        .file-icon {
            font-size: 1.5em;
            margin-right: 10px;
        }
        .file-info {
            display: flex;
            align-items: center;
        }
        .file-preview {
            max-width: 100px;
            max-height: 100px;
            margin-top: 8px;
            border-radius: 4px;
            border: 1px solid #ddd;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
        }
        .metadata {
            color: #666;
            font-size: 0.9em;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <h1>Uploaded Files</h1>
    
    <?php if ($uploadResults): ?>
        <!-- Display upload results if just uploaded -->
        <?php if (!empty($uploadResults['success_files'])): ?>
            <div class="alert alert-success">
                <h3>Files Uploaded Successfully</h3>
                <p>Successfully uploaded <?= count($uploadResults['success_files']) ?> file(s).</p>
                
                <?php if (!empty($uploadResults['metadata']['description'])): ?>
                    <div class="metadata">
                        <div><strong>Description:</strong> <?= htmlspecialchars($uploadResults['metadata']['description']) ?></div>
                        <div><strong>Category:</strong> <?= htmlspecialchars($uploadResults['metadata']['category']) ?></div>
                        <div><strong>Uploaded at:</strong> <?= htmlspecialchars($uploadResults['metadata']['timestamp']) ?></div>
                    </div>
                <?php endif; ?>
            </div>
        <?php endif; ?>
        
        <?php if (!empty($uploadResults['errors'])): ?>
            <div class="alert alert-error">
                <h3>Upload Errors</h3>
                <ul>
                    <?php foreach ($uploadResults['errors'] as $error): ?>
                        <li><?= htmlspecialchars($error) ?></li>
                    <?php endforeach; ?>
                </ul>
            </div>
        <?php endif; ?>
    <?php endif; ?>
    
    <div class="card">
        <h2>All Uploaded Files</h2>
        
        <?php if (empty($allFiles)): ?>
            <p>No files have been uploaded yet.</p>
        <?php else: ?>
            <p>Showing <?= count($allFiles) ?> file(s) in the uploads directory.</p>
            
            <table>
                <thead>
                    <tr>
                        <th>File</th>
                        <th>Type</th>
                        <th>Size</th>
                        <th>Uploaded</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <?php foreach ($allFiles as $file): ?>
                        <tr>
                            <td>
                                <div class="file-info">
                                    <span class="file-icon"><?= getFileIcon($file['type']) ?></span>
                                    <div>
                                        <?= htmlspecialchars($file['name']) ?>
                                        <?php if (strpos($file['type'], 'image/') === 0): ?>
                                            <div>
                                                <img src="<?= htmlspecialchars($file['path']) ?>" class="file-preview" alt="Preview">
                                            </div>
                                        <?php endif; ?>
                                    </div>
                                </div>
                            </td>
                            <td><?= htmlspecialchars(getFileTypeLabel($file['type'])) ?></td>
                            <td><?= formatBytes($file['size']) ?></td>
                            <td><?= date('Y-m-d H:i:s', $file['time']) ?></td>
                            <td>
                                <a href="<?= htmlspecialchars($file['path']) ?>" target="_blank">View</a>
                            </td>
                        </tr>
                    <?php endforeach; ?>
                </tbody>
            </table>
        <?php endif; ?>
        
        <div>
            <a href="/upload" class="button">Upload More Files</a>
            <a href="/" class="button">Back to Home</a>
        </div>
    </div>
    
    <div class="card">
        <h2>How $_FILES Works</h2>
        <p>When files are uploaded, PHP provides the file information in the $_FILES superglobal array:</p>
        <pre><?php
            $exampleFilesArray = [
                'single_file' => [
                    'name' => 'example.jpg',
                    'type' => 'image/jpeg',
                    'tmp_name' => '/tmp/phpxyz123',
                    'error' => 0,
                    'size' => 123456
                ],
                'multi_files' => [
                    'name' => ['file1.jpg', 'file2.pdf'],
                    'type' => ['image/jpeg', 'application/pdf'],
                    'tmp_name' => ['/tmp/phpxyz124', '/tmp/phpxyz125'],
                    'error' => [0, 0],
                    'size' => [123456, 234567]
                ]
            ];
            echo htmlspecialchars(var_export($exampleFilesArray, true));
        ?></pre>
    </div>
</body>
</html> 