<?php
/**
 * Frango Render Helper
 * 
 * Helper functions for accessing variables from Go in PHP templates.
 */

/**
 * Gets a variable passed from Go.
 * 
 * @param string $name    The name of the variable
 * @param mixed  $default Default value if the variable doesn't exist
 * @return mixed The variable value or the default value
 */
function go_var($name, $default = null) {
    $jsonValue = $_ENV['frango_VAR_' . $name] ?? null;
    if ($jsonValue === null) {
        return $default;
    }
    
    $value = json_decode($jsonValue, true);
    // Return default if JSON decode failed
    if ($value === null && json_last_error() !== JSON_ERROR_NONE) {
        return $default;
    }
    
    return $value;
}

/**
 * Gets all variables passed from Go.
 * 
 * @return array Associative array of all variables
 */
function go_vars() {
    $vars = [];
    foreach ($_ENV as $key => $value) {
        if (strpos($key, 'frango_VAR_') === 0) {
            $name = substr($key, 11); // Remove 'frango_VAR_' prefix
            $decodedValue = json_decode($value, true);
            if ($decodedValue !== null || json_last_error() === JSON_ERROR_NONE) {
                $vars[$name] = $decodedValue;
            }
        }
    }
    return $vars;
}

/**
 * Gets a debugging view of all variables passed from Go.
 * 
 * @return string HTML representation of all variables for debugging
 */
function go_debug() {
    $vars = go_vars();
    $output = '<div class="go-debug">';
    $output .= '<h3>Variables from Go</h3>';
    
    if (empty($vars)) {
        $output .= '<p>No variables passed from Go.</p>';
    } else {
        $output .= '<ul>';
        foreach ($vars as $name => $value) {
            $output .= '<li><strong>' . htmlspecialchars($name) . '</strong>: ';
            $output .= '<pre>' . htmlspecialchars(json_encode($value, JSON_PRETTY_PRINT)) . '</pre>';
            $output .= '</li>';
        }
        $output .= '</ul>';
    }
    
    $output .= '</div>';
    return $output;
} 