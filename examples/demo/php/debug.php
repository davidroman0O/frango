<?php
/**
 * Frango Debug Panel
 * 
 * This file provides a reusable debug panel that shows PHP environment variables
 * Include this in any page to see the current environment state
 */

// Start session right at the beginning before any output
if (session_status() == PHP_SESSION_NONE) {
    // Only start session if it wasn't already started
    @session_start(); // Using @ to suppress any errors if session can't be started
}

// Process special debug actions
if (isset($_GET['__debug_clear_session']) && session_status() === PHP_SESSION_ACTIVE) {
    // Clear session data
    $_SESSION = array();
    
    // Delete the session cookie
    if (ini_get("session.use_cookies")) {
        $params = session_get_cookie_params();
        setcookie(session_name(), '', time() - 42000,
            $params["path"], $params["domain"],
            $params["secure"], $params["httponly"]
        );
    }
    
    // Finally, destroy the session
    session_destroy();
    
    // Redirect to remove the query parameter
    header('Location: ' . strtok($_SERVER['REQUEST_URI'], '?'));
    exit;
}

// Start timer for execution time measurement
$debugPanelStartTime = microtime(true);
$debugPanelStartMemory = memory_get_usage();

// Initialize superglobals if they don't exist
if (!isset($_PATH)) $_PATH = [];
if (!isset($_PATH_SEGMENTS)) $_PATH_SEGMENTS = [];
if (!isset($_PATH_SEGMENT_COUNT)) $_PATH_SEGMENT_COUNT = 0;
if (!isset($_JSON)) $_JSON = [];
if (!isset($_FORM)) $_FORM = [];
if (!isset($_URL)) $_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_CURRENT_URL)) $_CURRENT_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_QUERY)) $_QUERY = isset($_GET) ? $_GET : [];
if (!isset($_FILES_INFO)) $_FILES_INFO = isset($_FILES) ? $_FILES : [];

// Define helper functions if they don't exist
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

if (!function_exists('path_param')) {
    function path_param($name, $default = null) {
        global $_PATH;
        return isset($_PATH[$name]) ? $_PATH[$name] : $default;
    }
}

if (!function_exists('has_path_param')) {
    function has_path_param($name) {
        global $_PATH;
        return isset($_PATH[$name]);
    }
}

// Extract HTTP scheme
$httpScheme = isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? 'https' : 'http';

// Format value for display (handles arrays and objects)
function formatValue($value) {
    if (is_array($value)) {
        if (count($value) > 10) {
            // Show just the first 10 elements for large arrays
            $subset = array_slice($value, 0, 10);
            $formatted = htmlspecialchars(print_r($subset, true)) . '... (' . count($value) . ' items total)';
        } else {
            $formatted = htmlspecialchars(print_r($value, true));
        }
        return '<pre class="debug-array">' . $formatted . '</pre>';
    } else if (is_object($value)) {
        return '<pre class="debug-object">' . htmlspecialchars(print_r($value, true)) . '</pre>';
    } else if (is_bool($value)) {
        return $value ? '<span class="debug-bool-true">true</span>' : '<span class="debug-bool-false">false</span>';
    } else if (is_null($value)) {
        return '<span class="debug-null">null</span>';
    } else if (is_string($value) && (
        str_starts_with($value, '{') || 
        str_starts_with($value, '[')
    )) {
        // Check if the string looks like JSON
        $decoded = json_decode($value, true);
        if (json_last_error() === JSON_ERROR_NONE) {
            // It's valid JSON, display it as pre-formatted
            return '<pre class="debug-json">' . htmlspecialchars($value) . '</pre>';
        }
    }
    
    return htmlspecialchars((string)$value);
}

// Get current route information
function getRouteInfo() {
    global $_PATH_SEGMENTS, $_SERVER, $_PATH;
    
    return [
        'method' => $_SERVER['REQUEST_METHOD'] ?? 'GET',
        'path' => $_SERVER['REQUEST_URI'] ?? '/',
        'segments' => $_PATH_SEGMENTS,
        'params' => $_PATH
    ];
}

// Function to get VFS information if available
function getVFSInfo() {
    $vfsInfo = [
        'available' => false,
        'paths' => [],
        'globals_file' => ''
    ];
    
    if (isset($_SERVER['FRANGO_VFS_NAME'])) {
        $vfsInfo['available'] = true;
        $vfsInfo['name'] = $_SERVER['FRANGO_VFS_NAME'] ?? 'unknown';
        $vfsInfo['temp_dir'] = $_SERVER['FRANGO_VFS_TEMP_DIR'] ?? 'unknown';
        $vfsInfo['globals_file'] = $_SERVER['FRANGO_VFS_GLOBALS_PATH'] ?? '/_frango_php_globals.php';
    }
    
    return $vfsInfo;
}

// Get information about session errors
function getSessionErrors() {
    $errors = [];
    
    if (headers_sent($filename, $linenum)) {
        $errors[] = "Headers already sent in $filename on line $linenum. Session must be started before any output.";
    }
    
    if (!extension_loaded('session')) {
        $errors[] = "PHP session extension is not loaded.";
    }
    
    // Check if session directory is writable
    if (session_save_path() !== '') {
        $save_path = session_save_path();
        if (!is_writable($save_path)) {
            $errors[] = "Session save path ($save_path) is not writable.";
        }
    } else {
        // If session.save_path is not set, PHP will use the system's temporary directory
        $system_tmp = sys_get_temp_dir();
        if (!is_writable($system_tmp)) {
            $errors[] = "System temporary directory ($system_tmp) is not writable.";
        }
    }
    
    return $errors;
}

// Function to add test data to session (for debugging)
function addSessionTestData() {
    if (session_status() === PHP_SESSION_ACTIVE) {
        $_SESSION['test_string'] = 'This is a test string';
        $_SESSION['test_array'] = ['item1', 'item2', 'item3'];
        $_SESSION['test_object'] = (object)['name' => 'Test Object', 'value' => 42];
        $_SESSION['test_timestamp'] = time();
        return true;
    }
    return false;
}
?>
<style>
.debug-panel {
    position: fixed;
    top: 10px;
    right: 10px;
    width: 350px;
    max-height: 95vh;
    overflow-y: auto;
    background: #fff;
    border-radius: 6px;
    box-shadow: 0 3px 15px rgba(0,0,0,0.25);
    z-index: 9999;
    font-family: system-ui, -apple-system, sans-serif;
    font-size: 12px;
    color: #333;
    opacity: 0.95;
    transition: opacity 0.3s, box-shadow 0.3s;
}
.debug-panel:hover {
    opacity: 1;
    box-shadow: 0 5px 20px rgba(0,0,0,0.3);
}
.debug-panel-header {
    padding: 10px 14px;
    background: #2c3e50;
    color: white;
    font-weight: bold;
    border-top-left-radius: 6px;
    border-top-right-radius: 6px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.debug-panel-body {
    padding: 0;
    max-height: 85vh;
    overflow-y: auto;
    border-left: 1px solid #eee;
    border-right: 1px solid #eee;
    border-bottom: 1px solid #eee;
    border-bottom-left-radius: 6px;
    border-bottom-right-radius: 6px;
}
.debug-panel-section {
    margin-bottom: 1px;
}
.debug-section-header {
    padding: 8px 14px;
    background: #f5f5f5;
    font-weight: bold;
    cursor: pointer;
    border-bottom: 1px solid #ddd;
    font-size: 11px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: background-color 0.2s;
}
.debug-section-header:hover {
    background: #e8e8e8;
}
.debug-section-content {
    padding: 6px 12px;
    display: none;
    font-size: 11px;
    background: #f9f9f9;
    max-height: 300px;
    overflow-y: auto;
    border-bottom: 1px solid #eee;
}
.env-var {
    padding: 4px 0;
    border-bottom: 1px solid #eee;
}
.env-var-name {
    font-weight: bold;
    color: #2980b9;
    display: inline-block;
    margin-bottom: 3px;
}
.env-value {
    color: #333;
    word-break: break-all;
}
.debug-path-var {
    margin: 3px 0;
    padding: 4px;
    background: #eaf2f8;
    border-radius: 3px;
    border-left: 3px solid #3498db;
}
.path-value {
    font-weight: bold;
}
.debug-container {
    margin-top: 4px;
}
.debug-bool-true {
    color: #27ae60;
    font-weight: bold;
}
.debug-bool-false {
    color: #e74c3c;
    font-weight: bold;
}
.debug-null {
    color: #7f8c8d;
    font-style: italic;
}
.debug-array, .debug-object, .debug-json {
    margin: 5px 0;
    padding: 5px;
    background: #f0f0f0;
    border-radius: 3px;
    border-left: 3px solid #9b59b6;
    font-family: monospace;
    white-space: pre-wrap;
    font-size: 11px;
    overflow: auto;
    max-height: 150px;
}
.debug-json {
    border-left-color: #f39c12;
}
.debug-metrics {
    display: flex;
    justify-content: space-between;
    padding: 5px 8px;
    background: #ecf0f1;
    font-size: 10px;
    border-top: 1px solid #ddd;
}
.debug-metric {
    display: inline-block;
}
.debug-badge {
    display: inline-block;
    padding: 2px 5px;
    border-radius: 3px;
    font-size: 10px;
    margin-left: 5px;
    font-weight: bold;
}
.badge-info {
    background: #3498db;
    color: white;
}
.badge-success {
    background: #2ecc71;
    color: white;
}
.badge-warning {
    background: #f39c12;
    color: white;
}
.badge-error {
    background: #e74c3c;
    color: white;
}
.badge-default {
    background: #95a5a6;
    color: white;
}
.btn-debug {
    padding: 2px 6px;
    border-radius: 3px;
    border: none;
    cursor: pointer;
    font-size: 10px;
    margin-left: 5px;
    background: #3498db;
    color: white;
}
.btn-debug:hover {
    background: #2980b9;
}
.debug-arrow {
    font-size: 11px;
    margin-left: 4px;
    transition: transform 0.2s;
}
.debug-arrow-down {
    transform: rotate(90deg);
}
.debug-status {
    padding: 10px;
    margin: 5px 0;
    border-radius: 3px;
}
.debug-status-ok {
    background: #d5f5e3;
    border-left: 3px solid #2ecc71;
}
.debug-status-warning {
    background: #fef9e7;
    border-left: 3px solid #f1c40f;
}
.debug-status-error {
    background: #fdedec;
    border-left: 3px solid #e74c3c;
}
</style>

<div class="debug-panel">
    <div class="debug-panel-header" onclick="toggleDebugPanel()">
        Frango Debug Panel
        <span id="debug-panel-toggle">[-]</span>
    </div>
    <div class="debug-panel-body" id="debug-panel-body">
        <!-- Request Summary -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('request-summary')">
                Request Summary <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="request-summary">
                <div class="debug-path-var">
                    <span class="env-var-name">Method:</span>
                    <span class="path-value"><?= htmlspecialchars($_SERVER['REQUEST_METHOD'] ?? 'GET') ?></span>
                </div>
                <div class="debug-path-var">
                    <span class="env-var-name">URL:</span>
                    <span class="path-value"><?= htmlspecialchars($_SERVER['REQUEST_URI'] ?? '/') ?></span>
                </div>
                <div class="debug-path-var">
                    <span class="env-var-name">Script:</span>
                    <span class="path-value"><?= htmlspecialchars($_SERVER['SCRIPT_FILENAME'] ?? 'Unknown') ?></span>
                </div>
                <div class="debug-path-var">
                    <span class="env-var-name">Client IP:</span>
                    <span class="path-value"><?= htmlspecialchars($_SERVER['REMOTE_ADDR'] ?? 'Unknown') ?></span>
                </div>
                <?php if (!empty($_SERVER['HTTP_USER_AGENT'])): ?>
                <div class="debug-path-var">
                    <span class="env-var-name">User Agent:</span>
                    <span class="path-value"><?= htmlspecialchars($_SERVER['HTTP_USER_AGENT']) ?></span>
                </div>
                <?php endif; ?>
            </div>
        </div>

        <!-- Path Parameters -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('path-params')">
                Path Parameters <span class="badge-info debug-badge"><?= count($_PATH) ?></span>
                <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="path-params">
                <?php if (!empty($_PATH)): ?>
                    <?php foreach ($_PATH as $key => $value): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= formatValue($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                <?php else: ?>
                    <div class="debug-path-var">No path parameters</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Path Segments -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('path-segments')">
                Path Segments <span class="badge-info debug-badge"><?= count($_PATH_SEGMENTS) ?></span>
                <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="path-segments">
                <?php if (!empty($_PATH_SEGMENTS)): ?>
                    <?php foreach ($_PATH_SEGMENTS as $index => $segment): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">[<?= $index ?>]:</span>
                        <span class="path-value"><?= formatValue($segment) ?></span>
                    </div>
                    <?php endforeach; ?>
                <?php else: ?>
                    <div class="debug-path-var">No path segments</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Query Data -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('query-data')">
                Query Parameters <span class="badge-info debug-badge"><?= count($_GET) ?></span>
                <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="query-data">
                <?php if (!empty($_GET)): ?>
                    <?php foreach ($_GET as $key => $value): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= formatValue($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                <?php else: ?>
                    <div class="debug-path-var">No query parameters</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Form Data -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('form-data')">
                Form Data <span class="badge-info debug-badge"><?= count($_POST) + count($_FORM ?? []) ?></span>
                <?= empty($_POST) && !empty($_SERVER['CONTENT_TYPE']) && strpos($_SERVER['CONTENT_TYPE'], 'form') !== false ? '<span class="badge-warning debug-badge">⚠️</span>' : '' ?>
                <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="form-data">
                <?php if (!empty($_POST) || !empty($_FORM ?? [])): ?>
                    <?php 
                    $formData = array_merge($_POST, (array)($_FORM ?? []));
                    foreach ($formData as $key => $value): 
                    ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= formatValue($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                <?php else: ?>
                    <?php
                    // Check for PHP_FORM_ variables that aren't in $_POST
                    $missingFormVars = [];
                    foreach ($_SERVER as $key => $value) {
                        if (strpos($key, 'PHP_FORM_') === 0) {
                            $paramName = substr($key, 10);
                            $missingFormVars[$paramName] = $value;
                        }
                    }
                    if (!empty($missingFormVars)):
                    ?>
                    <div class="debug-status debug-status-warning">
                        <strong>Warning:</strong> PHP_FORM_* variables exist but $_POST is empty. Form fix not applied.
                    </div>
                    <?php foreach ($missingFormVars as $key => $value): ?>
                    <div class="debug-path-var" style="border-left-color: #e74c3c;">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= formatValue($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                    <?php else: ?>
                    <div class="debug-path-var">No form data</div>
                    <?php endif; ?>
                <?php endif; ?>
            </div>
        </div>

        <!-- JSON Data -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('json-data')">
                JSON Data <span class="badge-info debug-badge"><?= is_array($_JSON) ? count($_JSON) : 0 ?></span>
                <?= !empty($_SERVER['CONTENT_TYPE']) && strpos($_SERVER['CONTENT_TYPE'], 'json') !== false && empty($_JSON) ? '<span class="badge-warning debug-badge">⚠️</span>' : '' ?>
                <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="json-data">
                <?php if (!empty($_JSON) && is_array($_JSON)): ?>
                    <?php foreach ($_JSON as $key => $value): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= formatValue($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                <?php else: ?>
                    <?php if (!empty($_SERVER['CONTENT_TYPE']) && strpos($_SERVER['CONTENT_TYPE'], 'json') !== false): ?>
                    <div class="debug-status debug-status-warning">
                        <strong>Warning:</strong> Content-Type is JSON but $_JSON is empty. JSON parsing might not be working.
                    </div>
                    <?php endif; ?>
                    <div class="debug-path-var">No JSON data</div>
                <?php endif; ?>
                
                <?php
                // Check for raw JSON input
                $rawInput = file_get_contents('php://input');
                if (!empty($rawInput) && (empty($_JSON) || !is_array($_JSON)) && json_decode($rawInput) !== null):
                ?>
                <div class="debug-status debug-status-info">
                    <strong>Raw JSON Input:</strong>
                    <pre><?= htmlspecialchars(substr($rawInput, 0, 500)) ?><?= strlen($rawInput) > 500 ? '...' : '' ?></pre>
                </div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- File Uploads -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('file-uploads')">
                File Uploads <span class="badge-info debug-badge"><?= count($_FILES) ?></span>
                <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="file-uploads">
                <?php if (!empty($_FILES)): ?>
                    <?php foreach ($_FILES as $key => $file): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <div class="debug-container">
                            <?php if (is_array($file['name'])): ?>
                                <!-- Multiple files with same input name -->
                                <?php for ($i = 0; $i < count($file['name']); $i++): ?>
                                    <?php if ($file['name'][$i] !== ''): ?>
                                    <div style="margin: 5px 0; padding-left: 5px; border-left: 2px solid #3498db;">
                                        <div><strong>Name:</strong> <?= htmlspecialchars($file['name'][$i]) ?></div>
                                        <div><strong>Type:</strong> <?= htmlspecialchars($file['type'][$i]) ?></div>
                                        <div><strong>Size:</strong> <?= number_format($file['size'][$i]) ?> bytes</div>
                                        <div><strong>Temp File:</strong> <?= htmlspecialchars($file['tmp_name'][$i]) ?></div>
                                        <div><strong>Error:</strong> <?= htmlspecialchars($file['error'][$i]) ?></div>
                                    </div>
                                    <?php endif; ?>
                                <?php endfor; ?>
                            <?php else: ?>
                                <!-- Single file -->
                                <?php if ($file['name'] !== ''): ?>
                                <div><strong>Name:</strong> <?= htmlspecialchars($file['name']) ?></div>
                                <div><strong>Type:</strong> <?= htmlspecialchars($file['type']) ?></div>
                                <div><strong>Size:</strong> <?= number_format($file['size']) ?> bytes</div>
                                <div><strong>Temp File:</strong> <?= htmlspecialchars($file['tmp_name']) ?></div>
                                <div><strong>Error:</strong> <?= htmlspecialchars($file['error']) ?></div>
                                <?php endif; ?>
                            <?php endif; ?>
                        </div>
                    </div>
                    <?php endforeach; ?>
                <?php else: ?>
                    <div class="debug-path-var">No file uploads</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- HTTP Headers -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('http-headers')">
                HTTP Headers <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="http-headers">
                <?php
                $headers = [];
                foreach ($_SERVER as $key => $value) {
                    if (substr($key, 0, 5) === 'HTTP_') {
                        $headerName = str_replace(' ', '-', ucwords(strtolower(str_replace('_', ' ', substr($key, 5)))));
                        $headers[$headerName] = $value;
                    } elseif (in_array($key, ['CONTENT_TYPE', 'CONTENT_LENGTH'])) {
                        $headerName = str_replace(' ', '-', ucwords(strtolower(str_replace('_', ' ', $key))));
                        $headers[$headerName] = $value;
                    }
                }
                ksort($headers);
                
                if (!empty($headers)):
                    foreach ($headers as $name => $value):
                ?>
                <div class="env-var">
                    <span class="env-var-name"><?= htmlspecialchars($name) ?>:</span><br>
                    <span class="env-value"><?= formatValue($value) ?></span>
                </div>
                <?php 
                    endforeach;
                else:
                ?>
                <div class="env-var">No HTTP headers found</div>
                <?php endif; ?>
            </div>
        </div>

        <!-- PHP Environment Variables -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('php-env')">
                PHP_ Variables <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="php-env">
                <?php
                $phpVars = array_filter($_SERVER, function($k) {
                    return strpos($k, 'PHP_') === 0;
                }, ARRAY_FILTER_USE_KEY);
                
                ksort($phpVars);
                
                if (!empty($phpVars)):
                    foreach ($phpVars as $key => $value):
                ?>
                <div class="env-var">
                    <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span><br>
                    <span class="env-value"><?= formatValue($value) ?></span>
                </div>
                <?php 
                    endforeach;
                else:
                ?>
                <div class="env-var">No PHP_ variables found</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- VFS Information -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('vfs-info')">
                VFS Information <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="vfs-info">
                <?php
                $vfsInfo = getVFSInfo();
                
                if ($vfsInfo['available']):
                ?>
                <div class="debug-status debug-status-ok">
                    <strong>VFS Configured</strong>
                </div>
                <div class="debug-path-var">
                    <span class="env-var-name">VFS Name:</span>
                    <span class="path-value"><?= htmlspecialchars($vfsInfo['name']) ?></span>
                </div>
                <div class="debug-path-var">
                    <span class="env-var-name">Temp Directory:</span>
                    <span class="path-value"><?= htmlspecialchars($vfsInfo['temp_dir']) ?></span>
                </div>
                <div class="debug-path-var">
                    <span class="env-var-name">PHP Globals File:</span>
                    <span class="path-value"><?= htmlspecialchars($vfsInfo['globals_file']) ?></span>
                </div>
                <?php else: ?>
                <div class="debug-status debug-status-warning">
                    <strong>VFS Not Configured</strong>
                    <div>The application does not appear to be using FrankenPHP VFS</div>
                </div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Session Data -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('session-data')">
                Session Data <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="session-data">
                <?php
                $sessionAvailable = (session_status() === PHP_SESSION_ACTIVE);
                $sessionId = session_id();
                $sessionErrors = getSessionErrors();
                
                // Handle test data request
                if (isset($_GET['__debug_add_session_test']) && $sessionAvailable) {
                    addSessionTestData();
                    // Redirect to remove the query parameter
                    header('Location: ' . strtok($_SERVER['REQUEST_URI'], '?'));
                    exit;
                }
                
                // Display session errors if any
                if (!empty($sessionErrors)):
                ?>
                <div class="debug-status debug-status-error">
                    <strong>Session Errors:</strong>
                    <ul style="margin: 5px 0; padding-left: 20px;">
                        <?php foreach($sessionErrors as $error): ?>
                        <li><?= htmlspecialchars($error) ?></li>
                        <?php endforeach; ?>
                    </ul>
                </div>
                <?php endif; ?>
                
                <?php if (!$sessionAvailable): ?>
                <div class="debug-status debug-status-warning">
                    <strong>Session Unavailable</strong>
                    <div>Sessions may be disabled or the session could not be started.</div>
                </div>
                <?php endif; ?>
                
                <div class="debug-path-var">
                    <span class="env-var-name">Session ID:</span>
                    <span class="path-value"><?= $sessionId ?: 'Not set' ?></span>
                </div>
                
                <div class="debug-path-var">
                    <span class="env-var-name">Session Status:</span>
                    <span class="path-value">
                        <?php 
                        switch (session_status()) {
                            case PHP_SESSION_DISABLED:
                                echo '<span class="debug-bool-false">Disabled</span> (Sessions are disabled on the server)';
                                break;
                            case PHP_SESSION_NONE:
                                echo '<span class="debug-bool-false">None</span> (No session started)';
                                break;
                            case PHP_SESSION_ACTIVE:
                                echo '<span class="debug-bool-true">Active</span> (Session has started)';
                                break;
                            default:
                                echo 'Unknown';
                        }
                        ?>
                    </span>
                </div>
                
                <?php if ($sessionAvailable && !empty($_SESSION)): ?>
                    <?php foreach ($_SESSION as $key => $value): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= formatValue($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                <?php elseif ($sessionAvailable): ?>
                    <div class="debug-path-var">Session is active but no data is stored</div>
                <?php endif; ?>
                
                <?php if ($sessionAvailable): ?>
                <div style="margin: 10px 0;">
                    <a href="?__debug_add_session_test=1" class="btn-debug" style="margin-left: 0; padding: 5px 10px;">
                        Add Test Session Data
                    </a>
                    <a href="?__debug_clear_session=1" class="btn-debug" style="background: #e74c3c; padding: 5px 10px;">
                        Clear Session
                    </a>
                </div>
                <?php endif; ?>
                
                <?php 
                // Show additional session information
                if ($sessionAvailable):
                    $sessionInfo = session_get_cookie_params();
                ?>
                <div class="debug-path-var">
                    <span class="env-var-name">Session Cookie Info:</span>
                    <div class="debug-container">
                        <div><strong>Lifetime:</strong> <?= $sessionInfo['lifetime'] ?> seconds</div>
                        <div><strong>Path:</strong> <?= htmlspecialchars($sessionInfo['path']) ?></div>
                        <div><strong>Domain:</strong> <?= htmlspecialchars($sessionInfo['domain'] ?: 'Not set') ?></div>
                        <div><strong>Secure:</strong> <?= $sessionInfo['secure'] ? 'Yes' : 'No' ?></div>
                        <div><strong>HttpOnly:</strong> <?= $sessionInfo['httponly'] ? 'Yes' : 'No' ?></div>
                        <?php if (isset($sessionInfo['samesite'])): ?>
                        <div><strong>SameSite:</strong> <?= htmlspecialchars($sessionInfo['samesite']) ?></div>
                        <?php endif; ?>
                    </div>
                </div>
                
                <div class="debug-path-var">
                    <span class="env-var-name">Session Configuration:</span>
                    <div class="debug-container">
                        <div><strong>Save Path:</strong> <?= htmlspecialchars(session_save_path() ?: 'Default system temp') ?></div>
                        <div><strong>Session Name:</strong> <?= htmlspecialchars(session_name()) ?></div>
                        <div><strong>Cache Limiter:</strong> <?= htmlspecialchars(session_cache_limiter()) ?></div>
                        <div><strong>Cache Expire:</strong> <?= session_cache_expire() ?> minutes</div>
                    </div>
                </div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Debug Environment Variables -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('debug-env')">
                DEBUG_ Variables <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="debug-env">
                <?php
                $debugVars = array_filter($_SERVER, function($k) {
                    return strpos($k, 'DEBUG_') === 0;
                }, ARRAY_FILTER_USE_KEY);
                
                ksort($debugVars);
                
                if (!empty($debugVars)):
                    foreach ($debugVars as $key => $value):
                ?>
                <div class="env-var">
                    <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span><br>
                    <span class="env-value"><?= formatValue($value) ?></span>
                </div>
                <?php 
                    endforeach;
                else:
                ?>
                <div class="env-var">No DEBUG_ variables found</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Server Variables -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('server-vars')">
                Server Variables <span class="debug-arrow">›</span>
            </div>
            <div class="debug-section-content" id="server-vars">
                <?php
                // Filter out variables already shown in other sections
                $serverVars = array_filter($_SERVER, function($k) {
                    return strpos($k, 'HTTP_') !== 0 
                        && strpos($k, 'PHP_') !== 0
                        && strpos($k, 'DEBUG_') !== 0
                        && !in_array($k, ['CONTENT_TYPE', 'CONTENT_LENGTH']);
                }, ARRAY_FILTER_USE_KEY);
                
                ksort($serverVars);
                
                if (!empty($serverVars)):
                    foreach ($serverVars as $key => $value):
                ?>
                <div class="env-var">
                    <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span><br>
                    <span class="env-value"><?= formatValue($value) ?></span>
                </div>
                <?php 
                    endforeach;
                else:
                ?>
                <div class="env-var">No server variables found</div>
                <?php endif; ?>
            </div>
        </div>
    </div>
    
    <!-- Metrics Bar -->
    <div class="debug-metrics">
        <div class="debug-metric">
            <strong>PHP:</strong> <?= phpversion() ?>
        </div>
        <div class="debug-metric">
            <strong>Memory:</strong> <?= number_format(memory_get_usage() / 1024 / 1024, 2) ?> MB
        </div>
        <div class="debug-metric">
            <strong>Time:</strong> <?= number_format((microtime(true) - $debugPanelStartTime) * 1000, 2) ?> ms
        </div>
    </div>
</div>

<script>
// Debug panel toggle
let panelVisible = true;
const sectionStates = JSON.parse(localStorage.getItem('frangoDebugPanelSections') || '{}');

function toggleDebugPanel() {
    const panel = document.getElementById('debug-panel-body');
    const toggle = document.getElementById('debug-panel-toggle');
    const metrics = document.querySelector('.debug-metrics');
    
    if (panelVisible) {
        panel.style.display = 'none';
        toggle.innerText = '[+]';
        if (metrics) metrics.style.display = 'none';
    } else {
        panel.style.display = 'block';
        toggle.innerText = '[-]';
        if (metrics) metrics.style.display = 'flex';
    }
    
    panelVisible = !panelVisible;
    localStorage.setItem('frangoDebugPanelVisible', panelVisible ? 'true' : 'false');
}

// Section toggle
function toggleSection(sectionId) {
    const section = document.getElementById(sectionId);
    const headerElement = section.previousElementSibling;
    const arrow = headerElement.querySelector('.debug-arrow');
    
    const isVisible = section.style.display === 'block';
    
    section.style.display = isVisible ? 'none' : 'block';
    
    if (arrow) {
        if (isVisible) {
            arrow.classList.remove('debug-arrow-down');
        } else {
            arrow.classList.add('debug-arrow-down');
        }
    }
    
    // Save section state to localStorage
    sectionStates[sectionId] = !isVisible;
    localStorage.setItem('frangoDebugPanelSections', JSON.stringify(sectionStates));
}

// Make debug panel draggable
let isDragging = false;
let dragOffsetX = 0;
let dragOffsetY = 0;

function initDragPanel() {
    const panel = document.querySelector('.debug-panel');
    const header = document.querySelector('.debug-panel-header');
    
    header.addEventListener('mousedown', function(e) {
        // Prevent dragging when clicking the toggle
        if (e.target.id === 'debug-panel-toggle') return;
        
        isDragging = true;
        dragOffsetX = e.clientX - panel.getBoundingClientRect().left;
        dragOffsetY = e.clientY - panel.getBoundingClientRect().top;
        
        // Prevent text selection during drag
        e.preventDefault();
    });
    
    document.addEventListener('mousemove', function(e) {
        if (!isDragging) return;
        
        const x = e.clientX - dragOffsetX;
        const y = e.clientY - dragOffsetY;
        
        // Keep panel within viewport
        const maxX = window.innerWidth - panel.offsetWidth;
        const maxY = window.innerHeight - panel.offsetHeight;
        
        const boundedX = Math.max(0, Math.min(x, maxX));
        const boundedY = Math.max(0, Math.min(y, maxY));
        
        panel.style.left = boundedX + 'px';
        panel.style.top = boundedY + 'px';
        panel.style.right = 'auto';
    });
    
    document.addEventListener('mouseup', function() {
        isDragging = false;
    });
}

// Initialize the debug panel
document.addEventListener('DOMContentLoaded', function() {
    // Restore panel visibility
    const savedVisibility = localStorage.getItem('frangoDebugPanelVisible');
    if (savedVisibility === 'false') {
        toggleDebugPanel();
    }
    
    // Show sections based on saved state (open 'request-summary' by default)
    const sectionsToOpen = Object.keys(sectionStates)
        .filter(sectionId => sectionStates[sectionId])
        .concat(['request-summary']);
    
    const uniqueSections = [...new Set(sectionsToOpen)];
    
    uniqueSections.forEach(sectionId => {
        const section = document.getElementById(sectionId);
        if (section) {
            section.style.display = 'block';
            const headerElement = section.previousElementSibling;
            const arrow = headerElement.querySelector('.debug-arrow');
            if (arrow) {
                arrow.classList.add('debug-arrow-down');
            }
        }
    });
    
    // Initialize draggable panel
    initDragPanel();
    
    // Add copy buttons to JSON/array values
    addCopyButtons();
});

// Add copy buttons to formatted values
function addCopyButtons() {
    document.querySelectorAll('.debug-array, .debug-object, .debug-json').forEach(el => {
        const btn = document.createElement('button');
        btn.className = 'btn-debug';
        btn.innerText = 'Copy';
        btn.style.position = 'absolute';
        btn.style.right = '5px';
        btn.style.top = '5px';
        
        // Create a wrapper to position the button properly
        const wrapper = document.createElement('div');
        wrapper.style.position = 'relative';
        el.parentNode.insertBefore(wrapper, el);
        wrapper.appendChild(el);
        wrapper.appendChild(btn);
        
        btn.addEventListener('click', function(e) {
            e.stopPropagation();
            
            const text = el.textContent;
            navigator.clipboard.writeText(text).then(
                function() {
                    btn.innerText = 'Copied!';
                    setTimeout(() => { btn.innerText = 'Copy'; }, 1500);
                },
                function() {
                    btn.innerText = 'Failed!';
                    setTimeout(() => { btn.innerText = 'Copy'; }, 1500);
                }
            );
        });
    });
}
</script> 