<?php
// Start session
session_start();

// Check if user is logged in
if (!isset($_SESSION['user'])) {
    // Not logged in, redirect to login page
    header('Location: /login');
    exit;
}

// Get user data from session
$user = $_SESSION['user'];

// Check if user has admin role
if ($user['role'] !== 'admin') {
    // Not an admin, show access denied
    header('Content-Type: text/html');
    ?>
    <!DOCTYPE html>
    <html>
    <head>
        <title>Access Denied - Frango Sessions Example</title>
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
            .error-message {
                background-color: #f8d7da;
                color: #721c24;
                padding: 15px;
                border-radius: 4px;
                margin-bottom: 20px;
                border-left: 4px solid #dc3545;
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
        </style>
    </head>
    <body>
        <div class="error-message">
            <h1>Access Denied</h1>
            <p>You do not have permission to access the Admin Panel.</p>
            <p>Your role is: <strong><?= htmlspecialchars($user['role']) ?></strong></p>
            <p>This page requires the <strong>admin</strong> role.</p>
        </div>
        
        <div class="card">
            <h2>What Happened?</h2>
            <p>This page demonstrates role-based access control using PHP sessions:</p>
            <ol>
                <li>You are logged in as <strong><?= htmlspecialchars($user['username']) ?></strong></li>
                <li>Your account has the <strong><?= htmlspecialchars($user['role']) ?></strong> role</li>
                <li>This page checks your role and denies access to non-admin users</li>
            </ol>
            
            <div>
                <a href="/dashboard" class="button">Return to Dashboard</a>
                <a href="/" class="button">Return to Home</a>
            </div>
        </div>
    </body>
    </html>
    <?php
    exit;
}

// User is an admin, show admin panel
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Admin Panel - Frango Sessions Example</title>
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
            width: 36px;
            height: 36px;
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
        .admin-badge {
            display: inline-block;
            background: #0366d6;
            color: white;
            padding: 3px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: bold;
            text-transform: uppercase;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 8px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f5f5f5;
        }
    </style>
</head>
<body>
    <div class="nav">
        <div>
            <h1 style="margin: 0;">Admin Panel</h1>
        </div>
        
        <div class="user-info">
            <div class="avatar"><?= strtoupper(substr($user['username'], 0, 1)) ?></div>
            <div>
                <div>
                    <?= htmlspecialchars($user['username']) ?>
                    <span class="admin-badge">Admin</span>
                </div>
                <div><a href="/logout">Logout</a></div>
            </div>
        </div>
    </div>
    
    <div class="card">
        <h2>Welcome to the Admin Panel</h2>
        <p>This page is only accessible to users with the <strong>admin</strong> role.</p>
        <p>Your session contains the following user information:</p>
        
        <table>
            <tr>
                <th>Username</th>
                <td><?= htmlspecialchars($user['username']) ?></td>
            </tr>
            <tr>
                <th>Full Name</th>
                <td><?= htmlspecialchars($user['full_name']) ?></td>
            </tr>
            <tr>
                <th>Email</th>
                <td><?= htmlspecialchars($user['email']) ?></td>
            </tr>
            <tr>
                <th>Role</th>
                <td><?= htmlspecialchars($user['role']) ?></td>
            </tr>
            <tr>
                <th>Login Time</th>
                <td><?= date('Y-m-d H:i:s', $user['login_time']) ?></td>
            </tr>
            <tr>
                <th>Last Activity</th>
                <td><?= date('Y-m-d H:i:s', $user['last_activity']) ?></td>
            </tr>
        </table>
    </div>
    
    <div class="card">
        <h2>Role-Based Access Control</h2>
        <p>This example demonstrates a simple implementation of role-based access control:</p>
        <pre style="background: #f6f8fa; padding: 10px; border-radius: 4px; overflow: auto;">// Check if user is logged in
if (!isset($_SESSION['user'])) {
    // Not logged in, redirect to login page
    header('Location: /login');
    exit;
}

// Get user data from session
$user = $_SESSION['user'];

// Check if user has admin role
if ($user['role'] !== 'admin') {
    // Not an admin, show access denied
    // ...
    exit;
}</pre>
        
        <p>This pattern can be used to protect any route that requires specific permissions or roles.</p>
        
        <div>
            <a href="/dashboard" class="button">Return to Dashboard</a>
            <a href="/" class="button">Return to Home</a>
        </div>
    </div>
</body>
</html> 