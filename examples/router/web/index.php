<?php
/**
 * Main index page that leverages the API endpoints
 */
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go-PHP Router Example</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        h1, h2, h3 {
            color: #2c3e50;
        }
        .container {
            display: flex;
            flex-wrap: wrap;
            gap: 20px;
        }
        .card {
            flex: 1 1 45%;
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 20px;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f8f9fa;
            font-weight: 600;
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
            cursor: pointer;
            border: none;
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
        form {
            margin-top: 20px;
            padding: 20px;
            background-color: #f8f9fa;
            border-radius: 8px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
        }
        input, select {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
        }
        .success-message {
            color: #27ae60;
            background-color: #edfbf0;
            border: 1px solid #27ae60;
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .error-message {
            color: #e74c3c;
            background-color: #fbeaeb;
            border: 1px solid #e74c3c;
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <h1>Go-PHP Router Example</h1>
    <p>This example demonstrates the integration between Go and PHP using REST APIs. The Go backend provides the data while PHP renders the UI.</p>
    
    <?php
    // Fetch data from our API endpoints
    $usersJson = file_get_contents('http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users');
    $itemsJson = file_get_contents('http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/items');
    
    // Parse the JSON responses
    $usersData = json_decode($usersJson, true);
    $itemsData = json_decode($itemsJson, true);
    
    // Get success/error messages from URL parameters
    $message = $_GET['message'] ?? '';
    $error = $_GET['error'] ?? '';
    $success = $_GET['success'] ?? '';
    
    // Display any messages
    if ($message) {
        echo '<div class="info-message">' . htmlspecialchars($message) . '</div>';
    }
    if ($error) {
        echo '<div class="error-message">' . htmlspecialchars($error) . '</div>';
    }
    if ($success) {
        echo '<div class="success-message">' . htmlspecialchars($success) . '</div>';
    }
    ?>
    
    <div class="container">
        <div class="card">
            <h2>Users</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Name</th>
                        <th>Email</th>
                        <th>Role</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <?php if (isset($usersData['users']) && is_array($usersData['users'])): ?>
                        <?php foreach ($usersData['users'] as $user): ?>
                            <tr>
                                <td><?= htmlspecialchars($user['id']) ?></td>
                                <td><?= htmlspecialchars($user['name']) ?></td>
                                <td><?= htmlspecialchars($user['email']) ?></td>
                                <td><?= htmlspecialchars($user['role']) ?></td>
                                <td>
                                    <a href="user_detail.php?id=<?= $user['id'] ?>" class="btn">View</a>
                                    <a href="user_edit.php?id=<?= $user['id'] ?>" class="btn">Edit</a>
                                    <form method="post" action="user_delete.php" style="display:inline">
                                        <input type="hidden" name="id" value="<?= $user['id'] ?>">
                                        <button type="submit" class="btn btn-danger" onclick="return confirm('Are you sure you want to delete this user?')">Delete</button>
                                    </form>
                                </td>
                            </tr>
                        <?php endforeach; ?>
                    <?php else: ?>
                        <tr>
                            <td colspan="5">No users found</td>
                        </tr>
                    <?php endif; ?>
                </tbody>
            </table>
            
            <h3>Add New User</h3>
            <form action="user_create.php" method="post">
                <div class="form-group">
                    <label for="name">Name:</label>
                    <input type="text" id="name" name="name" required>
                </div>
                <div class="form-group">
                    <label for="email">Email:</label>
                    <input type="email" id="email" name="email" required>
                </div>
                <div class="form-group">
                    <label for="role">Role:</label>
                    <select id="role" name="role">
                        <option value="user">User</option>
                        <option value="admin">Admin</option>
                    </select>
                </div>
                <button type="submit" class="btn">Create User</button>
            </form>
        </div>
        
        <div class="card">
            <h2>Items</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Name</th>
                        <th>Description</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <?php if (isset($itemsData['items']) && is_array($itemsData['items'])): ?>
                        <?php foreach ($itemsData['items'] as $item): ?>
                            <tr>
                                <td><?= htmlspecialchars($item['id']) ?></td>
                                <td><?= htmlspecialchars($item['name']) ?></td>
                                <td><?= htmlspecialchars($item['description']) ?></td>
                                <td>
                                    <a href="item_detail.php?id=<?= $item['id'] ?>" class="btn">View</a>
                                </td>
                            </tr>
                        <?php endforeach; ?>
                    <?php else: ?>
                        <tr>
                            <td colspan="4">No items found</td>
                        </tr>
                    <?php endif; ?>
                </tbody>
            </table>
            
            <h3>Add New Item</h3>
            <form action="item_create.php" method="post">
                <div class="form-group">
                    <label for="item_name">Name:</label>
                    <input type="text" id="item_name" name="name" required>
                </div>
                <div class="form-group">
                    <label for="description">Description:</label>
                    <input type="text" id="description" name="description">
                </div>
                <button type="submit" class="btn">Create Item</button>
            </form>
        </div>
    </div>
    
    <footer style="margin-top: 40px; text-align: center; color: #7f8c8d; font-size: 14px;">
        <p>Page generated at: <?= date('Y-m-d H:i:s') ?></p>
        <p>API endpoints available at: /api/users and /api/items</p>
    </footer>
</body>
</html> 