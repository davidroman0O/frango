<?php
// Start session to maintain state
session_start();

// Get session information if available
$loggedIn = isset($_SESSION['user']);
$username = $loggedIn ? $_SESSION['user']['username'] : '';

header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango - Session Management Example</title>
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
        .nav {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 1px solid #eee;
        }
        .user-info {
            display: flex;
            align-items: center;
        }
        .avatar {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            background: #0366d6;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 10px;
            font-weight: bold;
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
        code {
            background: #f6f8fa;
            padding: 2px 4px;
            border-radius: 3px;
            font-family: monospace;
        }
        pre {
            background: #f6f8fa;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
    </style>
</head>
<body>
    <div class="nav">
        <div>
            <h1 style="margin: 0;">Frango Sessions</h1>
        </div>
        
        <div class="user-info">
            <?php if ($loggedIn): ?>
                <div class="avatar"><?= strtoupper(substr($username, 0, 1)) ?></div>
                <div>
                    <div><?= htmlspecialchars($username) ?></div>
                    <div><a href="/logout">Logout</a></div>
                </div>
            <?php else: ?>
                <a href="/login" class="button">Login</a>
            <?php endif; ?>
        </div>
    </div>
    
    <div class="card">
        <h2>Session Management Example</h2>
        <p>This example demonstrates how to use PHP sessions with Frango, including:</p>
        <ul>
            <li>Starting and accessing sessions</li>
            <li>Storing and retrieving session data</li>
            <li>User authentication with sessions</li>
            <li>Session state persistence across requests</li>
            <li>Protected routes requiring authentication</li>
            <li>Session destruction (logout)</li>
        </ul>
        
        <div>
            <a href="/login" class="button">Login Example</a>
            <a href="/counter" class="button">Session Counter</a>
            <?php if ($loggedIn): ?>
                <a href="/dashboard" class="button">User Dashboard</a>
            <?php endif; ?>
        </div>
    </div>
    
    <div class="card">
        <h2>How Sessions Work in PHP</h2>
        <p>Sessions in PHP provide a way to preserve data across page requests:</p>
        <ol>
            <li>When a session starts, PHP generates a unique session ID</li>
            <li>This ID is stored in a cookie on the client's browser</li>
            <li>On the server, data is stored in files or a database, associated with this ID</li>
            <li>The <code>$_SESSION</code> superglobal provides access to session data</li>
        </ol>
        
        <p>Basic session usage:</p>
        <pre><code>// Start or resume a session
session_start();

// Store data in the session
$_SESSION['username'] = 'user123';

// Retrieve data from the session
$username = $_SESSION['username'];

// Remove data from the session
unset($_SESSION['username']);

// Destroy the entire session
session_destroy();</code></pre>
    </div>
    
    <?php if ($loggedIn): ?>
    <div class="card">
        <h2>Your Current Session</h2>
        <p>You are currently logged in as: <strong><?= htmlspecialchars($username) ?></strong></p>
        <p>Session ID: <code><?= session_id() ?></code></p>
        <p>Session Data:</p>
        <pre><?php var_export($_SESSION); ?></pre>
    </div>
    <?php endif; ?>
</body>
</html> 