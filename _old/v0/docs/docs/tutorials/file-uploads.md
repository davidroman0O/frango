# File Upload Handling in Frango

This tutorial demonstrates how to implement file uploads in your PHP applications using the Frango middleware.

## Table of Contents

- [Overview](#overview)
- [Basic File Upload](#basic-file-upload)
- [Multiple File Uploads](#multiple-file-uploads)
- [File Validation](#file-validation)
- [Storing Uploaded Files](#storing-uploaded-files)
- [Error Handling](#error-handling)
- [Advanced Techniques](#advanced-techniques)
- [Troubleshooting](#troubleshooting)

## Overview

File uploads are a common requirement in web applications. Frango seamlessly handles file uploads by making uploaded files available through PHP's `$_FILES` superglobal, just as they would be in a traditional PHP application.

PHP files in Frango have access to all standard PHP file upload functionality. The Frango middleware handles the multipart/form-data parsing and makes the files accessible to your PHP scripts.

## Basic File Upload

Let's start with a simple file upload form:

### HTML Form (upload_form.php)

```php
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Example</title>
</head>
<body>
    <h1>Upload a File</h1>
    
    <form action="/upload.php" method="post" enctype="multipart/form-data">
        <div>
            <label for="user_file">Select a file:</label>
            <input type="file" name="user_file" id="user_file">
        </div>
        <div style="margin-top: 10px;">
            <input type="submit" value="Upload File">
        </div>
    </form>
</body>
</html>
```

### PHP Handler (upload.php)

```php
<?php
// Check if a file was uploaded
if (isset($_FILES['user_file']) && $_FILES['user_file']['error'] === UPLOAD_ERR_OK) {
    // Get file details
    $file_name = $_FILES['user_file']['name'];
    $file_tmp = $_FILES['user_file']['tmp_name'];
    $file_size = $_FILES['user_file']['size'];
    $file_type = $_FILES['user_file']['type'];

    // Display file information
    echo "<h2>File Uploaded Successfully</h2>";
    echo "<p><strong>File Name:</strong> " . htmlspecialchars($file_name) . "</p>";
    echo "<p><strong>File Size:</strong> " . number_format($file_size / 1024, 2) . " KB</p>";
    echo "<p><strong>File Type:</strong> " . htmlspecialchars($file_type) . "</p>";
    
    // Example of reading the file content (for text files)
    if (strpos($file_type, 'text/') === 0) {
        $content = file_get_contents($file_tmp);
        echo "<h3>File Preview:</h3>";
        echo "<pre>" . htmlspecialchars($content) . "</pre>";
    }
} else {
    // Handle upload errors
    echo "<h2>Upload Error</h2>";
    if (isset($_FILES['user_file'])) {
        switch ($_FILES['user_file']['error']) {
            case UPLOAD_ERR_INI_SIZE:
                $message = "The uploaded file exceeds the upload_max_filesize directive in php.ini";
                break;
            case UPLOAD_ERR_FORM_SIZE:
                $message = "The uploaded file exceeds the MAX_FILE_SIZE directive in the HTML form";
                break;
            case UPLOAD_ERR_PARTIAL:
                $message = "The uploaded file was only partially uploaded";
                break;
            case UPLOAD_ERR_NO_FILE:
                $message = "No file was uploaded";
                break;
            case UPLOAD_ERR_NO_TMP_DIR:
                $message = "Missing a temporary folder";
                break;
            case UPLOAD_ERR_CANT_WRITE:
                $message = "Failed to write file to disk";
                break;
            case UPLOAD_ERR_EXTENSION:
                $message = "A PHP extension stopped the file upload";
                break;
            default:
                $message = "Unknown upload error";
        }
        echo "<p>" . $message . "</p>";
    } else {
        echo "<p>No file was uploaded</p>";
    }
}
?>

<p><a href="/upload_form.php">Back to upload form</a></p>
```

### Go Setup

In your Go application, set up your routes to serve both the form and handle the upload:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/davidroman0O/go-php/v1"
)

func main() {
    // Initialize Frango middleware
    php, err := frango.New(
        frango.WithSourceDir("./php-files"),
        frango.WithDevelopmentMode(true),
    )
    if err != nil {
        log.Fatalf("Failed to initialize Frango: %v", err)
    }
    defer php.Shutdown()
    
    // Set up routes
    http.Handle("/upload_form.php", php.For("/upload_form.php"))
    http.Handle("/upload.php", php.For("/upload.php"))
    
    // Start the server
    log.Println("Server running on http://localhost:8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

## Multiple File Uploads

Handling multiple file uploads is similar to single file uploads, but with some differences in the HTML form and PHP processing code.

### HTML Form (multiple_upload_form.php)

```php
<!DOCTYPE html>
<html>
<head>
    <title>Multiple File Upload Example</title>
</head>
<body>
    <h1>Upload Multiple Files</h1>
    
    <form action="/multiple_upload.php" method="post" enctype="multipart/form-data">
        <div>
            <label for="user_files">Select files:</label>
            <input type="file" name="user_files[]" id="user_files" multiple>
        </div>
        <div style="margin-top: 10px;">
            <input type="submit" value="Upload Files">
        </div>
    </form>
</body>
</html>
```

### PHP Handler (multiple_upload.php)

```php
<?php
// Check if files were uploaded
if (isset($_FILES['user_files']) && !empty($_FILES['user_files']['name'][0])) {
    echo "<h2>Files Uploaded</h2>";
    echo "<ul>";
    
    // Count files
    $file_count = count($_FILES['user_files']['name']);
    
    for ($i = 0; $i < $file_count; $i++) {
        // Check for errors
        if ($_FILES['user_files']['error'][$i] === UPLOAD_ERR_OK) {
            $file_name = $_FILES['user_files']['name'][$i];
            $file_tmp = $_FILES['user_files']['tmp_name'][$i];
            $file_size = $_FILES['user_files']['size'][$i];
            $file_type = $_FILES['user_files']['type'][$i];
            
            echo "<li>";
            echo "<strong>" . htmlspecialchars($file_name) . "</strong> - ";
            echo number_format($file_size / 1024, 2) . " KB, ";
            echo htmlspecialchars($file_type);
            echo "</li>";
            
            // Process file here...
        } else {
            echo "<li>Error uploading file #" . ($i + 1) . "</li>";
        }
    }
    echo "</ul>";
} else {
    echo "<p>No files were uploaded</p>";
}
?>

<p><a href="/multiple_upload_form.php">Back to upload form</a></p>
```

Update your Go routes to include these new files:

```go
// Add these to your existing routes
http.Handle("/multiple_upload_form.php", php.For("/multiple_upload_form.php"))
http.Handle("/multiple_upload.php", php.For("/multiple_upload.php"))
```

## File Validation

It's essential to validate uploaded files to ensure they meet your requirements and to prevent security issues. Here's an example of basic file validation:

```php
<?php
// File validation function
function validateFile($file) {
    // Initialize result array
    $result = [
        'valid' => false,
        'error' => ''
    ];
    
    // Check for upload errors
    if ($file['error'] !== UPLOAD_ERR_OK) {
        $result['error'] = 'File upload error: ' . $file['error'];
        return $result;
    }
    
    // Check file size (max 5MB)
    $maxFileSize = 5 * 1024 * 1024; // 5MB in bytes
    if ($file['size'] > $maxFileSize) {
        $result['error'] = 'File is too large (max ' . ($maxFileSize / 1024 / 1024) . 'MB)';
        return $result;
    }
    
    // Check file type/extension
    $allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'application/pdf'];
    if (!in_array($file['type'], $allowedTypes)) {
        $result['error'] = 'Invalid file type. Allowed types: JPEG, PNG, GIF, PDF';
        return $result;
    }
    
    // Additional validation: check file extension based on actual content
    $finfo = new finfo(FILEINFO_MIME_TYPE);
    $fileContentsType = $finfo->file($file['tmp_name']);
    if (!in_array($fileContentsType, $allowedTypes)) {
        $result['error'] = 'File content does not match the expected type';
        return $result;
    }
    
    // All validations passed
    $result['valid'] = true;
    return $result;
}

// Usage in a file upload handler
if (isset($_FILES['user_file'])) {
    $validation = validateFile($_FILES['user_file']);
    
    if ($validation['valid']) {
        // Process the valid file
        echo "<p>File validation passed!</p>";
        // Continue with file processing...
    } else {
        // Display validation error
        echo "<p>Validation error: " . htmlspecialchars($validation['error']) . "</p>";
    }
}
?>
```

## Storing Uploaded Files

Uploaded files in PHP are initially stored in a temporary location. You need to move them to a permanent location to preserve them. Here's how to do it in Frango:

```php
<?php
// Check if a file was uploaded
if (isset($_FILES['user_file']) && $_FILES['user_file']['error'] === UPLOAD_ERR_OK) {
    // Validate file first
    $validation = validateFile($_FILES['user_file']);
    
    if ($validation['valid']) {
        // Create uploads directory if it doesn't exist
        $uploadDir = './uploads/';
        if (!is_dir($uploadDir)) {
            mkdir($uploadDir, 0755, true);
        }
        
        // Generate a unique filename to prevent overwrites
        $filename = uniqid() . '_' . basename($_FILES['user_file']['name']);
        $targetPath = $uploadDir . $filename;
        
        // Attempt to move the file from temp location to target location
        if (move_uploaded_file($_FILES['user_file']['tmp_name'], $targetPath)) {
            echo "<h2>File Uploaded and Stored Successfully</h2>";
            echo "<p>Your file has been saved as: " . htmlspecialchars($filename) . "</p>";
            
            // If it's an image, display it
            $fileType = $_FILES['user_file']['type'];
            if (strpos($fileType, 'image/') === 0) {
                echo "<h3>Image Preview:</h3>";
                echo "<img src='/uploads/" . htmlspecialchars($filename) . "' style='max-width: 500px;'>";
            }
        } else {
            echo "<h2>Error</h2>";
            echo "<p>Failed to store the uploaded file.</p>";
        }
    } else {
        echo "<h2>Validation Error</h2>";
        echo "<p>" . htmlspecialchars($validation['error']) . "</p>";
    }
} else {
    echo "<p>No file was uploaded or an error occurred.</p>";
}
?>
```

In your Go application, you'll need to add a route to serve the uploaded files:

```go
// Add this to your routes
http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./php-files/uploads"))))
```

## Error Handling

PHP provides specific error codes for file uploads. Here's a comprehensive error handling function:

```php
<?php
function getFileUploadError($error_code) {
    switch ($error_code) {
        case UPLOAD_ERR_OK:
            return 'No error, the file was uploaded successfully.';
        case UPLOAD_ERR_INI_SIZE:
            return 'The uploaded file exceeds the upload_max_filesize directive in php.ini.';
        case UPLOAD_ERR_FORM_SIZE:
            return 'The uploaded file exceeds the MAX_FILE_SIZE directive in the HTML form.';
        case UPLOAD_ERR_PARTIAL:
            return 'The uploaded file was only partially uploaded.';
        case UPLOAD_ERR_NO_FILE:
            return 'No file was uploaded.';
        case UPLOAD_ERR_NO_TMP_DIR:
            return 'Missing a temporary folder.';
        case UPLOAD_ERR_CANT_WRITE:
            return 'Failed to write file to disk.';
        case UPLOAD_ERR_EXTENSION:
            return 'A PHP extension stopped the file upload.';
        default:
            return 'Unknown upload error.';
    }
}

// Usage
if (isset($_FILES['user_file']['error'])) {
    echo getFileUploadError($_FILES['user_file']['error']);
}
?>
```

## Advanced Techniques

### AJAX File Uploads

For modern web applications, AJAX file uploads provide a better user experience. Here's a simple example using JavaScript's Fetch API:

#### HTML Form (ajax_upload_form.php)

```php
<!DOCTYPE html>
<html>
<head>
    <title>AJAX File Upload</title>
    <style>
        .progress-bar {
            width: 300px;
            height: 20px;
            border: 1px solid #ccc;
            border-radius: 5px;
            margin-top: 10px;
            display: none;
        }
        .progress-bar-fill {
            height: 100%;
            background-color: #4CAF50;
            width: 0%;
            border-radius: 5px;
            transition: width 0.3s;
        }
        #upload-status {
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <h1>AJAX File Upload</h1>
    
    <form id="upload-form">
        <div>
            <label for="ajax-file">Select a file:</label>
            <input type="file" name="ajax_file" id="ajax-file">
        </div>
        <div style="margin-top: 10px;">
            <button type="submit">Upload File</button>
        </div>
    </form>
    
    <div class="progress-bar" id="progress-bar">
        <div class="progress-bar-fill" id="progress-bar-fill"></div>
    </div>
    
    <div id="upload-status"></div>
    
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            const form = document.getElementById('upload-form');
            const progressBar = document.getElementById('progress-bar');
            const progressBarFill = document.getElementById('progress-bar-fill');
            const statusDiv = document.getElementById('upload-status');
            
            form.addEventListener('submit', function(e) {
                e.preventDefault();
                
                const fileInput = document.getElementById('ajax-file');
                const file = fileInput.files[0];
                
                if (!file) {
                    statusDiv.innerHTML = '<p style="color: red;">Please select a file</p>';
                    return;
                }
                
                // Create FormData object
                const formData = new FormData();
                formData.append('ajax_file', file);
                
                // Show progress bar
                progressBar.style.display = 'block';
                progressBarFill.style.width = '0%';
                statusDiv.innerHTML = '<p>Uploading...</p>';
                
                // Send AJAX request
                fetch('/ajax_upload.php', {
                    method: 'POST',
                    body: formData,
                })
                .then(response => response.json())
                .then(data => {
                    // Update progress to 100%
                    progressBarFill.style.width = '100%';
                    
                    // Show result
                    if (data.success) {
                        statusDiv.innerHTML = `
                            <p style="color: green;">File uploaded successfully!</p>
                            <p><strong>Name:</strong> ${data.fileName}</p>
                            <p><strong>Size:</strong> ${data.fileSize} KB</p>
                            <p><strong>Type:</strong> ${data.fileType}</p>
                        `;
                        
                        // Clear form
                        form.reset();
                    } else {
                        statusDiv.innerHTML = `<p style="color: red;">Error: ${data.error}</p>`;
                    }
                })
                .catch(error => {
                    progressBar.style.display = 'none';
                    statusDiv.innerHTML = `<p style="color: red;">Upload failed: ${error.message}</p>`;
                });
            });
        });
    </script>
</body>
</html>
```

#### PHP Handler (ajax_upload.php)

```php
<?php
// Set content type to JSON
header('Content-Type: application/json');

// Prepare response array
$response = [
    'success' => false,
    'error' => '',
    'fileName' => '',
    'fileSize' => 0,
    'fileType' => ''
];

// Check if file was uploaded
if (isset($_FILES['ajax_file']) && $_FILES['ajax_file']['error'] === UPLOAD_ERR_OK) {
    // Get file details
    $file_name = $_FILES['ajax_file']['name'];
    $file_tmp = $_FILES['ajax_file']['tmp_name'];
    $file_size = $_FILES['ajax_file']['size'];
    $file_type = $_FILES['ajax_file']['type'];
    
    // Create uploads directory if it doesn't exist
    $uploadDir = './uploads/';
    if (!is_dir($uploadDir)) {
        mkdir($uploadDir, 0755, true);
    }
    
    // Generate a unique filename
    $filename = uniqid() . '_' . basename($file_name);
    $targetPath = $uploadDir . $filename;
    
    // Attempt to move the file
    if (move_uploaded_file($file_tmp, $targetPath)) {
        $response['success'] = true;
        $response['fileName'] = $file_name;
        $response['fileSize'] = round($file_size / 1024, 2); // Convert to KB
        $response['fileType'] = $file_type;
    } else {
        $response['error'] = 'Failed to store the uploaded file';
    }
} else {
    // Handle upload error
    if (isset($_FILES['ajax_file'])) {
        $response['error'] = getFileUploadError($_FILES['ajax_file']['error']);
    } else {
        $response['error'] = 'No file was uploaded';
    }
}

// Helper function for error messages
function getFileUploadError($error_code) {
    switch ($error_code) {
        case UPLOAD_ERR_OK:
            return 'No error';
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

// Send JSON response
echo json_encode($response);
?>
```

Add the route for the AJAX upload in your Go application:

```go
http.Handle("/ajax_upload_form.php", php.For("/ajax_upload_form.php"))
http.Handle("/ajax_upload.php", php.For("/ajax_upload.php"))
```

### Image Processing with Uploaded Files

You can use PHP's GD library to process uploaded images. Here's an example that creates a thumbnail:

```php
<?php
// Check if an image was uploaded
if (isset($_FILES['image']) && $_FILES['image']['error'] === UPLOAD_ERR_OK) {
    // Get file details
    $file_name = $_FILES['image']['name'];
    $file_tmp = $_FILES['image']['tmp_name'];
    $file_type = $_FILES['image']['type'];
    
    // Check if the file is an image
    if (strpos($file_type, 'image/') === 0) {
        // Create upload directories
        $uploadDir = './uploads/';
        $thumbDir = './uploads/thumbs/';
        
        if (!is_dir($uploadDir)) {
            mkdir($uploadDir, 0755, true);
        }
        
        if (!is_dir($thumbDir)) {
            mkdir($thumbDir, 0755, true);
        }
        
        // Generate unique filenames
        $filename = uniqid() . '_' . basename($file_name);
        $filePath = $uploadDir . $filename;
        $thumbPath = $thumbDir . $filename;
        
        // Move the uploaded file
        if (move_uploaded_file($file_tmp, $filePath)) {
            // Create thumbnail
            createThumbnail($filePath, $thumbPath, 200, 200);
            
            echo "<h2>Image Uploaded Successfully</h2>";
            echo "<div style='display: flex; gap: 20px;'>";
            echo "<div>";
            echo "<h3>Original Image:</h3>";
            echo "<img src='/uploads/" . htmlspecialchars($filename) . "' style='max-width: 500px;'>";
            echo "</div>";
            echo "<div>";
            echo "<h3>Thumbnail:</h3>";
            echo "<img src='/uploads/thumbs/" . htmlspecialchars($filename) . "'>";
            echo "</div>";
            echo "</div>";
        } else {
            echo "<p>Failed to move the uploaded file.</p>";
        }
    } else {
        echo "<p>The uploaded file is not an image.</p>";
    }
} else {
    echo "<p>No image was uploaded or an error occurred.</p>";
}

/**
 * Creates a thumbnail from an image
 */
function createThumbnail($source, $destination, $maxWidth, $maxHeight) {
    // Get image info
    list($width, $height, $type) = getimagesize($source);
    
    // Calculate new dimensions
    $ratio = min($maxWidth / $width, $maxHeight / $height);
    $newWidth = $width * $ratio;
    $newHeight = $height * $ratio;
    
    // Create new image resource
    $thumb = imagecreatetruecolor($newWidth, $newHeight);
    
    // Load source image
    switch ($type) {
        case IMAGETYPE_JPEG:
            $sourceImg = imagecreatefromjpeg($source);
            break;
        case IMAGETYPE_PNG:
            $sourceImg = imagecreatefrompng($source);
            // Preserve transparency
            imagealphablending($thumb, false);
            imagesavealpha($thumb, true);
            break;
        case IMAGETYPE_GIF:
            $sourceImg = imagecreatefromgif($source);
            break;
        default:
            return false;
    }
    
    // Resize image
    imagecopyresampled($thumb, $sourceImg, 0, 0, 0, 0, $newWidth, $newHeight, $width, $height);
    
    // Save thumbnail
    switch ($type) {
        case IMAGETYPE_JPEG:
            imagejpeg($thumb, $destination, 90);
            break;
        case IMAGETYPE_PNG:
            imagepng($thumb, $destination);
            break;
        case IMAGETYPE_GIF:
            imagegif($thumb, $destination);
            break;
    }
    
    // Free up memory
    imagedestroy($sourceImg);
    imagedestroy($thumb);
    
    return true;
}
?>
```

Add routes for the thumb directory:

```go
http.Handle("/uploads/thumbs/", http.StripPrefix("/uploads/thumbs/", http.FileServer(http.Dir("./php-files/uploads/thumbs"))))
```

## Troubleshooting

### Common Issues and Solutions

1. **File size exceeds limit**

   PHP has several configuration directives that limit file upload sizes:
   
   - `upload_max_filesize` in php.ini (default is often 2MB)
   - `post_max_size` in php.ini (must be larger than upload_max_filesize)
   - `memory_limit` in php.ini
   
   Solution: In your Go application, you can set these values when initializing Frango:
   
   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(map[string]string{
           "upload_max_filesize": "20M",
           "post_max_size": "25M",
           "memory_limit": "128M",
       }),
   )
   ```

2. **Files not uploading**

   Check that your HTML form has the correct enctype attribute:
   
   ```html
   <form action="/upload.php" method="post" enctype="multipart/form-data">
   ```
   
   The `enctype="multipart/form-data"` is essential for file uploads.

3. **Permission issues**

   Ensure the directory where files are being uploaded to has the correct permissions.
   
   ```
   chmod 755 ./php-files/uploads
   ```
   
   The PHP process needs write permissions to the directory.

4. **Temporary directory issues**

   If you see errors about missing temporary directories, check that the PHP configuration has a valid temporary directory setting.
   
   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(map[string]string{
           "upload_tmp_dir": "/tmp",
       }),
   )
   ```

### Debugging File Uploads

For debugging file uploads, it's helpful to display the full contents of the `$_FILES` array:

```php
<?php
// Debugging: Display the $_FILES array
echo "<h2>Debug Info:</h2>";
echo "<pre>";
var_dump($_FILES);
echo "</pre>";
?>
```

You can also check PHP's error log for issues:

```php
<?php
// Write to error log
error_log("Upload attempt: " . json_encode($_FILES));
?>
```

## Conclusion

File uploads in Frango work just like regular PHP applications. The middleware transparently handles all the details, allowing you to focus on your application logic. By following the best practices and examples in this tutorial, you can implement secure, robust file upload functionality in your Frango applications.

Remember to always validate files before storing them, handle errors gracefully, and follow security best practices to prevent vulnerabilities like unrestricted file uploads.

## Next Steps

- Check out the [Form Handling](./form-handling.md) tutorial for more information on handling form data
- See the [Virtual Filesystem](./virtual-filesystem.md) tutorial to learn more about managing files
- For server configuration recommendations, refer to the [Deployment](../guides/deployment.md) guide 