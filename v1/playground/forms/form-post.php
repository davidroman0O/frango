<?php
/**
 * POST Form Test
 * 
 * Simple test focused solely on POST request handling
 */

// Include the globals fix at the top of the file
require_once __DIR__ . '/globals_fix.php';

// Debug variables
$debugInfo = [];
$debugInfo['raw_form_vars'] = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $debugInfo['raw_form_vars'][$key] = $value;
    }
}
$debugInfo['post_count'] = count($_POST);
$debugInfo['form_count'] = isset($_FORM) ? count($_FORM) : 0;
?>
<!DOCTYPE html>
<html>
<head>
    <title>POST Form Test</title>
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
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input, textarea {
            width: 100%;
            padding: 8px;
            margin-bottom: 15px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        button {
            background: #2ecc71;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #27ae60;
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
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
            background: #2ecc71;
            color: white;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">POST</span> Form Test</h1>
        <p>This form submits data using the POST method. The data will be sent in the request body, not visible in the URL.</p>
        
        <form action="/forms/form-post" method="POST">
            <label for="username">Username:</label>
            <input type="text" id="username" name="username" value="test_user">
            
            <label for="email">Email:</label>
            <input type="email" id="email" name="email" value="test@example.com">
            
            <label for="message">Message:</label>
            <textarea id="message" name="message" rows="4">This is a test message to verify POST form handling</textarea>
            
            <button type="submit">Submit POST Form</button>
        </form>
        
        <?php if ($_SERVER['REQUEST_METHOD'] === 'POST'): ?>
        <div class="result">
            <h2>POST Results</h2>
            
            <h3>$_POST Superglobal:</h3>
            <pre><?php var_export($_POST); ?></pre>
            
            <h3>$_FORM Superglobal (Frango alias):</h3>
            <pre><?php var_export($_FORM ?? []); ?></pre>
            
            <h3>$_REQUEST Superglobal:</h3>
            <pre><?php var_export($_REQUEST); ?></pre>
            
            <h3>PHP_FORM_* Variables:</h3>
            <pre><?php var_export($debugInfo['raw_form_vars']); ?></pre>
            
            <h3>Debug Info:</h3>
            <pre><?php var_export($debugInfo); ?></pre>
            
            <h3>Raw POST Input:</h3>
            <pre><?php
                $rawInput = file_get_contents('php://input');
                echo htmlspecialchars($rawInput ?: 'Empty (normal for form submissions)');
            ?></pre>
        </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms/form-index">Back to Form Tests</a></p>
        <p><a href="/forms">Back to Forms</a></p>
        <p><a href="/debug.php" target="_blank">View Debug Info</a></p>
        <p><a href="/forms/form_debug.php" target="_blank">Form Debug Tool</a></p>
    </div>

    <?php include_once(__DIR__ . '/../debug_panel.php'); ?>
</body>
</html> 