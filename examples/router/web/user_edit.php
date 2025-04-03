<?php
/**
 * User edit page and form handler
 */

// Get user ID from URL segment (users/123/edit -> segment 1 is "123")
$userId = $_SERVER['FRANGO_URL_SEGMENT_1'] ?? null;

// Initialize debug variables
$debug = "URL Segments: ";
for ($i = 0; $i < ($_SERVER['FRANGO_URL_SEGMENT_COUNT'] ?? 0); $i++) {
    $debug .= "[$i]=" . ($_SERVER["FRANGO_URL_SEGMENT_$i"] ?? 'none') . " ";
}

// Initialize API debug info (to prevent undefined variable errors)
$apiDebug = "";

// Initialize userData with defaults to prevent undefined variable errors
$userData = [
    'id' => $userId,
    'name' => '',
    'email' => '',
    'role' => 'user'
];

// Initialize error and success messages
$error = null;
$success = null;

// Redirect if no user ID
if (!$userId) {
    $error = 'User ID is required';
} 
// Process form submission
else if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $apiDebug .= "Form submitted via POST\n";
    
    // Dump all superglobals for debugging
    $apiDebug .= "POST data: " . print_r($_POST, true) . "\n";
    $apiDebug .= "REQUEST data: " . print_r($_REQUEST, true) . "\n";
    $apiDebug .= "SERVER data: " . print_r($_SERVER, true) . "\n";
    
    // Get and sanitize POST data - direct access with fallbacks
    // IMPORTANT: Form values are in $_SERVER with FRANGO_FORM_ prefix, not in $_POST!
    $name = isset($_SERVER['FRANGO_FORM_name']) ? trim($_SERVER['FRANGO_FORM_name']) : '';
    $email = isset($_SERVER['FRANGO_FORM_email']) ? trim($_SERVER['FRANGO_FORM_email']) : '';
    $role = isset($_SERVER['FRANGO_FORM_role']) ? trim($_SERVER['FRANGO_FORM_role']) : 'user';
    
    $apiDebug .= "Form data from SERVER variables:\n";
    $apiDebug .= "  name = " . $name . "\n";
    $apiDebug .= "  email = " . $email . "\n";
    $apiDebug .= "  role = " . $role . "\n";
    
    // Debug dump all FRANGO_FORM_ prefixed values
    $apiDebug .= "\nAll FRANGO_FORM_ variables:\n";
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'FRANGO_FORM_') === 0) {
            $apiDebug .= "  $key = $value\n";
        }
    }
    
    // Validate form input
    if (empty($name) || empty($email)) {
        $error = 'Name and email are required';
        $apiDebug .= "Validation error: $error\n";
    } else {
        // Prepare data for API
        $userData = [
            'name' => $name,
            'email' => $email,
            'role' => $role
        ];
        
        // Call the API to update user using file_get_contents instead of cURL
        $apiUrl = 'http://localhost:' . ($_SERVER["SERVER_PORT"] ?? 8082) . '/api/users/' . $userId;
        $apiDebug .= "PUT to URL: $apiUrl\n";
        
        // Prepare the JSON data
        $jsonData = json_encode($userData);
        $apiDebug .= "JSON data to send: " . $jsonData . "\n";
        
        // Create stream context for PUT request
        $options = [
            'http' => [
                'method' => 'PUT',
                'header' => [
                    'Content-Type: application/json',
                    'Accept: application/json',
                    'Content-Length: ' . strlen($jsonData)
                ],
                'content' => $jsonData,
                'ignore_errors' => true,
                'timeout' => 15
            ]
        ];
        $context = stream_context_create($options);
        
        // Execute the request
        $apiDebug .= "Sending API request using file_get_contents...\n";
        
        // Suppress warnings
        $oldErrorReporting = error_reporting(0);
        $response = @file_get_contents($apiUrl, false, $context);
        error_reporting($oldErrorReporting);
        
        // Get HTTP response code
        $httpCode = $http_response_header ? intval(substr($http_response_header[0], 9, 3)) : 0;
        
        $apiDebug .= "API Response HTTP Code: $httpCode\n";
        $apiDebug .= "API Response Headers: " . json_encode($http_response_header ?? []) . "\n";
        $apiDebug .= "API Raw Response: " . ($response ?: '[empty response]') . "\n";
        
        // Parse the response if we got one
        if ($response !== false) {
            $result = json_decode($response, true);
            $jsonError = json_last_error();
            $apiDebug .= "JSON Decode Result: " . ($jsonError === JSON_ERROR_NONE ? "Success" : json_last_error_msg()) . "\n";
            
            if ($jsonError !== JSON_ERROR_NONE) {
                $error = 'Failed to decode API response: ' . json_last_error_msg();
                $apiDebug .= "JSON decode error: " . json_last_error_msg() . "\n";
            } else if (isset($result['user'])) {
                $success = 'User updated successfully';
                $userData = $result['user']; // Update local data with response
                $apiDebug .= "User updated successfully. New data: " . json_encode($userData) . "\n";
            } else {
                $error = isset($result['error']) ? $result['error'] : 'Unknown API error';
                $apiDebug .= "API returned error: " . ($error ?? 'unknown') . "\n";
            }
        } else {
            $error = "API request failed";
            $apiDebug .= "API request failed. HTTP code: $httpCode\n";
            if (isset($http_response_header)) {
                $apiDebug .= "Response headers: " . print_r($http_response_header, true) . "\n";
            }
        }
    }
} else {
    // Fetch user data from API
    $apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/users/' . $userId;
    
    $apiDebug .= "GET request to URL: $apiUrl\n";
    
    // Make the API request with better error handling
    $response = @file_get_contents($apiUrl);
    $statusCode = $http_response_header ? intval(substr($http_response_header[0], 9, 3)) : 0;
    $apiDebug .= "API Response Status Code: $statusCode\n";
    $apiDebug .= "API Response Success: " . ($response !== false ? "Yes" : "No") . "\n";
    
    if ($response === false) {
        $error = 'Failed to fetch user data from API';
        $apiDebug .= "Error: " . (error_get_last() ? error_get_last()['message'] : 'Unknown error') . "\n";
    } else {
        // Parse the response
        $result = json_decode($response, true);
        $jsonError = json_last_error();
        $apiDebug .= "JSON Decode Result: " . ($jsonError === JSON_ERROR_NONE ? "Success" : json_last_error_msg()) . "\n";
        $apiDebug .= "Raw response: " . substr($response, 0, 200) . (strlen($response) > 200 ? '...' : '') . "\n";
        $apiDebug .= "Parsed result: " . json_encode($result) . "\n";
        $apiDebug .= "Result contains 'user': " . (isset($result['user']) ? "Yes" : "No") . "\n";
        
        // Check if user was found
        if ($jsonError !== JSON_ERROR_NONE) {
            $error = 'Failed to parse API response: ' . json_last_error_msg();
            $apiDebug .= "JSON parsing error: " . json_last_error_msg() . "\n";
        } else if (!isset($result['user']) || !is_array($result['user'])) {
            $error = 'User data not found in API response';
            $apiDebug .= "User not found in response. Response structure: " . json_encode(array_keys($result)) . "\n";
        } else {
            $userData = $result['user'];
            $apiDebug .= "User data successfully loaded: " . json_encode($userData) . "\n";
        }
    }
}
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit User<?= !empty($userData['name']) ? ' - ' . htmlspecialchars($userData['name']) : '' ?></title>
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
        <form method="post" action="">
            <div class="form-group">
                <label for="name">Name:</label>
                <input type="text" id="name" name="name" value="<?= htmlspecialchars($userData['name'] ?? '') ?>">
            </div>
            
            <div class="form-group">
                <label for="email">Email:</label>
                <input type="email" id="email" name="email" value="<?= htmlspecialchars($userData['email'] ?? '') ?>">
            </div>
            
            <div class="form-group">
                <label for="role">Role:</label>
                <select id="role" name="role">
                    <option value="user" <?= (isset($userData['role']) && strtolower($userData['role']) === 'user') ? 'selected' : '' ?>>User</option>
                    <option value="admin" <?= (isset($userData['role']) && strtolower($userData['role']) === 'admin') ? 'selected' : '' ?>>Admin</option>
                </select>
            </div>
            
            <div>
                <button type="submit" class="btn">Update User</button>
                <a href="/users/<?= htmlspecialchars($userData['id'] ?? $userId) ?>" class="btn">Cancel</a>
                <a href="/" class="btn">Back to Home</a>
            </div>
        </form>
    </div>
    
    <div>
        <a href="/" class="btn">Back to List</a>
    </div>
    
    <!-- Debug info -->
    <div style="margin-top: 30px; padding: 15px; background: #f5f5f5; border: 1px solid #ddd; border-radius: 5px; font-family: monospace; font-size: 12px;">
        <h3>Debug Information</h3>
        <p><strong>URL Segments:</strong> <?= htmlspecialchars($debug ?? '') ?></p>
        <p><strong>Raw URL Path:</strong> <?= htmlspecialchars($_SERVER['FRANGO_URL_PATH'] ?? 'not available') ?></p>
        <pre style="white-space: pre-wrap; word-break: break-all;"><?= htmlspecialchars($apiDebug ?? '') ?></pre>
    </div>
</body>
</html> 