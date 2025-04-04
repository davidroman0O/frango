<?php
/**
 * JSON Request Test
 * 
 * Simple test focused solely on JSON request handling
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>JSON Request Test</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        button {
            background: #9b59b6;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
            margin-bottom: 10px;
        }
        button:hover {
            background: #8e44ad;
        }
        pre {
            background: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .result {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #eee;
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
            background: #9b59b6;
            color: white;
        }
        #requestDataContainer {
            margin-top: 15px;
        }
        textarea {
            width: 100%;
            height: 200px;
            font-family: monospace;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">JSON</span> Request Test</h1>
        <p>This test demonstrates how to send and receive JSON data in PHP. The data is sent via a POST request with Content-Type: application/json.</p>
        
        <div id="requestDataContainer">
            <h3>Edit JSON Request Data:</h3>
            <textarea id="jsonData">{
  "user": "johndoe",
  "action": "update",
  "timestamp": "2023-04-04T12:00:00Z",
  "data": {
    "id": 123,
    "status": "active",
    "items": [1, 2, 3],
    "settings": {
      "notifications": true,
      "theme": "dark"
    }
  }
}</textarea>
            <button id="sendJsonBtn">Send JSON Request</button>
        </div>
        
        <div class="code-block">
// JavaScript code to send JSON data
fetch('/forms/form-json', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
});</div>
        
        <div class="code-block">
// PHP code to access JSON data
$user = $_JSON['user'];
$action = $_JSON['action'];
$data = $_JSON['data'];</div>
        
        <div id="jsonResult" class="result" style="display: none;">
            <h2>Server Response</h2>
            <pre id="serverResponse"></pre>
        </div>
        
        <?php if ($_SERVER['REQUEST_METHOD'] === 'POST' && strpos($_SERVER['CONTENT_TYPE'] ?? '', 'application/json') !== false): ?>
        <div class="result">
            <h2>JSON Request Data</h2>
            
            <h3>$_JSON Superglobal:</h3>
            <pre><?php var_export($_JSON ?? []); ?></pre>
            
            <h3>Raw JSON Data:</h3>
            <pre><?php 
                $rawInput = file_get_contents('php://input');
                echo htmlspecialchars($rawInput);
            ?></pre>
            
            <h3>PHP_JSON Variables:</h3>
            <pre><?php 
                $jsonVars = [];
                foreach ($_SERVER as $key => $value) {
                    if (strpos($key, 'PHP_JSON_') === 0) {
                        $jsonVars[$key] = $value;
                    }
                }
                var_export($jsonVars);
            ?></pre>
            
            <?php
            // Send JSON response back
            if (!headers_sent()) {
                header('Content-Type: application/json');
                $response = [
                    'success' => true,
                    'message' => 'JSON data received successfully',
                    'receivedData' => $_JSON ?? json_decode($rawInput, true),
                    'timestamp' => date('c'),
                    'server' => 'Frango PHP'
                ];
                echo json_encode($response, JSON_PRETTY_PRINT);
                exit;
            }
            ?>
        </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms/form-index">Back to Form Tests</a></p>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
    
    <script>
        document.getElementById('sendJsonBtn').addEventListener('click', function() {
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
            
            fetch('/forms/form-json', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(jsonData)
            })
            .then(response => response.json())
            .then(data => {
                document.getElementById('serverResponse').textContent = JSON.stringify(data, null, 2);
                document.getElementById('jsonResult').style.display = 'block';
            })
            .catch(error => {
                document.getElementById('serverResponse').textContent = 'Error: ' + error.message;
                document.getElementById('jsonResult').style.display = 'block';
            });
        });
    </script>
</body>
</html> 