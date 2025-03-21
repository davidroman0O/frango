<?php
// Define helper functions to access the variables injected from Go

/**
 * Gets a variable injected from Go
 * 
 * @param string $name The variable name
 * @param mixed $default The default value to return if the variable is not found
 * @return mixed The variable value
 */
function go_var($name, $default = null) {
    $envKey = "GOPHP_VAR_" . $name;
    
    if (!isset($_SERVER[$envKey])) {
        return $default;
    }
    
    $jsonValue = $_SERVER[$envKey];
    $value = json_decode($jsonValue, true);
    
    // Handle JSON decode errors
    if ($value === null && json_last_error() !== JSON_ERROR_NONE) {
        error_log("Error decoding Go variable {$name}: " . json_last_error_msg());
        return $default;
    }
    
    return $value;
}

/**
 * Gets all variables injected from Go
 * 
 * @return array An associative array of all injected variables
 */
function go_vars() {
    $vars = [];
    $prefix = "GOPHP_VAR_";
    $prefixLen = strlen($prefix);
    
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, $prefix) === 0) {
            $name = substr($key, $prefixLen);
            $jsonValue = $value;
            
            $decodedValue = json_decode($jsonValue, true);
            
            // Handle JSON decode errors
            if ($decodedValue === null && json_last_error() !== JSON_ERROR_NONE) {
                error_log("Error decoding Go variable {$name}: " . json_last_error_msg());
                continue;
            }
            
            $vars[$name] = $decodedValue;
        }
    }
    
    return $vars;
}

// Debug information
echo "<!-- Server variables: \n";
foreach ($_SERVER as $key => $value) {
    if (substr($key, 0, 6) === "DEBUG_") {
        echo "{$key}: {$value}\n";
    }
}
echo "-->\n";

// Get variables from Go
$title = go_var('title', 'Default Title');
$user = go_var('user', []);
$items = go_var('items', []);

// Get all variables at once
$allVars = go_vars();
?>
<!DOCTYPE html>
<html>
<head>
    <title><?= htmlspecialchars($title) ?></title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .user-info { background: #f0f0f0; padding: 10px; margin-bottom: 20px; }
        .items { list-style-type: none; padding: 0; }
        .item { border-bottom: 1px solid #eee; padding: 5px 0; }
    </style>
</head>
<body>
    <h1><?= htmlspecialchars($title) ?></h1>
    
    <?php if (!empty($user)): ?>
    <div class="user-info">
        <h2>User Information</h2>
        <p>Name: <?= htmlspecialchars($user['name']) ?></p>
        <p>Email: <?= htmlspecialchars($user['email']) ?></p>
        <p>Role: <?= htmlspecialchars($user['role']) ?></p>
    </div>
    <?php endif; ?>
    
    <?php if (!empty($items)): ?>
    <h2>Items</h2>
    <ul class="items">
        <?php foreach ($items as $item): ?>
        <li class="item">
            <strong><?= htmlspecialchars($item['name']) ?></strong>
            <p><?= htmlspecialchars($item['description']) ?></p>
            <p>Price: $<?= number_format($item['price'], 2) ?></p>
        </li>
        <?php endforeach; ?>
    </ul>
    <?php endif; ?>
    
    <footer>
        <p>Page rendered at: <?= date('Y-m-d H:i:s') ?></p>
    </footer>
</body>
</html> 