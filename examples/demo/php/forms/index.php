<?php
/**
 * Frango v2 Demo - Forms Index
 * 
 * This page provides access to various form handling examples.
 */
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Frango v2 - Forms Demo</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>Forms Processing Demo</h1>
        <p>This demo showcases how Frango handles various types of form submissions and data processing.</p>

        <div class="grid">
            <!-- GET Form Demo -->
            <div class="card">
                <h2>GET Form Example</h2>
                <p>Demonstrates processing URL parameters via <code>$_GET</code>.</p>
                
                <form action="/forms/get" method="GET">
                    <div style="margin-bottom: 15px;">
                        <label for="name">Name:</label>
                        <input type="text" id="name" name="name" value="John Doe" style="width: 100%; padding: 8px; margin-top: 5px;">
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <label for="category">Category:</label>
                        <select id="category" name="category" style="width: 100%; padding: 8px; margin-top: 5px;">
                            <option value="electronics">Electronics</option>
                            <option value="clothing">Clothing</option>
                            <option value="books">Books</option>
                            <option value="food">Food</option>
                        </select>
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <label>Interests:</label>
                        <div style="margin-top: 5px;">
                            <input type="checkbox" id="int_php" name="interests[]" value="php" checked>
                            <label for="int_php">PHP</label>
                            
                            <input type="checkbox" id="int_go" name="interests[]" value="go" checked>
                            <label for="int_go">Go</label>
                            
                            <input type="checkbox" id="int_js" name="interests[]" value="javascript">
                            <label for="int_js">JavaScript</label>
                        </div>
                    </div>
                    
                    <button type="submit">Submit GET Form</button>
                </form>
            </div>

            <!-- POST Form Demo -->
            <div class="card">
                <h2>POST Form Example</h2>
                <p>Demonstrates processing form data via <code>$_POST</code>.</p>
                
                <form action="/forms/post" method="POST">
                    <div style="margin-bottom: 15px;">
                        <label for="email">Email:</label>
                        <input type="email" id="email" name="email" value="user@example.com" style="width: 100%; padding: 8px; margin-top: 5px;">
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <label for="message">Message:</label>
                        <textarea id="message" name="message" rows="3" style="width: 100%; padding: 8px; margin-top: 5px;">Hello from Frango v2!</textarea>
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <label for="priority">Priority:</label>
                        <div style="margin-top: 5px;">
                            <input type="radio" id="priority_low" name="priority" value="low">
                            <label for="priority_low">Low</label>
                            
                            <input type="radio" id="priority_medium" name="priority" value="medium" checked>
                            <label for="priority_medium">Medium</label>
                            
                            <input type="radio" id="priority_high" name="priority" value="high">
                            <label for="priority_high">High</label>
                        </div>
                    </div>
                    
                    <button type="submit">Submit POST Form</button>
                </form>
            </div>
        </div>
        
        <div class="grid">
            <!-- JSON Data Demo -->
            <div class="card">
                <h2>JSON Request Example</h2>
                <p>Demonstrates processing JSON data sent to the server and accessed via <code>$_JSON</code>.</p>
                
                <div style="margin-bottom: 15px;">
                    <label for="jsonData">JSON Data:</label>
                    <textarea id="jsonData" rows="8" style="width: 100%; padding: 8px; margin-top: 5px; font-family: monospace;">{
  "user": {
    "name": "Jane Smith",
    "email": "jane@example.com"
  },
  "action": "update",
  "items": [1, 2, 3],
  "timestamp": "2023-09-15T10:30:00Z"
}</textarea>
                </div>
                
                <button id="sendJsonBtn">Send JSON Request</button>
                
                <div id="jsonResponse" style="margin-top: 15px; display: none;">
                    <h3>Server Response:</h3>
                    <pre id="jsonResponseText" style="background: #f1f3f5; padding: 10px; border-radius: 4px; overflow-x: auto;"></pre>
                </div>
            </div>

            <!-- File Upload to PHP -->
            <div class="card">
                <h2>File Upload (PHP Handler)</h2>
                <p>Demonstrates file uploads processed by PHP using <code>$_FILES</code>.</p>
                
                <form action="/forms/upload-php" method="POST" enctype="multipart/form-data">
                    <div style="margin-bottom: 15px;">
                        <label for="userfile">Select File:</label>
                        <input type="file" id="userfile" name="userfile" style="width: 100%; padding: 8px; margin-top: 5px;">
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <label for="description">Description:</label>
                        <input type="text" id="description" name="description" value="Uploaded via PHP handler" style="width: 100%; padding: 8px; margin-top: 5px;">
                    </div>
                    
                    <button type="submit">Upload to PHP</button>
                </form>
            </div>
        </div>
        
        <div class="grid">
            <!-- File Upload to Go -->
            <div class="card">
                <h2>File Upload (Go Handler)</h2>
                <p>Demonstrates file uploads processed directly by Go without PHP involvement.</p>
                
                <form action="/forms/upload-go" method="POST" enctype="multipart/form-data">
                    <div style="margin-bottom: 15px;">
                        <label for="userfile2">Select File:</label>
                        <input type="file" id="userfile2" name="userfile" style="width: 100%; padding: 8px; margin-top: 5px;">
                    </div>
                    
                    <div style="margin-bottom: 15px;">
                        <label for="description2">Description:</label>
                        <input type="text" id="description2" name="description" value="Uploaded via Go handler" style="width: 100%; padding: 8px; margin-top: 5px;">
                    </div>
                    
                    <button type="submit">Upload to Go</button>
                </form>
            </div>
            
            <!-- Raw Request Demo (php://input) -->
            <div class="card">
                <h2>Raw Request Body Example</h2>
                <p>Demonstrates handling raw request body via <code>php://input</code> (for PUT, PATCH, etc.).</p>
                
                <div style="margin-bottom: 15px;">
                    <label for="rawBody">Raw Request Body:</label>
                    <textarea id="rawBody" rows="5" style="width: 100%; padding: 8px; margin-top: 5px; font-family: monospace;">This is raw text data being sent in the request body.</textarea>
                </div>
                
                <div style="margin-bottom: 15px;">
                    <label for="method">HTTP Method:</label>
                    <select id="method" style="width: 100%; padding: 8px; margin-top: 5px;">
                        <option value="POST">POST</option>
                        <option value="PUT">PUT</option>
                        <option value="PATCH">PATCH</option>
                        <option value="DELETE">DELETE</option>
                    </select>
                </div>
                
                <button id="sendRawBtn">Send Raw Request</button>
                
                <div id="rawResponse" style="margin-top: 15px; display: none;">
                    <h3>Server Response:</h3>
                    <pre id="rawResponseText" style="background: #f1f3f5; padding: 10px; border-radius: 4px; overflow-x: auto;"></pre>
                </div>
            </div>
        </div>
        
        <div class="card" style="margin-top: 20px;">
            <h2>Navigation</h2>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(__DIR__) . '/debug_panel.php'; ?>
    
    <script>
        // JSON Request Handler
        document.getElementById('sendJsonBtn').addEventListener('click', function() {
            const jsonData = document.getElementById('jsonData').value;
            let parsedData;
            
            try {
                parsedData = JSON.parse(jsonData);
            } catch (e) {
                alert('Invalid JSON: ' + e.message);
                return;
            }
            
            fetch('/forms/json', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: jsonData
            })
            .then(response => response.json())
            .then(data => {
                document.getElementById('jsonResponseText').textContent = JSON.stringify(data, null, 2);
                document.getElementById('jsonResponse').style.display = 'block';
            })
            .catch(error => {
                document.getElementById('jsonResponseText').textContent = 'Error: ' + error.message;
                document.getElementById('jsonResponse').style.display = 'block';
            });
        });
        
        // Raw Request Handler
        document.getElementById('sendRawBtn').addEventListener('click', function() {
            const rawBody = document.getElementById('rawBody').value;
            const method = document.getElementById('method').value;
            
            fetch('/forms/raw', {
                method: method,
                headers: {
                    'Content-Type': 'text/plain'
                },
                body: rawBody
            })
            .then(response => response.text())
            .then(data => {
                document.getElementById('rawResponseText').textContent = data;
                document.getElementById('rawResponse').style.display = 'block';
            })
            .catch(error => {
                document.getElementById('rawResponseText').textContent = 'Error: ' + error.message;
                document.getElementById('rawResponse').style.display = 'block';
            });
        });
    </script>
</body>
</html> 