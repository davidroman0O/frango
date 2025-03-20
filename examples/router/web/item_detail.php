<?php
/**
 * Item detail page
 * 
 * Displays detailed information about a specific item
 */

// Get item ID from query parameters
$itemId = $_GET['id'] ?? null;

if (!$itemId) {
    header('Location: index.php?error=Item ID is required');
    exit;
}

// Fetch item data from API
$apiUrl = 'http://localhost:' . ($_SERVER['SERVER_PORT'] ?? 8082) . '/api/items/' . $itemId;
$response = @file_get_contents($apiUrl);

if ($response === false) {
    header('Location: index.php?error=Failed to fetch item data');
    exit;
}

// Parse the response
$itemData = json_decode($response, true);

// Check if item was found
if (!isset($itemData['item'])) {
    header('Location: index.php?error=Item not found');
    exit;
}

$item = $itemData['item'];
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Item Details - <?= htmlspecialchars($item['name']) ?></title>
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
    </style>
</head>
<body>
    <h1>Item Details</h1>
    
    <div class="card">
        <h2><?= htmlspecialchars($item['name']) ?></h2>
        
        <div class="field">
            <div class="field-label">ID:</div>
            <div><?= htmlspecialchars($item['id']) ?></div>
        </div>
        
        <div class="field">
            <div class="field-label">Description:</div>
            <div><?= htmlspecialchars($item['description']) ?></div>
        </div>
        
        <div class="field">
            <div class="field-label">Created:</div>
            <div><?= htmlspecialchars($item['created_at']) ?></div>
        </div>
        
        <div style="margin-top: 20px;">
            <a href="index.php" class="btn">Back to List</a>
        </div>
    </div>
</body>
</html> 