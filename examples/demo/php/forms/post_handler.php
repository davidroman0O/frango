<?php
/**
 * Frango v2 Demo - POST Form Handler
 * 
 * Demonstrates processing form data submitted via POST method.
 */
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>POST Form Results - Frango v2</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>POST Form Results</h1>
        <p>This page demonstrates how Frango processes form data via the <code>$_POST</code> superglobal.</p>
        
        <?php if ($_SERVER['REQUEST_METHOD'] !== 'POST' || empty($_POST)): ?>
            <div class="card">
                <div style="background: #f8d7da; padding: 15px; border-left: 4px solid #dc3545; margin-bottom: 20px;">
                    <strong>No POST data received.</strong> Please submit the form on the <a href="/forms">Forms Demo</a> page.
                </div>
            </div>
        <?php else: ?>
            <div class="card">
                <div style="background: #d4edda; padding: 15px; border-left: 4px solid #28a745; margin-bottom: 20px;">
                    <strong>Success!</strong> POST data received and processed.
                </div>
                
                <h2>Received Form Data</h2>
                <p>The following data was submitted via POST:</p>
                
                <h3>Raw $_POST Superglobal</h3>
                <pre><?php var_export($_POST); ?></pre>
                
                <h3>Form Field Details</h3>
                <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                    <tr style="background: #f8f9fa;">
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Field Name</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Type</th>
                    </tr>
                    <?php foreach ($_POST as $key => $value): ?>
                        <tr>
                            <td style="padding: 8px; border: 1px solid #dee2e6;"><code><?= htmlspecialchars($key) ?></code></td>
                            <td style="padding: 8px; border: 1px solid #dee2e6;">
                                <?php if (is_array($value)): ?>
                                    <ul style="margin: 0; padding-left: 20px;">
                                        <?php foreach ($value as $item): ?>
                                            <li><?= htmlspecialchars($item) ?></li>
                                        <?php endforeach; ?>
                                    </ul>
                                <?php elseif (is_object($value)): ?>
                                    <?= htmlspecialchars(json_encode($value)) ?>
                                <?php else: ?>
                                    <?= htmlspecialchars($value) ?>
                                <?php endif; ?>
                            </td>
                            <td style="padding: 8px; border: 1px solid #dee2e6;">
                                <?php 
                                    if (is_array($value)) {
                                        echo 'Array (' . count($value) . ' items)';
                                    } elseif (is_object($value)) {
                                        echo 'Object (' . get_class($value) . ')';
                                    } else {
                                        echo gettype($value);
                                    }
                                ?>
                            </td>
                        </tr>
                    <?php endforeach; ?>
                </table>
                
                <h3>Processed Form Data</h3>
                <div style="background: #f8f9fa; padding: 15px; border-radius: 4px; margin-bottom: 20px;">
                    <?php if (isset($_POST['email'])): ?>
                        <p><strong>Email:</strong> <?= htmlspecialchars($_POST['email']) ?></p>
                    <?php endif; ?>
                    
                    <?php if (isset($_POST['message'])): ?>
                        <p><strong>Message:</strong></p>
                        <div style="background: white; padding: 10px; border: 1px solid #dee2e6; border-radius: 4px; margin-bottom: 10px;">
                            <?= nl2br(htmlspecialchars($_POST['message'])) ?>
                        </div>
                    <?php endif; ?>
                    
                    <?php if (isset($_POST['priority'])): ?>
                        <p><strong>Priority:</strong> 
                            <span style="display: inline-block; padding: 3px 8px; border-radius: 4px; background: 
                                <?php 
                                    switch($_POST['priority']) {
                                        case 'high': echo '#dc3545'; break;
                                        case 'medium': echo '#ffc107'; break;
                                        case 'low': echo '#28a745'; break;
                                        default: echo '#6c757d';
                                    }
                                ?>;
                                color: white;">
                                <?= htmlspecialchars(ucfirst($_POST['priority'])) ?>
                            </span>
                        </p>
                    <?php endif; ?>
                </div>
            </div>
            
            <div class="card">
                <h2>Content-Type & $_REQUEST</h2>
                <p>The <code>Content-Type</code> for form submissions is <code>application/x-www-form-urlencoded</code> (or <code>multipart/form-data</code> for file uploads).</p>
                <p>Form data is also accessible via the <code>$_REQUEST</code> superglobal, which contains values from <code>$_GET</code>, <code>$_POST</code>, and <code>$_COOKIE</code>:</p>
                <pre><?php var_export($_REQUEST); ?></pre>
            </div>
        <?php endif; ?>
        
        <div class="card">
            <h2>Technical Details</h2>
            <h3>Request Information</h3>
            <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                <tr style="background: #f8f9fa;">
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Property</th>
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>REQUEST_METHOD</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['REQUEST_METHOD']) ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>CONTENT_TYPE</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'Not available') ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>CONTENT_LENGTH</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['CONTENT_LENGTH'] ?? $_SERVER['HTTP_CONTENT_LENGTH'] ?? 'Not available') ?></td>
                </tr>
            </table>
            
            <h3>Code Example</h3>
            <p>This is how you access POST data in PHP:</p>
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// Check if form was submitted via POST
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    // Access simple parameter with fallback
    $email = $_POST['email'] ?? 'Not provided';
    
    // Validate an email address
    if (filter_var($email, FILTER_VALIDATE_EMAIL)) {
        echo "Valid email: $email\n";
    }
    
    // Access textarea content
    $message = $_POST['message'] ?? '';
    $messageLength = strlen($message);
    
    // Access radio button selection
    $priority = $_POST['priority'] ?? 'normal';
    
    // Process based on form data
    if ($priority === 'high') {
        // Handle high priority submission
    }
}</pre>
        </div>
        
        <div class="card">
            <h2>Navigation</h2>
            <p><a href="/forms">Back to Forms Demo</a></p>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(__DIR__) . '/debug_panel.php'; ?>
</body>
</html> 