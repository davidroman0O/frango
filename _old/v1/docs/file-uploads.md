# File Upload Handling

This guide covers how to handle file uploads with Frango, both from the PHP side and the Go side.

## Creating a File Upload Form in PHP

First, create an HTML form in your PHP script that supports file uploads:

```php
<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Example</title>
</head>
<body>
    <h1>Upload File</h1>
    
    <form action="/upload" method="POST" enctype="multipart/form-data">
        <div>
            <label for="file">Select file:</label>
            <input type="file" name="file" id="file" required>
        </div>
        
        <div>
            <label for="description">Description:</label>
            <input type="text" name="description" id="description">
        </div>
        
        <button type="submit">Upload</button>
    </form>
</body>
</html>
```

## Handling File Uploads in PHP

Create a PHP script to process the uploaded file:

```php
<?php
header('Content-Type: text/html');

// Check if a file was uploaded
if (isset($_FILES['file']) && $_FILES['file']['error'] === UPLOAD_ERR_OK) {
    // Get uploaded file details
    $fileName = $_FILES['file']['name'];
    $fileType = $_FILES['file']['type'];
    $fileSize = $_FILES['file']['size'];
    $fileTmpPath = $_FILES['file']['tmp_name'];
    
    // Get form fields
    $description = $_POST['description'] ?? '';
    
    // Create a safe filename
    $safeFileName = preg_replace('/[^a-zA-Z0-9_.-]/', '_', $fileName);
    
    // Move the file to a permanent location
    $uploadDir = '/tmp/uploads/';
    $destinationPath = $uploadDir . $safeFileName;
    
    // Make sure the upload directory exists
    if (!is_dir($uploadDir)) {
        mkdir($uploadDir, 0755, true);
    }
    
    // Move the file
    $success = move_uploaded_file($fileTmpPath, $destinationPath);
    
    if ($success) {
        echo "<h2>File Uploaded Successfully</h2>";
        echo "<p>File: " . htmlspecialchars($fileName) . "</p>";
        echo "<p>Size: " . htmlspecialchars(number_format($fileSize / 1024, 2)) . " KB</p>";
        echo "<p>Type: " . htmlspecialchars($fileType) . "</p>";
        echo "<p>Description: " . htmlspecialchars($description) . "</p>";
    } else {
        echo "<h2>Upload Failed</h2>";
        echo "<p>There was an error moving the uploaded file.</p>";
    }
} else {
    // Display upload errors
    $errorCode = $_FILES['file']['error'] ?? UPLOAD_ERR_NO_FILE;
    
    echo "<h2>Upload Error</h2>";
    echo "<p>Error code: " . $errorCode . "</p>";
    
    switch ($errorCode) {
        case UPLOAD_ERR_INI_SIZE:
            echo "<p>The uploaded file exceeds the upload_max_filesize directive in php.ini.</p>";
            break;
        case UPLOAD_ERR_FORM_SIZE:
            echo "<p>The uploaded file exceeds the MAX_FILE_SIZE directive in the HTML form.</p>";
            break;
        case UPLOAD_ERR_PARTIAL:
            echo "<p>The uploaded file was only partially uploaded.</p>";
            break;
        case UPLOAD_ERR_NO_FILE:
            echo "<p>No file was uploaded.</p>";
            break;
        case UPLOAD_ERR_NO_TMP_DIR:
            echo "<p>Missing a temporary folder.</p>";
            break;
        case UPLOAD_ERR_CANT_WRITE:
            echo "<p>Failed to write file to disk.</p>";
            break;
        case UPLOAD_ERR_EXTENSION:
            echo "<p>A PHP extension stopped the file upload.</p>";
            break;
        default:
            echo "<p>Unknown upload error.</p>";
    }
}
?>
```

## Setting Up Routes in Go

Set up the routes in your Go application:

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create HTTP server
	mux := http.NewServeMux()

	// Route for the upload form
	mux.Handle("/", php.For("upload_form.php"))
	
	// Route for handling the upload
	mux.Handle("/upload", php.For("upload_handler.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Handling File Uploads in Go

You can also handle file uploads directly in Go and then execute PHP scripts:

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create HTTP server
	mux := http.NewServeMux()

	// Route for the upload form
	mux.Handle("/", php.For("upload_form.php"))
	
	// Handle uploads in Go
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST method
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse multipart form, 10MB max memory
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get file from form
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read form fields
		description := r.FormValue("description")

		// Create a unique filename
		timestamp := time.Now().UnixNano()
		fileName := fmt.Sprintf("%d_%s", timestamp, header.Filename)
		uploadDir := "./uploads"
		filePath := filepath.Join(uploadDir, fileName)

		// Create upload directory if it doesn't exist
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			http.Error(w, "Failed to create upload directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Create the file
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to create file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the destination file
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Run PHP with the upload information
		vfs := php.NewVFS()
		defer vfs.Cleanup()

		// Create dynamic PHP script to display upload result
		phpScript := `<?php
		header('Content-Type: text/html');
		?>
		<!DOCTYPE html>
		<html>
		<head>
			<title>Upload Result</title>
		</head>
		<body>
			<h1>File Uploaded Successfully</h1>
			<p>File: <?= htmlspecialchars($fileName) ?></p>
			<p>Size: <?= htmlspecialchars(number_format($fileSize / 1024, 2)) ?> KB</p>
			<p>Type: <?= htmlspecialchars($fileType) ?></p>
			<p>Description: <?= htmlspecialchars($description) ?></p>
			<p>Saved to: <?= htmlspecialchars($filePath) ?></p>
			<p><a href="/">Upload Another File</a></p>
		</body>
		</html>`

		// Create the PHP script in the VFS
		vfs.CreateVirtualFile("/result.php", []byte(phpScript))

		// Execute PHP with template data
		renderData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"fileName":    header.Filename,
				"fileSize":    header.Size,
				"fileType":    header.Header.Get("Content-Type"),
				"description": description,
				"filePath":    filePath,
			}
		}

		php.ExecutePHP("/result.php", vfs, renderData, w, r)
	})

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Multiple File Uploads

To handle multiple file uploads, use an array notation in your form:

```php
<form action="/upload" method="POST" enctype="multipart/form-data">
    <div>
        <label for="files">Select files:</label>
        <input type="file" name="files[]" id="files" multiple required>
    </div>
    
    <button type="submit">Upload</button>
</form>
```

And in your PHP handler:

```php
<?php
// Check for multiple file uploads
if (isset($_FILES['files'])) {
    $fileCount = count($_FILES['files']['name']);
    
    for ($i = 0; $i < $fileCount; $i++) {
        // Check if this file was uploaded successfully
        if ($_FILES['files']['error'][$i] === UPLOAD_ERR_OK) {
            $fileName = $_FILES['files']['name'][$i];
            $fileTmpPath = $_FILES['files']['tmp_name'][$i];
            $fileSize = $_FILES['files']['size'][$i];
            $fileType = $_FILES['files']['type'][$i];
            
            // Process each file...
            echo "<p>Uploaded: " . htmlspecialchars($fileName) . " (" . 
                 number_format($fileSize / 1024, 2) . " KB)</p>";
        } else {
            echo "<p>Error uploading " . htmlspecialchars($_FILES['files']['name'][$i]) . "</p>";
        }
    }
}
?>
```

## Best Practices for File Uploads

1. **Validate file types** - Always verify that uploaded files are of the expected type
2. **Limit file sizes** - Set maximum upload size limits to prevent server overload
3. **Use unique filenames** - Generate unique names to prevent overwriting existing files
4. **Sanitize filenames** - Remove special characters from filenames
5. **Store files outside web root** - When possible, store uploaded files outside the web-accessible directory
6. **Validate image files** - For images, verify they are valid by attempting to process them
7. **Set appropriate permissions** - Ensure uploaded files have the correct permissions 