<?php
// Start session
session_start();

// Check if user is already logged in
if (isset($_SESSION['user'])) {
    // Already logged in, redirect to dashboard
    header('Location: /dashboard');
    exit;
}

// Check for login error message
$error = $_GET['error'] ?? '';
$errorMessage = '';

if ($error === 'invalid') {
    $errorMessage = 'Invalid username or password';
} elseif ($error === 'empty') {
    $errorMessage = 'Please enter both username and password';
}

header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Login - Frango Sessions Example</title>
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
        h1, h2, h3 {
            color: #333;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: bold;
        }
        input {
            width: 100%;
            padding: 8px;
            margin-bottom: 16px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        button {
            background: #0366d6;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 1em;
        }
        button:hover {
            background: #0250be;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .error-message {
            color: #e53e3e;
            padding: 10px;
            background-color: #fff5f5;
            border-radius: 4px;
            margin-bottom: 16px;
        }
        .login-note {
            background: #f5f9ff;
            padding: 12px;
            border-radius: 4px;
            border-left: 4px solid #0366d6;
            margin-top: 20px;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <h1>Login</h1>
    
    <div class="card">
        <?php if (!empty($errorMessage)): ?>
            <div class="error-message">
                <?= htmlspecialchars($errorMessage) ?>
            </div>
        <?php endif; ?>
        
        <form action="/process-login" method="POST">
            <div>
                <label for="username">Username</label>
                <input type="text" id="username" name="username" placeholder="Enter your username" autofocus>
            </div>
            
            <div>
                <label for="password">Password</label>
                <input type="password" id="password" name="password" placeholder="Enter your password">
            </div>
            
            <button type="submit">Login</button>
        </form>
        
        <div class="login-note">
            <h3>Demo Credentials</h3>
            <p>For this example, you can use these credentials:</p>
            <ul>
                <li><strong>Username:</strong> admin</li>
                <li><strong>Password:</strong> password</li>
            </ul>
            <p>Or</p>
            <ul>
                <li><strong>Username:</strong> user</li>
                <li><strong>Password:</strong> password</li>
            </ul>
        </div>
        
        <a href="/" class="back-link">Back to Home</a>
    </div>
</body>
</html> 