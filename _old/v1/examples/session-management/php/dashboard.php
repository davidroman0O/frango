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
$visitCount = $_SESSION['visit_count'] ?? 1;
$loginHistory = $_SESSION['login_history'] ?? [];

// Update last activity time
$_SESSION['user']['last_activity'] = time();

header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard - Frango Sessions Example</title>
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
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .stat-card {
            padding: 15px;
            border-radius: 4px;
            background: #f6f8fa;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .stat-label {
            color: #666;
            font-size: 14px;
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
        .role-badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: bold;
            text-transform: uppercase;
        }
        .role-admin {
            background: #0366d6;
            color: white;
        }
        .role-user {
            background: #6c757d;
            color: white;
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
        .session-info {
            background: #f5f9ff;
            padding: 12px;
            border-radius: 4px;
            border-left: 4px solid #0366d6;
            margin-bottom: 16px;
        }
    </style>
</head>
<body>
    <div class="nav">
        <div>
            <h1 style="margin: 0;">User Dashboard</h1>
        </div>
        
        <div class="user-info">
            <div class="avatar"><?= strtoupper(substr($user['username'], 0, 1)) ?></div>
            <div>
                <div><?= htmlspecialchars($user['username']) ?></div>
                <div><a href="/logout">Logout</a></div>
            </div>
        </div>
    </div>
    
    <div class="card">
        <h2>Welcome, <?= htmlspecialchars($user['full_name']) ?></h2>
        <p>You are logged in as <span class="role-badge role-<?= $user['role'] ?>"><?= htmlspecialchars($user['role']) ?></span></p>
        
        <div class="session-info">
            <p><strong>Session ID:</strong> <?= session_id() ?></p>
            <p><strong>Login Time:</strong> <?= date('Y-m-d H:i:s', $user['login_time']) ?></p>
            <p><strong>Last Activity:</strong> <?= date('Y-m-d H:i:s', $user['last_activity']) ?></p>
        </div>
        
        <div class="grid">
            <div class="stat-card">
                <div class="stat-value"><?= $visitCount ?></div>
                <div class="stat-label">Visit Count</div>
            </div>
            
            <div class="stat-card">
                <div class="stat-value"><?= count($loginHistory) ?></div>
                <div class="stat-label">Login Count</div>
            </div>
            
            <div class="stat-card">
                <div class="stat-value"><?= session_status() === PHP_SESSION_ACTIVE ? 'Active' : 'Inactive' ?></div>
                <div class="stat-label">Session Status</div>
            </div>
        </div>
        
        <?php if ($user['role'] === 'admin'): ?>
            <p>As an administrator, you have access to the following administrative features:</p>
            <div>
                <a href="/admin" class="button">Admin Panel</a>
                <a href="/settings" class="button">System Settings</a>
            </div>
        <?php else: ?>
            <p>As a regular user, you have access to the following features:</p>
            <div>
                <a href="/profile" class="button">Your Profile</a>
                <a href="/settings" class="button">Account Settings</a>
            </div>
        <?php endif; ?>
    </div>
    
    <?php if (!empty($loginHistory)): ?>
    <div class="card">
        <h2>Login History</h2>
        <table>
            <thead>
                <tr>
                    <th>#</th>
                    <th>Time</th>
                    <th>IP Address</th>
                </tr>
            </thead>
            <tbody>
                <?php foreach (array_reverse($loginHistory) as $index => $login): ?>
                    <tr>
                        <td><?= count($loginHistory) - $index ?></td>
                        <td><?= date('Y-m-d H:i:s', $login['time']) ?></td>
                        <td><?= htmlspecialchars($login['ip']) ?></td>
                    </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    </div>
    <?php endif; ?>
    
    <div class="card">
        <h2>Session Data</h2>
        <p>This is the raw data stored in your session:</p>
        <pre style="background: #f6f8fa; padding: 10px; border-radius: 4px; overflow: auto;"><?php var_export($_SESSION); ?></pre>
        
        <div>
            <a href="/" class="button">Back to Home</a>
            <a href="/counter" class="button">Session Counter</a>
            <a href="/logout" class="button">Logout</a>
        </div>
    </div>
</body>
</html> 