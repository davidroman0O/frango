<?php
// Main PHP script that includes other files
header('Content-Type: text/html; charset=UTF-8');

// Simple variable to check scope
$mainVar = "Main script variable";

// Include the header file - use relative path from this script
include __DIR__ . '/header.php';

// Include the functions file - use relative path from this script
require __DIR__ . '/functions.php';

// Show the page content
?>
<div id="content">
    <h2>Include/Require Test</h2>
    <p>This page demonstrates including and requiring other PHP files.</p>
    
    <div class="result">
        <h3>Test Results:</h3>
        <ul>
            <li>Main variable: <?= $mainVar ?></li>
            <li>Header variable: <?= $headerVar ?></li>
            <li>Function result: <?= addNumbers(5, 3) ?></li>
            <li>Constant from required file: <?= FUNCTION_FILE_CONSTANT ?></li>
        </ul>
    </div>
</div>

<?php
// Include the footer - use relative path from this script
include __DIR__ . '/footer.php';
?> 