<?php
/**
 * PHP-to-Go File Upload Demo
 * 
 * Demonstrates uploading a file from PHP to a Go endpoint
 */

// ./v1/playground/demo/forms/upload_to_go.php
?>
<!DOCTYPE html>
<html>
<head>
    <title>PHP-to-Go File Upload</title>
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
        h3 { color: #16a085; }
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
            background: #16a085;
            color: white;
        }
        .code-block {
            background: #2c3e50;
            color: white;
            padding: 15px;
            border-radius: 4px;
            margin: 15px 0;
            font-family: monospace;
            white-space: pre-wrap;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input[type="file"], input[type="text"] {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-bottom: 10px;
        }
        button {
            background: #16a085;
            color: white;
            border: none;
            padding: 8px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        #response {
            background: #f8f9fa;
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 15px;
            margin-top: 20px;
            display: none;
        }
        .key-difference {
            background-color: #e8f4f8;
            border-left: 4px solid #3498db;
            padding: 15px;
            margin: 15px 0;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">GO</span> PHP-to-Go File Upload</h1>
        <p>This example demonstrates uploading a file from a PHP page directly to a Go handler - bypassing PHP processing.</p>
        
        <div class="key-difference">
            <h3>Key Difference</h3>
            <p>Unlike the PHP-to-PHP upload example, this form submits directly to a Go endpoint. The file is processed by Go code instead of PHP, demonstrating how to integrate with native Go handlers.</p>
        </div>
        
        <form id="uploadForm" action="/upload" method="POST" enctype="multipart/form-data">
            <div class="form-group">
                <label for="file">Select File:</label>
                <input type="file" id="file" name="file">
            </div>
            
            <div class="form-group">
                <label for="description">File Description:</label>
                <input type="text" id="description" name="description" value="File uploaded to Go endpoint">
            </div>
            
            <div class="form-group">
                <button type="submit">Upload to Go Endpoint</button>
            </div>
        </form>
        
        <div id="response"></div>
        
        <h2>How It Works</h2>
        <p>This form submits to a Go handler at <code>/upload</code> which:</p>
        <ol>
            <li>Extracts the file from the multipart form data</li>
            <li>Reads metadata like filename, size, and content type</li>
            <li>Processes the file (in this case, just displays information)</li>
            <li>Returns a JSON response with the results</li>
        </ol>
        
        <h3>Go Handler Code</h3>
        <div class="code-block">
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Only allow POST method
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse multipart form, 10 MB max memory
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Get the file from the form
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Read form fields
    description := r.FormValue("description")

    // Get file details
    fileInfo := map[string]interface{}{
        "filename":    header.Filename,
        "size":        header.Size,
        "contentType": header.Header.Get("Content-Type"),
        "description": description,
        "uploadTime":  time.Now().Format(time.RFC3339),
    }

    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "message": "File uploaded successfully",
        "fileInfo": fileInfo,
    })
}</div>
        
        <h2>Navigation:</h2>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
    
    <script>
        document.getElementById('uploadForm').addEventListener('submit', function(e) {
            e.preventDefault(); // Prevent traditional form submission
            
            const formData = new FormData(this);
            const responseDiv = document.getElementById('response');
            
            // Reset response area
            responseDiv.style.display = 'none';
            responseDiv.innerHTML = '';
            
            // Show loading indicator
            const button = this.querySelector('button');
            const originalText = button.textContent;
            button.textContent = 'Uploading...';
            button.disabled = true;
            
            fetch('/upload', {
                method: 'POST',
                body: formData
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                // Create a nice display of the response
                responseDiv.innerHTML = `
                    <h3>Upload Result</h3>
                    <div style="color: ${data.success ? '#155724' : '#721c24'}; font-weight: bold; margin-bottom: 10px;">
                        ${data.message}
                    </div>
                    
                    ${data.fileInfo ? `
                    <h4>File Information</h4>
                    <ul>
                        <li><strong>Filename:</strong> ${data.fileInfo.filename}</li>
                        <li><strong>Size:</strong> ${formatFileSize(data.fileInfo.size)}</li>
                        <li><strong>Content Type:</strong> ${data.fileInfo.contentType}</li>
                        <li><strong>Description:</strong> ${data.fileInfo.description}</li>
                        <li><strong>Upload Time:</strong> ${formatDate(data.fileInfo.uploadTime)}</li>
                    </ul>
                    ` : ''}
                    
                    <h4>Full Response</h4>
                    <pre>${JSON.stringify(data, null, 2)}</pre>
                `;
                responseDiv.style.display = 'block';
            })
            .catch(error => {
                responseDiv.innerHTML = `
                    <h3>Error</h3>
                    <div style="color: #721c24; font-weight: bold; margin-bottom: 10px;">
                        ${error.message}
                    </div>
                `;
                responseDiv.style.display = 'block';
            })
            .finally(() => {
                // Restore button
                button.textContent = originalText;
                button.disabled = false;
            });
        });
        
        // Helper function to format file size
        function formatFileSize(bytes) {
            if (bytes >= 1073741824) {
                return (bytes / 1073741824).toFixed(2) + ' GB';
            } else if (bytes >= 1048576) {
                return (bytes / 1048576).toFixed(2) + ' MB';
            } else if (bytes >= 1024) {
                return (bytes / 1024).toFixed(2) + ' KB';
            } else {
                return bytes + ' bytes';
            }
        }
        
        // Helper function to format date
        function formatDate(dateString) {
            try {
                const date = new Date(dateString);
                return date.toLocaleString();
            } catch (e) {
                return dateString;
            }
        }
    </script>
</body>
</html> 