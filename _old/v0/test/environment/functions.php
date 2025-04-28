<?php
// Functions file for require test

// Define a constant
define('FUNCTION_FILE_CONSTANT', 'Constant from functions.php');

// Define a simple function
function addNumbers($a, $b) {
    return $a + $b;
}

// This file doesn't output any HTML, it just defines functions and constants
?> 