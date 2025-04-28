<?php
/**
 * Frango v2 Demo - Debug Panel
 *
 * Include this in any page to see the current environment state.
 */

// Function to safely display variables, handling non-existent keys
function debug_var_export($var) {
    if ($var === null || $var === []) {
        return '<em>(empty or not set)</em>';
    }
    return '<pre>' . htmlspecialchars(var_export($var, true)) . '</pre>';
}

// Ensure globals provided by Frango are initialized (even if empty) for display
if (!isset($_GET)) $_GET = [];
if (!isset($_POST)) $_POST = [];
if (!isset($_FILES)) $_FILES = [];
if (!isset($_COOKIE)) $_COOKIE = [];
if (!isset($_REQUEST)) $_REQUEST = [];
if (!isset($_SERVER)) $_SERVER = [];
if (!isset($_ENV)) $_ENV = [];
if (!isset($GLOBALS['_PATH'])) $GLOBALS['_PATH'] = []; // Path params extracted by Frango
if (!isset($GLOBALS['_PATH_SEGMENTS'])) $GLOBALS['_PATH_SEGMENTS'] = []; // Path segments extracted by Frango
if (!isset($GLOBALS['_JSON'])) $GLOBALS['_JSON'] = null; // JSON body parsed by Frango
if (!isset($GLOBALS['_FORM'])) $GLOBALS['_FORM'] = []; // Form data from Frango (should match $_POST)
if (!isset($GLOBALS['_QUERY'])) $GLOBALS['_QUERY'] = []; // Query data from Frango (should match $_GET)
if (!isset($GLOBALS['_TEMPLATE'])) $GLOBALS['_TEMPLATE'] = []; // Data from Go via Render()

// Filter $_SERVER for display (optional: exclude sensitive info)
$display_server = array_filter($_SERVER, function($key) {
    // Add any keys you want to explicitly hide from the debug panel
    // return !in_array($key, ['DATABASE_PASSWORD', 'SECRET_KEY']);
    return true;
}, ARRAY_FILTER_USE_KEY);
ksort($display_server);

?>
<style>
.frango-debug-panel {
    position: fixed;
    top: 0;
    right: 0;
    width: 350px;
    max-height: 100vh;
    overflow-y: auto;
    background: #fff;
    border-left: 1px solid #ccc;
    box-shadow: -2px 0 5px rgba(0,0,0,0.1);
    z-index: 9999;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    font-size: 12px;
    color: #333;
    opacity: 0.95;
    transition: opacity 0.2s, transform 0.3s ease-out;
    transform: translateX(100%); /* Start hidden */
}
.frango-debug-panel.visible {
    transform: translateX(0);
}
.frango-debug-panel:hover {
    opacity: 1;
}
.frango-debug-header {
    padding: 8px 12px;
    background: #343a40;
    color: white;
    font-weight: bold;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    position: sticky;
    top: 0;
    z-index: 10;
}
.frango-debug-body {
    padding: 0;
}
.frango-debug-section details {
    margin-bottom: 0;
    border-bottom: 1px solid #eee;
}
.frango-debug-section summary {
    padding: 6px 12px;
    background: #f8f9fa;
    font-weight: bold;
    cursor: pointer;
    font-size: 11px;
    outline: none;
    list-style: none; /* Remove default arrow in some browsers */
}
.frango-debug-section summary::-webkit-details-marker { display: none; }
.frango-debug-section summary::before {
    content: '\25B6'; /* Right-pointing triangle */
    display: inline-block;
    margin-right: 5px;
    transition: transform 0.2s;
    font-size: 0.8em;
}
.frango-debug-section details[open] summary::before {
    transform: rotate(90deg);
}
.frango-debug-section-content {
    padding: 8px 12px;
    font-size: 11px;
    background: #fff;
    max-height: 300px;
    overflow-y: auto;
    border-top: 1px solid #eee;
}
.frango-debug-section-content pre {
    background: #f1f3f5;
    padding: 5px;
    border-radius: 3px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-all;
    margin: 5px 0;
    font-size: 10px;
}
.frango-debug-section-content em {
    color: #888;
}
.frango-debug-toggle-btn {
    position: fixed;
    top: 10px;
    right: 10px;
    background: #007bff;
    color: white;
    border: none;
    padding: 5px 10px;
    border-radius: 4px;
    cursor: pointer;
    z-index: 10000; /* Above panel */
    font-size: 12px;
}
</style>

<button class="frango-debug-toggle-btn" onclick="toggleFrangoDebugPanel()">Debug</button>

<div class="frango-debug-panel" id="frango-debug-panel">
    <div class="frango-debug-header" onclick="toggleFrangoDebugPanel()">
        Frango v2 Debug
        <span>&times;</span>
    </div>
    <div class="frango-debug-body">

        <!-- Standard Superglobals -->
        <div class="frango-debug-section">
            <details>
                <summary>Standard Globals</summary>
                <div class="frango-debug-section-content">
                    <strong>$_GET:</strong> <?= debug_var_export($_GET) ?>
                    <strong>$_POST:</strong> <?= debug_var_export($_POST) ?>
                    <strong>$_FILES:</strong> <?= debug_var_export($_FILES) ?>
                    <strong>$_COOKIE:</strong> <?= debug_var_export($_COOKIE) ?>
                    <strong>$_REQUEST:</strong> <?= debug_var_export($_REQUEST) ?>
                    <strong>$_ENV:</strong> <?= debug_var_export($_ENV) ?>
                </div>
            </details>
        </div>

        <!-- Frango Specific Globals -->
        <div class="frango-debug-section">
            <details>
                <summary>Frango Globals</summary>
                <div class="frango-debug-section-content">
                    <strong>$GLOBALS['_PATH']:</strong> <?= debug_var_export($GLOBALS['_PATH']) ?>
                    <strong>$GLOBALS['_PATH_SEGMENTS']:</strong> <?= debug_var_export($GLOBALS['_PATH_SEGMENTS']) ?>
                    <strong>$GLOBALS['_JSON']:</strong> <?= debug_var_export($GLOBALS['_JSON']) ?>
                    <strong>$GLOBALS['_FORM']:</strong> <?= debug_var_export($GLOBALS['_FORM']) ?>
                    <strong>$GLOBALS['_QUERY']:</strong> <?= debug_var_export($GLOBALS['_QUERY']) ?>
                    <strong>$GLOBALS['_TEMPLATE']:</strong> <?= debug_var_export($GLOBALS['_TEMPLATE']) ?>
                </div>
            </details>
        </div>

        <!-- Key Server Variables -->
        <div class="frango-debug-section">
            <details>
                <summary>$_SERVER</summary>
                <div class="frango-debug-section-content">
                     <?= debug_var_export($display_server) ?>
                </div>
            </details>
        </div>

    </div>
</div>

<script>
let isPanelVisible = false;
function toggleFrangoDebugPanel() {
    const panel = document.getElementById('frango-debug-panel');
    isPanelVisible = !isPanelVisible;
    if (isPanelVisible) {
        panel.classList.add('visible');
    } else {
        panel.classList.remove('visible');
    }
}
// Optional: Close panel if clicking outside
document.addEventListener('click', function(event) {
    const panel = document.getElementById('frango-debug-panel');
    const toggleButton = document.querySelector('.frango-debug-toggle-btn');
    if (isPanelVisible && !panel.contains(event.target) && !toggleButton.contains(event.target)) {
        toggleFrangoDebugPanel();
    }
});
</script> 