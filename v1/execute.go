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

	"mime/multipart"

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
	FileUploads  map[string][]*multipart.FileHeader
}

// ExecutePHP handles execution of a PHP script through the VFS
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	const (
		emptyJSON      = "{}"
		emptyJSONArray = "[]"
		wrapperPrefix  = "_wrapper_"
		globalsFile    = "/_frango_php_globals.php" // Path to PHP globals script defined in php_globals.go
	)

	m.logger.Printf("========== EXECUTING PHP SCRIPT ==========")
	m.logger.Printf("ExecutePHP: Executing script '%s' with VFS %s", scriptPath, vfs.name)
	m.logger.Printf("ExecutePHP: HTTP Request %s %s", r.Method, r.URL.String())

	// Ensure the VFS has the PHP globals script installed from php_globals.go
	// This creates/_frango_php_globals.php in the VFS which provides:
	// - PHP superglobals ($_PATH, $_GET, $_POST, etc.)
	// - Helper functions (path_param, has_path_param, path_segments)
	// - Utility variables for convenient access ($_URL, $_CURRENT_URL, $_QUERY)
	if err := UpdateVFS(vfs); err != nil {
		m.logger.Printf("Warning: Failed to update VFS with PHP globals: %v", err)
	}

	// 1. Extract request data and prepare environment variables
	requestData := extractRequestData(r)
	envData := prepareEnvironmentData(m, requestData, scriptPath, r, renderFn, w)

	// 2. Resolve PHP script path
	phpFilePath, err := resolveScriptPath(m, vfs, scriptPath, requestData.Path)
	if err != nil {
		http.Error(w, "Server error locating PHP script", http.StatusInternalServerError)
		return
	}

	// 3. Set up document root and script name for FrankenPHP
	documentRoot := filepath.Dir(phpFilePath)
	scriptName := "/" + filepath.Base(phpFilePath)

	m.logger.Printf("Executing PHP script in env: '%s' (from source: '%s')", phpFilePath, scriptPath)
	m.logger.Printf("FrankenPHP Setup: DocumentRoot='%s', ScriptName='%s', URL='%s'",
		documentRoot, scriptName, r.URL.String())

	// 4. Verify and ensure the PHP file exists
	phpFilePath = ensurePhpFileExists(m, phpFilePath, scriptPath)
	if phpFilePath == "" {
		http.Error(w, "Server error: Failed to locate PHP file", http.StatusInternalServerError)
		return
	}

	// 5. Check if wrapper already exists, and create it if needed
	wrapperFilename := wrapperPrefix + filepath.Base(phpFilePath)
	wrapperPath := filepath.Join(vfs.tempDir, wrapperFilename)

	// Check if wrapper exists and is up to date
	recreateWrapper := false
	wrapperInfo, wrapperErr := os.Stat(wrapperPath)
	if os.IsNotExist(wrapperErr) {
		// Wrapper doesn't exist
		recreateWrapper = true
		m.logger.Printf("Wrapper doesn't exist, will create it")
	} else if wrapperErr == nil {
		// Wrapper exists, check if source file is newer
		sourceInfo, sourceErr := os.Stat(phpFilePath)
		if sourceErr == nil && sourceInfo.ModTime().After(wrapperInfo.ModTime()) {
			// Source file is newer than wrapper, recreate
			recreateWrapper = true
			m.logger.Printf("Source file modified after wrapper, will recreate")
		}
	}

	if recreateWrapper {
		// Create or update the wrapper
		if err := createPhpWrapper(vfs, phpFilePath, wrapperPath, globalsFile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		m.logger.Printf("Created/updated wrapper: %s", wrapperPath)
	} else {
		m.logger.Printf("Using existing up-to-date wrapper: %s", wrapperPath)
	}

	// 6. Update paths to use our wrapper
	phpFilePath = wrapperPath
	documentRoot = vfs.tempDir
	scriptName = "/" + filepath.Base(wrapperPath)

	// 7. Set up PHP environment variables
	phpEnv := buildPhpEnvironment(m, phpFilePath, scriptName, documentRoot, requestData,
		scriptPath, vfs.name, envData)

	// 8. Execute the PHP script
	if err := executePhpRequest(m, w, r, documentRoot, scriptName, phpEnv); err != nil {
		http.Error(w, fmt.Sprintf("PHP execution error: %v", err), http.StatusInternalServerError)
		return
	}

	m.logger.Printf("PHP execution completed successfully for '%s'", scriptPath)
	m.logger.Printf("========== PHP EXECUTION COMPLETE ==========")
}

// prepareEnvironmentData extracts and prepares all environment data for PHP execution
func prepareEnvironmentData(m *Middleware, requestData *RequestData, scriptPath string,
	r *http.Request, renderFn RenderData, w http.ResponseWriter) map[string]string {

	const (
		emptyJSON      = "{}"
		emptyJSONArray = "[]"
	)

	envData := make(map[string]string)

	// --- PATH PARAMETERS ---
	extractPathParameters(m, requestData, scriptPath, r, envData)

	// --- URL AND PATH SEGMENTS ---
	marshalToEnv(requestData.PathSegments, "_PATH_SEGMENTS", emptyJSONArray, envData)
	envData["_PATH_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))

	// --- QUERY PARAMETERS ---
	marshalToEnv(requestData.QueryParams, "_GET", emptyJSON, envData)

	// --- FORM DATA ---
	formJSON, err := json.Marshal(requestData.FormData)
	if err == nil {
		formJSONStr := string(formJSON)
		envData["_POST"] = formJSONStr
		envData["_FORM"] = formJSONStr // $_FORM is same as $_POST but preserves naming convention
	} else {
		envData["_POST"] = emptyJSON
		envData["_FORM"] = emptyJSON
	}

	// --- JSON BODY ---
	marshalToEnv(requestData.JSONBody, "_JSON", emptyJSON, envData)

	// --- REQUEST HEADERS ---
	for key, values := range requestData.Headers {
		if len(values) > 0 {
			headerKey := strings.ReplaceAll(strings.ToUpper(key), "-", "_")
			envData["PHP_HEADER_"+headerKey] = values[0]
		}
	}

	// --- FILE UPLOADS ---
	addFileUploads(requestData.FileUploads, envData)

	// --- TEMPLATE VARIABLES ---
	addTemplateVariables(m, renderFn, w, r, envData)

	return envData
}

// marshalToEnv marshals a value to JSON and adds it to the environment data with fallback
func marshalToEnv(value interface{}, key, fallback string, envData map[string]string) {
	if jsonVal, err := json.Marshal(value); err == nil {
		envData[key] = string(jsonVal)
	} else {
		envData[key] = fallback
	}
}

// extractPathParameters extracts path parameters from the request pattern or script path
func extractPathParameters(m *Middleware, requestData *RequestData, scriptPath string,
	r *http.Request, envData map[string]string) {

	var pathParams map[string]string
	pattern := r.Pattern

	// If no pattern set but scriptPath contains parameters, use scriptPath as pattern
	if pattern == "" && strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}") {
		pattern = scriptPath
		m.logger.Printf("Using scriptPath as pattern: %s", pattern)
	}

	if pattern != "" {
		m.logger.Printf("Using pattern: %s", pattern)

		// Use the pattern to extract path parameters
		pathParams = extractPathParams(pattern, requestData.Path)

		// Pre-compute $_PATH superglobal
		if pathParams != nil && len(pathParams) > 0 {
			marshalToEnv(pathParams, "_PATH", "{}", envData)
			m.logger.Printf("Extracted path parameters: %v", pathParams)
		} else {
			envData["_PATH"] = "{}"
		}
	} else {
		envData["_PATH"] = "{}"
		m.logger.Printf("No pattern available, using URL path without parameter extraction: %s", requestData.Path)
	}
}

// addFileUploads adds file upload information to environment data
func addFileUploads(fileUploads map[string][]*multipart.FileHeader, envData map[string]string) {
	if len(fileUploads) > 0 {
		filesData := make(map[string]map[string]interface{})

		for fieldName, fileHeaders := range fileUploads {
			if len(fileHeaders) > 0 {
				fileHeader := fileHeaders[0] // Handle the first file for now (PHP style)

				// Build the file info structure
				fileInfo := map[string]interface{}{
					"name":     fileHeader.Filename,
					"type":     fileHeader.Header.Get("Content-Type"),
					"size":     fileHeader.Size,
					"tmp_name": "/tmp/" + fileHeader.Filename,
					"error":    0,
				}

				filesData[fieldName] = fileInfo
			}
		}

		marshalToEnv(filesData, "_FILES", "{}", envData)
	} else {
		envData["_FILES"] = "{}"
	}
}

// addTemplateVariables adds template variables to environment data
func addTemplateVariables(m *Middleware, renderFn RenderData, w http.ResponseWriter,
	r *http.Request, envData map[string]string) {

	if renderFn == nil {
		envData["_TEMPLATE"] = "{}"
		return
	}

	m.logger.Printf("Calling render function")
	data := renderFn(w, r)
	m.logger.Printf("Render data keys: %v", getMapKeys(data))

	// Pre-compute $_TEMPLATE
	templateVars := make(map[string]interface{})

	for key, value := range data {
		templateVars[key] = value

		// Also add as individual variables
		jsonData, err := json.Marshal(value)
		if err != nil {
			m.logger.Printf("Error marshaling render data for '%s': %v", key, err)
			continue
		}
		m.logger.Printf("Render data for '%s': %s", key, string(jsonData))
		renderVarKey := "PHP_VAR_" + key
		envData[renderVarKey] = string(jsonData)
	}

	marshalToEnv(templateVars, "_TEMPLATE", "{}", envData)
}

// resolveScriptPath resolves the PHP script path in the VFS
func resolveScriptPath(m *Middleware, vfs *VFS, scriptPath string, requestPath string) (string, error) {
	// Try to resolve the script path in the VFS
	phpFilePath, err := vfs.ResolvePath(scriptPath)
	if err == nil {
		return phpFilePath, nil
	}

	m.logger.Printf("Error resolving script path '%s': %v", scriptPath, err)

	// Try to find the file in the source directory
	if m.sourceDir != "" && strings.HasPrefix(scriptPath, "/") {
		sourcePath := filepath.Join(m.sourceDir, filepath.FromSlash(strings.TrimPrefix(scriptPath, "/")))
		m.logger.Printf("Looking for script in source directory: %s", sourcePath)

		if _, err := os.Stat(sourcePath); err == nil {
			// Add the file to the VFS
			if err := vfs.AddSourceFile(sourcePath, scriptPath); err != nil {
				m.logger.Printf("Error adding source file to VFS: %v", err)
			} else {
				// Try to resolve path again
				phpFilePath, err = vfs.ResolvePath(scriptPath)
				if err == nil {
					return phpFilePath, nil
				}
				m.logger.Printf("Error resolving script path after adding from source: %v", err)
			}
		}
	}

	// For script paths with parameters, try to find a physical file
	if strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}") {
		return resolveParameterizedPath(m, vfs, scriptPath)
	}

	return "", fmt.Errorf("script path not resolved: %s", scriptPath)
}

// resolveParameterizedPath tries to find a matching file for a parameterized path pattern
func resolveParameterizedPath(m *Middleware, vfs *VFS, scriptPath string) (string, error) {
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
				phpFilePath, err := vfs.ResolvePath(f)
				if err == nil {
					return phpFilePath, nil
				}
			}
		}
	}

	// If we still couldn't find a match, try looking for files with {param} literally in the name
	phpFilePath, err := vfs.ResolvePathLiteral(scriptPath)
	m.logger.Printf("Tried literal path resolution: %v", err)

	if err != nil {
		return "", err
	}

	return phpFilePath, nil
}

// ensurePhpFileExists verifies that the PHP file exists and attempts to locate it if not
func ensurePhpFileExists(m *Middleware, phpFilePath, scriptPath string) string {
	m.logger.Printf("Checking if resolved phpFilePath exists: %s", phpFilePath)
	_, fileErr := os.Stat(phpFilePath)
	if fileErr == nil {
		m.logger.Printf("Resolved phpFilePath exists: %s", phpFilePath)
		return phpFilePath
	}

	m.logger.Printf("WARNING: Cannot access phpFilePath: %v", fileErr)

	// Try to find it in the source directory
	if m.sourceDir != "" && !filepath.IsAbs(phpFilePath) {
		// Try direct path in source directory
		sourcePath := filepath.Join(m.sourceDir, phpFilePath)
		m.logger.Printf("Trying source path: %s", sourcePath)

		if _, err := os.Stat(sourcePath); err == nil {
			m.logger.Printf("Found file in source directory: %s", sourcePath)
			return sourcePath
		}

		// Try stripping a prefix
		parts := strings.SplitN(phpFilePath, "/", 2)
		if len(parts) > 1 {
			sourcePath = filepath.Join(m.sourceDir, parts[1])
			m.logger.Printf("Trying source path (without prefix): %s", sourcePath)
			if _, err := os.Stat(sourcePath); err == nil {
				m.logger.Printf("Found file after stripping prefix: %s", sourcePath)
				return sourcePath
			}
		}
	}

	// Final verification
	_, fileErr = os.Stat(phpFilePath)
	if fileErr != nil {
		m.logger.Printf("ERROR: Failed to locate PHP file after all resolution attempts: %v", fileErr)
		return ""
	}

	return phpFilePath
}

// createPhpWrapper copies the PHP file to the VFS temp directory and creates a wrapper script
func createPhpWrapper(vfs *VFS, phpFilePath, wrapperPath, globalsFile string) error {
	// 1. Copy the target PHP file to the VFS temp directory
	targetFilename := filepath.Base(phpFilePath)
	targetPath := filepath.Join(vfs.tempDir, targetFilename)

	// Read the original file content
	originalContent, err := os.ReadFile(phpFilePath)
	if err != nil {
		return fmt.Errorf("Server error reading PHP file: %v", err)
	}

	// Write it to the target location
	if err := os.WriteFile(targetPath, originalContent, 0644); err != nil {
		return fmt.Errorf("Server error preparing PHP file: %v", err)
	}

	// Path to the globals file - handle the leading slash by trimming it for filepath.Join
	globalsFilePath := filepath.Join(vfs.tempDir, strings.TrimPrefix(globalsFile, "/"))

	// Create a simple wrapper that includes the LOCAL copy of the original script
	wrapperContent := fmt.Sprintf(`<?php
// Auto-generated wrapper
require_once '%s'; // Load globals
include './%s'; // Include the local copy
?>`, globalsFilePath, targetFilename)

	// Create the wrapper script
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		return fmt.Errorf("Server error creating wrapper: %v", err)
	}

	return nil
}

// buildPhpEnvironment builds the PHP environment variables for execution
func buildPhpEnvironment(m *Middleware, phpFilePath, scriptName, documentRoot string,
	requestData *RequestData, scriptPath, vfsName string, envData map[string]string) map[string]string {

	// Base PHP environment
	phpEnv := map[string]string{
		// CRITICAL: Set SCRIPT_FILENAME explicitly to the absolute file path
		"SCRIPT_FILENAME": phpFilePath, // Absolute path to the script

		// CRITICAL: These environment variables must be set correctly
		"SCRIPT_NAME":    scriptName,          // Must match the basename of the script
		"PHP_SELF":       scriptName,          // Match SCRIPT_NAME
		"DOCUMENT_ROOT":  documentRoot,        // Must be the parent directory of the script
		"REQUEST_URI":    requestData.FullURL, // Use the same full URL
		"REQUEST_METHOD": requestData.Method,
		"QUERY_STRING":   requestData.FullURL, // This will be parsed from the URL in phpServerEnv
		"HTTP_HOST":      requestData.Headers.Get("Host"),
		"REMOTE_ADDR":    requestData.RemoteAddr,

		// Debugging info
		"DEBUG_DOCUMENT_ROOT": documentRoot,
		"DEBUG_SCRIPT_NAME":   scriptName,
		"DEBUG_PHP_FILE_PATH": phpFilePath, // Full path for debugging
		"DEBUG_SOURCE_PATH":   scriptPath,
		"DEBUG_ENV_ID":        vfsName,
	}

	// Add extracted data
	for key, value := range envData {
		phpEnv[key] = value
	}

	// Set PHP configuration options
	if m.developmentMode {
		phpEnv["PHP_FCGI_MAX_REQUESTS"] = "1" // Disable PHP-FPM keepalive
	} else {
		phpEnv["PHP_OPCACHE_ENABLE"] = "1" // Enable opcache in production
	}

	// Set explicit PHP timeouts
	phpEnv["PHP_MAX_EXECUTION_TIME"] = "10" // 10 second timeout
	phpEnv["PHP_DEFAULT_SOCKET_TIMEOUT"] = "10"

	// Set PHP error display settings
	if m.displayErrors {
		phpEnv["display_errors"] = "1"
		phpEnv["display_startup_errors"] = "1"
		phpEnv["error_reporting"] = "E_ALL"
	} else {
		phpEnv["display_errors"] = "0"
		phpEnv["display_startup_errors"] = "0"
		phpEnv["error_reporting"] = "E_ERROR | E_PARSE"
	}

	return phpEnv
}

// executePhpRequest executes the PHP script using FrankenPHP
func executePhpRequest(m *Middleware, w http.ResponseWriter, r *http.Request,
	documentRoot, scriptName string, phpEnv map[string]string) error {

	// Log environment setup
	m.logger.Printf("Total PHP environment variables: %d", len(phpEnv))

	// CRITICAL: Modify the request clone path to match the script name
	reqClone := r.Clone(r.Context())
	reqClone.URL.Path = scriptName
	m.logger.Printf("Modified request clone path for FrankenPHP: %s", reqClone.URL.Path)

	// Dump environment variables for debugging
	logEnvironmentVariables(m, phpEnv)

	// Create FrankenPHP request
	req, err := frankenphp.NewRequestWithContext(
		reqClone, // Use the modified request with the script path
		frankenphp.WithRequestDocumentRoot(documentRoot, false), // Exact document root
		frankenphp.WithRequestEnv(phpEnv),                       // All environment variables
	)
	if err != nil {
		m.logger.Printf("Error creating PHP request: %v", err)
		return fmt.Errorf("Server error creating PHP request: %v", err)
	}

	// Execute the PHP script
	if err := frankenphp.ServeHTTP(w, req); err != nil {
		m.logger.Printf("Error executing PHP script: %v", err)
		return err
	}

	return nil
}

// logEnvironmentVariables logs all environment variables in sorted order
func logEnvironmentVariables(m *Middleware, phpEnv map[string]string) {
	m.logger.Printf("ExecutePHP: ======= FULL PHP ENVIRONMENT =======")
	var keys []string
	for k := range phpEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := phpEnv[k]
		m.logger.Printf("  %s = %s", k, v)
	}
	m.logger.Printf("ExecutePHP: ===================================")
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
		JSONBody:    make(map[string]interface{}),
		FormData:    make(map[string][]string),
		FileUploads: make(map[string][]*multipart.FileHeader),
	}

	// Check if method might include a request body (most methods except GET and HEAD)
	// Include DELETE explicitly since modern APIs often use request bodies with DELETE
	if r.Method != "GET" && r.Method != "HEAD" {
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
		} else if strings.Contains(contentType, "multipart/form-data") {
			// For multipart form data with file uploads
			if err := r.ParseMultipartForm(32 << 20); err == nil { // 32MB max memory
				// Get form values
				if r.MultipartForm != nil {
					// Extract form values
					for key, values := range r.MultipartForm.Value {
						data.FormData[key] = values
					}

					// Extract file uploads
					for key, fileHeaders := range r.MultipartForm.File {
						data.FileUploads[key] = fileHeaders
					}
				}
			}
		} else {
			// For regular form data
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
