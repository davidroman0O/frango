<?php
/**
 * POST Form Handler
 * 
 * Demonstrates using $_POST with the improved form handling
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

// Process form submission
$requestMethod = $_SERVER['REQUEST_METHOD'];
$contentType = $_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? '';
$rawInput = file_get_contents('php://input');

// Manually parse POST data as a backup
$manuallyParsed = [];
if ($requestMethod === 'POST' && !empty($rawInput)) {
    parse_str($rawInput, $manuallyParsed);
}

// Initialize superglobals if they don't exist
if (!isset($_FORM)) $_FORM = isset($_POST) ? $_POST : [];
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
        a { color: #3498db; }
        .result-item {
            padding: 10px;
            margin: 10px 0;
            border-left: 4px solid #3498db;
            background: #eaf2f8;
        }
        .explanation {
            background: #f9f9f9;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
            border-left: 4px solid #2ecc71;
        }
        .success-banner {
            background: #d4efdf;
            color: #27ae60;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            border: 1px solid #27ae60;
        }
        .warning-banner {
            background: #fef9e7;
            color: #b7950b;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            border: 1px solid #b7950b;
        }
        .debug-info {
            background: #f5eef8;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
            border-left: 4px solid #8e44ad;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>POST Form Results</h1>
        
        <?php if ($requestMethod === 'POST'): ?>
            <?php if (empty($_POST) && empty($_FORM)): ?>
                <div class="warning-banner">
                    <h3>Form submission detected, but no data received in $_POST or $_FORM</h3>
                    <p>This might indicate an issue with form data processing in the Frango middleware.</p>
                </div>
            <?php else: ?>
                <div class="success-banner">
                    <h3>Form submitted successfully!</h3>
                    <p>Your form data has been received using standard <code>$_POST</code> superglobal.</p>
                </div>
            <?php endif; ?>
            
            <h2>Submitted POST Data</h2>
            
            <div class="result-item">
                <h3>$_POST Superglobal:</h3>
                <pre><?php var_export($_POST); ?></pre>
            </div>
            
            <?php if (!empty($_POST)): ?>
            <div class="result-item">
                <h3>Individual POST Parameters:</h3>
                <ul>
                    <?php foreach ($_POST as $key => $value): ?>
                        <li><strong><?= htmlspecialchars($key) ?>:</strong> <?= htmlspecialchars($value) ?></li>
                    <?php endforeach; ?>
                </ul>
            </div>
            <?php endif; ?>
            
            <div class="result-item">
                <h3>$_FORM Superglobal (Alias to $_POST):</h3>
                <pre><?php var_export($_FORM); ?></pre>
            </div>
            
            <div class="result-item">
                <h3>$_REQUEST Superglobal:</h3>
                <pre><?php var_export($_REQUEST); ?></pre>
            </div>
            
            <?php if (!empty($manuallyParsed)): ?>
            <div class="result-item">
                <h3>Manually Parsed Form Data:</h3>
                <pre><?php var_export($manuallyParsed); ?></pre>
            </div>
            <?php endif; ?>
            
            <div class="result-item">
                <h3>Raw POST Input:</h3>
                <pre><?php echo htmlspecialchars($rawInput ?: 'Empty (this is normal for form submissions)'); ?></pre>
            </div>
            
            <div class="debug-info">
                <h3>Request Debug Information:</h3>
                <ul>
                    <li><strong>Request Method:</strong> <?= htmlspecialchars($requestMethod) ?></li>
                    <li><strong>Content-Type:</strong> <?= htmlspecialchars($contentType) ?></li>
                    <li><strong>Raw Input Length:</strong> <?= strlen($rawInput) ?> bytes</li>
                </ul>
                <h4>Request Headers:</h4>
                <pre><?php 
                    $headers = [];
                    foreach ($_SERVER as $key => $value) {
                        if (str_starts_with($key, 'HTTP_') || str_starts_with($key, 'PHP_HEADER_')) {
                            $headers[$key] = $value;
                        }
                    }
                    var_export($headers);
                ?></pre>
            </div>
            
            <div class="explanation">
                <p>Notice that with Frango v1's improved form handling:</p>
                <ul>
                    <li><code>$_POST</code> automatically contains all form data submitted via POST</li>
                    <li><code>$_FORM</code> is an alias to <code>$_POST</code> for convenience</li>
                    <li><code>$_REQUEST</code> contains all GET, POST, and COOKIE data, just like standard PHP</li>
                    <li>Even though <code>php://input</code> might be empty in some cases, we still have access to POST data</li>
                </ul>
            </div>
        <?php else: ?>
            <div class="result-item">
                <p>Please submit the POST form from the forms index page.</p>
                <p><a href="/forms/" class="button">Go to Forms</a></p>
            </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms/">Back to Forms</a></p>
        <p><a href="/">Back to Home</a></p>
        <p><a href="/forms/json">Try JSON example</a></p>
    </div>
    
    <?php include_once(__DIR__ . '/../debug_panel.php'); ?>
</body>
</html> 