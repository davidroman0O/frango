<?php
/**
 * Frango v2 Demo - Superglobals Test
 * 
 * This page demonstrates that all standard PHP superglobals
 * are properly initialized and populated.
 */

// Initialize any test values that might not be present
if (empty($_POST['test_post'])) {
    $_POST['test_post'] = 'This is populated dynamically to show $_POST structure';
}

// Test arrays in superglobals
$arrayKey = 'test_array';
if (empty($_GET[$arrayKey])) {
    $_GET[$arrayKey] = ['item1', 'item2', 'item3'];
}

// Insert a test cookie if none exist
if (empty($_COOKIE)) {
    setcookie('test_cookie', 'cookie_value', time() + 3600);
    $_COOKIE['test_cookie'] = 'cookie_value'; // Also set it for current request
}

// Ensure $_REQUEST contains merged values
$_REQUEST = array_merge($_COOKIE, $_GET, $_POST);

?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Frango v2 - Superglobals Test</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>Superglobals Compliance Test</h1>
        <p>This page verifies that all PHP superglobals are properly initialized and populated according to the <a href="https://github.com/php/php-src/blob/master/README.GLOBALS" target="_blank">PHP specification</a>.</p>

        <!-- Test form for GET and POST -->
        <div class="card">
            <h2>Test Forms</h2>
            <h3>GET Form</h3>
            <form action="/checklist/superglobals" method="GET">
                <div style="margin-bottom: 15px;">
                    <label>Simple Parameter:</label>
                    <input type="text" name="name" value="John Doe" placeholder="Your name">
                </div>
                
                <div style="margin-bottom: 15px;">
                    <label>Array Parameter:</label>
                    <div>
                        <input type="checkbox" name="interests[]" value="php" checked> PHP
                        <input type="checkbox" name="interests[]" value="go" checked> Go
                        <input type="checkbox" name="interests[]" value="js"> JavaScript
                    </div>
                </div>
                
                <div style="margin-bottom: 15px;">
                    <label>Nested Array:</label>
                    <div>
                        <input type="text" name="user[name]" value="Jane" placeholder="User name">
                        <input type="text" name="user[email]" value="jane@example.com" placeholder="User email">
                    </div>
                </div>
                
                <button type="submit">Submit GET Form</button>
            </form>
            
            <h3>POST Form</h3>
            <form action="/checklist/superglobals" method="POST">
                <div style="margin-bottom: 15px;">
                    <label>Simple Parameter:</label>
                    <input type="text" name="email" value="email@example.com" placeholder="Your email">
                </div>
                
                <div style="margin-bottom: 15px;">
                    <label>Array Parameter:</label>
                    <div>
                        <input type="checkbox" name="subscriptions[]" value="newsletter" checked> Newsletter
                        <input type="checkbox" name="subscriptions[]" value="updates" checked> Updates
                    </div>
                </div>
                
                <div style="margin-bottom: 15px;">
                    <label>Nested Array:</label>
                    <div>
                        <input type="text" name="address[street]" value="123 Main St" placeholder="Street">
                        <input type="text" name="address[city]" value="Springfield" placeholder="City">
                    </div>
                </div>
                
                <input type="hidden" name="test_post" value="This was submitted via POST form">
                <button type="submit">Submit POST Form</button>
            </form>
        </div>

        <!-- Superglobal Values Display -->
        <div class="card">
            <h2>$_GET</h2>
            <p>Query string and path parameters.</p>
            <pre><?php var_export($_GET); ?></pre>
            
            <h3>Testing $_GET Arrays</h3>
            <?php if (!empty($_GET['interests'])): ?>
            <ul>
                <?php foreach ($_GET['interests'] as $interest): ?>
                <li><?php echo htmlspecialchars($interest); ?></li>
                <?php endforeach; ?>
            </ul>
            <?php endif; ?>
            
            <?php if (!empty($_GET['user'])): ?>
            <p>Nested array access: <?php echo htmlspecialchars($_GET['user']['name'] ?? '(not set)'); ?></p>
            <?php endif; ?>
        </div>
        
        <div class="card">
            <h2>$_POST</h2>
            <p>Form data submitted via POST.</p>
            <pre><?php var_export($_POST); ?></pre>
            
            <?php if (!empty($_POST['subscriptions'])): ?>
            <ul>
                <?php foreach ($_POST['subscriptions'] as $subscription): ?>
                <li><?php echo htmlspecialchars($subscription); ?></li>
                <?php endforeach; ?>
            </ul>
            <?php endif; ?>
        </div>
        
        <div class="card">
            <h2>$_COOKIE</h2>
            <p>HTTP cookies sent by the client. A test cookie was added if none existed.</p>
            <pre><?php var_export($_COOKIE); ?></pre>
        </div>
        
        <div class="card">
            <h2>$_REQUEST</h2>
            <p>Merged values from $_GET, $_POST, and $_COOKIE.</p>
            <pre><?php var_export($_REQUEST); ?></pre>
        </div>
        
        <div class="card">
            <h2>$_FILES</h2>
            <p>Uploaded files information. Use the form below to test.</p>
            <pre><?php var_export($_FILES); ?></pre>
            
            <h3>Upload Test</h3>
            <form action="/checklist/superglobals" method="POST" enctype="multipart/form-data">
                <div style="margin-bottom: 15px;">
                    <label>Upload a file:</label>
                    <input type="file" name="testfile">
                </div>
                <button type="submit">Upload</button>
            </form>
        </div>
        
        <div class="card">
            <h2>$_SERVER (Key Variables)</h2>
            <p>Here are the most important $_SERVER variables:</p>
            <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
                <tr style="background: #f8f9fa;">
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Variable</th>
                    <th style="text-align: left; padding: 8px; border: 1px solid #dee2e6;">Value</th>
                </tr>
                <?php
                $important_vars = [
                    'REQUEST_METHOD',
                    'REQUEST_URI',
                    'QUERY_STRING',
                    'SCRIPT_NAME',
                    'SCRIPT_FILENAME',
                    'DOCUMENT_ROOT',
                    'SERVER_PROTOCOL',
                    'REMOTE_ADDR',
                    'HTTP_USER_AGENT',
                    'HTTP_ACCEPT',
                    'HTTP_HOST'
                ];
                
                foreach ($important_vars as $var) {
                    echo '<tr>';
                    echo '<td style="padding: 8px; border: 1px solid #dee2e6;"><code>' . $var . '</code></td>';
                    echo '<td style="padding: 8px; border: 1px solid #dee2e6;">' . 
                         (isset($_SERVER[$var]) ? htmlspecialchars($_SERVER[$var]) : '<em>Not set</em>') . '</td>';
                    echo '</tr>';
                }
                ?>
            </table>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(__DIR__) . '/debug_panel.php'; ?>
</body>
</html> 