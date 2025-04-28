<?php
/**
 * Frango v2 Demo - User Profile Page
 * 
 * Demonstrates path parameter extraction from URLs like /users/{userID}
 */

// Extract path parameters from the URL
// For a URL like /users/123, the {userID} parameter should be available in $_GET
$userID = $_GET['userID'] ?? 'unknown';

// Sample user data - in a real app this would come from a database
$users = [
    '123' => [
        'id' => 123,
        'name' => 'John Smith',
        'email' => 'john@example.com',
        'role' => 'Administrator',
        'joined' => '2022-03-15',
        'avatar' => 'male'
    ],
    '456' => [
        'id' => 456,
        'name' => 'Jane Doe',
        'email' => 'jane@example.com',
        'role' => 'Editor',
        'joined' => '2022-06-22',
        'avatar' => 'female'
    ],
    '789' => [
        'id' => 789,
        'name' => 'Alex Johnson',
        'email' => 'alex@example.com',
        'role' => 'Subscriber',
        'joined' => '2023-01-10',
        'avatar' => 'male'
    ]
];

// Get the current user data or use placeholder data
$userData = $users[$userID] ?? [
    'id' => $userID,
    'name' => 'Unknown User',
    'email' => 'no-email@example.com',
    'role' => 'Guest',
    'joined' => 'Unknown',
    'avatar' => 'unknown'
];

// Access the raw path segments if needed
$pathSegments = $GLOBALS['_PATH_SEGMENTS'] ?? [];

// Function to get a random user ID for demo navigation
function getRandomUserID($exclude = null) {
    global $users;
    $ids = array_keys($users);
    if ($exclude !== null) {
        $ids = array_filter($ids, function($id) use ($exclude) {
            return $id != $exclude;
        });
    }
    return $ids[array_rand($ids)] ?? '123';
}
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>User Profile - <?= htmlspecialchars($userData['name']) ?></title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <h1>User Profile</h1>
        <p>Demonstrating path parameter extraction from URLs like <code>/users/{userID}</code>.</p>
        
        <div class="card">
            <div style="display: flex; align-items: center;">
                <div style="margin-right: 20px;">
                    <?php
                    // Show an avatar image based on user data
                    $avatarType = $userData['avatar'] ?? 'unknown';
                    $avatarColor = '';
                    
                    switch($avatarType) {
                        case 'male': $avatarColor = '#007bff'; break;
                        case 'female': $avatarColor = '#e83e8c'; break;
                        default: $avatarColor = '#6c757d'; break;
                    }
                    ?>
                    <div style="width: 100px; height: 100px; border-radius: 50%; background-color: <?= $avatarColor ?>; color: white; display: flex; justify-content: center; align-items: center; font-size: 40px;">
                        <?= strtoupper(substr($userData['name'], 0, 1)) ?>
                    </div>
                </div>
                <div>
                    <h2><?= htmlspecialchars($userData['name']) ?></h2>
                    <p><strong>ID:</strong> <?= htmlspecialchars($userData['id']) ?></p>
                    <p><strong>Email:</strong> <?= htmlspecialchars($userData['email']) ?></p>
                    <p><strong>Role:</strong> <?= htmlspecialchars($userData['role']) ?></p>
                    <p><strong>Joined:</strong> <?= htmlspecialchars($userData['joined']) ?></p>
                </div>
            </div>
        </div>
        
        <div class="card">
            <h2>URL Path Parameters</h2>
            <p>This page received the following parameter from the URL:</p>
            <ul>
                <li><strong>userID:</strong> <?= htmlspecialchars($userID) ?></li>
            </ul>
            
            <h3>How It Works</h3>
            <p>When you navigate to <code>/users/123</code>, Frango extracts the <code>123</code> value from the URL pattern <code>/users/{userID}</code> and makes it available as <code>$_GET['userID']</code> in your PHP code.</p>
            
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// Access path parameters from the URL pattern:
$userID = $_GET['userID'] ?? 'default';

// Use it in your code:
$userData = getUserData($userID);
            </pre>
        </div>
        
        <div class="card">
            <h2>Path Segments</h2>
            <p>For more advanced routing needs, you can access the raw path segments:</p>
            <pre><?php var_export($pathSegments); ?></pre>
            
            <h3>Code Example</h3>
            <pre style="background: #2c3e50; color: #fff; padding: 15px; border-radius: 4px;">
// Access all path segments:
$segments = $GLOBALS['_PATH_SEGMENTS'] ?? [];

// Access specific segment by index:
$userIDSegment = $segments[1] ?? null; // Index 1 is the second segment
            </pre>
        </div>
        
        <div class="card">
            <h2>URL Components</h2>
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
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>SCRIPT_NAME</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['SCRIPT_NAME']) ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>SCRIPT_FILENAME</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['SCRIPT_FILENAME']) ?></td>
                </tr>
                <tr>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><code>PATH_INFO</code></td>
                    <td style="padding: 8px; border: 1px solid #dee2e6;"><?= htmlspecialchars($_SERVER['PATH_INFO'] ?? 'Not set') ?></td>
                </tr>
            </table>
        </div>
        
        <div class="card">
            <h2>Try Different Users</h2>
            <ul>
                <?php foreach ($users as $id => $user): ?>
                    <li><a href="/users/<?= htmlspecialchars($id) ?>"><?= htmlspecialchars($user['name']) ?> (ID: <?= htmlspecialchars($id) ?>)</a></li>
                <?php endforeach; ?>
                <li><a href="/users/999">Unknown User (ID: 999)</a></li>
            </ul>
        </div>
        
        <div class="card">
            <h2>Navigation</h2>
            <p><a href="/products/<?= htmlspecialchars($userID) ?>?owner=true">View User's Products</a></p>
            <p><a href="/categories/electronics/smartphones">Browse Categories</a></p>
            <p><a href="/">Back to Dashboard</a></p>
        </div>
    </div>
    
    <!-- Include the debug panel -->
    <?php include dirname(dirname(__FILE__)) . '/debug_panel.php'; ?>
</body>
</html> 