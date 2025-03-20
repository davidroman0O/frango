<?php
// test.php - Used to test require_once functionality

// Define a global variable to verify inclusion
$INCLUDED_TEST_FILE = true;

// Define a function to demonstrate include works correctly
function getTestMessage() {
    return "This function is from test.php - require_once works correctly!";
}

// Set a marker in the global scope
global $testInclusionMarker;
$testInclusionMarker = "TEST_INCLUSION_MARKER_" . mt_rand(1000, 9999);

// Write to a log file to verify this was executed
$logMessage = "Test file included at " . date('Y-m-d H:i:s') . 
             " - Marker: " . $testInclusionMarker . 
             " - Script: " . $_SERVER['SCRIPT_FILENAME'] . 
             " - Doc Root: " . $_SERVER['DOCUMENT_ROOT'];

// Return a value that can be used in the including file
return [
    'timestamp' => time(),
    'message' => 'Successfully included test.php',
    'marker' => $testInclusionMarker
];
?>
