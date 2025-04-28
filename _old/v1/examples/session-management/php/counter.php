<?php
// Start session
session_start();

// Initialize or increment counter
if (!isset($_SESSION['counter'])) {
    $_SESSION['counter'] = 1;
} else {
    $_SESSION['counter']++;
}

// Get counter value
$counter = $_SESSION['counter'];

// Initialize or update page view history
if (!isset($_SESSION['page_views'])) {
    $_SESSION['page_views'] = [];
}

// Add this page view to history
$_SESSION['page_views'][] = [
    'page' => 'counter',
    'time' => time(),
    'counter' => $counter
];

// Get login status
$loggedIn = isset($_SESSION['user']);
$username = $loggedIn ? $_SESSION['user']['username'] : 'Guest';

header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Session Counter - Frango Example</title>
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
        .counter-display {
            text-align: center;
            padding: 30px;
            background: #f6f8fa;
            border-radius: 4px;
            margin: 20px 0;
        }
        .counter-value {
            font-size: 72px;
            font-weight: bold;
            color: #0366d6;
        }
        .counter-label {
            font-size: 18px;
            color: #666;
            margin-top: 10px;
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
        .action-buttons {
            display: flex;
            justify-content: center;
            margin: 20px 0;
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
            <h1 style="margin: 0;">Session Counter</h1>
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
        <h2>Session Counter Example</h2>
        <p>This page demonstrates a simple counter that persists across page refreshes using PHP sessions.</p>
        <p>Each time you visit or refresh this page, the counter will increment by 1.</p>
        
        <div class="counter-display">
            <div class="counter-value"><?= $counter ?></div>
            <div class="counter-label">Page Views</div>
        </div>
        
        <div class="action-buttons">
            <a href="/counter" class="button">Refresh Page (Increment Counter)</a>
            <a href="/counter?reset=1" class="button">Reset Counter</a>
        </div>
        
        <p>This simple example shows how sessions maintain state between HTTP requests, without requiring cookies or URL parameters to be manually managed.</p>
    </div>
    
    <?php if (!empty($_SESSION['page_views'])): ?>
    <div class="card">
        <h2>Page View History</h2>
        <p>The session has recorded the following page views:</p>
        
        <table>
            <thead>
                <tr>
                    <th>#</th>
                    <th>Page</th>
                    <th>Counter Value</th>
                    <th>Time</th>
                </tr>
            </thead>
            <tbody>
                <?php foreach (array_reverse($_SESSION['page_views']) as $index => $view): ?>
                    <tr>
                        <td><?= count($_SESSION['page_views']) - $index ?></td>
                        <td><?= htmlspecialchars($view['page']) ?></td>
                        <td><?= $view['counter'] ?></td>
                        <td><?= date('Y-m-d H:i:s', $view['time']) ?></td>
                    </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    </div>
    <?php endif; ?>
    
    <div class="card">
        <h2>Session Information</h2>
        <p><strong>Session ID:</strong> <?= session_id() ?></p>
        <p><strong>User:</strong> <?= htmlspecialchars($username) ?></p>
        <p><strong>Session Status:</strong> <?= session_status() === PHP_SESSION_ACTIVE ? 'Active' : 'Inactive' ?></p>
        
        <div>
            <a href="/" class="button">Back to Home</a>
            <?php if ($loggedIn): ?>
                <a href="/dashboard" class="button">Dashboard</a>
            <?php else: ?>
                <a href="/login" class="button">Login</a>
            <?php endif; ?>
        </div>
    </div>
</body>
</html>

<?php
// Handle reset request (must be after the HTML to show the counter first)
if (isset($_GET['reset']) && $_GET['reset'] === '1') {
    $_SESSION['counter'] = 0;
    header('Location: /counter');
    exit;
}
?> 