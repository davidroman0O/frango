<?php
/**
 * User edit page and form handler
 */

// Get user ID from query parameters
$userId = $_GET['id'] ?? null;

if (!$userId) {
    header('Location: index.php?error=User ID is required');
    exit;
}

// Initialize error and success messages
$error = null;
$success = null;

// Process form submission
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    // Validate form input
    if (empty($_POST['name']) || empty($_POST['email'])) {
        $error = 'Name and email are required';
    } else {
        // Prepare data for API
        $userData = [
            'name' => $_POST['name'],
            'email' => $_POST['email'],
            'role' => $_POST['role'] ?? 'user'
        ];
        
        // Call the API to update user
        $apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users/' . $userId;
        
        // Create stream context for PUT request
        $options = [
            'http' => [
                'method' => 'PUT',
                'header' => 'Content-Type: application/json',
                'content' => json_encode($userData),
                'ignore_errors' => true
            ]
        ];
        $context = stream_context_create($options);
        
        // Execute request
        $response = file_get_contents($apiUrl, false, $context);
        $statusCode = $http_response_header ? intval(substr($http_response_header[0], 9, 3)) : 500;
        
        // Parse response
        $result = json_decode($response, true);
        
        if ($statusCode === 200) {
            $success = 'User updated successfully';
            $userData = $result['user']; // Update local data with response
        } else {
            $error = isset($result['error']) ? $result['error'] : 'Failed to update user';
        }
    }
} else {
    // Fetch user data from API
    $apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users/' . $userId;
    $response = @file_get_contents($apiUrl);
    
    if ($response === false) {
        header('Location: index.php?error=Failed to fetch user data');
        exit;
    }
    
    // Parse the response
    $result = json_decode($response, true);
    
    // Check if user was found
    if (!isset($result['user'])) {
        header('Location: index.php?error=User not found');
        exit;
    }
    
    $userData = $result['user'];
}
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit User - <?= htmlspecialchars($userData['name']) ?></title>
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
    <h1>Edit User</h1>
    
    <?php if ($error): ?>
        <div class="error-message"><?= htmlspecialchars($error) ?></div>
    <?php endif; ?>
    
    <?php if ($success): ?>
        <div class="success-message"><?= htmlspecialchars($success) ?></div>
    <?php endif; ?>
    
    <div class="card">
        <form method="post">
            <div class="form-group">
                <label for="name">Name:</label>
                <input type="text" id="name" name="name" value="<?= htmlspecialchars($userData['name']) ?>" required>
            </div>
            
            <div class="form-group">
                <label for="email">Email:</label>
                <input type="email" id="email" name="email" value="<?= htmlspecialchars($userData['email']) ?>" required>
            </div>
            
            <div class="form-group">
                <label for="role">Role:</label>
                <select id="role" name="role">
                    <option value="user" <?= $userData['role'] === 'user' ? 'selected' : '' ?>>User</option>
                    <option value="admin" <?= $userData['role'] === 'admin' ? 'selected' : '' ?>>Admin</option>
                </select>
            </div>
            
            <div>
                <button type="submit" class="btn">Update User</button>
                <a href="user_detail.php?id=<?= $userData['id'] ?>" class="btn">Cancel</a>
            </div>
        </form>
    </div>
    
    <div>
        <a href="index.php" class="btn">Back to List</a>
    </div>
</body>
</html> 