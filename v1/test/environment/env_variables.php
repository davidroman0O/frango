<?php
// Environment variable access test
header('Content-Type: text/html; charset=UTF-8');

// Get some environment variables
$phpVersion = phpversion();
$serverSoftware = $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown';
$serverProtocol = $_SERVER['SERVER_PROTOCOL'] ?? 'Unknown';
$requestMethod = $_SERVER['REQUEST_METHOD'] ?? 'Unknown';
$documentRoot = $_SERVER['DOCUMENT_ROOT'] ?? 'Unknown';
$scriptFilename = $_SERVER['SCRIPT_FILENAME'] ?? 'Unknown';

// Get custom environment variables from system
$customVar1 = getenv('TEST_ENV_VAR1') ?: 'Not set';
$customVar2 = getenv('TEST_ENV_VAR2') ?: 'Not set';

// For test output only - no assertions on these
$hasPath = isset($_PATH) ? 'Yes' : 'No';
$hasPathSegments = isset($_PATH_SEGMENTS) ? 'Yes' : 'No';
$hasJSON = isset($_JSON) ? 'Yes' : 'No';

// Output formatted HTML
echo "<!DOCTYPE html>
<html>
<head>
    <title>PHP Environment Variables Test</title>
</head>
<body>
    <h1>PHP Environment Variables Test</h1>
    
    <div id='results'>
        <h2>PHP Information</h2>
        <p>PHP Version: " . htmlspecialchars($phpVersion) . "</p>
        
        <h2>Standard Server Variables</h2>
        <p>Server Software: " . htmlspecialchars($serverSoftware) . "</p>
        <p>Server Protocol: " . htmlspecialchars($serverProtocol) . "</p>
        <p>Request Method: " . htmlspecialchars($requestMethod) . "</p>
        <p>Document Root: " . htmlspecialchars($documentRoot) . "</p>
        <p>Script Filename: " . htmlspecialchars($scriptFilename) . "</p>
        
        <h2>Custom Environment Variables</h2>
        <p>TEST_ENV_VAR1: " . htmlspecialchars($customVar1) . "</p>
        <p>TEST_ENV_VAR2: " . htmlspecialchars($customVar2) . "</p>
    </div>
</body>
</html>";
