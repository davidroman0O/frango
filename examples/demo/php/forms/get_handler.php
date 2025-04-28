<?php
/**
 * Frango v2 Demo - GET Form Handler
 * 
 * Demonstrates processing URL parameters received via $_GET.
 */
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GET Form Results - Frango v2</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>GET Form Results</h1>
        <p>This page demonstrates how Frango processes URL parameters via the <code>$_GET</code> superglobal.</p>
        
        <?php if (empty($_GET)): ?>
            <div class="card">
                <div style="background: #fff3cd; padding: 15px; border-left: 4px solid #ffc107; margin-bottom: 20px;">
                    <strong>No parameters received.</strong> Please submit the form on the <a href="/forms">Forms Demo</a> page.
                </div>
            </div>
        <?php else: ?>
            <div class="card">
                <h2>Received Parameters</h2>
                <p>The following parameters were received via the GET request:</p>
                
                <h3>Raw $_GET Superglobal</h3>
                <pre><?php var_export($_GET); ?></pre>
                
                <h3>Parameter Details</h3>
                <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                    <tr style="background: #f8f9fa;">
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Parameter</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                        <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Type</th>
                    </tr>
                    <?php foreach ($_GET as $key => $value): ?>
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
                
                <h3>Demonstrating Array Access</h3>
                <?php if (isset($_GET['interests']) && is_array($_GET['interests'])): ?>
                    <p>You selected these interests:</p>
                    <ul>
                        <?php foreach ($_GET['interests'] as $interest): ?>
                            <li><strong><?= htmlspecialchars($interest) ?></strong></li>
                        <?php endforeach; ?>
                    </ul>
                <?php endif; ?>
                
                <?php if (isset($_GET['user']) && is_array($_GET['user'])): ?>
                    <p>User information:</p>
                    <ul>
                        <?php foreach ($_GET['user'] as $key => $value): ?>
                            <li><strong><?= htmlspecialchars($key) ?>:</strong> <?= htmlspecialchars($value) ?></li>
                        <?php endforeach; ?>
                    </ul>
                <?php endif; ?>
            </div>
        <?php endif; ?>
        
        <div class="card">
            <h2>Technical Details</h2>
            <h3>URL Components</h3>
            <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                <tr style="background: #f8f9fa;">
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Component</th>
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>REQUEST_URI</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['REQUEST_URI']) ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>QUERY_STRING</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['QUERY_STRING'] ?? '') ?></td>
                </tr>
            </table>
            
            <h3>Code Example</h3>
            <p>This is how you access GET parameters in PHP:</p>
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// Access simple parameter
$name = $_GET['name'] ?? 'Default Name';
echo $name;

// Access array parameter
if (isset($_GET['interests']) && is_array($_GET['interests'])) {
    foreach ($_GET['interests'] as $interest) {
        echo "Interest: $interest\n";
    }
}

// Access nested array
if (isset($_GET['user']) && is_array($_GET['user'])) {
    $userName = $_GET['user']['name'] ?? 'Unknown';
    $userEmail = $_GET['user']['email'] ?? 'No email';
    echo "User: $userName ($userEmail)";
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