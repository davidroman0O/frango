<?php
/**
 * Form Handling Examples
 * 
 * Demonstrates the improved form handling capabilities in Frango v1
 */

// ./v1/playground/demo/forms/index.php
// Include the globals fix
include_once __DIR__ . '/globals_fix.php';

// Notification for developers
$fixApplied = '<div style="position: fixed; top: 0; right: 0; background: #f39c12; color: white; padding: 5px 10px; z-index: 9999; font-size: 12px;">Form Data Fix Applied</div>';

// Initialize superglobals if they don't exist
if (!isset($_PATH)) $_PATH = [];
if (!isset($_PATH_SEGMENTS)) $_PATH_SEGMENTS = [];
if (!isset($_PATH_SEGMENT_COUNT)) $_PATH_SEGMENT_COUNT = 0;
if (!isset($_JSON)) $_JSON = [];
if (!isset($_FORM)) $_FORM = [];
if (!isset($_URL)) $_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_CURRENT_URL)) $_CURRENT_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_QUERY)) $_QUERY = isset($_GET) ? $_GET : [];
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango Form Handling Examples</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
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
        a { color: #3498db; }
        .form-example {
            margin-bottom: 20px;
            padding-bottom: 20px;
            border-bottom: 1px solid #eee;
        }
        form {
            margin: 15px 0;
        }
        input[type="text"], input[type="email"], textarea, input[type="file"] {
            width: 100%;
            padding: 8px;
            margin-bottom: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        button {
            background: #3498db;
            color: white;
            border: none;
            padding: 8px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #2980b9;
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
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
        }
        .get { background: #3498db; color: white; }
        .post { background: #2ecc71; color: white; }
        .json { background: #9b59b6; color: white; }
        .form { background: #f39c12; color: white; }
        .upload { background: #e74c3c; color: white; }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
            color: #555;
        }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
            gap: 15px;
            margin-top: 20px;
        }
        .feature-box {
            background: white;
            border-radius: 6px;
            padding: 15px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .feature-box:hover {
            transform: translateY(-3px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.15);
        }
        .feature-box h3 {
            margin-top: 0;
            display: flex;
            align-items: center;
        }
        .feature-box p {
            margin-bottom: 10px;
            color: #666;
        }
        .btn {
            display: inline-block;
            padding: 8px 16px;
            margin-right: 10px;
            border-radius: 4px;
            text-decoration: none;
            font-weight: 500;
            cursor: pointer;
        }
        .btn-primary {
            background-color: #3498db;
            color: white;
        }
        .btn-secondary {
            background-color: #7f8c8d;
            color: white;
        }
        .btn:hover {
            opacity: 0.9;
        }
        .button-group {
            margin: 10px 0;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>Frango Form Handling Examples</h1>
        <p>This page demonstrates the different ways to handle form submissions and data in Frango v1.</p>
        
        <div class="features">
            <div class="feature-box">
                <h3><span class="method get">GET</span> PHP Endpoint</h3>
                <p>Send GET data to a PHP endpoint</p>
                <a href="#get-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method get">GET</span> Go Endpoint</h3>
                <p>Send GET data to a Go endpoint</p>
                <a href="#get-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method post">POST</span> PHP Endpoint</h3>
                <p>Send POST data to a PHP endpoint</p>
                <a href="#post-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method post">POST</span> Go Endpoint</h3>
                <p>Send POST data to a Go endpoint</p>
                <a href="#post-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method form">FORM</span> PHP Endpoint</h3>
                <p>Send form data to a PHP endpoint</p>
                <a href="#general-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method form">FORM</span> Go Endpoint</h3>
                <p>Send form data to a Go endpoint</p>
                <a href="#general-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method json">JSON</span> PHP Endpoint</h3>
                <p>Send JSON data to a PHP endpoint</p>
                <a href="#json-data">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method json">JSON</span> Go Endpoint</h3>
                <p>Send JSON data to a Go endpoint</p>
                <a href="#json-data">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method upload">UPLOAD</span> PHP Endpoint</h3>
                <p>Upload files to a PHP endpoint</p>
                <a href="#php-to-php-upload">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method upload">UPLOAD</span> Go Endpoint</h3>
                <p>Upload files to a Go endpoint</p>
                <a href="#php-to-go-upload">View Example</a>
            </div>
        </div>
    </div>
    
    <!-- Debug Tools -->
    <div class="card" style="border-left: 5px solid #3498db;">
        <h2>Debug Tools</h2>
        <p>Use these tools to help diagnose form processing issues:</p>
        <div class="button-group">
            <a href="/forms/debug" class="btn btn-primary" target="_blank">Form Debug Tool</a>
            <a href="/debug" class="btn btn-secondary" target="_blank">PHP Environment Debug</a>
        </div>
        <div style="background-color: #f8f9fa; padding: 10px; border-radius: 4px; margin-top: 10px;">
            <strong>Form Data Status:</strong>
            <ul>
                <li>$_POST count: <?= count($_POST) ?> item(s)</li>
                <li>$_FORM count: <?= count($_FORM ?? []) ?> item(s)</li>
                <li>PHP_FORM_* variables: <?= count(array_filter(array_keys($_SERVER), function($key) { return strpos($key, 'PHP_FORM_') === 0; })) ?> found</li>
            </ul>
        </div>
    </div>
    
    <div class="card" id="get-form">
        <h2><span class="method get">GET</span> Form Examples</h2>
        
        <!-- PHP-to-PHP GET Example -->
        <div class="form-example">
            <h3>PHP-to-PHP GET Example</h3>
            <p>GET forms submit data through URL parameters, which can be accessed via <code>$_GET</code> or <code>$_QUERY</code>.</p>
            
            <form action="/forms/get_display" method="GET">
                <div class="form-group">
                    <label for="name">Name:</label>
                    <input type="text" id="name" name="name" placeholder="Your name" value="Test User">
                </div>
                
                <div class="form-group">
                    <label for="category">Category:</label>
                    <input type="text" id="category" name="category" placeholder="Category" value="testing">
                </div>
                
                <div class="form-group">
                    <label for="limit">Results Limit:</label>
                    <input type="text" id="limit" name="limit" placeholder="Result limit" value="10">
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method get">GET</span> Submit to PHP</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access (PHP)
$name = $_GET['name'];       # Standard PHP way
$category = $_QUERY['category']; # Frango alias</div>
        </div>
        
        <!-- PHP-to-Go GET Example -->
        <div class="form-example">
            <h3>PHP-to-Go GET Example</h3>
            <p>This example sends GET parameters to a Go handler instead of PHP.</p>
            
            <form action="/api/get" method="GET">
                <div class="form-group">
                    <label for="go_name">Name:</label>
                    <input type="text" id="go_name" name="name" placeholder="Your name" value="Go Test User">
                </div>
                
                <div class="form-group">
                    <label for="go_category">Category:</label>
                    <input type="text" id="go_category" name="category" placeholder="Category" value="go-testing">
                </div>
                
                <div class="form-group">
                    <label for="go_limit">Results Limit:</label>
                    <input type="text" id="go_limit" name="limit" placeholder="Result limit" value="5">
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method get">GET</span> Submit to Go</button>
                </div>
            </form>
            
            <div id="getGoResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="getGoResponse"></pre>
            </div>
            
            <div class="code-block">
# Server-side access (Go)
func getHandler(w http.ResponseWriter, r *http.Request) {
    // Access URL query parameters
    name := r.URL.Query().Get("name")
    category := r.URL.Query().Get("category")
    limit := r.URL.Query().Get("limit")
    
    // Process and return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "received": true,
        "method": "GET",
        "params": map[string]string{
            "name": name,
            "category": category,
            "limit": limit,
        },
    })
}</div>
        </div>
    </div>
    
    <div class="card" id="post-form">
        <h2><span class="method post">POST</span> Form Examples</h2>
        
        <!-- PHP-to-PHP POST Example -->
        <div class="form-example">
            <h3>PHP-to-PHP POST Example</h3>
            <p>POST forms submit data in the request body, which can be accessed via <code>$_POST</code>.</p>
            
            <form action="/forms/post_display" method="POST">
                <div class="form-group">
                    <label for="username">Username:</label>
                    <input type="text" id="username" name="username" placeholder="Your username" value="test_user">
                </div>
                
                <div class="form-group">
                    <label for="email">Email:</label>
                    <input type="email" id="email" name="email" placeholder="Your email" value="test@example.com">
                </div>
                
                <div class="form-group">
                    <label for="comment">Comment:</label>
                    <textarea id="comment" name="comment" rows="3" placeholder="Your comment">This is a test comment to verify POST form handling</textarea>
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method post">POST</span> Submit to PHP</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access (PHP)
$username = $_POST['username']; # Standard PHP way
$email = $_POST['email'];       # Access email field</div>
        </div>
        
        <!-- PHP-to-Go POST Example -->
        <div class="form-example">
            <h3>PHP-to-Go POST Example</h3>
            <p>This example sends POST data to a Go handler instead of PHP.</p>
            
            <form id="goPostForm" action="/api/post" method="POST">
                <div class="form-group">
                    <label for="go_username">Username:</label>
                    <input type="text" id="go_username" name="username" placeholder="Your username" value="go_test_user">
                </div>
                
                <div class="form-group">
                    <label for="go_email">Email:</label>
                    <input type="email" id="go_email" name="email" placeholder="Your email" value="go_test@example.com">
                </div>
                
                <div class="form-group">
                    <label for="go_comment">Comment:</label>
                    <textarea id="go_comment" name="comment" rows="3" placeholder="Your comment">This is a test comment sent to a Go handler</textarea>
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method post">POST</span> Submit to Go</button>
                </div>
            </form>
            
            <div id="postGoResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="postGoResponse"></pre>
            </div>
            
            <div class="code-block">
# Server-side access (Go)
func postHandler(w http.ResponseWriter, r *http.Request) {
    // Parse form data
    err := r.ParseForm()
    if err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }
    
    // Access form values
    username := r.FormValue("username")
    email := r.FormValue("email")
    comment := r.FormValue("comment")
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "received": true,
        "method": "POST",
        "data": map[string]string{
            "username": username,
            "email": email,
            "comment": comment,
        },
    })
}</div>
        </div>
    </div>
    
    <div class="card" id="general-form">
        <h2><span class="method form">FORM</span> General Form Examples</h2>
        
        <!-- PHP-to-PHP General Form Example -->
        <div class="form-example">
            <h3>PHP-to-PHP General Form Example</h3>
            <p>General form data can be accessed via the <code>$_FORM</code> superglobal in Frango, which works with both GET and POST methods.</p>
            
            <form action="/forms/form_display" method="POST">
                <div class="form-group">
                    <label for="product">Product:</label>
                    <input type="text" id="product" name="product" placeholder="Product name" value="Test Product">
                </div>
                
                <div class="form-group">
                    <label for="quantity">Quantity:</label>
                    <input type="text" id="quantity" name="quantity" placeholder="Quantity" value="5">
                </div>
                
                <div class="form-group">
                    <label for="notes">Notes:</label>
                    <textarea id="notes" name="notes" rows="3" placeholder="Additional notes">Testing the $_FORM superglobal with this data</textarea>
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method form">FORM</span> Submit to PHP</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access with $_FORM superglobal (PHP)
$product = $_FORM['product'];   # Works with both GET and POST
$quantity = $_FORM['quantity']; # Access any form field</div>
        </div>
        
        <!-- PHP-to-Go General Form Example -->
        <div class="form-example">
            <h3>PHP-to-Go General Form Example</h3>
            <p>This example sends form data to a Go handler which handles both GET and POST methods.</p>
            
            <form id="goGeneralForm" action="/api/form" method="POST">
                <div class="form-group">
                    <label for="go_product">Product:</label>
                    <input type="text" id="go_product" name="product" placeholder="Product name" value="Go Test Product">
                </div>
                
                <div class="form-group">
                    <label for="go_quantity">Quantity:</label>
                    <input type="text" id="go_quantity" name="quantity" placeholder="Quantity" value="10">
                </div>
                
                <div class="form-group">
                    <label for="go_notes">Notes:</label>
                    <textarea id="go_notes" name="notes" rows="3" placeholder="Additional notes">Testing Go form handling with this data</textarea>
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method form">FORM</span> Submit to Go</button>
                </div>
            </form>
            
            <div id="formGoResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="formGoResponse"></pre>
            </div>
            
            <div class="code-block">
# Server-side access (Go)
func formHandler(w http.ResponseWriter, r *http.Request) {
    // Handle both GET and POST
    var data map[string]string
    
    if r.Method == "GET" {
        // For GET requests, use query parameters
        data = map[string]string{
            "product": r.URL.Query().Get("product"),
            "quantity": r.URL.Query().Get("quantity"),
            "notes": r.URL.Query().Get("notes"),
        }
    } else {
        // For POST requests, parse form data
        r.ParseForm()
        data = map[string]string{
            "product": r.FormValue("product"),
            "quantity": r.FormValue("quantity"),
            "notes": r.FormValue("notes"),
        }
    }
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "received": true,
        "method": r.Method,
        "data": data,
    })
}</div>
        </div>
    </div>
    
    <div class="card" id="json-data">
        <h2><span class="method json">JSON</span> Data Examples</h2>
        
        <!-- PHP-to-PHP JSON Example -->
        <div class="form-example">
            <h3>PHP-to-PHP JSON Example</h3>
            <p>JSON data sent with Content-Type: application/json can be accessed via <code>$_JSON</code>.</p>
            
            <div class="form-group">
                <label for="jsonData">JSON Data:</label>
                <textarea id="jsonData" rows="8">{
  "user": "johndoe",
  "action": "update",
  "data": {
    "id": 123,
    "status": "active",
    "items": [1, 2, 3]
  }
}</textarea>
            </div>
            
            <div class="form-group">
                <button id="sendJson"><span class="method json">JSON</span> Send to PHP</button>
            </div>
            
            <div class="code-block">
# Server-side access with $_JSON superglobal (PHP)
$user = $_JSON['user'];
$action = $_JSON['action'];
$data = $_JSON['data'];</div>
            
            <div id="jsonResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="jsonResponse"></pre>
            </div>
        </div>
        
        <!-- PHP-to-Go JSON Example -->
        <div class="form-example">
            <h3>PHP-to-Go JSON Example</h3>
            <p>This example sends JSON data directly to a Go handler.</p>
            
            <div class="form-group">
                <label for="goJsonData">JSON Data:</label>
                <textarea id="goJsonData" rows="8">{
  "user": "janedoe",
  "action": "create",
  "data": {
    "id": 456,
    "status": "pending",
    "items": [4, 5, 6]
  }
}</textarea>
            </div>
            
            <div class="form-group">
                <button id="sendGoJson"><span class="method json">JSON</span> Send to Go</button>
            </div>
            
            <div class="code-block">
# Server-side access (Go)
func jsonHandler(w http.ResponseWriter, r *http.Request) {
    // Read the request body
    var data map[string]interface{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&data)
    if err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Process the JSON data
    // (In a real app, you'd do more with this data)
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "received": true,
        "method": "JSON",
        "data": data,
    })
}</div>
            
            <div id="goJsonResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="goJsonResponse"></pre>
            </div>
        </div>
    </div>
    
    <div class="card" id="php-to-php-upload">
        <h2><span class="method upload">UPLOAD</span> PHP-to-PHP Upload Example</h2>
        <div class="form-example">
            <p>This example demonstrates uploading a file from one PHP script to another PHP script.</p>
            
            <form id="uploadForm" action="/forms/php_receiver" method="POST" enctype="multipart/form-data">
                <div class="form-group">
                    <label for="userfile">Select File:</label>
                    <input type="file" id="userfile" name="userfile">
                </div>
                
                <div class="form-group">
                    <label for="php_description">File Description:</label>
                    <input type="text" id="php_description" name="description" value="File uploaded via PHP form">
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method upload">UPLOAD</span> To PHP Endpoint</button>
                </div>
            </form>
            
            <div class="code-block">
# PHP-to-PHP Upload
# In the receiver file (php_receiver.php):
$uploadedFile = $_FILES['userfile'];
$description = $_POST['description'];

// Validate upload
if ($uploadedFile['error'] === 0) {
    // Process the file
    $fileTmpPath = $uploadedFile['tmp_name'];
    $fileName = $uploadedFile['name'];
    $fileSize = $uploadedFile['size'];
    $fileType = $uploadedFile['type'];
    
    // File was successfully uploaded
    echo "Upload successful!";
}</div>
        </div>
    </div>
    
    <div class="card" id="php-to-go-upload">
        <h2><span class="method upload">UPLOAD</span> PHP-to-Go Upload Example</h2>
        <div class="form-example">
            <p>This example demonstrates uploading a file from a PHP page directly to a Go handler.</p>
            
            <form id="goUploadForm" action="/upload" method="POST" enctype="multipart/form-data">
                <div class="form-group">
                    <label for="go_file">Select File:</label>
                    <input type="file" id="go_file" name="file">
                </div>
                
                <div class="form-group">
                    <label for="go_description">File Description:</label>
                    <input type="text" id="go_description" name="description" value="File uploaded to Go endpoint">
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method upload">UPLOAD</span> To Go Endpoint</button>
                </div>
            </form>
            
            <div id="goUploadResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="goUploadResponse"></pre>
            </div>
            
            <div class="code-block">
# Go Handler Code
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    r.ParseMultipartForm(32 << 20)
    
    // Get the file from the form
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Read form fields
    description := r.FormValue("description")
    
    // Process the file...
    
    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "message": "File uploaded successfully",
        "fileInfo": map[string]interface{}{
            "filename": header.Filename,
            "size": header.Size,
        },
    })
}</div>
        </div>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/">Back to Home</a></p>
    </div>
    
    <script>
        // PHP JSON example functionality
        document.getElementById('sendJson').addEventListener('click', function() {
            const jsonTextarea = document.getElementById('jsonData');
            let jsonData;
            
            try {
                jsonData = JSON.parse(jsonTextarea.value);
            } catch (error) {
                alert('Invalid JSON: ' + error.message);
                return;
            }
            
            // Add timestamp if not present
            if (!jsonData.timestamp) {
                jsonData.timestamp = new Date().toISOString();
            }
            
            fetch('/forms/json', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify(jsonData)
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                document.getElementById('jsonResponse').textContent = JSON.stringify(data, null, 2);
                document.getElementById('jsonResult').style.display = 'block';
            })
            .catch(error => {
                console.error('Error:', error);
                document.getElementById('jsonResponse').textContent = 'Error: ' + error.message;
                document.getElementById('jsonResult').style.display = 'block';
            });
        });
        
        // Go JSON example functionality
        document.getElementById('sendGoJson').addEventListener('click', function() {
            const jsonTextarea = document.getElementById('goJsonData');
            let jsonData;
            
            try {
                jsonData = JSON.parse(jsonTextarea.value);
            } catch (error) {
                alert('Invalid JSON: ' + error.message);
                return;
            }
            
            // Add timestamp if not present
            if (!jsonData.timestamp) {
                jsonData.timestamp = new Date().toISOString();
            }
            
            fetch('/api/json', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify(jsonData)
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                document.getElementById('goJsonResponse').textContent = JSON.stringify(data, null, 2);
                document.getElementById('goJsonResult').style.display = 'block';
            })
            .catch(error => {
                console.error('Error:', error);
                document.getElementById('goJsonResponse').textContent = 'Error: ' + error.message;
                document.getElementById('goJsonResult').style.display = 'block';
            });
        });
        
        // Go upload form handler with AJAX
        document.getElementById('goUploadForm').addEventListener('submit', function(e) {
            e.preventDefault(); // Prevent traditional form submission
            
            const formData = new FormData(this);
            const responseDiv = document.getElementById('goUploadResult');
            
            // Reset response area
            responseDiv.style.display = 'none';
            document.getElementById('goUploadResponse').textContent = '';
            
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
                document.getElementById('goUploadResponse').textContent = JSON.stringify(data, null, 2);
                responseDiv.style.display = 'block';
            })
            .catch(error => {
                document.getElementById('goUploadResponse').textContent = 'Error: ' + error.message;
                responseDiv.style.display = 'block';
            })
            .finally(() => {
                // Restore button
                button.textContent = originalText;
                button.disabled = false;
            });
        });

        // Go POST form handler with AJAX
        document.getElementById('goPostForm').addEventListener('submit', function(e) {
            e.preventDefault(); // Prevent traditional form submission
            
            const formData = new FormData(this);
            const responseDiv = document.getElementById('postGoResult');
            
            // Create URL encoded version as backup
            const urlEncodedData = new URLSearchParams();
            for (const [key, value] of formData.entries()) {
                urlEncodedData.append(key, value);
                console.log(`Form data: ${key}=${value}`); // Debug log
            }
            
            // Reset response area
            responseDiv.style.display = 'none';
            document.getElementById('postGoResponse').textContent = '';
            
            // Show loading indicator
            const button = this.querySelector('button');
            const originalText = button.textContent;
            button.textContent = 'Sending...';
            button.disabled = true;
            
            // Use URL encoded format instead of FormData
            fetch('/api/post', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: urlEncodedData
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                document.getElementById('postGoResponse').textContent = JSON.stringify(data, null, 2);
                responseDiv.style.display = 'block';
            })
            .catch(error => {
                document.getElementById('postGoResponse').textContent = 'Error: ' + error.message;
                responseDiv.style.display = 'block';
            })
            .finally(() => {
                // Restore button
                button.textContent = originalText;
                button.disabled = false;
            });
        });
        
        // Go General FORM handler with AJAX
        document.getElementById('goGeneralForm').addEventListener('submit', function(e) {
            e.preventDefault(); // Prevent traditional form submission
            
            const formData = new FormData(this);
            const responseDiv = document.getElementById('formGoResult');
            
            // Create URL encoded version for submission
            const urlEncodedData = new URLSearchParams();
            for (const [key, value] of formData.entries()) {
                urlEncodedData.append(key, value);
                console.log(`Form data: ${key}=${value}`); // Debug log
            }
            
            // Reset response area
            responseDiv.style.display = 'none';
            document.getElementById('formGoResponse').textContent = '';
            
            // Show loading indicator
            const button = this.querySelector('button');
            const originalText = button.textContent;
            button.textContent = 'Sending...';
            button.disabled = true;
            
            // Use URL encoded format instead of FormData
            fetch('/api/form', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: urlEncodedData
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                document.getElementById('formGoResponse').textContent = JSON.stringify(data, null, 2);
                responseDiv.style.display = 'block';
            })
            .catch(error => {
                document.getElementById('formGoResponse').textContent = 'Error: ' + error.message;
                responseDiv.style.display = 'block';
            })
            .finally(() => {
                // Restore button
                button.textContent = originalText;
                button.disabled = false;
            });
        });
        
        // Intercept GET forms to show response inline for Go examples
        document.addEventListener('DOMContentLoaded', function() {
            // Helper to extract query params from URL
            function getQueryParams(url) {
                const paramObj = {};
                const searchParams = new URLSearchParams(url.split('?')[1]);
                for (const [key, value] of searchParams.entries()) {
                    paramObj[key] = value;
                }
                return paramObj;
            }
            
            // Get the form element by ID
            const getGoForm = document.querySelector('form[action="/api/get"]');
            if (getGoForm) {
                getGoForm.addEventListener('submit', function(e) {
                    e.preventDefault();
                    
                    // Get form action with query params
                    const formData = new FormData(this);
                    let url = this.action + '?';
                    for (const [key, value] of formData.entries()) {
                        url += encodeURIComponent(key) + '=' + encodeURIComponent(value) + '&';
                    }
                    url = url.slice(0, -1); // Remove trailing &
                    
                    // Show loading
                    const button = this.querySelector('button');
                    const originalText = button.textContent;
                    button.textContent = 'Sending...';
                    button.disabled = true;
                    
                    // Fetch data
                    fetch(url)
                    .then(response => {
                        if (!response.ok) {
                            throw new Error(`HTTP error! Status: ${response.status}`);
                        }
                        return response.json();
                    })
                    .then(data => {
                        document.getElementById('getGoResponse').textContent = JSON.stringify(data, null, 2);
                        document.getElementById('getGoResult').style.display = 'block';
                    })
                    .catch(error => {
                        document.getElementById('getGoResponse').textContent = 'Error: ' + error.message;
                        document.getElementById('getGoResult').style.display = 'block';
                    })
                    .finally(() => {
                        // Restore button
                        button.textContent = originalText;
                        button.disabled = false;
                    });
                });
            }
        });
    </script>
</body>
</html> 