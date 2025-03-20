<?php
// Start session for counter
session_start();

// Initialize or increment counter
if (!isset($_SESSION['page_counter'])) {
    $_SESSION['page_counter'] = 1;
} else {
    $_SESSION['page_counter']++;
}

// Handle reset action
if (isset($_GET['action']) && $_GET['action'] === 'reset') {
    $_SESSION['page_counter'] = 0;
    header('Location: counter.php');
    exit;
}

// Get counter value
$counter = $_SESSION['page_counter'];
?>
<!DOCTYPE html>
<html>
<head>
    <title>Counter - Static Files PHP Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }
        h1 {
            color: #333;
        }
        .counter-display {
            margin: 30px 0;
            text-align: center;
        }
        .counter-value {
            font-size: 72px;
            font-weight: bold;
            color: #2196F3;
            padding: 20px;
            background-color: #f5f5f5;
            border-radius: 8px;
            display: inline-block;
            min-width: 100px;
        }
        .counter-text {
            margin-top: 10px;
            font-size: 16px;
            color: #666;
        }
        .actions {
            display: flex;
            justify-content: center;
            gap: 20px;
            margin: 30px 0;
        }
        .btn {
            display: inline-block;
            padding: 10px 20px;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            cursor: pointer;
            font-weight: bold;
        }
        .btn-primary {
            background-color: #4CAF50;
        }
        .btn-danger {
            background-color: #f44336;
        }
        .btn:hover {
            opacity: 0.9;
        }
        .session-info {
            margin-top: 30px;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 4px;
        }
        .nav {
            margin-top: 30px;
            text-align: center;
        }
        .nav a {
            display: inline-block;
            padding: 10px 15px;
            background-color: #2196F3;
            color: white;
            text-decoration: none;
            border-radius: 4px;
        }
        .nav a:hover {
            background-color: #0b7dda;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Session Counter Example</h1>
        
        <div class="counter-display">
            <div class="counter-value"><?php echo $counter; ?></div>
            <div class="counter-text">Page views this session</div>
        </div>
        
        <div class="actions">
            <a href="counter.php" class="btn btn-primary">Refresh Page (Increment)</a>
            <a href="counter.php?action=reset" class="btn btn-danger">Reset Counter</a>
        </div>
        
        <div class="session-info">
            <h3>Session Information</h3>
            <p>Session ID: <?php echo session_id(); ?></p>
            <p>PHP Session Path: <?php echo session_save_path(); ?></p>
            <p>This counter persists until you close your browser or the session expires.</p>
        </div>
        
        <div class="nav">
            <a href="/">Back to Home</a>
        </div>
    </div>
</body>
</html> 