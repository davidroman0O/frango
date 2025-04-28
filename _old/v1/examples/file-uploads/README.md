# File Upload Example

This example demonstrates how to handle file uploads with Frango, showing the integration between Go's file handling and PHP's file processing capabilities.

## What This Example Shows

- Creating file upload forms in PHP
- Processing uploaded files with PHP using `$_FILES`
- Validating file types and sizes
- Moving uploaded files to a permanent location
- Accessing file system from PHP
- Serving uploaded files via Go
- Single and multiple file uploads
- Displaying file listings and previews

## Files

- `main.go`: Go HTTP server with file upload routes
- `php/index.php`: Home page with explanation of file uploads
- `php/upload.php`: Form for uploading files
- `php/process-upload.php`: Script to process uploaded files
- `php/view-files.php`: Shows uploaded files and upload results

## File Upload Process

1. The user submits files via an HTML form with `enctype="multipart/form-data"`
2. PHP receives the files in the `$_FILES` superglobal
3. PHP validates and processes the files (type checking, size limits, etc.)
4. PHP moves valid files from temporary storage to a permanent location
5. Go serves the uploaded files from the uploads directory

## Key Features

1. **Directory Access**: PHP can access the uploads directory using file system paths
2. **Multiple Upload Types**: Both single and multiple file uploads are supported
3. **Validation**: File types and sizes are validated before storing
4. **File Serving**: Go serves the uploaded files via a standard file server

## Running the Example

1. Navigate to this directory
2. Run:
   ```bash
   go run main.go
   ```
3. Open a browser and visit http://localhost:8080
4. Try uploading files and viewing the results

## Important Code Samples

```go
// Create uploads directory if it doesn't exist
uploadsDir := "./uploads"
if err := os.MkdirAll(uploadsDir, 0755); err != nil {
    log.Fatalf("Failed to create uploads directory: %v", err)
}

// Serve uploaded files
mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))
```

```php
// Accessing uploaded files in PHP
$file = $_FILES['single_file'];
if ($file['error'] === UPLOAD_ERR_OK) {
    // Process the file
    move_uploaded_file($file['tmp_name'], $destination);
}
``` 