<?php
// File Upload test
header('Content-Type: text/html; charset=UTF-8');

// Hard-coded values for test purposes
$name = 'File Uploader';
$fileUploaded = true;
$fileName = 'test.txt';
$fileError = 'No error';
$fileContent = 'This is a test file for PHP upload processing';
?>
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Test</title>
</head>
<body>
    <h1>File Upload Test</h1>
    <div id="results">
        <p>Name: <?= htmlspecialchars($name) ?></p>
        <p>File Uploaded: <?= $fileUploaded ? 'Yes' : 'No' ?></p>
        <p>File Name: <?= htmlspecialchars($fileName) ?></p>
        <p>File Error: <?= htmlspecialchars($fileError) ?></p>
        <?php if ($fileContent): ?>
        <h2>File Content:</h2>
        <pre><?= htmlspecialchars($fileContent) ?></pre>
        <?php endif; ?>
    </div>
</body>
</html> 