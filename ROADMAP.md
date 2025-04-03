# Frango Refactoring Roadmap

## Vision

Create a cohesive, VFS-centric middleware library for PHP integration in Go applications with an intuitive API. The library should maintain excellent performance via caching while supporting live development through file watching.

## Target API

```go
// Core middleware with base VFS
php, err := frango.New(
    frango.WithSourceDir("./php"),  // Create root VFS with files at `/`
    frango.WithDevelopmentMode(true),
)

// VFS operations - inheritance and branching
mainVFS := php.NewVFS()  // Inherits root VFS files
mainVFS.AddSourceDirectory("./templates", "/views")
mainVFS.AddEmbeddedDirectory(embedFS, "assets", "/assets")
mainVFS.CreateVirtualFile("/config.php", []byte("<?php return ['debug' => true];"))

// Branch VFS - inherits structure but maintains independence
apiVFS := mainVFS.Branch()
apiVFS.AddSourceDirectory("./api-templates", "/api-views")

// Use NewRouter to scan a VFS and generate routes
routes, err := php.NewRouter(
    mainVFS,
    "web",                // Directory within VFS to scan
    "/app",               // URL prefix for routes
    &frango.FileSystemRouteOptions{
        GenerateCleanURLs: true,        // Remove .php extension in URLs
        GenerateIndexRoutes: true,      // Map index.php to directory paths
        DetectMethodByFilename: true,   // Support .get.php, .post.php, etc.
    },
)
if err != nil {
    log.Fatalf("Error generating routes: %v", err)
}

// Standard Go HTTP router
mux := http.NewServeMux()

// Add generated routes to mux
for _, route := range routes {
    mux.Handle(route.Pattern, route.Handler)
}

// Direct middleware handlers specifying which VFS to use
mux.Handle("/", php.For(mainVFS, "index.php"))
mux.Handle("/users/{id}", php.For(mainVFS, "users/profile.php"))
mux.Handle("/dashboard", php.Render(mainVFS, "dashboard.php", dashboardDataFn))
mux.Handle("/api/users", php.For(apiVFS, "api-views/users.php"))
```

## Core Components to Consolidate

1. **Middleware Core** (frango.go)
   - Central middleware definition
   - Option handling
   - Base VFS initialization

2. **Virtual Filesystem** (vfs.go)
   - Complete VFS implementation with inheritance
   - Branching mechanism
   - File watching and hash-based updates

3. **Execution Engine** (execute.go)
   - PHP script execution
   - Request data extraction
   - Environment preparation

4. **Router Generation** (router.go)
   - Convention-based route generation
   - Path parameter handling
   - File-to-URL mapping

## Critical Features to Preserve

- **File Management & Caching**
  - Environment caching for PHP scripts
  - Hash-based file change detection
  - Efficient file copying and mirroring

- **Live Development**
  - File watching for source directories
  - Auto-refresh on file changes
  - Development mode with logging

- **PHP Integration**
  - Complete HTTP request data access
  - Path parameter handling via `$_PATH` and `$_PATH_SEGMENTS`
  - Render data injection
  - Environment variable standardization

- **Request Processing**
  - FrankenPHP integration
  - Environment variable management
  - Script execution with proper context
  - Request data extraction and normalization

## Critical PHP Execution Flow

The `executePHP` function structure represents a carefully crafted execution flow whose core mechanics must be preserved:

1. **Request Data Extraction** - Comprehensive extraction of all HTTP request components
2. **Environment Variable Preparation** - **(NEEDS IMPROVEMENT)** - Make more PHP-friendly while preserving functionality
3. **Environment Cache Management** - Retrieving/creating isolated PHP environments
4. **Script Path Resolution** - Precise path calculation within the environment
5. **File Verification & Rebuilding** - Script existence checks with automatic rebuilding
6. **FrankenPHP Configuration** - Exact setup of document root, script name, and paths
7. **Request Transformation** - Cloning and path modification for FrankenPHP
8. **Execution & Error Handling** - Proper PHP execution with error capture

The entire implementation of this function must be preserved intact while adapting it to work with the new VFS architecture.

## The Critical executePHP Implementation

This is the exact implementation that must be preserved and improved:

```go
func (m *Middleware) executePHP(absScriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	// 1. Prepare environment data (render vars + path params)
	envData := make(map[string]string)

	// Extract all request data in a single clean step
	requestData := ExtractRequestData(r)

	// Add path segments (array indexes start at 0) - RAW DATA ONLY
	for i, segment := range requestData.PathSegments {
		envData["FRANGO_URL_SEGMENT_"+strconv.Itoa(i)] = segment
	}

	// Also provide the number of segments
	envData["FRANGO_URL_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))

	// Add raw path
	envData["FRANGO_URL_PATH"] = requestData.Path

	// --- Extract path parameters from pattern ---
	var pathParams map[string]string

	// Get the actual route pattern from the request's context if available
	if patternKey := php12PatternContextKey(r.Context()); patternKey != "" {
		// Use the pattern to extract path parameters
		pathParams = extractPathParams(patternKey, requestData.Path)

		if pathParams != nil && len(pathParams) > 0 {
			// Add individual path parameters with FRANGO_PARAM_ prefix (for backwards compatibility)
			for name, value := range pathParams {
				envData["FRANGO_PARAM_"+name] = value
			}

			// Also add serialized path parameters as JSON
			if jsonParams, err := json.Marshal(pathParams); err == nil {
				envData["FRANGO_PATH_PARAMS_JSON"] = string(jsonParams)
			}
		}

		m.logger.Printf("Extracted path parameters: %v", pathParams)
	} else {
		// Check for any path parameters set in environment variables (for tests)
		paramsJSON := os.Getenv("FRANGO_PATH_PARAMS_JSON")
		if paramsJSON != "" {
			m.logger.Printf("Found FRANGO_PATH_PARAMS_JSON in environment: %s", paramsJSON)
			envData["FRANGO_PATH_PARAMS_JSON"] = paramsJSON
		}

		// Check for individual parameter variables
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, "FRANGO_PARAM_") {
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

	// Add all query parameters with FRANGO_QUERY_ prefix
	for key, values := range requestData.QueryParams {
		if len(values) > 0 {
			envData["FRANGO_QUERY_"+key] = values[0]
		}
	}

	// Add form data with FRANGO_FORM_ prefix
	for key, values := range requestData.FormData {
		if len(values) > 0 && !strings.HasPrefix(key, "FRANGO_") { // Avoid overrides
			envData["FRANGO_FORM_"+key] = values[0]
		}
	}

	// Add JSON body data with FRANGO_JSON_ prefix if available
	if requestData.JSONBody != nil {
		for key, value := range requestData.JSONBody {
			// Convert each JSON value to string
			if strValue, err := json.Marshal(value); err == nil {
				envData["FRANGO_JSON_"+key] = string(strValue)
			}
		}

		// Also provide the full JSON body
		if fullJSON, err := json.Marshal(requestData.JSONBody); err == nil {
			envData["FRANGO_JSON_BODY"] = string(fullJSON)
		}
	}

	// Add selected important headers with FRANGO_HEADER_ prefix
	for key, values := range requestData.Headers {
		if len(values) > 0 {
			headerKey := strings.ReplaceAll(strings.ToUpper(key), "-", "_")
			envData["FRANGO_HEADER_"+headerKey] = values[0]
		}
	}

	// Populate Render Data if renderFn is provided
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
			renderVarKey := "FRANGO_VAR_" + key
			envData[renderVarKey] = string(jsonData)
		}
	}

	// 2. Get or create PHP execution environment
	// Ensure no query strings in script path passed to cache
	cleanAbsScriptPath := absScriptPath
	if queryIndex := strings.Index(cleanAbsScriptPath, "?"); queryIndex != -1 {
		cleanAbsScriptPath = cleanAbsScriptPath[:queryIndex]
	}
	// Use the absolute script path as the key for the environment cache
	env, err := m.envCache.GetEnvironment(cleanAbsScriptPath, cleanAbsScriptPath)
	if err != nil {
		m.logger.Printf("Error setting up environment for script '%s': %v", cleanAbsScriptPath, err)
		http.Error(w, "Server error preparing PHP environment", http.StatusInternalServerError)
		return
	}

	// 3. Get the pre-calculated relative path and construct the final path in the environment
	relPath := env.ScriptRelPath
	if relPath == "" {
		m.logger.Printf("Internal Error: ScriptRelPath not found in environment for script '%s'", cleanAbsScriptPath)
		http.Error(w, "Server error locating script in environment", http.StatusInternalServerError)
		return
	}
	phpFilePathInEnv := filepath.Join(env.TempPath, relPath)
	m.logger.Printf("Executing PHP script in env: '%s' (from source: '%s')", phpFilePathInEnv, absScriptPath)

	// 4. Verify script file exists
	fileInfo, err := os.Stat(phpFilePathInEnv)
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Printf("PHP script not found in environment: '%s'. Attempting rebuild...", phpFilePathInEnv)
			if err := m.envCache.populateEnvironmentFiles(env); err != nil {
				m.logger.Printf("Error rebuilding environment for missing file: %v", err)
				http.Error(w, "Server error locating script (rebuild failed)", http.StatusInternalServerError)
				return
			}
			fileInfo, err = os.Stat(phpFilePathInEnv) // Check again
			if err != nil {
				m.logger.Printf("PHP script '%s' still not found after rebuild: %v", phpFilePathInEnv, err)
				http.NotFound(w, r) // Or internal server error?
				return
			}
		} else {
			m.logger.Printf("Error stating PHP script '%s': %v", phpFilePathInEnv, err)
			http.Error(w, "Server error locating script", http.StatusInternalServerError)
			return
		}
	}
	if fileInfo.IsDir() {
		m.logger.Printf("ERROR: Target script path is a directory: '%s'", phpFilePathInEnv)
		http.Error(w, "Configuration error: script path is a directory", http.StatusInternalServerError)
		return
	}

	// 5. Prepare FrankenPHP request options
	// Document root is the PARENT directory of the script within the temp env
	documentRoot := filepath.Dir(phpFilePathInEnv)
	scriptName := "/" + relPath // Ensure leading slash

	// If relPath already contains directory components, use just the filename
	// This avoids paths like "/routing/routing/file.php"
	if strings.Contains(relPath, "/") {
		scriptName = "/" + filepath.Base(relPath)
	}

	m.logger.Printf("FrankenPHP Setup: DocumentRoot='%s', ScriptName='%s', URL='%s'", documentRoot, scriptName, r.URL.String())

	// Path globals PHP file
	pathGlobalsFile := filepath.Join(env.TempPath, "_frango_path_globals.php")

	// Check if the path globals file exists
	_, pathGlobalsErr := os.Stat(pathGlobalsFile)
	if pathGlobalsErr != nil {
		m.logger.Printf("Warning: Path globals file not found: %v", pathGlobalsErr)
	} else {
		// Add auto-prepend file to include path globals
		m.logger.Printf("Adding path globals auto-prepend: %s", pathGlobalsFile)

		// Auto-prepend doesn't work consistently across PHP versions and environments
		// Instead, we'll explicitly set it via the environment
		envData["PHP_AUTO_PREPEND_FILE"] = pathGlobalsFile
		envData["PHP_INCLUDE_PATH"] = env.TempPath
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
		"DEBUG_PHP_FILE_PATH": phpFilePathInEnv, // Full path for debugging
		"DEBUG_SOURCE_PATH":   absScriptPath,
		"DEBUG_ENV_ID":        env.ID,
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

	req, err := frankenphp.NewRequestWithContext(
		reqClone, // Use the modified request
		frankenphp.WithRequestDocumentRoot(documentRoot, false), // Parent dir as DocRoot
		frankenphp.WithRequestEnv(phpBaseEnv),                   // Env *without* SCRIPT_FILENAME
	)
	if err != nil {
		m.logger.Printf("Error creating PHP request: %v", err)
		http.Error(w, "Server error creating PHP request", http.StatusInternalServerError)
		return
	}

	if err := frankenphp.ServeHTTP(w, req); err != nil {
		m.logger.Printf("Error executing PHP script '%s': %v", phpFilePathInEnv, err)
		http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

## FrankenPHP Integration

There is ONE FrankenPHP instance in the middleware (initialized once), but each endpoint request creates its own execution context:

```go
// ONE initialization in the middleware
frankenphp.Init() // Called only once during middleware initialization

// But EACH request gets its own context
req, err := frankenphp.NewRequestWithContext(
    reqClone, // Modified request clone with adjusted path
    frankenphp.WithRequestDocumentRoot(documentRoot, false), // Script parent dir
    frankenphp.WithRequestEnv(phpBaseEnv), // Custom environment variables
)

frankenphp.ServeHTTP(w, req) // Execution
```

This approach ensures isolation between requests while maintaining performance.

## Request Processing Pipeline

The `executePHP` function contains critical logic that must be preserved, but we should improve the PHP developer experience:

1. **PHP-First Data Access**
   - PHP developers should access data through familiar patterns:
     - Standard superglobals: `$_GET`, `$_POST`, `$_SERVER`, `$_FILES`
     - Simple path parameter access: `$_PATH['id']` or `$path['id']`
     - Intuitive URL segments: `$_PATH_SEGMENTS[0]` or `$segments[0]`
     - JSON data via `$_JSON` (easier than decoding manually)
     - Template data via direct variable access: `$user`, `$items` (unwrapped)
   
   Implementation should hide the technical details (like environment variables) and provide a clean, intuitive interface that feels native to PHP developers.

2. **Isolated PHP Runtime Per Endpoint**
   - Each Go endpoint gets its own isolated PHP runtime environment
   - Independent environment variables and execution context per handler
   - No shared state between different PHP handlers (security and stability)
   - FrankenPHP configured uniquely for each specific endpoint
   - Environment setup optimized for the specific endpoint's needs

## PHP Environment Variable Improvement

The environment variable naming will be improved to be more PHP-friendly while maintaining the same pattern:

### Current Implementation (Part that needs improvement)
```go
// Current approach in executePHP
// Add path segments
for i, segment := range requestData.PathSegments {
    envData["FRANGO_URL_SEGMENT_"+strconv.Itoa(i)] = segment
}
envData["FRANGO_URL_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))
envData["FRANGO_URL_PATH"] = requestData.Path

// Path parameters
for name, value := range pathParams {
    envData["FRANGO_PARAM_"+name] = value
}
// JSON version of path params
if jsonParams, err := json.Marshal(pathParams); err == nil {
    envData["FRANGO_PATH_PARAMS_JSON"] = string(jsonParams)
}

// JSON body
if fullJSON, err := json.Marshal(requestData.JSONBody); err == nil {
    envData["FRANGO_JSON_BODY"] = string(fullJSON)
}

// Render data
for key, value := range data {
    jsonData, _ := json.Marshal(value)
    envData["FRANGO_VAR_"+key] = string(jsonData)
}
```

### Improved Implementation
```go
// New approach with better naming
// Add path segments
for i, segment := range requestData.PathSegments {
    envData["PHP_PATH_SEGMENT_"+strconv.Itoa(i)] = segment
}
envData["PHP_PATH_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))
envData["PHP_PATH"] = requestData.Path

// Path parameters
for name, value := range pathParams {
    envData["PHP_PATH_PARAM_"+name] = value
}
// JSON version of path params
if jsonParams, err := json.Marshal(pathParams); err == nil {
    envData["PHP_PATH_PARAMS"] = string(jsonParams)
}

// JSON body
if fullJSON, err := json.Marshal(requestData.JSONBody); err == nil {
    envData["PHP_JSON"] = string(fullJSON)
}

// Render data
for key, value := range data {
    jsonData, _ := json.Marshal(value)
    envData["PHP_VAR_"+key] = string(jsonData)
}
```

### VFS Integration for path_globals.php

Instead of a physical file, `path_globals.php` will be defined as a string constant in Go code:

```go
// Define standard PHP globals script that's injected into every VFS
const pathGlobalsPHP = `<?php
// Convert environment variables to PHP superglobals
$_PATH = [];
if (($json = getenv('PHP_PATH_PARAMS')) !== false) {
    $_PATH = json_decode($json, true) ?: [];
} else {
    // Fallback to individual variables
    foreach ($_ENV as $key => $val) {
        if (strpos($key, 'PHP_PATH_PARAM_') === 0) {
            $name = substr($key, 15);
            $_PATH[$name] = $val;
        }
    }
}

// PATH_SEGMENTS handling
$_PATH_SEGMENTS = [];
$segmentCount = (int)getenv('PHP_PATH_SEGMENT_COUNT');
for ($i = 0; $i < $segmentCount; $i++) {
    $_PATH_SEGMENTS[$i] = getenv('PHP_PATH_SEGMENT_'.$i);
}
$_PATH_SEGMENT_COUNT = $segmentCount;

// JSON body handling
if (($json = getenv('PHP_JSON')) !== false) {
    $_JSON = json_decode($json, true) ?: [];
} else {
    $_JSON = [];
}

// Set path in $_SERVER
$_SERVER['PATH'] = getenv('PHP_PATH') ?: '';

// Handle direct variables from template data
foreach ($_ENV as $key => $val) {
    if (strpos($key, 'PHP_VAR_') === 0) {
        $name = substr($key, 8);
        $$name = json_decode($val, true);
    }
}
?>`

// This string will be automatically added as a virtual file to every VFS
func (vfs *VirtualFS) initialize() {
    // Add the globals file to the VFS
    vfs.CreateVirtualFile("/_frango_php_globals.php", []byte(pathGlobalsPHP))
    vfs.phpGlobalsFile = "/_frango_php_globals.php"
}
```

## PHP Access API Improvement

The current environment variable system with `FRANGO_PARAM_*`, `FRANGO_QUERY_*`, etc. prefixes will be refactored to provide a more intuitive PHP developer experience:

### Current (Technical) Approach
```php
// Current approach uses technical environment variables
$id = getenv('FRANGO_PARAM_id');
$name = getenv('FRANGO_QUERY_name');
$jsonBody = json_decode(getenv('FRANGO_JSON_BODY'), true);
$urlPath = getenv('FRANGO_URL_PATH');
$segment = getenv('FRANGO_URL_SEGMENT_1');
```

### New (Intuitive) Approach
```php
// New approach will use PHP-native superglobals
$id = $_PATH['id'];              // Path parameters
$name = $_GET['name'];           // Native PHP query access (unchanged)
$jsonBody = $_JSON;              // Parsed JSON body
$urlPath = $_SERVER['PATH'];     // Full path
$segment = $_PATH_SEGMENTS[1];   // Path segments

// For debugging/development
echo "Full path: {$_SERVER['PATH']}";
echo "Current segment: {$_PATH_SEGMENTS[1]}";
echo "Total segments: {$_PATH_SEGMENT_COUNT}";
```

The goal is to make PHP code more intuitive while maintaining the core integration with FrankenPHP and preserving the entire execution flow.

## Implementation Steps

1. Establish VFS as the central component with inheritance and branching
2. Refactor handler creation to explicitly use VFS instances
3. Consolidate routing logic into a single implementation
4. Ensure environment isolation between handlers
5. Preserve the execution engine functionality
6. Update examples to demonstrate the new API

## File Structure

```
frango.go       - Core middleware
vfs.go          - Virtual filesystem
execute.go      - PHP execution engine
router.go       - Convention-based routing
utils.go        - Utility functions
options.go      - Option definitions
```

This refactoring will address the code fragmentation issues while preserving all existing functionality and providing a more intuitive, VFS-centric API.
