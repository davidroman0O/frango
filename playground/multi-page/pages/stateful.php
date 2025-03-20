<?php
// Start session for PHP-based state management
session_start();

// State management approaches
// 1. Using URL query parameters (already demonstrated in dynamic.php)
// 2. Using PHP session variables
// 3. Using cookies
// 4. Using hidden form fields
// 5. Reading environment variables set by Go
// 6. API-based state managed by Go

// Process form submission for session-based counter
if (isset($_POST['increment_session'])) {
    $_SESSION['counter'] = ($_SESSION['counter'] ?? 0) + 1;
}

// Process form submission for cookie-based counter
if (isset($_POST['increment_cookie'])) {
    $cookie_counter = isset($_COOKIE['counter']) ? (int)$_COOKIE['counter'] + 1 : 1;
    setcookie('counter', $cookie_counter, time() + 3600, '/');
    // Also set immediately for current request
    $_COOKIE['counter'] = $cookie_counter;
}

// Process form submission for hidden field counter
$hidden_counter = 0;
if (isset($_POST['hidden_counter'])) {
    $hidden_counter = (int)$_POST['hidden_counter'] + 1;
}

// Get environment variables (potentially set by Go)
$go_environment = [];
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'GO_') === 0) {
        $go_environment[$key] = $value;
    }
}

// Get server port from Go
$server_port = $_SERVER['GO_SERVER_PORT'] ?? '8082';

// Function to fetch current API counter from Go
function getApiCounter($port) {
    $url = "http://localhost:{$port}/api/counter";
    $options = [
        'http' => [
            'method' => 'GET',
            'header' => 'Content-type: application/json'
        ]
    ];
    $context = stream_context_create($options);
    $result = @file_get_contents($url, false, $context);
    
    if ($result === false) {
        return ['counter' => 'Error fetching counter', 'time' => ''];
    }
    
    return json_decode($result, true);
}

// Function to increment API counter via Go API
function incrementApiCounter($port, $increment = 1) {
    $url = "http://localhost:{$port}/api/counter" . ($increment > 1 ? "?add={$increment}" : "");
    $options = [
        'http' => [
            'method' => 'POST',
            'header' => 'Content-type: application/json'
        ]
    ];
    $context = stream_context_create($options);
    $result = @file_get_contents($url, false, $context);
    
    if ($result === false) {
        return ['counter' => 'Error incrementing counter', 'time' => ''];
    }
    
    return json_decode($result, true);
}

// Handle API counter increment
$api_counter_result = null;
if (isset($_POST['increment_api'])) {
    $increment = isset($_POST['api_increment']) ? (int)$_POST['api_increment'] : 1;
    $api_counter_result = incrementApiCounter($server_port, $increment);
} else {
    // Just fetch the current value
    $api_counter_result = getApiCounter($server_port);
}

// Get the current timestamp for this page load
$timestamp = date('Y-m-d H:i:s');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Stateful Page - PHP Multi-Page Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }
        h1, h2, h3 {
            color: #333;
        }
        .section {
            background-color: #f9f9f9;
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
            border-left: 4px solid #2196F3;
        }
        .counter {
            font-size: 24px;
            font-weight: bold;
            color: #2196F3;
            padding: 10px;
            background-color: #e9e9e9;
            border-radius: 4px;
            display: inline-block;
            min-width: 50px;
            text-align: center;
        }
        form {
            margin: 15px 0;
        }
        button {
            padding: 8px 15px;
            background-color: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #45a049;
        }
        .nav-links {
            margin-top: 30px;
            display: flex;
            gap: 10px;
        }
        a {
            display: inline-block;
            padding: 10px 15px;
            background-color: #2196F3;
            color: white;
            text-decoration: none;
            border-radius: 4px;
        }
        a:hover {
            background-color: #0b7dda;
        }
        pre {
            background-color: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .api-actions {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-top: 10px;
        }
        input[type="number"] {
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            width: 60px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Stateful PHP Page</h1>
        <p>This page demonstrates different ways to maintain state in a PHP application.</p>
        <p>Page loaded at: <?php echo $timestamp; ?></p>

        <div class="section">
            <h2>7. Go API Counter</h2>
            <p>This counter is maintained by a Go API endpoint and can be read/updated from PHP:</p>
            <div class="counter"><?php echo isset($api_counter_result['counter']) ? $api_counter_result['counter'] : 'N/A'; ?></div>
            <p><small>Last updated: <?php echo isset($api_counter_result['time']) ? $api_counter_result['time'] : 'N/A'; ?></small></p>
            
            <form method="POST">
                <div class="api-actions">
                    <label for="api_increment">Add: </label>
                    <input type="number" id="api_increment" name="api_increment" min="1" max="100" value="1">
                    <button type="submit" name="increment_api">Update API Counter</button>
                </div>
            </form>
            
            <p>
                <strong>Features:</strong>
                <ul>
                    <li>Counter is maintained on the Go server</li>
                    <li>PHP can read the current value via API call</li>
                    <li>PHP can update the value via POST request</li>
                    <li>The server port (<?php echo $server_port; ?>) is injected by Go</li>
                </ul>
            </p>
        </div>
        
        <div class="section">
            <h2>1. Session-based State</h2>
            <p>This counter persists across page loads using PHP session variables:</p>
            <div class="counter"><?php echo $_SESSION['counter'] ?? 0; ?></div>
            <form method="POST">
                <button type="submit" name="increment_session">Increment Session Counter</button>
            </form>
            <p><strong>Note:</strong> Sessions persist until the browser is closed or the session expires.</p>
        </div>
        
        <div class="section">
            <h2>2. Cookie-based State</h2>
            <p>This counter persists across page loads using cookies:</p>
            <div class="counter"><?php echo $_COOKIE['counter'] ?? 0; ?></div>
            <form method="POST">
                <button type="submit" name="increment_cookie">Increment Cookie Counter</button>
            </form>
            <p><strong>Note:</strong> Cookies persist until they expire (set to 1 hour here) or are manually cleared.</p>
        </div>
        
        <div class="section">
            <h2>3. Hidden Form Field State</h2>
            <p>This counter persists only during form submissions:</p>
            <div class="counter"><?php echo $hidden_counter; ?></div>
            <form method="POST">
                <input type="hidden" name="hidden_counter" value="<?php echo $hidden_counter; ?>">
                <button type="submit">Increment Hidden Counter</button>
            </form>
            <p><strong>Note:</strong> Hidden fields only maintain state during form submissions, not across separate page loads.</p>
        </div>
        
        <div class="section">
            <h2>4. URL Query Parameters</h2>
            <p>This demonstrates state through URL parameters (like in dynamic.php):</p>
            <a href="<?php echo '/stateful?counter=' . (($_GET['counter'] ?? 0) + 1); ?>">
                Increment URL Counter (Currently: <?php echo $_GET['counter'] ?? 0; ?>)
            </a>
            <p><strong>Note:</strong> URL parameters are visible in the address bar and are lost when navigating away.</p>
        </div>
        
        <div class="section">
            <h2>5. Go Environment Variables</h2>
            <p>Variables potentially set by the Go application:</p>
            <?php if (empty($go_environment)): ?>
                <p><em>No custom Go environment variables detected. These would have a GO_ prefix.</em></p>
            <?php else: ?>
                <pre><?php print_r($go_environment); ?></pre>
            <?php endif; ?>
            <p><strong>Note:</strong> The Go application can inject custom environment variables that PHP can read.</p>
        </div>
        
        <div class="section">
            <h2>6. Server Information</h2>
            <p>Information about the current server and request:</p>
            <ul>
                <li>Server Software: <?php echo $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown'; ?></li>
                <li>PHP Version: <?php echo phpversion(); ?></li>
                <li>Request Method: <?php echo $_SERVER['REQUEST_METHOD']; ?></li>
                <li>Request URI: <?php echo $_SERVER['REQUEST_URI']; ?></li>
            </ul>
        </div>
        
        <div class="nav-links">
            <a href="/">Home Page</a>
            <a href="/demo">Demo Page</a>
            <a href="/dynamic">Dynamic Page</a>
        </div>
    </div>
</body>
</html> 