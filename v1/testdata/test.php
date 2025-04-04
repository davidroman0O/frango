<?php
// Test PHP file for frango testing
echo "This is a test PHP file for frango";

// Display any path parameters that might be set
if (isset($_PATH) && count($_PATH) > 0) {
    echo "\nPath parameters:\n";
    foreach ($_PATH as $key => $value) {
        echo "$key: $value\n";
    }
}

// Display path segments if available
if (isset($_PATH_SEGMENTS) && count($_PATH_SEGMENTS) > 0) {
    echo "\nPath segments:\n";
    foreach ($_PATH_SEGMENTS as $index => $segment) {
        echo "[$index]: $segment\n";
    }
}

// Display JSON data if available
if (isset($_JSON) && count($_JSON) > 0) {
    echo "\nJSON data:\n";
    foreach ($_JSON as $key => $value) {
        if (is_string($value) || is_numeric($value) || is_bool($value)) {
            echo "$key: $value\n";
        } else {
            echo "$key: " . json_encode($value) . "\n";
        }
    }
}
?>