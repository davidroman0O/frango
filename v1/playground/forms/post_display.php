<?php
/**
 * POST Form Display
 * 
 * Displays the results of a POST form submission
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
    <title>POST Form Results</title>
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
            background: #2ecc71;
            color: white;
        }
        .result-section {
            margin-top: 20px;
            padding-top: 15px;
            border-top: 1px solid #eee;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
        }
        table, th, td {
            border: 1px solid #ddd;
        }
        th, td {
            padding: 10px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">POST</span> Form Results</h1>
        
        <?php if ($_SERVER['REQUEST_METHOD'] !== 'POST' || empty($_POST)): ?>
            <div style="background: #fff4e5; padding: 15px; border-left: 4px solid #f39c12; margin: 20px 0;">
                <strong>No POST data received.</strong> Please submit the form from the forms page.
            </div>
        <?php else: ?>
            <div style="background: #e8f8f5; padding: 15px; border-left: 4px solid #2ecc71; margin: 20px 0;">
                <strong>Success!</strong> POST data received and available in $_POST superglobal.
            </div>
            
            <div class="result-section">
                <h2>POST Parameters</h2>
                <table>
                    <tr>
                        <th>Parameter</th>
                        <th>Value</th>
                    </tr>
                    <?php foreach ($_POST as $key => $value): ?>
                        <tr>
                            <td><strong><?= htmlspecialchars($key) ?></strong></td>
                            <td><?= htmlspecialchars($value) ?></td>
                        </tr>
                    <?php endforeach; ?>
                </table>
            </div>
            
            <div class="result-section">
                <h2>Data Access Examples</h2>
                
                <h3>Standard $_POST Access:</h3>
                <pre><?php
                    $username = isset($_POST['username']) ? $_POST['username'] : 'Not provided';
                    $email = isset($_POST['email']) ? $_POST['email'] : 'Not provided';
                    $comment = isset($_POST['comment']) ? $_POST['comment'] : 'Not provided';
                    
                    echo "\$username = \$_POST['username'] ?? 'Not provided';\n";
                    echo "// Result: $username\n\n";
                    
                    echo "\$email = \$_POST['email'] ?? 'Not provided';\n";
                    echo "// Result: $email\n\n";
                    
                    echo "\$comment = \$_POST['comment'] ?? 'Not provided';\n";
                    echo "// Result: " . substr($comment, 0, 30) . (strlen($comment) > 30 ? '...' : '');
                ?></pre>
            </div>
            
            <div class="result-section">
                <h2>Environment Variables</h2>
                
                <h3>PHP_FORM_* Variables (Raw Source):</h3>
                <pre><?php var_export($debugInfo['raw_form_vars']); ?></pre>
                
                <h3>Debug Info:</h3>
                <pre><?php var_export($debugInfo); ?></pre>
                
                <h3>Raw POST Input:</h3>
                <pre><?php
                    $rawInput = file_get_contents('php://input');
                    echo htmlspecialchars($rawInput ?: '(empty - this is normal for form submissions)');
                ?></pre>
            </div>
            
            <div class="result-section">
                <h2>Request Information</h2>
                <table>
                    <tr>
                        <th>Property</th>
                        <th>Value</th>
                    </tr>
                    <tr>
                        <td>Request Method</td>
                        <td><?= $_SERVER['REQUEST_METHOD'] ?></td>
                    </tr>
                    <tr>
                        <td>Content Type</td>
                        <td><?= $_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'Not available' ?></td>
                    </tr>
                    <tr>
                        <td>Content Length</td>
                        <td><?= $_SERVER['CONTENT_LENGTH'] ?? $_SERVER['HTTP_CONTENT_LENGTH'] ?? 'Not available' ?></td>
                    </tr>
                </table>
            </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms">Back to Forms</a></p>
        <p><a href="/debug.php" target="_blank">View Debug Info</a></p>
    </div>
</body>
</html> 