<?php
// Include our simple Redis client
require_once 'SimpleRedis.php';

// Set appropriate headers
header('Content-Type: text/html; charset=utf-8');

// Initialize Redis connection variables
$redisHost = '127.0.0.1';
$redisPort = 6379;
$redisConnected = false;
$redisInfo = [];
$redisKeys = [];
$redisError = '';

// Try to connect to Redis
try {
    $redis = new SimpleRedis($redisHost, $redisPort, 2.0);
    $redisConnected = $redis->connect();
    
    if ($redisConnected) {
        // Get Redis server info
        $redisInfo = $redis->info();
        
        // Set a test key if it doesn't exist
        if (!$redis->exists('gophp_test_counter')) {
            $redis->set('gophp_test_counter', 0);
        }
        
        // Increment the counter
        $counter = $redis->incr('gophp_test_counter');
        
        // Set a timestamp for this visit
        $timestamp = time();
        $redis->set('gophp_last_visit', $timestamp);
        $redis->set('gophp_last_visit_formatted', date('Y-m-d H:i:s', $timestamp));
        
        // Get all keys matching the gophp pattern
        $redisKeys = $redis->keys('gophp_*');
        if (!is_array($redisKeys)) {
            $redisKeys = [];
        }
        sort($redisKeys);
    }
} catch (Exception $e) {
    $redisError = $e->getMessage();
    $redisConnected = false;
}

// Current timestamp for debugging
$timestamp = date('Y-m-d H:i:s');
?>
<!DOCTYPE html>
<html>
<head>
    <title>GoPHP Redis Example</title>
    <meta http-equiv="Cache-Control" content="no-cache, no-store, must-revalidate">
    <meta http-equiv="Pragma" content="no-cache">
    <meta http-equiv="Expires" content="0">
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 5px;
            background-color: #fff;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
        }
        h2 {
            color: #555;
            margin-top: 20px;
        }
        pre {
            background: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow-x: auto;
        }
        .status {
            padding: 5px 10px;
            border-radius: 4px;
            display: inline-block;
            font-weight: bold;
        }
        .status-ok {
            background-color: #d4edda;
            color: #155724;
        }
        .status-error {
            background-color: #f8d7da;
            color: #721c24;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>GoPHP Redis Example</h1>
        
        <h2>PHP Information</h2>
        <table>
            <tr>
                <th>PHP Version</th>
                <td><?php echo phpversion(); ?></td>
            </tr>
            <tr>
                <th>Current Time</th>
                <td><?php echo $timestamp; ?></td>
            </tr>
            <tr>
                <th>Redis Client</th>
                <td>SimpleRedis (pure PHP implementation)</td>
            </tr>
        </table>

        <h2>Redis Connection</h2>
        <table>
            <tr>
                <th>Redis Host</th>
                <td><?php echo $redisHost; ?></td>
            </tr>
            <tr>
                <th>Redis Port</th>
                <td><?php echo $redisPort; ?></td>
            </tr>
            <tr>
                <th>Connection Status</th>
                <td>
                    <span class="status <?php echo $redisConnected ? 'status-ok' : 'status-error'; ?>">
                        <?php echo $redisConnected ? 'Connected' : 'Not Connected'; ?>
                    </span>
                    <?php if (!empty($redisError)): ?>
                        <p><?php echo htmlspecialchars($redisError); ?></p>
                    <?php endif; ?>
                </td>
            </tr>
        </table>

        <?php if ($redisConnected): ?>
        <h2>Redis Test Counter</h2>
        <p>This page has been viewed <?php echo $counter; ?> times since the counter was initialized.</p>
        <p>Last visit: <?php echo $redis->get('gophp_last_visit_formatted'); ?></p>

        <h2>Redis Keys</h2>
        <table>
            <tr>
                <th>Key</th>
                <th>Value</th>
            </tr>
            <?php foreach ($redisKeys as $key): ?>
            <tr>
                <td><?php echo htmlspecialchars($key); ?></td>
                <td>
                    <?php 
                    echo htmlspecialchars($redis->get($key));
                    ?>
                </td>
            </tr>
            <?php endforeach; ?>
        </table>

        <h2>Redis Server Info</h2>
        <pre><?php print_r($redisInfo); ?></pre>
        <?php endif; ?>
    </div>
</body>
</html>
