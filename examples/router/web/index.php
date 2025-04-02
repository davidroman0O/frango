<?php
/**
 * Main index page that leverages the API endpoints
 */

// Get data from API
$usersUrl = "http://localhost:" . ($_SERVER["SERVER_PORT"] ?? 8082) . "/api/users";
$itemsUrl = "http://localhost:" . ($_SERVER["SERVER_PORT"] ?? 8082) . "/api/items";

$usersJson = @file_get_contents($usersUrl);
$itemsJson = @file_get_contents($itemsUrl);

$usersData = json_decode($usersJson, true);
$itemsData = json_decode($itemsJson, true);

// Get flash messages from Go (passed via RenderHandlerFor)
$flashMessages = json_decode($_SERVER['FRANGO_VAR_flash_messages'] ?? '[]', true) ?? [];

// Ensure $flashMessages is always an array
if (!is_array($flashMessages)) {
    $flashMessages = [];
}
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
    // Display flash messages - handle both uppercase and lowercase field names
    foreach ($flashMessages as $message): 
        // Try both uppercase and lowercase keys
        $type = $message['Type'] ?? $message['type'] ?? 'info';
        $content = $message['Content'] ?? $message['content'] ?? '';
        $msgClass = $type . '-message';
    ?>
        <div class="<?= $msgClass ?>"><?= htmlspecialchars($content) ?></div>
    <?php endforeach; ?>
    
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
                                    <a href="/users/<?= $user['id'] ?>" class="btn">View</a>
                                    <a href="/users/<?= $user['id'] ?>/edit" class="btn">Edit</a>
                                    <button type="button" class="btn btn-danger" onclick="deleteUser(<?= $user['id'] ?>)">Delete (JS)</button>
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
            <form action="/api/users" method="post" id="addUserForm">
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
                                    <a href="/items/<?= $item['id'] ?>" class="btn">View</a>
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
            <form action="/api/items" method="post" id="addItemForm">
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
        <p>API endpoints available at: /api/*</p>
    </footer>

    <script>
        // Basic JS for Add User form submission (using fetch)
        document.getElementById('addUserForm').addEventListener('submit', function(e) {
            e.preventDefault();
            const formData = new FormData(this);
            const data = Object.fromEntries(formData.entries());
            
            fetch('/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            })
            .then(response => response.json())
            .then(result => {
                console.log('Success:', result);
                // Use our Go message endpoint to set the message
                window.location.href = '/message?type=success&content=' + 
                    encodeURIComponent('User ' + (result.user?.name || '') + ' created');
            })
            .catch((error) => {
                console.error('Error:', error);
                window.location.href = '/message?type=error&content=' + 
                    encodeURIComponent('Failed to create user');
            });
        });
        
        // Basic JS for Add Item form submission
        document.getElementById('addItemForm').addEventListener('submit', function(e) {
            e.preventDefault();
            const formData = new FormData(this);
            const data = Object.fromEntries(formData.entries());

            fetch('/api/items', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            })
            .then(response => response.json())
            .then(result => {
                console.log('Success:', result);
                window.location.href = '/message?type=success&content=' + 
                    encodeURIComponent('Item ' + (result.item?.name || '') + ' created');
            })
            .catch((error) => {
                console.error('Error:', error);
                window.location.href = '/message?type=error&content=' + 
                    encodeURIComponent('Failed to create item');
            });
        });

        // Basic JS for Delete User button
        function deleteUser(userId) {
            if (!confirm('Are you sure you want to delete user ' + userId + '?')) {
                return;
            }
            fetch('/api/users/' + userId, {
                method: 'DELETE',
            })
            .then(response => response.json())
            .then(result => {
                console.log('Success:', result);
                window.location.href = '/message?type=success&content=' + 
                    encodeURIComponent('User ' + userId + ' deleted');
            })
            .catch((error) => {
                console.error('Error:', error);
                window.location.href = '/message?type=error&content=' + 
                    encodeURIComponent('Failed to delete user ' + userId);
            });
        }
    </script>

</body>
</html> 