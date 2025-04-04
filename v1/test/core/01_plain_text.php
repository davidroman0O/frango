<?php
// Set the content type
header('Content-Type: text/plain');

// Add error reporting
error_reporting(E_ALL);
ini_set('display_errors', 1);

// Output plain text
echo "Hello from PHP!\n";
echo "Running PHP version: " . phpversion() . "\n";
echo "Request time: " . date('Y-m-d H:i:s') . "\n";
?> 