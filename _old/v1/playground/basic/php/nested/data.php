<?php
// Set content type to JSON for debugging
header('Content-Type: application/json');

// Try to get ID from path parameters
$id = $_PATH['id'];

// For backward compatibility with original code
echo "\nNested " . ($id ?? 'NOT FOUND');
?>