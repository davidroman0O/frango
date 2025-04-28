<?php
// Process file uploads

// Configuration
$maxFileSize = 5 * 1024 * 1024; // 5MB
$allowedExtensions = ['jpg', 'jpeg', 'png', 'gif', 'pdf'];
// Use absolute path to the uploads directory
$uploadsDir = dirname($_SERVER['DOCUMENT_ROOT']) . '/uploads/';

// Initialize error and success messages
$errors = [];
$successFiles = [];

// Check if the form was submitted
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: /upload');
    exit;
}

// Get form metadata
$description = $_POST['description'] ?? '';
$category = $_POST['category'] ?? 'other';

// Function to validate and move a file
function processUploadedFile($file, $allowedExtensions, $maxFileSize, $uploadsDir, $fileNumber = '') {
    $errors = [];
    
    // Check if file was uploaded successfully
    if ($file['error'] !== UPLOAD_ERR_OK) {
        $errorMessages = [
            UPLOAD_ERR_INI_SIZE => 'The uploaded file exceeds the upload_max_filesize directive in php.ini',
            UPLOAD_ERR_FORM_SIZE => 'The uploaded file exceeds the MAX_FILE_SIZE directive in the HTML form',
            UPLOAD_ERR_PARTIAL => 'The uploaded file was only partially uploaded',
            UPLOAD_ERR_NO_FILE => 'No file was uploaded',
            UPLOAD_ERR_NO_TMP_DIR => 'Missing a temporary folder',
            UPLOAD_ERR_CANT_WRITE => 'Failed to write file to disk',
            UPLOAD_ERR_EXTENSION => 'A PHP extension stopped the file upload'
        ];
        
        return [
            'success' => false,
            'error' => $errorMessages[$file['error']] ?? 'Unknown upload error',
            'file_info' => null
        ];
    }
    
    // Get file info
    $fileName = $file['name'];
    $fileSize = $file['size'];
    $fileTmpPath = $file['tmp_name'];
    $fileType = $file['type'];
    
    // Extract file extension and sanitize filename
    $fileExt = strtolower(pathinfo($fileName, PATHINFO_EXTENSION));
    $newFileName = uniqid() . '_' . preg_replace('/[^a-zA-Z0-9_.-]/', '', $fileName);
    
    // Validate file extension
    if (!in_array($fileExt, $allowedExtensions)) {
        return [
            'success' => false,
            'error' => 'File type not allowed. Allowed types: ' . implode(', ', $allowedExtensions),
            'file_info' => null
        ];
    }
    
    // Validate file size
    if ($fileSize > $maxFileSize) {
        return [
            'success' => false,
            'error' => 'File size exceeds the limit of ' . formatBytes($maxFileSize),
            'file_info' => null
        ];
    }
    
    // Try to move the file to the uploads directory
    $destination = $uploadsDir . $newFileName;
    
    if (move_uploaded_file($fileTmpPath, $destination)) {
        return [
            'success' => true,
            'error' => null,
            'file_info' => [
                'original_name' => $fileName,
                'new_name' => $newFileName,
                'size' => $fileSize,
                'type' => $fileType,
                'path' => '/uploads/' . $newFileName
            ]
        ];
    } else {
        return [
            'success' => false,
            'error' => 'Failed to move uploaded file. Check directory permissions.',
            'file_info' => null
        ];
    }
}

// Helper function to format file size
function formatBytes($bytes, $precision = 2) {
    $units = ['B', 'KB', 'MB', 'GB', 'TB'];
    $bytes = max($bytes, 0);
    $pow = floor(($bytes ? log($bytes) : 0) / log(1024));
    $pow = min($pow, count($units) - 1);
    return round($bytes / (1024 ** $pow), $precision) . ' ' . $units[$pow];
}

// Process single file upload
if (isset($_FILES['single_file']) && $_FILES['single_file']['error'] !== UPLOAD_ERR_NO_FILE) {
    $result = processUploadedFile($_FILES['single_file'], $allowedExtensions, $maxFileSize, $uploadsDir);
    
    if ($result['success']) {
        $successFiles[] = $result['file_info'];
    } else {
        $errors[] = 'Single file: ' . $result['error'];
    }
}

// Process multiple file uploads
if (isset($_FILES['multi_files'])) {
    $fileCount = count($_FILES['multi_files']['name']);
    
    for ($i = 0; $i < $fileCount; $i++) {
        // Skip empty file slots (no file uploaded)
        if ($_FILES['multi_files']['error'][$i] === UPLOAD_ERR_NO_FILE) {
            continue;
        }
        
        // Create a single file array from the multiple upload
        $file = [
            'name' => $_FILES['multi_files']['name'][$i],
            'type' => $_FILES['multi_files']['type'][$i],
            'tmp_name' => $_FILES['multi_files']['tmp_name'][$i],
            'error' => $_FILES['multi_files']['error'][$i],
            'size' => $_FILES['multi_files']['size'][$i]
        ];
        
        $result = processUploadedFile($file, $allowedExtensions, $maxFileSize, $uploadsDir, $i + 1);
        
        if ($result['success']) {
            $successFiles[] = $result['file_info'];
        } else {
            $errors[] = 'File #' . ($i + 1) . ' (' . $_FILES['multi_files']['name'][$i] . '): ' . $result['error'];
        }
    }
}

// Start session to store upload results
session_start();
$_SESSION['upload_results'] = [
    'success_files' => $successFiles,
    'errors' => $errors,
    'metadata' => [
        'description' => $description,
        'category' => $category,
        'timestamp' => date('Y-m-d H:i:s')
    ]
];

// Redirect to the view-files page
header('Location: /view-files');
exit; 