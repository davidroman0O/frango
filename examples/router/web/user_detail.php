<?php
/**
 * User detail page
 * 
 * Displays detailed information about a specific user
 */

// Get user ID from query parameters
$userId = $_GET['id'] ?? null;

if (!$userId) {
    header('Location: index.php?error=User ID is required');
    exit;
}

// Fetch user data from API
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users/' . $userId;
$response = @file_get_contents($apiUrl);

if ($response === false) {
    header('Location: index.php?error=Failed to fetch user data');
    exit;
}

// Parse the response
$userData = json_decode($response, true);

// Check if user was found
if (!isset($userData['user'])) {
    header('Location: index.php?error=User not found');
    exit;
}

$user = $userData['user'];
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>User Details - <?= htmlspecialchars($user['name']) ?></title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1, h2 {
            color: #2c3e50;
        }
        .card {
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .field {
            margin-bottom: 15px;
        }
        .field-label {
            font-weight: bold;
            color: #555;
        }
        .btn {
            display: inline-block;
            background-color: #3498db;
            color: white;
            padding: 8px 16px;
            border-radius: 4px;
            text-decoration: none;
            font-size: 14px;
            margin-right: 8px;
        }
        .btn:hover {
            background-color: #2980b9;
        }
        .btn-danger {
            background-color: #e74c3c;
        }
        .btn-danger:hover {
            background-color: #c0392b;
        }
    </style>
</head>
<body>
    <h1>User Details</h1>
    
    <div class="card">
        <h2><?= htmlspecialchars($user['name']) ?></h2>
        
        <div class="field">
            <div class="field-label">ID:</div>
            <div><?= htmlspecialchars($user['id']) ?></div>
        </div>
        
        <div class="field">
            <div class="field-label">Email:</div>
            <div><?= htmlspecialchars($user['email']) ?></div>
        </div>
        
        <div class="field">
            <div class="field-label">Role:</div>
            <div><?= htmlspecialchars($user['role']) ?></div>
        </div>
        
        <div class="field">
            <div class="field-label">Created:</div>
            <div><?= htmlspecialchars($user['created_at']) ?></div>
        </div>
        
        <?php if (isset($user['updated_at'])): ?>
        <div class="field">
            <div class="field-label">Last Updated:</div>
            <div><?= htmlspecialchars($user['updated_at']) ?></div>
        </div>
        <?php endif; ?>
        
        <div style="margin-top: 20px;">
            <a href="index.php" class="btn">Back to List</a>
            <a href="user_edit.php?id=<?= $user['id'] ?>" class="btn">Edit User</a>
            <form method="post" action="user_delete.php" style="display:inline">
                <input type="hidden" name="id" value="<?= $user['id'] ?>">
                <button type="submit" class="btn btn-danger" onclick="return confirm('Are you sure you want to delete this user?')">Delete User</button>
            </form>
        </div>
    </div>
</body>
</html> 