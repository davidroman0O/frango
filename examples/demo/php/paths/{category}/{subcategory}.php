<?php 

// Use script_path helper function to properly resolve the debug panel path
// This ensures it works regardless of the current directory or path parameters
$debug_panel_path = __DIR__ . '/../../debug.php';
if (function_exists('script_path')) {
    $debug_panel_path = script_path('../../debug.php');
}


var_dump($_SERVER['_PATH']);


?>



<!DOCTYPE html>
<html>
<head>
    <title>Category: <?= $_PATH['category'] ?? 'All' ?> > <?= $_PATH['subcategory'] ?? 'All' ?></title>
</head>
<body>
<?php
// Include the debug panel using the resolved path 
if (file_exists($debug_panel_path)) {
    include_once($debug_panel_path);
} else {
    echo "<div style='color:red'>Debug panel not found at: $debug_panel_path</div>";
}
?>
</body>
</html> 