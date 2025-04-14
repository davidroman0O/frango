package executor

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// prepareEnvironmentData extracts and prepares all environment data needed for PHP execution.
// It aggregates data from the request, config, and potentially render functions.
func (e *Executor) prepareEnvironmentData(requestData *RequestData, scriptPath, physicalScriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) map[string]string {
	envData := make(map[string]string)
	config := e.config
	logger := config.Logger

	if logger != nil {
		logger.Printf("Executor: Preparing environment data for script: %s", scriptPath)
	}

	const (
		emptyJSON      = "{}"
		emptyJSONArray = "[]"
	)

	// --- PATH PARAMETERS ---
	// Extract path parameters from route pattern or script path
	pathParams := e.extractPathParamsFromRequest(requestData, scriptPath, r)
	if pathParams != nil && len(pathParams) > 0 {
		// Store path params in _PATH for backward compatibility
		marshalToEnv(pathParams, "_PATH", emptyJSON, envData)

		// IMPORTANT: In PHP, path parameters should be merged into $_GET
		// We'll handle the merging in PHP globals script via:
		// $_GET = array_merge($_GET, $_PATH);
	} else {
		envData["_PATH"] = emptyJSON
	}

	// --- QUERY PARAMETERS ---
	marshalToEnv(requestData.QueryParams, "_GET", emptyJSON, envData)

	// --- POST DATA ---
	marshalToEnv(requestData.FormData, "_POST", emptyJSON, envData)

	// --- JSON BODY ---
	// Make JSON body available in $_JSON (non-standard but useful)
	marshalToEnv(requestData.JSONBody, "_JSON", emptyJSON, envData)

	// --- PATH SEGMENTS ---
	marshalToEnv(requestData.PathSegments, "_PATH_SEGMENTS", emptyJSONArray, envData)
	envData["_PATH_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))

	// --- FILE UPLOADS ---
	// Build $_FILES structure according to PHP standard
	e.addStandardPHPFiles(requestData.FileUploads, envData)

	// --- TEMPLATE VARIABLES ---
	if renderFn != nil {
		e.addTemplateVariables(renderFn, w, r, envData)
	} else {
		envData["_TEMPLATE"] = emptyJSON
	}

	if logger != nil {
		// Log key environment variables
		var keys []string
		for k := range envData {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		logger.Printf("Executor: Prepared environment data with keys: %v", keys)
	}

	return envData
}

// extractPathParamsFromRequest extracts path parameters from the request
func (e *Executor) extractPathParamsFromRequest(requestData *RequestData, scriptPath string, r *http.Request) map[string]string {
	logger := e.config.Logger

	// Try to get pattern from context (set by router middleware)
	pattern := ""
	if patternCtx, ok := r.Context().Value("pattern").(string); ok && patternCtx != "" {
		pattern = patternCtx
		if logger != nil {
			logger.Printf("Using pattern from context: %s", pattern)
		}
	} else if strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}") {
		// If scriptPath contains parameters, use it as pattern
		pattern = scriptPath
		if logger != nil {
			logger.Printf("Using scriptPath as pattern: %s", pattern)
		}
	}

	if pattern == "" {
		return nil
	}

	// Extract parameters using the pattern
	pathParams := e.extractPathParams(pattern, requestData.Path)
	if logger != nil && pathParams != nil {
		logger.Printf("Extracted path parameters: %v", pathParams)
	}

	return pathParams
}

// addStandardPHPFiles formats file uploads in standard PHP $_FILES structure
func (e *Executor) addStandardPHPFiles(fileUploads map[string][]*multipart.FileHeader, envData map[string]string) {
	if len(fileUploads) == 0 {
		envData["_FILES"] = "{}"
		return
	}

	// Prepare FILES structure
	filesData := make(map[string]map[string]interface{})

	for fieldName, headers := range fileUploads {
		if len(headers) == 0 {
			continue
		}

		// Handle single file upload (PHP style)
		fileHeader := headers[0]

		// Build the standard PHP file structure with all required fields
		fileInfo := map[string]interface{}{
			"name":     fileHeader.Filename,
			"type":     fileHeader.Header.Get("Content-Type"),
			"tmp_name": "/tmp/" + fileHeader.Filename, // Standard PHP temp file location
			"error":    0,                             // UPLOAD_ERR_OK
			"size":     fileHeader.Size,
		}

		filesData[fieldName] = fileInfo
	}

	marshalToEnv(filesData, "_FILES", "{}", envData)
}

// addTemplateVariables adds template variables to environment data
func (e *Executor) addTemplateVariables(renderFn RenderData, w http.ResponseWriter, r *http.Request, envData map[string]string) {
	logger := e.config.Logger

	if logger != nil {
		logger.Printf("Executing render function to get template variables")
	}

	// Get template data from the render function
	data := renderFn(w, r)
	if data == nil {
		envData["_TEMPLATE"] = "{}"
		return
	}

	// Convert to PHP-accessible format
	templateVars := make(map[string]interface{})
	for key, value := range data {
		templateVars[key] = value

		// Also add individual variables with PHP_VAR_ prefix
		jsonData, err := json.Marshal(value)
		if err != nil {
			if logger != nil {
				logger.Printf("Error marshaling template value for '%s': %v", key, err)
			}
			continue
		}

		envData["PHP_VAR_"+key] = string(jsonData)
	}

	// Add complete template data
	marshalToEnv(templateVars, "_TEMPLATE", "{}", envData)
}

// marshalToEnv marshals a value to JSON and adds it to the environment data with fallback.
func marshalToEnv(value interface{}, key, fallback string, envData map[string]string) {
	// Handle nil values gracefully
	if value == nil {
		envData[key] = fallback
		return
	}

	// Marshal to JSON
	jsonVal, err := json.Marshal(value)
	if err != nil {
		envData[key] = fallback
		return
	}

	envData[key] = string(jsonVal)
}

// buildPhpEnvironment builds the final PHP environment variables for execution
func (e *Executor) buildPhpEnvironment(
	requestData *RequestData,
	goEnvData map[string]string,
	phpFilePath string, // Actual path to the wrapper script for execution
	documentRoot string, // Must be the parent directory of the script
	originalScriptName string, // The original script name for PHP_SELF, etc.
	r *http.Request,
) map[string]string {
	config := e.config
	logger := config.Logger

	// Get host and URI data from request
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	requestURI := r.URL.String()

	// Extract query string
	queryString := ""
	if i := strings.Index(requestURI, "?"); i > -1 {
		queryString = requestURI[i+1:]
	}

	// Extract remote address
	remoteAddr := r.RemoteAddr
	remoteHost := remoteAddr
	if i := strings.LastIndex(remoteAddr, ":"); i > -1 {
		remoteHost = remoteAddr[:i] // Remove port
	}

	// Calculate path info if applicable
	pathInfo := ""
	if len(r.URL.Path) > len(originalScriptName) && strings.HasPrefix(r.URL.Path, originalScriptName) {
		pathInfo = r.URL.Path[len(originalScriptName):]
	}

	// Base PHP environment with all standard PHP variables
	phpEnv := map[string]string{
		// Core PHP script paths
		"SCRIPT_FILENAME": phpFilePath,        // Absolute physical path to script
		"SCRIPT_NAME":     originalScriptName, // Web-accessible path (logical)
		"PHP_SELF":        originalScriptName, // Same as SCRIPT_NAME in most cases
		"DOCUMENT_ROOT":   documentRoot,       // Document root directory

		// Request information
		"REQUEST_METHOD":  r.Method,
		"REQUEST_URI":     requestURI,  // Full URI with query string
		"QUERY_STRING":    queryString, // Just the query part
		"SERVER_PROTOCOL": r.Proto,     // HTTP/1.1 or HTTP/2
		"PATH_INFO":       pathInfo,    // Path after the script name

		// Server info
		"SERVER_NAME":     host, // From Host header
		"SERVER_PORT":     extractPortFromHost(host),
		"SERVER_SOFTWARE": "frango/FrankenPHP",       // Server software identifier
		"HTTPS":           boolToOnOff(r.TLS != nil), // "on" if HTTPS, "off" otherwise

		// Client info
		"REMOTE_ADDR": remoteHost, // Client IP address
		"REMOTE_PORT": extractPortFromAddr(remoteAddr),
	}

	// Add HTTP headers (converted to HTTP_* format)
	for name, values := range r.Header {
		if len(values) > 0 {
			headerName := "HTTP_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
			phpEnv[headerName] = values[0]
		}
	}

	// Handle special headers that don't get HTTP_ prefix
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		phpEnv["CONTENT_TYPE"] = contentType
	}
	if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
		phpEnv["CONTENT_LENGTH"] = contentLength
	}

	// Add values from Go environment data (path params, etc.)
	for key, value := range goEnvData {
		// Only add PHP variables (prefixed with _) and not GO_* variables
		if !strings.HasPrefix(key, "GO_") {
			phpEnv[key] = value
		}
	}

	// Set PHP configuration options
	if config.DevelopmentMode {
		phpEnv["PHP_FCGI_MAX_REQUESTS"] = "1" // Disable PHP-FPM keepalive in dev mode
	} else {
		phpEnv["PHP_OPCACHE_ENABLE"] = "1" // Enable opcache in production
	}

	// Set explicit PHP timeouts
	phpEnv["PHP_MAX_EXECUTION_TIME"] = "10" // 10 second timeout
	phpEnv["PHP_DEFAULT_SOCKET_TIMEOUT"] = "10"

	// Set PHP error display settings - use ini_set directly for more reliable error control
	if config.DisplayErrors {
		// Display all errors for debugging
		phpEnv["auto_prepend_text"] = "<?php ini_set('display_errors', '1'); ini_set('display_startup_errors', '1'); error_reporting(E_ALL); ?>"
	} else {
		// Hide all errors for production
		phpEnv["auto_prepend_text"] = "<?php ini_set('display_errors', 'Off'); ini_set('display_startup_errors', 'Off'); ?>"
	}

	// Always set these error handling options regardless of display_errors
	// to ensure stack traces and error information are preserved
	phpEnv["log_errors"] = "1"      // Log errors to help with debugging
	phpEnv["report_memleaks"] = "1" // Report memory leaks
	phpEnv["html_errors"] = "1"     // Use HTML formatting for errors
	phpEnv["docref_root"] = "https://www.php.net/manual/en/"
	phpEnv["docref_ext"] = ".php"
	phpEnv["error_log"] = "/dev/stderr" // Log errors to stderr for containerized environments

	// Settings specifically for better stack traces
	phpEnv["zend.exception_ignore_args"] = "0"             // Don't ignore function arguments in stack traces
	phpEnv["zend.exception_string_param_max_len"] = "1024" // Show more of string parameters
	phpEnv["zend.exception_class_max_len"] = "1024"        // Show more of class names in exceptions

	// Additional debug settings for better stack traces
	phpEnv["xdebug.cli_color"] = "1"       // Enable colorized output for CLI
	phpEnv["xdebug.show_local_vars"] = "1" // Show local variables in stack traces

	if logger != nil && config.DevelopmentMode {
		logger.Printf("Built PHP environment with %d variables", len(phpEnv))
		e.logEnvironmentVariables(phpEnv)
	}

	return phpEnv
}

// Extract port from a host string (hostname:port)
func extractPortFromHost(host string) string {
	if i := strings.LastIndex(host, ":"); i > -1 {
		return host[i+1:]
	}
	// Default ports
	return "80"
}

// Extract port from a remote address (host:port)
func extractPortFromAddr(addr string) string {
	if i := strings.LastIndex(addr, ":"); i > -1 {
		return addr[i+1:]
	}
	return ""
}

// Convert boolean to "on" or "off" for PHP environment variables
func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// logEnvironmentVariables logs environment variables in sorted order
// for debugging purposes, masking sensitive information
func (e *Executor) logEnvironmentVariables(phpEnv map[string]string) {
	logger := e.config.Logger
	if logger == nil || !e.config.DevelopmentMode {
		return
	}

	logger.Printf("======= PHP ENVIRONMENT VARIABLES =======")

	// Get sorted keys for consistent logging
	var keys []string
	for k := range phpEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := phpEnv[k]

		// Mask sensitive information (like passwords or auth headers)
		if isSensitiveVariable(k) {
			v = maskSensitiveValue(v)
		}

		// Truncate long values (JSON data, etc.)
		if len(v) > 200 {
			v = v[:200] + "... [truncated]"
		}

		logger.Printf("  %s = %s", k, v)
	}
	logger.Printf("======= END PHP ENVIRONMENT =======")
}

// isSensitiveVariable checks if a variable name indicates sensitive content
func isSensitiveVariable(name string) bool {
	sensitiveNames := []string{
		"HTTP_AUTHORIZATION",
		"HTTP_COOKIE",
		"PASSWORD",
		"SECRET",
		"TOKEN",
		"KEY",
	}

	lowerName := strings.ToLower(name)
	for _, sensitive := range sensitiveNames {
		if strings.Contains(lowerName, strings.ToLower(sensitive)) {
			return true
		}
	}

	return false
}

// maskSensitiveValue masks a sensitive value for logging
func maskSensitiveValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}

	// Show first 4 and last 4 characters only
	return value[:4] + "****" + value[len(value)-4:]
}

// setupPhpEnvironment prepares the PHP execution environment
func (e *Executor) setupPhpEnvironment(requestData *RequestData, scriptPath, resolvedPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) (map[string]string, error) {
	logger := e.config.Logger

	// Prepare Go-specific environment data
	goEnvData := e.prepareEnvironmentData(requestData, scriptPath, resolvedPath, renderFn, w, r)

	// Set up document root and script name
	documentRoot := filepath.Dir(resolvedPath)
	originalScriptName := "/" + filepath.Base(scriptPath)

	if logger != nil {
		logger.Printf("Executor: Setup paths - DocumentRoot='%s', ScriptName='%s'",
			documentRoot, originalScriptName)
	}

	// Build PHP environment variables
	phpEnv := e.buildPhpEnvironment(requestData, goEnvData, resolvedPath, documentRoot, originalScriptName, r)

	// Add additional environment variables
	e.addScriptEnvironment(phpEnv, resolvedPath, scriptPath)

	// Log environment variables in debug mode
	e.logEnvironmentVariables(phpEnv)

	return phpEnv, nil
}

// addScriptEnvironment adds script-specific environment variables
func (e *Executor) addScriptEnvironment(phpEnv map[string]string, resolvedPath, scriptPath string) {
	// Add globals file path
	globalsFile := "/_frango_php_globals.php"
	phpEnv["PHP_GLOBALS_FILE"] = globalsFile

	// Set auto_prepend_file for globals
	globalsFilePath := filepath.Join(e.vfs.GetTempDir(), strings.TrimPrefix(globalsFile, "/"))
	phpEnv["auto_prepend_file"] = globalsFilePath

	// Add script directory for includes
	scriptDir := filepath.Dir(resolvedPath)
	phpEnv["SCRIPT_DIR"] = scriptDir

	// Add logical filename and dirname for __FILE__ and __DIR__
	phpEnv["FRANGO_LOGICAL_FILENAME"] = resolvedPath
	phpEnv["FRANGO_LOGICAL_DIR"] = scriptDir

	// Add environment variable for chdir
	phpEnv["FRANGO_DO_CHDIR"] = "1"
}

// extractPathParams extracts path parameters from a URL pattern and actual path
func (e *Executor) extractPathParams(pattern, path string) map[string]string {
	// Extract HTTP method if pattern includes it
	patternPath := pattern
	if parts := strings.SplitN(pattern, " ", 2); len(parts) > 1 {
		patternPath = parts[1]
	}

	// Special case for patterns with multiple parameters in the same segment
	if strings.Contains(patternPath, "}-{") {
		return e.extractMultipleParamsPerSegment(patternPath, path)
	}

	// Check if pattern contains a catchall parameter
	if e.containsCatchAllParam(patternPath) {
		return e.extractCatchAllParams(patternPath, path)
	}

	// Split pattern and path into segments
	patternSegments := strings.Split(strings.Trim(patternPath, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Create parameters map
	params := make(map[string]string)

	// Handle empty parameter case
	if len(pathSegments) == len(patternSegments)-1 &&
		len(patternSegments) > 0 &&
		strings.HasPrefix(patternSegments[len(patternSegments)-1], "{") &&
		strings.HasSuffix(patternSegments[len(patternSegments)-1], "}") {
		paramName := patternSegments[len(patternSegments)-1][1 : len(patternSegments[len(patternSegments)-1])-1]
		params[paramName] = ""
		return params
	}

	// Standard case: extract parameters from matching segments
	maxSegments := len(patternSegments)
	if maxSegments > len(pathSegments) {
		maxSegments = len(pathSegments)
	}

	// Extract parameters from matching segments
	for i := 0; i < maxSegments; i++ {
		patternSegment := patternSegments[i]
		pathSegment := pathSegments[i]

		// Check for parameter pattern {name}
		if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
			// Extract parameter name without braces
			paramName := patternSegment[1 : len(patternSegment)-1]

			// Handle special case for catchall params with asterisk
			if strings.HasPrefix(paramName, "*") {
				paramName = paramName[1:] // Remove the asterisk
			}

			if paramName != "" {
				params[paramName] = pathSegment
			}
		} else if patternSegment != pathSegment {
			// Non-parameter segments must match exactly
			return nil
		}
	}

	return params
}

// containsCatchAllParam checks if the pattern contains a catchall parameter
func (e *Executor) containsCatchAllParam(pattern string) bool {
	// Check for catchall syntax with asterisk
	if strings.Contains(pattern, "{*") {
		return true
	}

	// Skip the check if the pattern contains multiple parameters
	if strings.Count(pattern, "{") > 1 || strings.Count(pattern, "}") > 1 {
		return false
	}

	// This is only for patterns like /path/{param}
	// Where the parameter is the last segment
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	for i, segment := range patternSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			// If the parameter is not the last segment, it's not a catchall
			if i < len(patternSegments)-1 {
				return false
			}
			// This is a parameter at the end of the pattern
			return true
		}
	}

	return false
}

// extractCatchAllParams extracts parameters for catchall patterns
func (e *Executor) extractCatchAllParams(pattern, path string) map[string]string {
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Validate segments count
	if len(patternSegments) < 1 || len(pathSegments) < len(patternSegments)-1 {
		return nil
	}

	// Check if all non-parameter segments match
	for i := 0; i < len(patternSegments)-1; i++ {
		if !strings.HasPrefix(patternSegments[i], "{") || !strings.HasSuffix(patternSegments[i], "}") {
			if patternSegments[i] != pathSegments[i] {
				return nil
			}
		}
	}

	// Extract the last parameter
	lastSegment := patternSegments[len(patternSegments)-1]
	if strings.HasPrefix(lastSegment, "{") && strings.HasSuffix(lastSegment, "}") {
		paramName := lastSegment[1 : len(lastSegment)-1]

		// Handle special case for catchall params with asterisk
		if strings.HasPrefix(paramName, "*") {
			paramName = paramName[1:] // Remove the asterisk
		}

		// Capture all remaining path segments
		remainingPath := strings.Join(pathSegments[len(patternSegments)-1:], "/")

		params := make(map[string]string)
		params[paramName] = remainingPath
		return params
	}

	return nil
}

// extractMultipleParamsPerSegment handles extraction of multiple parameters in the same segment
func (e *Executor) extractMultipleParamsPerSegment(pattern, path string) map[string]string {
	// Split pattern and path into segments
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	// Basic validation - must have same number of segments
	if len(patternSegments) != len(pathSegments) {
		return nil
	}

	params := make(map[string]string)

	// First, check all regular segments (those without multiple params)
	for i, patternSegment := range patternSegments {
		// Skip segments with multiple parameters for now
		if strings.Contains(patternSegment, "}-{") {
			continue
		}

		// Handle regular parameter segments
		if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
			paramName := patternSegment[1 : len(patternSegment)-1]
			if paramName != "" {
				params[paramName] = pathSegments[i]
			}
		} else if patternSegment != pathSegments[i] {
			// Non-parameter segments must match exactly
			return nil
		}
	}

	// Now handle segments with multiple parameters
	for i, patternSegment := range patternSegments {
		if strings.Contains(patternSegment, "}-{") {
			// Process the segment and extract parameter names
			parts := strings.Split(patternSegment, "}-{")

			// Simple handling for segments like {param1}-{param2}-{param3}
			if len(parts) >= 2 && strings.Count(pathSegments[i], "-") == len(parts)-1 {
				values := strings.Split(pathSegments[i], "-")

				for j, part := range parts {
					paramName := ""
					if j == 0 && strings.HasPrefix(part, "{") {
						paramName = part[1:]
					} else if j == len(parts)-1 && strings.HasSuffix(part, "}") {
						paramName = part[:len(part)-1]
					} else {
						paramName = part
					}

					if paramName != "" && j < len(values) {
						params[paramName] = values[j]
					}
				}
			}
		}
	}

	return params
}
