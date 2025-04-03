<?php
// Include utility functions
require_once __DIR__ . '/../../lib/utils.php';

// Include config
require_once __DIR__ . '/../../config/app.php';

// Prepare content
$title = 'Virtual FS Home';
$header = 'Virtual FS Demo';
$content = <<<HTML
<div>
    <h2>Welcome to Virtual FS Demo</h2>
    <p>This is a demonstration of the Frango virtual filesystem capabilities.</p>
    <p>This page is loaded from a physical PHP file, but it's using:</p>
    <ul>
        <li><strong>Embedded utility:</strong> <code>{$_SERVER['DOCUMENT_ROOT']}/lib/utils.php</code></li>
        <li><strong>Virtual config:</strong> <code>{$_SERVER['DOCUMENT_ROOT']}/config/app.php</code></li>
        <li><strong>Embedded template:</strong> <code>{$_SERVER['DOCUMENT_ROOT']}/templates/layout.php</code></li>
    </ul>
    <p>Message: <strong><?= format_message('Hello from Virtual FS!') ?></strong></p>
    
    <h3>Available Routes:</h3>
    <ul>
        <li><a href="/">/</a> - This page</li>
        <li><a href="/pages/about">/pages/about</a> - About page</li>
        <li><a href="/api/users">/api/users</a> - API endpoint (GET)</li>
        <li><a href="/api/users/create">/api/users/create</a> - API endpoint (POST)</li>
        <li><a href="/status">/status</a> - Go endpoint</li>
        <li><a href="/admin/dashboard">/admin/dashboard</a> - Go admin endpoint</li>
    </ul>
</div>
HTML;

// Render using template
include __DIR__ . '/../../templates/layout.php';
?> 