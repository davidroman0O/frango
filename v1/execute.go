package frango

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

// ContextKey type is used for context value keys
type ContextKey string

// The path globals PHP script to be injected into VFS instances
const pathGlobalsPHP = `<?php
// Frango v1 path globals initialization
// 
// Initializes the following PHP superglobals:
// - $_PATH: Contains path parameters extracted from URL patterns
// - $_PATH_SEGMENTS: Contains URL path segments
// - $_JSON: Contains parsed JSON request body

// Initialize $_PATH superglobal for path parameters
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available
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

// Initialize $_JSON for parsed JSON request body
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

// Helper function to get path segments
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

// Initialize template variables from PHP_VAR_ environment variables
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_VAR_') === 0) {
        $varName = substr($key, strlen('PHP_VAR_'));
        $varValue = json_decode($value, true);
        $GLOBALS[$varName] = $varValue;
    }
}
`

// ExecutePHP handles execution of a PHP script through the VFS
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	// 1. Extract all request data in a clean step
	requestData := extractRequestData(r)

	// 2. Prepare environment variables
	envData := make(map[string]string)

	// Add path segments (array indexes start at 0)
	for i, segment := range requestData.PathSegments {
		envData["PHP_PATH_SEGMENT_"+strconv.Itoa(i)] = segment
	}

	// Also provide the number of segments
	envData["PHP_PATH_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))

	// Add raw path
	envData["PHP_PATH"] = requestData.Path

	// --- Extract path parameters from pattern ---
	var pathParams map[string]string

	// Get the actual route pattern from the request's context if available
	if patternKey := php12PatternContextKey(r.Context()); patternKey != "" {
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
		}

		m.logger.Printf("Extracted path parameters: %v", pathParams)
	} else {
		// Check for any path parameters set in environment variables (for tests)
		paramsJSON := os.Getenv("PHP_PATH_PARAMS")
		if paramsJSON != "" {
			m.logger.Printf("Found PHP_PATH_PARAMS in environment: %s", paramsJSON)
			envData["PHP_PATH_PARAMS"] = paramsJSON
		}

		// Check for individual parameter variables
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, "PHP_PATH_PARAM_") {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					key := parts[0]
					value := parts[1]
					m.logger.Printf("Found param in environment: %s=%s", key, value)
					envData[key] = value
				}
			}
		}
	}

	// Add JSON body if available
	if requestData.JSONBody != nil {
		if fullJSON, err := json.Marshal(requestData.JSONBody); err == nil {
			envData["PHP_JSON"] = string(fullJSON)
		}
	}

	// Populate render data if renderFn is provided
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
		http.Error(w, "Server error locating PHP script", http.StatusInternalServerError)
		return
	}

	// 4. Verify script file exists
	fileInfo, err := os.Stat(phpFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Printf("PHP script not found: '%s'", phpFilePath)
			http.NotFound(w, r)
			return
		}
		m.logger.Printf("Error stating PHP script '%s': %v", phpFilePath, err)
		http.Error(w, "Server error locating script", http.StatusInternalServerError)
		return
	}
	if fileInfo.IsDir() {
		m.logger.Printf("ERROR: Target script path is a directory: '%s'", phpFilePath)
		http.Error(w, "Configuration error: script path is a directory", http.StatusInternalServerError)
		return
	}

	// 5. Prepare FrankenPHP request options
	// Document root is the PARENT directory of the script
	documentRoot := filepath.Dir(phpFilePath)
	scriptName := "/" + filepath.Base(phpFilePath) // Just the filename with leading slash

	m.logger.Printf("FrankenPHP Setup: DocumentRoot='%s', ScriptName='%s', URL='%s'", documentRoot, scriptName, r.URL.String())

	// Access PHP globals file from VFS (should be injected during initialization)
	pathGlobalsFile, pathGlobalsErr := vfs.ResolvePath("/_frango_php_globals.php")
	if pathGlobalsErr != nil {
		// If the globals file doesn't exist yet, create it
		m.logger.Printf("Creating PHP globals file in VFS")
		err := vfs.CreateVirtualFile("/_frango_php_globals.php", []byte(pathGlobalsPHP))
		if err != nil {
			m.logger.Printf("Warning: Failed to create PHP globals file: %v", err)
		} else {
			pathGlobalsFile, _ = vfs.ResolvePath("/_frango_php_globals.php")
		}
	}

	// Add auto-prepend file to include path globals if available
	if pathGlobalsFile != "" {
		m.logger.Printf("Adding path globals auto-prepend: %s", pathGlobalsFile)
		envData["PHP_AUTO_PREPEND_FILE"] = pathGlobalsFile
	}

	// Inject envData (render vars, path params) and query params
	phpBaseEnv := map[string]string{
		// *** DO NOT SET SCRIPT_FILENAME here *** - Rely on DocRoot + modified request path
		"SCRIPT_NAME":    scriptName,          // e.g., /index.php
		"PHP_SELF":       scriptName,          // Match SCRIPT_NAME
		"DOCUMENT_ROOT":  documentRoot,        // Parent dir of script
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
	}

	// Add in all our extracted data
	for key, value := range envData {
		phpBaseEnv[key] = value
	}

	if !m.developmentMode {
		phpBaseEnv["PHP_OPCACHE_ENABLE"] = "1"
	} else {
		phpBaseEnv["PHP_FCGI_MAX_REQUESTS"] = "1"
	}
	m.logger.Printf("Total PHP environment variables: %d", len(phpBaseEnv))

	// 6. Create and execute FrankenPHP request
	reqClone := r.Clone(r.Context())
	// *** Modify the cloned request path to match the script name ***
	reqClone.URL.Path = scriptName
	m.logger.Printf("Modified request clone path for FrankenPHP: %s", reqClone.URL.Path)

	// This variable is only set to true in test_mock.go which is included with the nowatcher tag
	// During normal builds, it remains false
	if isMockBuild {
		// Testing: use mock handlers
		mockExecutePHP(documentRoot, phpBaseEnv, w, reqClone, m.logger)
		return
	}

	// Production: use real FrankenPHP
	req, err := frankenphp.NewRequestWithContext(
		reqClone,
		frankenphp.WithRequestDocumentRoot(documentRoot, false),
		frankenphp.WithRequestEnv(phpBaseEnv),
	)
	if err != nil {
		m.logger.Printf("Error creating PHP request: %v", err)
		http.Error(w, "Server error creating PHP request", http.StatusInternalServerError)
		return
	}

	if err := frankenphp.ServeHTTP(w, req); err != nil {
		m.logger.Printf("Error executing PHP script '%s': %v", phpFilePath, err)
		http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
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
			if paramName != "" && paramName != "$" { // Skip special {$} if it exists
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

// Set by test files for mocking
var isMockBuild = false

// mockExecutePHP is called by tests to properly mock PHP execution
func mockExecutePHP(documentRoot string, env map[string]string, w http.ResponseWriter, r *http.Request, logger *log.Logger) {
	logger.Printf("Using mock PHP executor")

	// Create a basic PHP-like response
	output := fmt.Sprintf("PHP output from %s in %s\n", r.URL.Path, documentRoot)

	// Add any template variables and path parameters
	for k, v := range env {
		if len(k) > 8 && k[:8] == "PHP_VAR_" {
			varName := k[8:]
			output += fmt.Sprintf("Template Variable: %s = %s\n", varName, v)
		}

		if len(k) > 15 && k[:15] == "PHP_PATH_PARAM_" {
			paramName := k[15:]
			output += fmt.Sprintf("Path Parameter: %s = %s\n", paramName, v)
		}
	}

	// Set headers and write response
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}
