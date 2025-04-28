<?php
// Start session
session_start();

// Check if user is logged in
$wasLoggedIn = isset($_SESSION['user']);
$username = $wasLoggedIn ? $_SESSION['user']['username'] : '';

// Unset all session variables
$_SESSION = array();

// If a session cookie is used, destroy it
if (ini_get("session.use_cookies")) {
    $params = session_get_cookie_params();
    setcookie(session_name(), '', time() - 42000,
        $params["path"], $params["domain"],
        $params["secure"], $params["httponly"]
    );
}

// Destroy the session
session_destroy();

// Set content type
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Logged Out - Frango Sessions Example</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .card {
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 15px;
            margin-bottom: 20px;
        }
        .success-message {
            background-color: #d4edda;
            color: #155724;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
            border-left: 4px solid #28a745;
        }
        h1, h2 {
            color: #333;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .button {
            display: inline-block;
            background: #0366d6;
            color: white;
            padding: 8px 16px;
            border-radius: 4px;
            text-decoration: none;
            margin-top: 10px;
            margin-right: 10px;
        }
        .button:hover {
            background: #0250be;
            text-decoration: none;
        }
        .timer {
            font-weight: bold;
            color: #0366d6;
        }
        .code-block {
            background: #f6f8fa;
            padding: 10px;
            border-radius: 4px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <h1>Logged Out</h1>
    
    <div class="success-message">
        <?php if ($wasLoggedIn): ?>
            <h2>Goodbye, <?= htmlspecialchars($username) ?>!</h2>
            <p>You have been successfully logged out.</p>
        <?php else: ?>
            <h2>Not Logged In</h2>
            <p>You were not logged in to begin with.</p>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h2>Session Status</h2>
        <p>Your session has been destroyed. All session data has been cleared.</p>
        <p>You will be redirected to the home page in <span id="countdown" class="timer">5</span> seconds...</p>
        
        <div>
            <a href="/" class="button">Return to Home</a>
            <a href="/login" class="button">Log in Again</a>
        </div>
    </div>
    
    <div class="card">
        <h2>How Session Destruction Works</h2>
        <p>When logging out, PHP performs these steps to completely destroy the session:</p>
        <ol>
            <li>Clear all session variables with <code>$_SESSION = array();</code></li>
            <li>Delete the session cookie from the user's browser</li>
            <li>Call <code>session_destroy()</code> to remove the session data from the server</li>
        </ol>
        
        <div class="code-block">
<pre>// Start session
session_start();

// Unset all session variables
$_SESSION = array();

// If a session cookie is used, destroy it
if (ini_get("session.use_cookies")) {
    $params = session_get_cookie_params();
    setcookie(session_name(), '', time() - 42000,
        $params["path"], $params["domain"],
        $params["secure"], $params["httponly"]
    );
}

// Destroy the session
session_destroy();</pre>
        </div>
    </div>
    
    <script>
        // Countdown and redirect
        let countdown = 5;
        const countdownElement = document.getElementById('countdown');
        
        const timer = setInterval(function() {
            countdown--;
            countdownElement.textContent = countdown;
            
            if (countdown <= 0) {
                clearInterval(timer);
                window.location.href = '/';
            }
        }, 1000);
    </script>
</body>
</html> 