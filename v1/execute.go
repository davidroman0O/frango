package frango

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dunglas/frankenphp"
)

// RequestData contains all information extracted from an HTTP request
type RequestData struct {
	Method       string
	FullURL      string
	Path         string
	RemoteAddr   string
	Headers      http.Header
	QueryParams  map[string][]string
	PathSegments []string
	JSONBody     map[string]interface{}
	FormData     map[string][]string
}

// The path globals PHP script to be injected into VFS instances
const pathGlobalsPHP = `<?php
/**
 * PHP globals initialization
 * 
 * This file is automatically included in all PHP scripts executed by Frango.
 * It initializes PHP superglobals and provides helper functions.
 */

// Initialize $_PATH superglobal for path parameters
// This holds all parameters extracted from URL patterns like "/users/{id}"
global $_PATH;
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available (preferred method)
    $pathParamsJson = $_SERVER['PHP_PATH_PARAMS'] ?? '{}';
    $decodedParams = json_decode($pathParamsJson, true);
    if (is_array($decodedParams)) {
        $_PATH = $decodedParams;
    }
    
    // Also add any PHP_PATH_PARAM_ variables from $_SERVER for backward compatibility
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'PHP_PATH_PARAM_') === 0) {
            $paramName = substr($key, strlen('PHP_PATH_PARAM_'));
            if (!isset($_PATH[$paramName])) {
                $_PATH[$paramName] = $value;
            }
        }
    }
    
    // Make sure $_PATH is globally accessible
    $GLOBALS['_PATH'] = $_PATH;
}

// Initialize $_PATH_SEGMENTS superglobal for URL segments
// Contains each segment of the URL path split by "/"
global $_PATH_SEGMENTS;
if (!isset($_PATH_SEGMENTS)) {
    $_PATH_SEGMENTS = [];
    
    // Get segment count
    $segmentCount = intval($_SERVER['PHP_PATH_SEGMENT_COUNT'] ?? 0);
    
    // Add segments to array
    for ($i = 0; $i < $segmentCount; $i++) {
        $segmentKey = "PHP_PATH_SEGMENT_$i";
        if (isset($_SERVER[$segmentKey])) {
            $_PATH_SEGMENTS[] = $_SERVER[$segmentKey];
        }
    }
    
    // Make sure $_PATH_SEGMENTS is globally accessible
    $GLOBALS['_PATH_SEGMENTS'] = $_PATH_SEGMENTS;
    $GLOBALS['_PATH_SEGMENT_COUNT'] = $segmentCount;
}

// Initialize $_JSON superglobal for parsed JSON request body
// Provides direct access to JSON data submitted in the request
global $_JSON;
if (!isset($_JSON)) {
    $_JSON = [];
    
    // Load from JSON if available
    $jsonBody = $_SERVER['PHP_JSON'] ?? '{}';
    $decodedJson = json_decode($jsonBody, true);
    if (is_array($decodedJson)) {
        $_JSON = $decodedJson;
    }
    
    // Make sure $_JSON is globally accessible
    $GLOBALS['_JSON'] = $_JSON;
}

// Initialize $_FORM superglobal for form data
// Similar to $_POST but guaranteed to contain form data regardless of request method
global $_FORM;
if (!isset($_FORM)) {
    $_FORM = [];
    
    // Copy existing form data
    $_FORM = $_POST;
    
    // Add PHP_FORM_ variables from $_SERVER
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'PHP_FORM_') === 0) {
            $formKey = substr($key, strlen('PHP_FORM_'));
            $_FORM[$formKey] = $value;
        }
    }
    
    // Make sure $_FORM is globally accessible
    $GLOBALS['_FORM'] = $_FORM;
}

// Initialize common helper functions
if (!function_exists('path_segments')) {
    /**
     * Returns the URL path segments as an array
     * @return array URL path segments
     */
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

if (!function_exists('path_param')) {
    /**
     * Gets a path parameter with optional default value
     * @param string $name Parameter name
     * @param mixed $default Default value if parameter doesn't exist
     * @return mixed Parameter value or default
     */
    function path_param($name, $default = null) {
        global $_PATH;
        return $_PATH[$name] ?? $default;
    }
}

if (!function_exists('has_path_param')) {
    /**
     * Checks if a path parameter exists
     * @param string $name Parameter name
     * @return bool True if parameter exists
     */
    function has_path_param($name) {
        global $_PATH;
        return isset($_PATH[$name]);
    }
}

// Initialize template variables from PHP_VAR_ environment variables
// Makes template variables directly accessible as PHP variables
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_VAR_') === 0) {
        $varName = substr($key, strlen('PHP_VAR_'));
        $varValue = json_decode($value, true);
        
        // Set variable in global scope for direct access
        $GLOBALS[$varName] = $varValue;
        
        // Also make available as a superglobal array
        if (!isset($GLOBALS['_TEMPLATE'])) {
            $GLOBALS['_TEMPLATE'] = [];
        }
        $GLOBALS['_TEMPLATE'][$varName] = $varValue;
    }
}

// Make common request data easily accessible
global $_URL, $_CURRENT_URL, $_QUERY;
$_URL = $_SERVER['PHP_PATH'] ?? $_SERVER['REQUEST_URI'] ?? '';
$_CURRENT_URL = $_SERVER['REQUEST_URI'] ?? '';
$_QUERY = $_GET;

$GLOBALS['_URL'] = $_URL;
$GLOBALS['_CURRENT_URL'] = $_CURRENT_URL;
$GLOBALS['_QUERY'] = $_QUERY;
`

// ExecutePHP handles execution of a PHP script through the VFS
// This version closely mimics the behavior of the original working version
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	m.logger.Printf("========== EXECUTING PHP SCRIPT ==========")
	m.logger.Printf("ExecutePHP: Executing script '%s' with VFS %s", scriptPath, vfs.name)
	m.logger.Printf("ExecutePHP: HTTP Request %s %s", r.Method, r.URL.String())

	// Ensure the VFS has the PHP globals script installed
	if err := UpdateVFS(vfs); err != nil {
		m.logger.Printf("Warning: Failed to update VFS with PHP globals: %v", err)
	}

	// 1. Extract all request data in a clean step
	requestData := extractRequestData(r)

	// 2. Prepare environment variables that will be used to create PHP superglobals
	// We now use PHP_ prefixes as specified in the roadmap for a more PHP-friendly approach
	envData := make(map[string]string)

	// --- URL AND PATH SEGMENTS ---
	// These become $_PATH_SEGMENTS in PHP
	for i, segment := range requestData.PathSegments {
		envData["PHP_PATH_SEGMENT_"+strconv.Itoa(i)] = segment
	}
	envData["PHP_PATH_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))
	envData["PHP_PATH"] = requestData.Path

	// --- PATH PARAMETERS ---
	// These become $_PATH in PHP
	var pathParams map[string]string

	// First try to get pattern from context (for backward compatibility)
	patternKey := extractPatternFromContext(r.Context())

	// If no pattern in context, try to extract it from the script path
	if patternKey == "" {
		// Remove file extension to get the pattern
		scriptExt := filepath.Ext(scriptPath)
		patternKey = strings.TrimSuffix(scriptPath, scriptExt)
		m.logger.Printf("No pattern in context, extracted pattern from script path: %s", patternKey)
	}

	if patternKey != "" {
		m.logger.Printf("Using pattern: %s", patternKey)

		// Use the pattern to extract path parameters
		pathParams = extractPathParams(patternKey, requestData.Path)

		if pathParams != nil && len(pathParams) > 0 {
			// Add individual path parameters with PHP_PATH_PARAM_ prefix
			for name, value := range pathParams {
				envData["PHP_PATH_PARAM_"+name] = value
			}

			// Also add serialized path parameters as JSON
			if jsonParams, err := json.Marshal(pathParams); err == nil {
				envData["PHP_PATH_PARAMS"] = string(jsonParams)
			}

			m.logger.Printf("Extracted path parameters: %v", pathParams)
		}
	} else {
		m.logger.Printf("No pattern available, using URL path without parameter extraction: %s", requestData.Path)
	}

	// --- QUERY PARAMETERS ---
	// These become available in both $_GET and $_QUERY in PHP
	for key, values := range requestData.QueryParams {
		if len(values) > 0 {
			envData["PHP_QUERY_"+key] = values[0]
		}
	}

	// --- FORM DATA ---
	// These become available in $_FORM in PHP
	for key, values := range requestData.FormData {
		if len(values) > 0 && !strings.HasPrefix(key, "PHP_") { // Avoid overrides
			envData["PHP_FORM_"+key] = values[0]
		}
	}

	// --- JSON BODY ---
	// This becomes $_JSON in PHP
	if requestData.JSONBody != nil {
		// Add individual JSON fields with PHP_JSON_ prefix
		for key, value := range requestData.JSONBody {
			// Convert each JSON value to string
			if strValue, err := json.Marshal(value); err == nil {
				envData["PHP_JSON_"+key] = string(strValue)
			}
		}

		// Also provide the full JSON body
		if fullJSON, err := json.Marshal(requestData.JSONBody); err == nil {
			envData["PHP_JSON"] = string(fullJSON)
		}
	}

	// --- REQUEST HEADERS ---
	// These become available as PHP_HEADER_* variables in $_SERVER
	for key, values := range requestData.Headers {
		if len(values) > 0 {
			headerKey := strings.ReplaceAll(strings.ToUpper(key), "-", "_")
			envData["PHP_HEADER_"+headerKey] = values[0]
		}
	}

	// --- TEMPLATE VARIABLES ---
	// These become direct variables in PHP global scope and also in $_TEMPLATE
	if renderFn != nil {
		m.logger.Printf("Calling render function")
		data := renderFn(w, r)
		m.logger.Printf("Render data keys: %v", getMapKeys(data))
		for key, value := range data {
			jsonData, err := json.Marshal(value)
			if err != nil {
				m.logger.Printf("Error marshaling render data for '%s': %v", key, err)
				continue
			}
			m.logger.Printf("Render data for '%s': %s", key, string(jsonData))
			renderVarKey := "PHP_VAR_" + key
			envData[renderVarKey] = string(jsonData)
		}
	}

	// 3. Resolve the script path in the VFS to its actual filesystem path
	phpFilePath, err := vfs.ResolvePath(scriptPath)
	if err != nil {
		m.logger.Printf("Error resolving script path '%s': %v", scriptPath, err)

		// If we couldn't find the file in the VFS, try to add it from the source directory
		if m.sourceDir != "" && strings.HasPrefix(scriptPath, "/") {
			// Try to find the file in the source directory
			sourcePath := filepath.Join(m.sourceDir, filepath.FromSlash(strings.TrimPrefix(scriptPath, "/")))
			m.logger.Printf("Looking for script in source directory: %s", sourcePath)
			if _, err := os.Stat(sourcePath); err == nil {
				// Add the file to the VFS
				if err := vfs.AddSourceFile(sourcePath, scriptPath); err != nil {
					m.logger.Printf("Error adding source file to VFS: %v", err)
				} else {
					// Try to resolve path again
					phpFilePath, err = vfs.ResolvePath(scriptPath)
					if err != nil {
						m.logger.Printf("Error resolving script path after adding from source: %v", err)
					}
				}
			}
		}

		// For script paths with parameters (e.g., /users/{userId}.php), try to find a physical file
		// by replacing parameters with wildcards - this helps with test cases and direct filepath lookup
		if err != nil && strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}") {
			m.logger.Printf("Script path contains parameters, trying to find a matching file pattern")

			// Extract the directory part of the path
			dirPath := filepath.Dir(scriptPath)
			fileName := filepath.Base(scriptPath)

			// List files in that directory
			files, err := vfs.listFilesIn(dirPath)
			if err == nil && len(files) > 0 {
				// Look for a file with the same pattern (ignoring parameter values)
				for _, f := range files {
					// If the base name matches our pattern when parameters are replaced with wildcards
					baseF := filepath.Base(f)
					paramPattern := regexp.MustCompile(`\{[^}]+\}`)
					patternRegex := "^" + paramPattern.ReplaceAllString(regexp.QuoteMeta(fileName), ".*") + "$"
					matched, _ := regexp.MatchString(patternRegex, baseF)

					if matched {
						m.logger.Printf("Found matching file pattern: %s", f)
						// Use this file instead
						scriptPath = f
						phpFilePath, err = vfs.ResolvePath(scriptPath)
						if err == nil {
							break
						}
					}
				}
			}

			// If we still couldn't find a match, try looking for actual files with {param} literally in the name
			if err != nil {
				phpFilePath, err = vfs.ResolvePathLiteral(scriptPath)
				m.logger.Printf("Tried literal path resolution: %v", err)
			}
		}

		if err != nil {
			http.Error(w, "Server error locating PHP script", http.StatusInternalServerError)
			return
		}
	}

	// CRITICAL: Set up document root and script name correctly for FrankenPHP
	// Document root must be the parent directory of the script
	documentRoot := filepath.Dir(phpFilePath)

	// CRITICAL: Script name must be the basename with a leading slash
	// This is required for FrankenPHP to properly locate the script
	scriptName := "/" + filepath.Base(phpFilePath)

	m.logger.Printf("Executing PHP script in env: '%s' (from source: '%s')", phpFilePath, scriptPath)
	m.logger.Printf("FrankenPHP Setup: DocumentRoot='%s', ScriptName='%s', URL='%s'", documentRoot, scriptName, r.URL.String())

	// CRITICAL: Create a wrapper script that explicitly includes the globals file
	// This ensures our PHP superglobals are properly initialized
	wrapperPath := filepath.Join(filepath.Dir(phpFilePath), "_wrapper_"+filepath.Base(phpFilePath))
	globalsFilePath := filepath.Join(vfs.tempDir, "_frango_php_globals.php")

	// Create a temporary wrapper script that includes the globals and then the main script
	// Using require_once instead of include to ensure globals are loaded even if the script has an error
	wrapperContent := fmt.Sprintf(`<?php
// Auto-generated wrapper to ensure PHP superglobals are initialized
require_once '%s'; // Load globals initialization
include '%s'; // Load main script
?>`, globalsFilePath, phpFilePath)

	// Write the wrapper script
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		m.logger.Printf("Warning: Failed to create PHP globals wrapper script: %v", err)
	} else {
		// Use the wrapper script instead of the original
		phpFilePath = wrapperPath
		// Update the script name to use the wrapper
		scriptName = "/" + filepath.Base(wrapperPath)
		m.logger.Printf("Created PHP globals wrapper script: %s", wrapperPath)
	}

	// Inject envData (render vars, path params) and query params
	phpBaseEnv := map[string]string{
		// CRITICAL: Set SCRIPT_FILENAME explicitly to the absolute file path
		// This is THE most important environment variable for PHP execution
		"SCRIPT_FILENAME": phpFilePath, // Absolute path to the script

		// CRITICAL: These environment variables must be set correctly
		"SCRIPT_NAME":    scriptName,          // Must match the basename of the script
		"PHP_SELF":       scriptName,          // Match SCRIPT_NAME
		"DOCUMENT_ROOT":  documentRoot,        // Must be the parent directory of the script
		"REQUEST_URI":    requestData.FullURL, // Use the same full URL
		"REQUEST_METHOD": requestData.Method,
		"QUERY_STRING":   r.URL.RawQuery,
		"HTTP_HOST":      r.Host,
		"REMOTE_ADDR":    requestData.RemoteAddr,
		// Debugging info
		"DEBUG_DOCUMENT_ROOT": documentRoot,
		"DEBUG_SCRIPT_NAME":   scriptName,
		"DEBUG_PHP_FILE_PATH": phpFilePath, // Full path for debugging
		"DEBUG_SOURCE_PATH":   scriptPath,
		"DEBUG_ENV_ID":        vfs.name,
	}

	// Add in all our extracted data (with PHP_ prefixes)
	for key, value := range envData {
		phpBaseEnv[key] = value
	}

	// Set up PHP configuration options
	if m.developmentMode {
		phpBaseEnv["PHP_FCGI_MAX_REQUESTS"] = "1" // Disable PHP-FPM keepalive
	} else {
		phpBaseEnv["PHP_OPCACHE_ENABLE"] = "1" // Enable opcache in production
	}

	// Set explicit PHP timeouts to prevent hanging
	phpBaseEnv["PHP_MAX_EXECUTION_TIME"] = "10" // 10 second timeout
	phpBaseEnv["PHP_DEFAULT_SOCKET_TIMEOUT"] = "10"

	m.logger.Printf("Total PHP environment variables: %d", len(phpBaseEnv))

	// CRITICAL: Modify the request clone path to match the script name
	// This ensures FrankenPHP looks for the correct file
	reqClone := r.Clone(r.Context())
	reqClone.URL.Path = scriptName
	m.logger.Printf("Modified request clone path for FrankenPHP: %s", reqClone.URL.Path)

	// Dump all environment variables in sorted order for debugging
	m.logger.Printf("ExecutePHP: ======= FULL PHP ENVIRONMENT =======")
	var keys []string
	for k := range phpBaseEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := phpBaseEnv[k]
		m.logger.Printf("  %s = %s", k, v)
	}
	m.logger.Printf("ExecutePHP: ===================================")

	// CRITICAL: Create FrankenPHP request with exact document root and environment
	req, err := frankenphp.NewRequestWithContext(
		reqClone, // Use the modified request with the script path
		frankenphp.WithRequestDocumentRoot(documentRoot, false), // Exact document root
		frankenphp.WithRequestEnv(phpBaseEnv),                   // All environment variables
	)
	if err != nil {
		m.logger.Printf("Error creating PHP request: %v", err)
		http.Error(w, "Server error creating PHP request", http.StatusInternalServerError)
		return
	}

	// Execute the PHP script
	if err := frankenphp.ServeHTTP(w, req); err != nil {
		m.logger.Printf("Error executing PHP script '%s': %v", phpFilePath, err)
		http.Error(w, fmt.Sprintf("PHP execution error: %v", err), http.StatusInternalServerError)
		return
	}

	m.logger.Printf("PHP execution completed successfully for '%s'", scriptPath)
	m.logger.Printf("========== PHP EXECUTION COMPLETE ==========")
}

// extractRequestData extracts all relevant data from an HTTP request
func extractRequestData(r *http.Request) *RequestData {
	// Create a new request data object
	data := &RequestData{
		Method:      r.Method,
		FullURL:     r.URL.String(),
		Path:        r.URL.Path,
		RemoteAddr:  r.RemoteAddr,
		Headers:     r.Header,
		QueryParams: r.URL.Query(),
		PathSegments: func() []string {
			segments := []string{}
			for _, segment := range strings.Split(strings.Trim(r.URL.Path, "/"), "/") {
				if segment != "" {
					segments = append(segments, segment)
				}
			}
			return segments
		}(),
		JSONBody: make(map[string]interface{}),
		FormData: make(map[string][]string),
	}

	// Parse form data if the method might include it
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		contentType := r.Header.Get("Content-Type")

		// For JSON requests, read and parse the body
		if strings.Contains(contentType, "application/json") {
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				// Restore the body for later PHP processing
				r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

				if err == nil && len(bodyBytes) > 0 {
					var jsonData map[string]interface{}
					if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
						data.JSONBody = jsonData
					}
				}
			}
		} else {
			// For form data, parse the form
			if err := r.ParseForm(); err == nil {
				data.FormData = r.Form
			}
		}
	}

	return data
}

// extractPathParams extracts path parameters from a URL pattern and actual path
// For example: extractPathParams("/users/{id}/posts/{postId}", "/users/42/posts/123")
// returns: map[string]string{"id": "42", "postId": "123"}
func extractPathParams(pattern, path string) map[string]string {
	// Extract HTTP method if pattern includes it
	patternPath := pattern
	if parts := strings.SplitN(pattern, " ", 2); len(parts) > 1 {
		patternPath = parts[1]
	}

	// Split pattern and path into segments
	patternSegments := strings.Split(strings.Trim(patternPath, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Check if segment counts don't match
	if len(patternSegments) != len(pathSegments) {
		return nil
	}

	// Extract parameters
	params := make(map[string]string)
	for i, patternSegment := range patternSegments {
		// Check for parameter pattern {name}
		if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
			// Extract parameter name without braces
			paramName := patternSegment[1 : len(patternSegment)-1]
			if paramName != "" {
				// Use actual path segment as parameter value
				params[paramName] = pathSegments[i]
			}
		} else if patternSegment != pathSegments[i] {
			// If a non-parameter segment doesn't match exactly, no match
			return nil
		}
	}

	return params
}

// getMapKeys is a helper function to get the keys of a map for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// extractPatternFromContext extracts the URL pattern from the request context

// Define the pattern key type for context values
