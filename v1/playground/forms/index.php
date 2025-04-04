<?php
/**
 * Form Handling Examples
 * 
 * Demonstrates the improved form handling capabilities in Frango v1
 */

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
                <h3><span class="method get">GET</span> URL Parameters</h3>
                <p>Test form data in URL query parameters</p>
                <a href="#get-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method post">POST</span> Form Data</h3>
                <p>Test standard POST form submissions</p>
                <a href="#post-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method form">FORM</span> General</h3>
                <p>Test general form data handling</p>
                <a href="#general-form">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method json">JSON</span> Data</h3>
                <p>Test JSON data handling</p>
                <a href="#json-data">View Example</a>
            </div>
            
            <div class="feature-box">
                <h3><span class="method upload">UPLOAD</span> Files</h3>
                <p>Test file upload functionality</p>
                <a href="#file-upload">View Example</a>
            </div>
        </div>
    </div>
    
    <!-- Debug Tools -->
    <div class="card" style="border-left: 5px solid #3498db;">
        <h2>Debug Tools</h2>
        <p>Use these tools to help diagnose form processing issues:</p>
        <div class="button-group">
            <a href="/forms/form_debug.php" class="btn btn-primary" target="_blank">Form Debug Tool</a>
            <a href="/debug.php" class="btn btn-secondary" target="_blank">PHP Environment Debug</a>
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
        <h2><span class="method get">GET</span> Form Example</h2>
        <div class="form-example">
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
                    <button type="submit"><span class="method get">GET</span> Submit Form</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access
$name = $_GET['name'];       # Standard PHP way
$category = $_QUERY['category']; # Frango alias</div>
        </div>
    </div>
    
    <div class="card" id="post-form">
        <h2><span class="method post">POST</span> Form Example</h2>
        <div class="form-example">
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
                    <button type="submit"><span class="method post">POST</span> Submit Form</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access
$username = $_POST['username']; # Standard PHP way
$email = $_POST['email'];       # Access email field</div>
        </div>
    </div>
    
    <div class="card" id="general-form">
        <h2><span class="method form">FORM</span> General Form Data</h2>
        <div class="form-example">
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
                    <button type="submit"><span class="method form">FORM</span> Submit Form</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access with $_FORM superglobal
$product = $_FORM['product'];   # Works with both GET and POST
$quantity = $_FORM['quantity']; # Access any form field</div>
        </div>
    </div>
    
    <div class="card" id="json-data">
        <h2><span class="method json">JSON</span> Data Example</h2>
        <div class="form-example">
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
                <button id="sendJson"><span class="method json">JSON</span> Send JSON Request</button>
            </div>
            
            <div class="code-block">
# Server-side access with $_JSON superglobal
$user = $_JSON['user'];
$action = $_JSON['action'];
$data = $_JSON['data'];</div>
            
            <div id="jsonResult" style="margin-top: 15px; display: none;">
                <h4>Response:</h4>
                <pre id="jsonResponse"></pre>
            </div>
        </div>
    </div>
    
    <div class="card" id="file-upload">
        <h2><span class="method upload">UPLOAD</span> File Example</h2>
        <div class="form-example">
            <p>File uploads are handled via <code>$_FILES</code> superglobal and multipart/form-data encoding.</p>
            
            <form action="/forms/upload_display" method="POST" enctype="multipart/form-data">
                <div class="form-group">
                    <label for="userfile">Select File:</label>
                    <input type="file" id="userfile" name="userfile">
                </div>
                
                <div class="form-group">
                    <label for="description">File Description:</label>
                    <input type="text" id="description" name="description" value="Test file upload">
                </div>
                
                <div class="form-group">
                    <button type="submit"><span class="method upload">UPLOAD</span> Upload File</button>
                </div>
            </form>
            
            <div class="code-block">
# Server-side access for file uploads
$fileName = $_FILES['userfile']['name'];
$fileType = $_FILES['userfile']['type'];
$fileSize = $_FILES['userfile']['size'];
$fileTmpPath = $_FILES['userfile']['tmp_name'];</div>
        </div>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/">Back to Home</a></p>
    </div>
    
    <script>
        // JSON example functionality
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
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(jsonData)
            })
            .then(response => response.json())
            .then(data => {
                document.getElementById('jsonResponse').textContent = JSON.stringify(data, null, 2);
                document.getElementById('jsonResult').style.display = 'block';
            })
            .catch(error => {
                document.getElementById('jsonResponse').textContent = 'Error: ' + error.message;
                document.getElementById('jsonResult').style.display = 'block';
            });
        });
    </script>
</body>
</html> 