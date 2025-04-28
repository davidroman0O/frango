<?php
// Include utility functions
require_once __DIR__ . '/../../lib/utils.php';

// Include config
require_once __DIR__ . '/../../config/app.php';

// Prepare content
$title = 'About - Virtual FS Demo';
$header = 'About Virtual FS Demo';
$content = <<<HTML
<div>
    <h2>About This Demo</h2>
    <p>This demo showcases the power of Frango's virtual filesystem for PHP applications in Go.</p>
    
    <h3>Features Demonstrated:</h3>
    <ul>
        <li><strong>VirtualFS</strong> - Combining source files, embedded files, and dynamically created files</li>
        <li><strong>File Operations</strong> - Moving, copying, and manipulating files in the virtual filesystem</li>
        <li><strong>Conventional Router</strong> - Mapping routes based on filesystem conventions</li>
        <li><strong>Mixed Endpoints</strong> - Combining PHP and Go handlers in a single router</li>
        <li><strong>File Watching</strong> - Automatic reloading when source files change</li>
    </ul>
    
    <h3>App Info:</h3>
    <pre><?php print_r(get_app_info()); ?></pre>
    
    <p><a href="/">Back to Home</a></p>
</div>
HTML;

// Render using template
include __DIR__ . '/../../templates/layout.php';
?> 