<?php
// Query parameters test
header('Content-Type: text/html; charset=UTF-8');

// Get query parameters
$name = $_GET['name'] ?? 'Guest';
$age = $_GET['age'] ?? 'Unknown';
$active = $_GET['active'] ?? '';
$interests = $_GET['interests'] ?? [];

// Also check the FRANGO_QUERY_ variables
$frangoName = $_SERVER['FRANGO_QUERY_name'] ?? 'Not set in FRANGO_QUERY';
?>
<!DOCTYPE html>
<html>
<head>
    <title>Query Parameters Test</title>
</head>
<body>
    <h1>Query Parameters Test</h1>
    
    <h2>Standard $_GET Access:</h2>
    <ul>
        <li>Name: <?= htmlspecialchars($name) ?></li>
        <li>Age: <?= htmlspecialchars($age) ?></li>
        <li>Active: <?= htmlspecialchars($active) ?></li>
        <li>Interests: 
            <?php if (is_array($interests)): ?>
                <ul>
                    <?php foreach ($interests as $interest): ?>
                        <li><?= htmlspecialchars($interest) ?></li>
                    <?php endforeach; ?>
                </ul>
            <?php else: ?>
                <?= htmlspecialchars($interests) ?>
            <?php endif; ?>
        </li>
    </ul>
    
    <h2>FRANGO_QUERY Access:</h2>
    <ul>
        <li>Name via FRANGO_QUERY: <?= htmlspecialchars($frangoName) ?></li>
    </ul>
    
    <p>Example URL: <code>?name=John&age=30&interests[]=coding&interests[]=hiking</code></p>
</body>
</html> 