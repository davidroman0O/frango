<?php
/**
 * Utility functions (embedded file)
 */

function format_message($message) {
    return "⚡ " . $message . " ⚡";
}

function get_app_info() {
    global $config;
    return [
        'name' => isset($config['app_name']) ? $config['app_name'] : 'Frango App',
        'version' => isset($config['version']) ? $config['version'] : '1.0.0',
        'debug' => isset($config['debug']) ? $config['debug'] : false,
    ];
}

/**
 * Start the page with a header
 */
function page_header($title = 'Virtual FS Demo', $header = null) {
    if ($header === null) {
        $header = $title;
    }
    
    // Start output buffering to capture content for the layout
    ob_start();
    
    // Set variables for the layout
    $GLOBALS['page_title'] = $title;
    $GLOBALS['page_header'] = $header;
}

/**
 * End the page with a footer
 */
function page_footer() {
    // Get the buffered content
    $content = ob_get_clean();
    
    // Extract variables to make them available in the layout
    $title = $GLOBALS['page_title'] ?? 'Virtual FS Demo';
    $header = $GLOBALS['page_header'] ?? $title;
    
    // Include the layout
    include($_SERVER['DOCUMENT_ROOT'] . '/templates/layout.php');
}
?> 