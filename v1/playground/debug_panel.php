<?php
/**
 * Frango Debug Panel
 * 
 * This file provides a reusable debug panel that shows PHP environment variables
 * Include this in any page to see the current environment state
 */

// Initialize superglobals if they don't exist
if (!isset($_PATH)) $_PATH = [];
if (!isset($_PATH_SEGMENTS)) $_PATH_SEGMENTS = [];
if (!isset($_PATH_SEGMENT_COUNT)) $_PATH_SEGMENT_COUNT = 0;
if (!isset($_JSON)) $_JSON = [];
if (!isset($_FORM)) $_FORM = [];
if (!isset($_URL)) $_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_CURRENT_URL)) $_CURRENT_URL = isset($_SERVER['REQUEST_URI']) ? $_SERVER['REQUEST_URI'] : '';
if (!isset($_QUERY)) $_QUERY = isset($_GET) ? $_GET : [];

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
?>
<style>
.debug-panel {
    position: fixed;
    top: 10px;
    right: 10px;
    width: 300px;
    max-height: 95vh;
    overflow-y: auto;
    background: #fff;
    border-radius: 5px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.2);
    z-index: 9999;
    font-family: system-ui, -apple-system, sans-serif;
    font-size: 12px;
    color: #333;
    opacity: 0.9;
    transition: opacity 0.2s;
}
.debug-panel:hover {
    opacity: 1;
}
.debug-panel-header {
    padding: 8px 12px;
    background: #2c3e50;
    color: white;
    font-weight: bold;
    border-top-left-radius: 5px;
    border-top-right-radius: 5px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.debug-panel-body {
    padding: 0;
    max-height: 85vh;
    overflow-y: auto;
}
.debug-panel-section {
    margin-bottom: 1px;
}
.debug-section-header {
    padding: 6px 12px;
    background: #f5f5f5;
    font-weight: bold;
    cursor: pointer;
    border-bottom: 1px solid #ddd;
    font-size: 11px;
}
.debug-section-content {
    padding: 6px 12px;
    display: none;
    font-size: 11px;
    background: #f9f9f9;
    max-height: 300px;
    overflow-y: auto;
}
.env-var {
    padding: 3px 0;
    border-bottom: 1px solid #eee;
}
.env-var-name {
    font-weight: bold;
    color: #2980b9;
}
.env-value {
    color: #333;
    word-break: break-all;
}
.debug-path-var {
    margin: 2px 0;
    padding: 3px;
    background: #eaf2f8;
    border-radius: 3px;
}
.path-value {
    font-weight: bold;
}
</style>

<div class="debug-panel">
    <div class="debug-panel-header" onclick="toggleDebugPanel()">
        Frango Debug Panel
        <span id="debug-panel-toggle">[-]</span>
    </div>
    <div class="debug-panel-body" id="debug-panel-body">
        <!-- Path Parameters -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('path-params')">
                Path Parameters
            </div>
            <div class="debug-section-content" id="path-params">
                <?php if (!empty($_PATH)): ?>
                    <?php foreach ($_PATH as $key => $value): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= htmlspecialchars($value) ?></span>
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
                Path Segments (<?= count($_PATH_SEGMENTS) ?>)
            </div>
            <div class="debug-section-content" id="path-segments">
                <?php if (!empty($_PATH_SEGMENTS)): ?>
                    <?php foreach ($_PATH_SEGMENTS as $index => $segment): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">[<?= $index ?>]:</span>
                        <span class="path-value"><?= htmlspecialchars($segment) ?></span>
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
                Query Parameters (<?= count($_GET) ?>)
            </div>
            <div class="debug-section-content" id="query-data">
                <?php if (!empty($_GET)): ?>
                    <?php foreach ($_GET as $key => $value): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= htmlspecialchars($value) ?></span>
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
                Form Data (<?= count($_POST) + count($_FORM ?? []) ?>) <?= empty($_POST) && !empty($debugInfo['raw_form_vars'] ?? []) ? '⚠️' : '✅' ?>
            </div>
            <div class="debug-section-content" id="form-data">
                <?php if (!empty($_POST) || !empty($_FORM ?? [])): ?>
                    <?php 
                    $formData = array_merge($_POST, (array)($_FORM ?? []));
                    foreach ($formData as $key => $value): 
                    ?>
                    <div class="debug-path-var">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= htmlspecialchars($value) ?></span>
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
                    <div style="background: #ffe5e5; padding: 5px; border-radius: 3px; margin-bottom: 5px;">
                        <strong>Warning:</strong> PHP_FORM_* variables exist but $_POST is empty. Form fix not applied.
                    </div>
                    <?php foreach ($missingFormVars as $key => $value): ?>
                    <div class="debug-path-var" style="border-left: 3px solid #e74c3c;">
                        <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span>
                        <span class="path-value"><?= htmlspecialchars($value) ?></span>
                    </div>
                    <?php endforeach; ?>
                    <?php else: ?>
                    <div class="debug-path-var">No form data</div>
                    <?php endif; ?>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- PHP Environment Variables -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('php-env')">
                PHP_ Variables
            </div>
            <div class="debug-section-content" id="php-env">
                <?php
                $phpVars = array_filter($_SERVER, function($k) {
                    return str_starts_with($k, 'PHP_');
                }, ARRAY_FILTER_USE_KEY);
                
                ksort($phpVars);
                
                if (!empty($phpVars)):
                    foreach ($phpVars as $key => $value):
                ?>
                <div class="env-var">
                    <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span><br>
                    <span class="env-value"><?= htmlspecialchars($value) ?></span>
                </div>
                <?php 
                    endforeach;
                else:
                ?>
                <div class="env-var">No PHP_ variables found</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Debug Environment Variables -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('debug-env')">
                DEBUG_ Variables
            </div>
            <div class="debug-section-content" id="debug-env">
                <?php
                $debugVars = array_filter($_SERVER, function($k) {
                    return str_starts_with($k, 'DEBUG_');
                }, ARRAY_FILTER_USE_KEY);
                
                ksort($debugVars);
                
                if (!empty($debugVars)):
                    foreach ($debugVars as $key => $value):
                ?>
                <div class="env-var">
                    <span class="env-var-name"><?= htmlspecialchars($key) ?>:</span><br>
                    <span class="env-value"><?= htmlspecialchars($value) ?></span>
                </div>
                <?php 
                    endforeach;
                else:
                ?>
                <div class="env-var">No DEBUG_ variables found</div>
                <?php endif; ?>
            </div>
        </div>
        
        <!-- Add a section to display session data -->
        <div class="debug-panel-section">
            <div class="debug-section-header" onclick="toggleSection('session-data')">
                Session Data
            </div>
            <div class="debug-section-content" id="session-data">
                <?php
                // Start session if not already started
                if (session_status() == PHP_SESSION_NONE) {
                    session_start();
                }
                
                if (!empty($_SESSION['form_data'])): 
                ?>
                    <h4>Form Data in Session:</h4>
                    <?php if (!empty($_SESSION['form_data']['POST'])): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">POST Data:</span>
                        <pre><?php var_export($_SESSION['form_data']['POST']); ?></pre>
                    </div>
                    <?php endif; ?>
                    
                    <?php if (!empty($_SESSION['form_data']['GET'])): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">GET Data:</span>
                        <pre><?php var_export($_SESSION['form_data']['GET']); ?></pre>
                    </div>
                    <?php endif; ?>
                    
                    <?php if (!empty($_SESSION['form_data']['JSON'])): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">JSON Data:</span>
                        <pre><?php var_export($_SESSION['form_data']['JSON']); ?></pre>
                    </div>
                    <?php endif; ?>
                    
                    <?php if (!empty($_SESSION['form_data']['debug'])): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">Debug Info:</span>
                        <pre><?php var_export($_SESSION['form_data']['debug']); ?></pre>
                    </div>
                    <?php endif; ?>
                    
                    <?php if (!empty($_SESSION['form_data']['debug_info'])): ?>
                    <div class="debug-path-var">
                        <span class="env-var-name">Debug Info (Extended):</span>
                        <pre><?php var_export($_SESSION['form_data']['debug_info']); ?></pre>
                    </div>
                    <?php endif; ?>
                    
                    <?php if (!empty($_SESSION['form_data']['error'])): ?>
                    <div class="debug-path-var" style="border-left-color: #e74c3c;">
                        <span class="env-var-name">Error:</span>
                        <pre><?php echo htmlspecialchars($_SESSION['form_data']['error']); ?></pre>
                    </div>
                    <?php endif; ?>
                    
                    <div class="debug-path-var">
                        <span class="env-var-name">Last Updated:</span>
                        <span class="path-value">
                            <?= !empty($_SESSION['form_data']['last_updated']) ? 
                                date('Y-m-d H:i:s', $_SESSION['form_data']['last_updated']) : 'Never' ?>
                        </span>
                    </div>
                <?php else: ?>
                    <div class="debug-path-var">No form data in session</div>
                <?php endif; ?>
                
                <h4>Session ID:</h4>
                <div class="debug-path-var">
                    <span class="path-value"><?= session_id() ?: 'Not set' ?></span>
                </div>
            </div>
        </div>
    </div>
</div>

<script>
// Debug panel toggle
let panelVisible = true;

function toggleDebugPanel() {
    const panel = document.getElementById('debug-panel-body');
    const toggle = document.getElementById('debug-panel-toggle');
    
    if (panelVisible) {
        panel.style.display = 'none';
        toggle.innerText = '[+]';
    } else {
        panel.style.display = 'block';
        toggle.innerText = '[-]';
    }
    
    panelVisible = !panelVisible;
}

// Section toggle
function toggleSection(sectionId) {
    const section = document.getElementById(sectionId);
    section.style.display = section.style.display === 'block' ? 'none' : 'block';
}

// Show first section by default
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('path-params').style.display = 'block';
});
</script> 