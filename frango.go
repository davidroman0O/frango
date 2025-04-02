package frango

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dunglas/frankenphp"
)

// --- Core Types (Exported) ---

// Middleware is the core PHP execution engine.
// It does not handle routing itself but provides http.Handler instances for integration.
type Middleware struct {
	sourceDir          string // Resolved absolute path to user's PHP source files
	tempDir            string // Base temporary directory for this instance
	logger             *log.Logger
	initialized        bool
	initLock           sync.Mutex
	developmentMode    bool
	blockDirectPHPURLs bool              // Whether to block direct .php access in URLs
	envCache           *environmentCache // Internal cache for PHP environments
}

// Option is a function that configures a Middleware.
type Option func(*Middleware)

// RenderData is a function that returns data to be passed to a PHP template.
// It's used with RenderHandlerFor.
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}

// --- Constructor (Exported) ---

// New creates a new PHP middleware instance (execution engine).
func New(opts ...Option) (*Middleware, error) {
	// Default configuration
	m := &Middleware{
		developmentMode:    true,
		blockDirectPHPURLs: true, // Default to blocking direct PHP access in URLs
		logger:             log.New(os.Stdout, "[frango] ", log.LstdFlags),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Resolve source directory (optional, can be empty)
	var absSourceDir string
	var err error
	if m.sourceDir == "" {
		// Create a minimal temp dir if no source provided (for embeds/cache)
		absSourceDir, err = os.MkdirTemp("", "frango-nosource-")
		if err != nil {
			return nil, fmt.Errorf("error creating temporary source directory: %w", err)
		}
		m.logger.Printf("No SourceDir provided, using temp dir: %s", absSourceDir)
	} else {
		absSourceDir, err = resolveDirectory(m.sourceDir)
		if err != nil {
			return nil, fmt.Errorf("error resolving source directory: %w", err)
		}
	}
	m.sourceDir = absSourceDir

	// Create base temporary directory for environments and embeds
	tempDir, err := os.MkdirTemp("", "frango-instance-")
	if err != nil {
		return nil, fmt.Errorf("error creating base temporary directory: %w", err)
	}
	m.tempDir = tempDir

	// Create dedicated subdirectory for embedded files
	embedTempDir := filepath.Join(m.tempDir, "_frango_embeds")
	if err := os.MkdirAll(embedTempDir, 0755); err != nil {
		os.RemoveAll(m.tempDir) // Cleanup base temp dir
		return nil, fmt.Errorf("error creating embeds temp directory: %w", err)
	}

	// Create environment cache
	m.envCache = newEnvironmentCache(m.sourceDir, m.tempDir, m.logger, m.developmentMode)

	return m, nil
}

// --- Public Methods (Exported) ---

// Shutdown cleans up resources (environments, temp files).
func (m *Middleware) Shutdown() {
	if m.initialized {
		frankenphp.Shutdown()
		m.initialized = false
	}
	if m.envCache != nil {
		m.envCache.Cleanup()
	}
	// Remove the base temp directory for this instance
	if err := os.RemoveAll(m.tempDir); err != nil {
		m.logger.Printf("Warning: Failed to remove base temp directory %s: %v", m.tempDir, err)
	}
}

// HandlerFor returns an http.Handler that executes the specified PHP script.
// scriptPath can be relative to the SourceDir or an absolute path.
// The registered pattern (e.g., "GET /users/{id}") must be passed for param extraction.
func (m *Middleware) HandlerFor(registeredPattern string, scriptPath string) http.Handler {
	// Resolve script path immediately if relative
	absScriptPath := m.resolveScriptPath(scriptPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block direct PHP access in URLs if enabled
		if m.blockDirectPHPURLs && strings.HasSuffix(strings.ToLower(r.URL.Path), ".php") {
			// Extract the registered URL pattern without method
			urlPattern := registeredPattern
			if parts := strings.SplitN(registeredPattern, " ", 2); len(parts) > 1 {
				urlPattern = parts[1]
			}

			// Special case: If this is explicitly registered as "/index.php" or similar, allow it
			if urlPattern == r.URL.Path {
				m.logger.Printf("Allowing explicitly registered PHP path: %s", r.URL.Path)
			} else {
				m.logger.Printf("Blocked direct access to PHP file in URL: %s", r.URL.Path)
				http.Error(w, "Not Found: Direct PHP file access is not allowed", http.StatusNotFound)
				return
			}
		}

		// Initialization check
		if !m.ensureInitialized(r.Context()) {
			http.Error(w, "PHP initialization error", http.StatusInternalServerError)
			return
		}
		// Execute PHP (pass pattern for param extraction)
		m.executePHP(registeredPattern, absScriptPath, nil, w, r)
	})
}

// RenderHandlerFor returns an http.Handler that executes the specified PHP script,
// passing data returned by the renderFn.
// scriptPath can be relative to the SourceDir or an absolute path.
// The registered pattern (e.g., "GET /posts/{id}") must be passed for param extraction.
func (m *Middleware) RenderHandlerFor(registeredPattern string, scriptPath string, renderFn RenderData) http.Handler {
	// Resolve script path immediately if relative
	absScriptPath := m.resolveScriptPath(scriptPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initialization check
		if !m.ensureInitialized(r.Context()) {
			http.Error(w, "PHP initialization error", http.StatusInternalServerError)
			return
		}
		// Execute PHP with render function (pass pattern for param extraction)
		m.executePHP(registeredPattern, absScriptPath, renderFn, w, r)
	})
}

// AddEmbeddedLibrary adds a PHP utility/library file from an embed.FS.
// It writes the file to a temporary location and registers it with the cache
// to be copied into PHP environments when they are created/updated.
// targetLibraryPath determines the path where the library will be available
// inside the PHP environment (e.g., "/lib/utils.php" -> envTmp/lib/utils.php).
func (m *Middleware) AddEmbeddedLibrary(embedFS embed.FS, embedPath string, targetLibraryPath string) (string, error) {
	content, err := embedFS.ReadFile(embedPath)
	if err != nil {
		m.logger.Printf("Error reading embedded library file %s: %v", embedPath, err)
		return "", fmt.Errorf("failed to read embedded library %s: %w", embedPath, err)
	}

	// Ensure targetLibraryPath is relative and clean
	relativeEmbedPath := strings.TrimPrefix(targetLibraryPath, "/")
	if relativeEmbedPath == "" {
		return "", fmt.Errorf("invalid empty target path for embedded library")
	}
	relativeEmbedPath = filepath.Clean(relativeEmbedPath)

	// Create the target path within the dedicated embeds temp directory
	embedTempBaseDir := filepath.Join(m.tempDir, "_frango_embeds")
	targetDiskPath := filepath.Join(embedTempBaseDir, relativeEmbedPath)

	// Create directory structure
	if targetDir := filepath.Dir(targetDiskPath); targetDir != "" {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			m.logger.Printf("Warning: Failed to create directory for embedded library %s: %v", targetDiskPath, err)
			// Proceed anyway, WriteFile might still work or fail clearly
		}
	}

	// Write file
	if err := os.WriteFile(targetDiskPath, content, 0644); err != nil {
		m.logger.Printf("Warning: Failed to write embedded library file %s: %v", targetDiskPath, err)
		return "", fmt.Errorf("failed to write embedded library file %s: %w", targetDiskPath, err)
	}

	m.logger.Printf("Added embedded PHP library for path %s (temp path: %s)", targetLibraryPath, targetDiskPath)

	// Register this library with the environment cache
	m.envCache.AddGlobalLibrary(relativeEmbedPath, targetDiskPath)

	return targetDiskPath, nil
}

// SourceDir returns the resolved absolute path to the source directory being used.
func (m *Middleware) SourceDir() string {
	return m.sourceDir
}

// --- Filesystem Routing Utility (Exported) ---

// FileSystemRoute represents a discovered route from the filesystem.
type FileSystemRoute struct {
	Method     string       // HTTP method (GET, POST, etc.) or "" for ANY
	Pattern    string       // The URL pattern (e.g., "/users/{id}", "/posts/welcome")
	Handler    http.Handler // The generated frango handler for the script
	ScriptPath string       // Source path of the script relative to the scanned filesystem root
}

// --- Filesystem Routing Options (Enums and Struct) ---

// OptionSetting defines explicit states for boolean-like options.
type OptionSetting int

const (
	// OptionDefault uses the function's default behavior.
	OptionDefault OptionSetting = iota // 0
	// OptionEnabled explicitly enables the feature.
	OptionEnabled // 1
	// OptionDisabled explicitly disables the feature.
	OptionDisabled // 2
)

// FileSystemRouteOptions provides configuration for MapFileSystemRoutes.
type FileSystemRouteOptions struct {
	// GenerateCleanURLs: Controls generation of routes without .php extension.
	// Default behavior is OptionEnabled.
	GenerateCleanURLs OptionSetting
	// GenerateIndexRoutes: Controls generation of routes for index.php at directory level.
	// Default behavior is OptionEnabled.
	GenerateIndexRoutes OptionSetting
	// DetectMethodByFilename: Controls checking for .METHOD.php patterns.
	// Default behavior is OptionDisabled.
	DetectMethodByFilename OptionSetting
}

// MapFileSystemRoutes scans a directory (`scanDir`) within a filesystem (`targetFS`)
// and generates a slice of FileSystemRoute structs based on the found PHP files.
// It maps these files to URL paths relative to `urlPrefix`.
// Assumes targetFS root corresponds to the frangoInstance's SourceDir for script path resolution.
func MapFileSystemRoutes(
	frangoInstance *Middleware,
	targetFS fs.FS, // Filesystem to scan (e.g., os.DirFS("pages"), embed.FS)
	scanDir string, // Subdirectory within targetFS to start scanning (e.g., ".")
	urlPrefix string, // URL prefix for generated routes (e.g., "/", "/app")
	options *FileSystemRouteOptions,
) ([]FileSystemRoute, error) {

	var routes []FileSystemRoute
	opt := options

	// Determine effective settings based on options or defaults
	generateCleanSetting := OptionEnabled
	generateIndexSetting := OptionEnabled
	detectMethodSetting := OptionDisabled
	if opt != nil {
		if opt.GenerateCleanURLs != OptionDefault {
			generateCleanSetting = opt.GenerateCleanURLs
		}
		if opt.GenerateIndexRoutes != OptionDefault {
			generateIndexSetting = opt.GenerateIndexRoutes
		}
		if opt.DetectMethodByFilename != OptionDefault {
			detectMethodSetting = opt.DetectMethodByFilename
		}
	}

	// Boolean flags derived from settings for use in logic
	generateClean := generateCleanSetting == OptionEnabled
	generateIndex := generateIndexSetting == OptionEnabled
	detectMethod := detectMethodSetting == OptionEnabled

	// Normalize urlPrefix
	urlPrefix = "/" + strings.Trim(urlPrefix, "/")
	if urlPrefix == "/" {
		urlPrefix = ""
	} // Avoid double slash at root

	scanDir = filepath.Clean(scanDir)

	frangoInstance.logger.Printf("Mapping filesystem routes: FS=%T, ScanDir='%s', Prefix='%s'", targetFS, scanDir, urlPrefix)

	walkErr := fs.WalkDir(targetFS, scanDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".php") {
			return nil // Skip directories and non-php files
		}

		scriptPathForHandler := path // Path relative to targetFS root

		// Calculate URL path relative to urlPrefix
		// Use filepath.Rel to get path relative to scanDir root
		relToScanDir, err := filepath.Rel(scanDir, path)
		if err != nil {
			// Log error but maybe continue? Skipping this file.
			frangoInstance.logger.Printf("Error calculating relative path for '%s' in '%s': %v. Skipping.", path, scanDir, err)
			return nil
		}
		// Ensure forward slashes for URL and join with prefix
		urlPath := urlPrefix + "/" + filepath.ToSlash(relToScanDir)
		urlPath = "/" + strings.Trim(urlPath, "/") // Clean final URL path

		// --- Detect Method (Optional) ---
		method := "" // Default: ANY method
		baseName := d.Name()
		patternPath := urlPath // Path part used in final registered pattern

		if detectMethod {
			// Check for pattern like `filename.METHOD.php`
			parts := strings.Split(baseName, ".")
			if len(parts) == 3 && strings.ToLower(parts[2]) == "php" {
				potentialMethod := strings.ToUpper(parts[1])
				if isHTTPMethod(potentialMethod) {
					method = potentialMethod
					// Adjust patternPath to remove method part for clean/index routes
					baseWithoutExt := strings.TrimSuffix(baseName, "."+parts[1]+".php")
					patternPath = filepath.Join(filepath.Dir(urlPath), baseWithoutExt)
					patternPath = strings.ReplaceAll(patternPath, string(os.PathSeparator), "/")
					patternPath = "/" + strings.Trim(patternPath, "/")
					frangoInstance.logger.Printf("Detected method '%s' for %s", method, path)
				}
			}
		}

		// --- Generate Handler & Base Route ---
		patternForHandler := patternPath // Base path pattern for HandlerFor
		if method != "" {
			patternForHandler = method + " " + patternPath // Add method prefix for HandlerFor
		}
		handler := frangoInstance.HandlerFor(patternForHandler, scriptPathForHandler)
		routes = append(routes, FileSystemRoute{Method: method, Pattern: patternPath, Handler: handler, ScriptPath: path})
		frangoInstance.logger.Printf("Mapped FS Route: [%s] %s -> %s", method, patternPath, path)

		// --- Generate Implicit Routes (if enabled and method allows) ---
		// Only generate clean/index for GET or ANY method routes
		if method == "" || method == http.MethodGet {
			if generateClean && strings.HasSuffix(patternPath, ".php") {
				cleanPattern := strings.TrimSuffix(patternPath, ".php")
				if cleanPattern != urlPrefix || len(cleanPattern) > 0 { // Avoid root conflict
					cleanPatternForHandler := cleanPattern
					if method != "" {
						cleanPatternForHandler = method + " " + cleanPattern
					}
					cleanHandler := frangoInstance.HandlerFor(cleanPatternForHandler, scriptPathForHandler)
					routes = append(routes, FileSystemRoute{Method: method, Pattern: cleanPattern, Handler: cleanHandler, ScriptPath: path})
					frangoInstance.logger.Printf("Mapped Clean URL: [%s] %s -> %s", method, cleanPattern, path)
				}
			}
			if generateIndex && filepath.Base(scriptPathForHandler) == "index.php" {
				dirPath := filepath.Dir(patternPath) // Dir of the pattern path
				if dirPath == "." {
					dirPath = "/" // Handle root case from filepath.Dir
				} else if !strings.HasSuffix(dirPath, "/") {
					dirPath += "/"
				}

				// Only skip registration if the calculated directory path is exactly the
				// same as a non-root urlPrefix (avoids double registration for prefix itself).
				// We WANT to register "/" if the prefix was "/" (empty string after norm)
				// and we found index.php at the root.
				shouldRegister := true
				if dirPath == urlPrefix && urlPrefix != "" {
					shouldRegister = false
				}

				if shouldRegister {
					dirPatternForHandler := dirPath
					if method != "" {
						dirPatternForHandler = method + " " + dirPath
					}
					dirHandler := frangoInstance.HandlerFor(dirPatternForHandler, scriptPathForHandler)
					routes = append(routes, FileSystemRoute{Method: method, Pattern: dirPath, Handler: dirHandler, ScriptPath: path})
					frangoInstance.logger.Printf("Mapped Index Dir: [%s] %s -> %s", method, dirPath, path)
				}
			}
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("error scanning directory '%s': %w", scanDir, walkErr)
	}

	return routes, nil
}

// --- Internal Methods ---

// initialize initializes the PHP environment (called lazily).
func (m *Middleware) initialize(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := frankenphp.Init(); err != nil {
		return fmt.Errorf("error initializing FrankenPHP: %w", err)
	}
	m.initialized = true
	return nil
}

// ensureInitialized checks if initialized and initializes if not.
// Returns true if ready, false on initialization error.
func (m *Middleware) ensureInitialized(ctx context.Context) bool {
	if !m.initialized {
		m.initLock.Lock()
		defer m.initLock.Unlock()
		if !m.initialized { // Double-check after lock
			m.logger.Println("Initializing FrankenPHP...")
			if err := m.initialize(ctx); err != nil {
				m.logger.Printf("Error initializing PHP environment: %v", err)
				return false
			}
			m.logger.Println("FrankenPHP initialized.")
		}
	}
	return true
}

// resolveScriptPath ensures the script path is absolute.
// If relative, it's joined with the SourceDir.
func (m *Middleware) resolveScriptPath(scriptPath string) string {
	if !filepath.IsAbs(scriptPath) {
		// Assume relative to SourceDir
		return filepath.Join(m.sourceDir, scriptPath)
	}
	return scriptPath // Already absolute
}

// executePHP handles the core logic of preparing the environment and executing a PHP script.
// Takes the absolute path to the PHP script to execute.
// registeredPattern is the original pattern string (e.g., "GET /users/{id}") used for extracting param names.
func (m *Middleware) executePHP(registeredPattern string, absScriptPath string, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
	// 1. Prepare environment data (render vars + path params from r.PathValue)
	envData := make(map[string]string)

	// Extract Path Parameters using Go 1.22+ r.PathValue
	paramNames := extractParamNames(registeredPattern) // Use helper on original registered pattern
	if len(paramNames) > 0 {
		m.logger.Printf("Extracting path parameters for pattern '%s': %v", registeredPattern, paramNames)
		for _, name := range paramNames {
			value := r.PathValue(name)
			if value != "" {
				paramVarKey := "FRANGO_PARAM_" + name
				envData[paramVarKey] = value
				m.logger.Printf("  Extracted param '%s' = '%s'", name, value)
			} else {
				m.logger.Printf("  Warning: Path parameter '%s' not found in request context for pattern '%s'", name, registeredPattern)
			}
		}
	}

	// Populate Render Data if renderFn is provided
	if renderFn != nil {
		m.logger.Printf("Calling render function for %s", registeredPattern)
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

	// 4. Ensure target script exists and is a file within the env
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

	// 5. Prepare FrankenPHP request options - MIMICKING OLD LOGIC
	// Document root is the PARENT directory of the script within the temp env
	documentRoot := filepath.Dir(phpFilePathInEnv)
	// Script name is just the basename of the script, prepended with /
	scriptName := "/" + filepath.Base(phpFilePathInEnv)
	m.logger.Printf("FrankenPHP Setup: DocumentRoot='%s', ScriptName='%s', URL='%s'", documentRoot, scriptName, r.URL.String())

	// Inject envData (render vars, path params) and query params
	phpBaseEnv := map[string]string{
		// *** DO NOT SET SCRIPT_FILENAME here *** - Rely on DocRoot + modified request path
		// "SCRIPT_FILENAME": phpFilePathInEnv,
		"SCRIPT_NAME":    scriptName,         // e.g., /index.php
		"PHP_SELF":       scriptName,         // Match SCRIPT_NAME
		"DOCUMENT_ROOT":  documentRoot,       // Parent dir of script
		"REQUEST_URI":    r.URL.RequestURI(), // Keep original request URI for PHP $_SERVER
		"REQUEST_METHOD": r.Method,
		"QUERY_STRING":   r.URL.RawQuery,
		"HTTP_HOST":      r.Host,
		// Debugging info
		"DEBUG_DOCUMENT_ROOT": documentRoot,
		"DEBUG_SCRIPT_NAME":   scriptName,
		"DEBUG_PHP_FILE_PATH": phpFilePathInEnv, // Full path for debugging
		"DEBUG_URL_PATTERN":   registeredPattern,
		"DEBUG_SOURCE_PATH":   absScriptPath,
		"DEBUG_ENV_ID":        env.ID,
	}
	if len(envData) > 0 {
		for key, value := range envData {
			phpBaseEnv[key] = value
		}
	}
	queryParams := r.URL.Query()
	for key, values := range queryParams {
		if len(values) > 0 {
			queryVarKey := "FRANGO_QUERY_" + key
			phpBaseEnv[queryVarKey] = values[0]
		}
	}
	if !m.developmentMode {
		phpBaseEnv["PHP_OPCACHE_ENABLE"] = "1"
	} else {
		phpBaseEnv["PHP_OPCACHE_ENABLE"] = "0"
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

// --- Internal Types (Lowercase) ---

// phpEnvironment represents a complete PHP execution environment
type phpEnvironment struct {
	ID               string
	OriginalPath     string // Absolute path to the source script
	EndpointPath     string // Key used for cache lookup (usually OriginalPath)
	TempPath         string // Path to the isolated temp dir for this env
	ScriptRelPath    string // Relative path of the main script within the temp dir
	LastUpdated      time.Time
	OriginalFileHash string // Hash of OriginalPath content
	mutex            sync.Mutex
}

// environmentCache manages all PHP execution environments
type environmentCache struct {
	sourceDir       string                     // User's main source dir
	baseDir         string                     // Base temp dir for this frango instance
	embedDir        string                     // Subdir in baseDir for embedded files (_frango_embeds)
	globalLibraries map[string]string          // relPath in env -> abs path on disk (_frango_embeds/...)
	environments    map[string]*phpEnvironment // Keyed by EndpointPath (abs script path)
	mutex           sync.RWMutex
	logger          *log.Logger
	developmentMode bool
}

// newEnvironmentCache creates a new environment cache
func newEnvironmentCache(sourceDir string, baseDir string, logger *log.Logger, developmentMode bool) *environmentCache {
	embedDir := filepath.Join(baseDir, "_frango_embeds")
	return &environmentCache{
		sourceDir:       sourceDir,
		baseDir:         baseDir,
		embedDir:        embedDir,
		environments:    make(map[string]*phpEnvironment),
		globalLibraries: make(map[string]string),
		logger:          logger,
		developmentMode: developmentMode,
	}
}

// AddGlobalLibrary tracks an embedded library file.
func (c *environmentCache) AddGlobalLibrary(targetRelPath string, sourceDiskPath string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.globalLibraries[targetRelPath] = sourceDiskPath
	c.logger.Printf("Tracking global library: %s -> %s", targetRelPath, sourceDiskPath)
}

// GetEnvironment retrieves or creates an environment for a specific PHP script.
// endpointPath key is typically the absolute path to the script.
func (c *environmentCache) GetEnvironment(endpointPath string, originalAbsPath string) (*phpEnvironment, error) {
	// Ensure no query strings in original path
	cleanOriginalPath := originalAbsPath
	if queryIndex := strings.Index(cleanOriginalPath, "?"); queryIndex != -1 {
		cleanOriginalPath = cleanOriginalPath[:queryIndex]
	}

	c.mutex.RLock()
	env, exists := c.environments[endpointPath]
	c.mutex.RUnlock()

	if exists {
		if c.developmentMode {
			if err := c.updateEnvironmentIfNeeded(env); err != nil {
				// Log update error but return existing env?
				c.logger.Printf("Warning: Failed to update environment for %s: %v", endpointPath, err)
				// return nil, err // Option: Fail request if update fails?
			}
		}
		return env, nil
	}

	// Create a new environment
	env, err := c.createEnvironment(endpointPath, cleanOriginalPath)
	if err != nil {
		return nil, err
	}

	// Store the environment
	c.mutex.Lock()
	c.environments[endpointPath] = env
	c.mutex.Unlock()

	return env, nil
}

// createEnvironment creates a new PHP execution environment
func (c *environmentCache) createEnvironment(endpointPath string, originalAbsPath string) (*phpEnvironment, error) {
	// Create a unique ID based *only* on a hash of the defining path
	h := sha256.Sum256([]byte(endpointPath))
	// Use a significant portion of the hash for the directory name to avoid collisions
	id := hex.EncodeToString(h[:16]) // Use first 16 bytes (32 hex chars)

	tempPath := filepath.Join(c.baseDir, id)
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return nil, fmt.Errorf("error creating environment directory '%s': %w", tempPath, err)
	}

	// Calculate initial file hash of the main script
	initialHash, err := calculateFileHash(originalAbsPath)
	if err != nil {
		os.RemoveAll(tempPath)
		return nil, fmt.Errorf("failed to calculate initial hash for '%s': %w", originalAbsPath, err)
	}

	// --- Calculate relative path BEFORE creating env struct ---
	relScriptPath, err := c.calculateRelPath(originalAbsPath)
	if err != nil {
		os.RemoveAll(tempPath)
		return nil, fmt.Errorf("cannot determine relative path for script '%s': %w", originalAbsPath, err)
	}
	// --- End calculate relative path ---

	env := &phpEnvironment{
		ID:               id,
		OriginalPath:     originalAbsPath,
		EndpointPath:     endpointPath,
		TempPath:         tempPath,
		ScriptRelPath:    relScriptPath, // Store relative path
		LastUpdated:      time.Now(),
		OriginalFileHash: initialHash,
	}

	// Copy necessary files to the environment
	if err := c.populateEnvironmentFiles(env); err != nil {
		os.RemoveAll(tempPath)
		return nil, fmt.Errorf("failed to populate environment '%s': %w", env.ID, err)
	}

	c.logger.Printf("Created environment for '%s' at '%s'", endpointPath, tempPath)
	return env, nil
}

// updateEnvironmentIfNeeded checks if an environment needs to be updated.
func (c *environmentCache) updateEnvironmentIfNeeded(env *phpEnvironment) error {
	env.mutex.Lock() // Lock specific env
	defer env.mutex.Unlock()

	// Hash check on main file only for now
	currentHash, err := calculateFileHash(env.OriginalPath)
	if err != nil {
		c.logger.Printf("Warning: Could not calculate hash for '%s' during update check: %v", env.OriginalPath, err)
		return nil // Don't fail update if hash check fails temporarily
	}

	if currentHash != env.OriginalFileHash {
		c.logger.Printf("Rebuilding environment for '%s' due to file content change (hash mismatch)", env.EndpointPath)
		if err := c.populateEnvironmentFiles(env); err != nil {
			return fmt.Errorf("error rebuilding environment files for '%s': %w", env.EndpointPath, err)
		}
		env.OriginalFileHash = currentHash
		env.LastUpdated = time.Now()
	}
	return nil
}

// calculateRelPath determines the relative path of a script based on source/embed dirs
func (c *environmentCache) calculateRelPath(absScriptPath string) (string, error) {
	var relPath string
	var err error
	if strings.HasPrefix(absScriptPath, c.embedDir) {
		relPath, err = filepath.Rel(c.embedDir, absScriptPath)
	} else {
		relPath, err = filepath.Rel(c.sourceDir, absScriptPath)
	}
	if err != nil {
		return "", err // Let caller handle specific error message
	}
	relPath = filepath.Clean(relPath)
	// Handle file at root of source/embed more carefully
	if relPath == "." {
		relPath = filepath.Base(absScriptPath)
	}
	return relPath, nil
}

// _mirrorDirectoryContent mirrors all files from a source directory to a destination directory.
// Used internally by populateEnvironmentFiles when dealing with SourceDir scripts.
func (c *environmentCache) _mirrorDirectoryContent(sourceDir string, destDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path from the source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("error calculating relative path during mirror: %w", err)
		}

		// Calculate the target path in the environment
		targetPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			// Create directories as needed
			// Use MkdirAll to handle nested directories properly
			if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("error creating directory during mirror '%s': %w", targetPath, err)
			}
			return nil // Don't copy directory itself, just ensure it exists
		}

		// If not a directory, copy the file
		if err := copyFile(path, targetPath); err != nil {
			return fmt.Errorf("error copying file during mirror '%s' to '%s': %w", path, targetPath, err)
		}

		return nil
	})
}

// populateEnvironmentFiles copies the necessary files into the environment.
// If the source is from SourceDir, mirrors the whole SourceDir.
// If the source is from EmbedDir, copies only the specific script.
// Then, overlays global libraries.
func (c *environmentCache) populateEnvironmentFiles(env *phpEnvironment) error {

	// Clear the temp directory first? Or assume it's fresh?
	// Assuming createEnvironment provides a fresh dir.
	// If updateEnvironmentIfNeeded calls this, maybe it should clear first?
	// For now, let's assume overwrite is okay.

	// 1. Handle main script source (Mirror sourceDir OR copy single embed script)
	if strings.HasPrefix(env.OriginalPath, c.embedDir) {
		// Source is an embedded file - copy only this file
		relEndpointPath := env.ScriptRelPath
		if relEndpointPath == "" {
			return fmt.Errorf("internal error: ScriptRelPath empty for embed env %s", env.ID)
		}
		targetEndpointPath := filepath.Join(env.TempPath, relEndpointPath)
		if err := copyFile(env.OriginalPath, targetEndpointPath); err != nil {
			return fmt.Errorf("failed to copy embedded endpoint file '%s' to '%s': %w", env.OriginalPath, targetEndpointPath, err)
		}
		c.logger.Printf("Populated env %s with single embedded script: %s", env.ID, relEndpointPath)

	} else if strings.HasPrefix(env.OriginalPath, c.sourceDir) || !filepath.IsAbs(env.OriginalPath) {
		// Source is from user's SourceDir (or was relative, assumed to be in sourceDir)
		// Mirror the entire source directory content
		c.logger.Printf("Populating env %s by mirroring SourceDir: %s", env.ID, c.sourceDir)
		if err := c._mirrorDirectoryContent(c.sourceDir, env.TempPath); err != nil {
			return fmt.Errorf("failed to mirror sourceDir '%s' to '%s': %w", c.sourceDir, env.TempPath, err)
		}
	} else {
		// Original path is absolute but not in embed dir - how should this be handled?
		// Copy just the single file for now.
		c.logger.Printf("Warning: Handling absolute script path '%s' outside known source/embed dirs. Copying only the single file.", env.OriginalPath)
		relEndpointPath := env.ScriptRelPath
		if relEndpointPath == "" {
			return fmt.Errorf("internal error: ScriptRelPath empty for absolute env %s", env.ID)
		}
		targetEndpointPath := filepath.Join(env.TempPath, relEndpointPath)
		if err := copyFile(env.OriginalPath, targetEndpointPath); err != nil {
			return fmt.Errorf("failed to copy absolute endpoint file '%s' to '%s': %w", env.OriginalPath, targetEndpointPath, err)
		}
	}

	// 2. Copy global libraries (overlaying potentially mirrored files)
	c.mutex.RLock() // Lock cache for reading libraries map
	libsToCopy := make(map[string]string)
	for target, source := range c.globalLibraries {
		libsToCopy[target] = source
	}
	c.mutex.RUnlock()

	for relLibPath, sourceLibPath := range libsToCopy {
		targetLibPath := filepath.Join(env.TempPath, relLibPath)
		if err := copyFile(sourceLibPath, targetLibPath); err != nil {
			// Log warning but maybe continue?
			c.logger.Printf("Warning: Failed to copy global library '%s' to '%s': %v", sourceLibPath, targetLibPath, err)
			// return fmt.Errorf("failed to copy global library ...") // Option: Fail hard
		}
	}

	return nil
}

// Cleanup removes all environments and the base temp dir.
func (c *environmentCache) Cleanup() {
	c.mutex.Lock() // Lock for modifying environments map
	defer c.mutex.Unlock()

	for key, env := range c.environments {
		// Use c.logger, not m.logger
		c.logger.Printf("Removing environment temp dir: %s (for %s)", env.TempPath, key)
		// os.RemoveAll(env.TempPath) // This is handled by baseDir removal
	}
	c.environments = make(map[string]*phpEnvironment) // Clear map

	// Use c.logger here too
	c.logger.Printf("Cleanup complete (base temp dir removal handled elsewhere).")
}

// calculateFileHash calculates the SHA256 hash of a file's content.
func calculateFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file '%s' for hashing: %w", filePath, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile utility copies a single file, creating destination directories.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory '%s': %w", filepath.Dir(dst), err)
	}
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

// --- Functional Options (Exported) ---

// WithSourceDir sets the source directory for PHP files.
func WithSourceDir(dir string) Option {
	return func(m *Middleware) {
		m.sourceDir = dir
	}
}

// WithDevelopmentMode enables immediate file change detection and disables caching.
func WithDevelopmentMode(enabled bool) Option {
	return func(m *Middleware) {
		m.developmentMode = enabled
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *log.Logger) Option {
	return func(m *Middleware) {
		m.logger = logger
	}
}

// WithDirectPHPURLsBlocking controls whether direct PHP file access in URLs should be blocked.
func WithDirectPHPURLsBlocking(block bool) Option {
	return func(m *Middleware) {
		m.blockDirectPHPURLs = block
	}
}

// NOTE: Implicit flags are removed as routing is external now.

// --- Internal Helpers ---

// isHTTPMethod checks if a string is a valid uppercase HTTP method name.
func isHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// resolveDirectory resolves a directory path, supporting both absolute and relative paths.
// It tries multiple strategies to find the directory:
// 1. Use the path directly if it exists (relative to CWD)
// 2. If relative, try to find it relative to runtime caller
// 3. If relative, try to find it relative to current working directory again
// 4. Falls back to the original path if nothing is found (will likely error later)
// NOTE: This function is restored from the original version to maintain behavior for examples run directly.
func resolveDirectory(path string) (string, error) {
	// If the path is absolute or explicitly relative (starts with ./ or ../)
	if filepath.IsAbs(path) || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("error resolving absolute/explicit relative path '%s': %w", path, err)
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath, nil
		} else if err != nil {
			return "", fmt.Errorf("error stating explicit path '%s': %w", absPath, err)
		} else {
			return "", fmt.Errorf("explicit path '%s' exists but is not a directory", absPath)
		}
	}

	// For a bare directory name (like "web" in examples), try multiple locations directly

	// 1. Try relative to CWD first.
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		absPath, absErr := filepath.Abs(path)
		if absErr == nil {
			return absPath, nil
		}
		// If os.Stat worked but Abs failed, that's weird, but report it.
		return "", fmt.Errorf("found path '%s' relative to CWD but failed to get absolute path: %w", path, absErr)
	}

	// 2. Try relative to Caller (primarily for examples)
	// Skip if path looks absolute or explicitly relative already.
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, ".") {
		// Use Caller(2) to get the caller of frango.New (or wherever resolveDirectory was called from)
		_, filename, _, ok := runtime.Caller(2)
		if ok {
			callerDir := filepath.Dir(filename)
			callerPath := filepath.Join(callerDir, path)
			absCallerPath, absErr := filepath.Abs(callerPath)
			if absErr == nil {
				if info, statErr := os.Stat(absCallerPath); statErr == nil && info.IsDir() {
					// Found relative to caller of New
					log.Printf("[frango] Info: Resolved path '%s' relative to caller (%s) -> %s", path, callerDir, absCallerPath)
					return absCallerPath, nil
				}
			}
		}
	}

	// 3. If neither worked, return the first error encountered (from CWD check)
	// ... rest of function

	return "", fmt.Errorf("directory '%s' not found relative to CWD or caller", path)
}

// extractParamNames parses a route pattern and returns a slice of parameter names.
func extractParamNames(pattern string) []string {
	var names []string
	parts := strings.Split(pattern, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := part[1 : len(part)-1]
			if name != "" && name != "$" { // Exclude special {$}
				names = append(names, name)
			}
		}
	}
	return names
}

// getMapKeys is a helper function to get the keys of a map for logging (internal)
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
