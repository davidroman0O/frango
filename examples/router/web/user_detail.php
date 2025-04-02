<?php
$userId = $_SERVER['FRANGO_PARAM_id'] ?? null;
if (!$userId) { die("User ID required."); }
$apiUrl = "http://localhost:" . ($_SERVER["SERVER_PORT"] ?? 8082) . "/api/users/" . $userId;
$userDataJson = @file_get_contents($apiUrl);
$userData = json_decode($userDataJson, true);
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>User Details - Go-PHP Router Example</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f7f9fc;
        }
        h1, h2 {
            color: #2c3e50;
            margin-bottom: 1rem;
        }
        .card {
            border: 1px solid #e1e4e8;
            border-radius: 10px;
            padding: 24px;
            margin-bottom: 24px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.05);
            background-color: white;
        }
        .user-header {
            display: flex;
            align-items: center;
            margin-bottom: 20px;
            border-bottom: 1px solid #eaeaea;
            padding-bottom: 15px;
        }
        .avatar {
            width: 80px;
            height: 80px;
            border-radius: 50%;
            background-color: #3498db;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 2rem;
            font-weight: bold;
            margin-right: 20px;
        }
        .user-info {
            flex: 1;
        }
        .user-info h2 {
            margin: 0 0 5px 0;
        }
        .user-role {
            display: inline-block;
            padding: 3px 10px;
            border-radius: 15px;
            font-size: 0.8rem;
            text-transform: uppercase;
            font-weight: 600;
            margin-bottom: 5px;
        }
        .role-admin {
            background-color: #e74c3c;
            color: white;
        }
        .role-user {
            background-color: #2ecc71;
            color: white;
        }
        .detail-row {
            margin-bottom: 12px;
            display: flex;
            border-bottom: 1px solid #f2f2f2;
            padding-bottom: 12px;
        }
        .detail-row:last-child {
            border-bottom: none;
        }
        .detail-label {
            width: 120px;
            font-weight: 600;
            color: #555;
        }
        .detail-value {
            flex: 1;
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
            transition: background-color 0.2s ease;
        }
        .btn:hover {
            background-color: #2980b9;
        }
        .btn-edit {
            background-color: #f39c12;
        }
        .btn-edit:hover {
            background-color: #d68910;
        }
        .error-card {
            background-color: #fbeaeb;
            border-left: 4px solid #e74c3c;
            padding: 15px;
            margin-bottom: 20px;
        }
        .action-buttons {
            margin-top: 20px;
            display: flex;
            gap: 10px;
        }
    </style>
</head>
<body>
    <h1>User Details</h1>
    
    <?php if ($userData && isset($userData["user"])): ?>
        <?php $user = $userData["user"]; ?>
        <div class="card">
            <div class="user-header">
                <div class="avatar"><?= strtoupper(substr($user["name"], 0, 1)) ?></div>
                <div class="user-info">
                    <h2><?= htmlspecialchars($user["name"]) ?></h2>
                    <span class="user-role role-<?= strtolower($user["role"]) ?>">
                        <?= htmlspecialchars($user["role"]) ?>
                    </span>
                    <div><?= htmlspecialchars($user["email"]) ?></div>
                </div>
            </div>
            
            <div class="detail-row">
                <div class="detail-label">User ID</div>
                <div class="detail-value"><?= htmlspecialchars($user["id"]) ?></div>
            </div>
            
            <div class="detail-row">
                <div class="detail-label">Email</div>
                <div class="detail-value"><?= htmlspecialchars($user["email"]) ?></div>
            </div>
            
            <div class="detail-row">
                <div class="detail-label">Role</div>
                <div class="detail-value"><?= htmlspecialchars($user["role"]) ?></div>
            </div>
            
            <div class="detail-row">
                <div class="detail-label">Created</div>
                <div class="detail-value">
                    <?php 
                        $date = new DateTime($user["created_at"]);
                        echo $date->format('F j, Y \a\t g:i a');
                    ?>
                </div>
            </div>
            
            <?php if (isset($user["updated_at"])): ?>
            <div class="detail-row">
                <div class="detail-label">Last Updated</div>
                <div class="detail-value">
                    <?php 
                        $updated = new DateTime($user["updated_at"]);
                        echo $updated->format('F j, Y \a\t g:i a');
                    ?>
                </div>
            </div>
            <?php endif; ?>
            
            <div class="action-buttons">
                <a href="/users/<?= $user["id"] ?>/edit" class="btn btn-edit">Edit User</a>
                <a href="/" class="btn">Back to List</a>
            </div>
        </div>
    <?php else: ?>
        <div class="error-card">
            <h2>User Not Found</h2>
            <p>The requested user was not found or there was an API error.</p>
            <pre><?= htmlspecialchars($userDataJson) ?></pre>
            <div class="action-buttons">
                <a href="/" class="btn">Back to List</a>
            </div>
        </div>
    <?php endif; ?>
</body>
</html>
