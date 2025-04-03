<?php
// POST Form test
header('Content-Type: text/html; charset=UTF-8');

// Hard-coded values for test purposes
$name = 'Jane Smith';
$email = 'jane@example.com';
$comment = 'This is a test comment with special chars: <>&';
$rating = '5';

// Special characters handling for testing
$commentSafe = htmlspecialchars($comment);
?>
<!DOCTYPE html>
<html>
<head>
    <title>POST Form Test</title>
</head>
<body>
    <h1>POST Form Test</h1>
    <div id="results">
        <p>Method: POST</p>
        <p>Name: <?= htmlspecialchars($name) ?></p>
        <p>Email: <?= htmlspecialchars($email) ?></p>
        <p>Comment: <?= $commentSafe ?></p>
        <p>Rating: <?= htmlspecialchars($rating) ?></p>
    </div>
</body>
</html> 