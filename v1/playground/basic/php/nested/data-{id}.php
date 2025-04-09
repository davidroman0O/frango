<?php
// Set content type to JSON for debugging
header('Content-Type: application/json');

// Try to get ID from path parameters
$id = $_PATH['id'];

// // If ID is not in $_PATH, try to extract from the path segments
if ($id === null && isset($_PATH_SEGMENTS) && count($_PATH_SEGMENTS) >= 2) {
    // For a URL like /nested/123, the ID would be in segment 1
    $id = $_PATH_SEGMENTS[1];
}

// Create a debug response
$response = [
    'id_value' => $id ?? 'NOT FOUND',
    'all_path_params' => $_PATH ?? [],
    'server_vars' => array_filter($_SERVER, function($key) {
        return strpos($key, 'PHP_PATH') === 0;
    }, ARRAY_FILTER_USE_KEY),
    'path_segments' => $_PATH_SEGMENTS ?? []
];

// Output as JSON
echo json_encode($response, JSON_PRETTY_PRINT);

// For backward compatibility with original code
echo "\nNested " . ($id ?? 'NOT FOUND');
?>