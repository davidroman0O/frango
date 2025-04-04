<?php
	// Set content type with charset to avoid potential encoding issues
	header('Content-Type: text/plain; charset=UTF-8');
	
	// Add error reporting
	error_reporting(E_ALL);
	ini_set('display_errors', 1);
	
	// Print basic info for diagnostic purposes
	echo "DIAGNOSTIC TEST\n";
	echo "PHP Version: " . phpversion() . "\n";
	echo "Request Time: " . date('Y-m-d H:i:s') . "\n";
	
	// Print all server variables to diagnose environment
	echo "\nSERVER VARIABLES:\n";
	foreach($_SERVER as $key => $value) {
		if (is_string($value)) {
			echo "$key: $value\n";
		}
	}
	
	// Send a response size large enough to flush output buffers
	echo str_repeat("*", 1024) . "\n";
	
	// Try to force flush any output buffers
	if (function_exists('ob_flush')) {
		ob_flush();
	}
	if (function_exists('flush')) {
		flush();
	}
	
	// Final message to confirm script completed
	echo "END OF DIAGNOSTIC TEST\n";
	?>