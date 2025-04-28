<?php
// Environment variable access test
header('Content-Type: text/html; charset=UTF-8');

// Get some environment variables
$phpVersion = phpversion();
$serverSoftware = $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown';
$serverProtocol = $_SERVER['SERVER_PROTOCOL'] ?? 'Unknown';
$requestMethod = $_SERVER['REQUEST_METHOD'] ?? 'Unknown';
$requestTime = $_SERVER['REQUEST_TIME'] ?? 0;
$documentRoot = $_SERVER['DOCUMENT_ROOT'] ?? 'Unknown';
$scriptFilename = $_SERVER['SCRIPT_FILENAME'] ?? 'Unknown';
$remoteAddr = $_SERVER['REMOTE_ADDR'] ?? 'Unknown';

// Get custom environment variables from system
$customVar1 = getenv('TEST_ENV_VAR1') ?: 'Not set';
$customVar2 = getenv('TEST_ENV_VAR2') ?: 'Not set';

// Get superglobals
$hasPath = isset($_PATH) ? 'Yes' : 'No';
$hasPathSegments = isset($_PATH_SEGMENTS) ? 'Yes' : 'No';
$hasJson = isset($_JSON) ? 'Yes' : 'No';
?>
<!DOCTYPE html>
<html>
<head>
    <title>PHP Environment Variables Test</title>
</head>
<body>
    <h1>PHP Environment Variables Test</h1>
    
    <div id="results">
        <h2>PHP Information</h2>
        <p>PHP Version: <?= htmlspecialchars($phpVersion) ?></p>
        
        <h2>Standard Server Variables</h2>
        <p>Server Software: <?= htmlspecialchars($serverSoftware) ?></p>
        <p>Server Protocol: <?= htmlspecialchars($serverProtocol) ?></p>
        <p>Request Method: <?= htmlspecialchars($requestMethod) ?></p>
        <p>Request Time: <?= date('Y-m-d H:i:s', $requestTime) ?></p>
        <p>Document Root: <?= htmlspecialchars($documentRoot) ?></p>
        <p>Script Filename: <?= htmlspecialchars($scriptFilename) ?></p>
        <p>Remote Address: <?= htmlspecialchars($remoteAddr) ?></p>
        
        <h2>Custom Environment Variables</h2>
        <p>TEST_ENV_VAR1: <?= htmlspecialchars($customVar1) ?></p>
        <p>TEST_ENV_VAR2: <?= htmlspecialchars($customVar2) ?></p>
        
        <h2>Frango Superglobals</h2>
        <p>$_PATH Available: <?= $hasPath ?></p>
        <p>$_PATH_SEGMENTS Available: <?= $hasPathSegments ?></p>
        <p>$_JSON Available: <?= $hasJson ?></p>
    </div>
</body>
</html> 