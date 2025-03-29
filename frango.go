// Package frango provides middleware for integrating PHP with Go HTTP servers using FrankenPHP.
package frango

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
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

// Middleware is the core PHP middleware type that implements http.Handler
type Middleware struct {
	sourceDir       string
	tempDir         string
	logger          *log.Logger
	initialized     bool
	initLock        sync.Mutex
	routes          map[string]string
	developmentMode bool
	envCache        *EnvironmentCache
}

// Config represents configuration options for the middleware
type Config struct {
	// SourceDir is the directory containing PHP files (empty for embedded files)
	SourceDir string
	// DevelopmentMode enables immediate file change detection and disables caching
	DevelopmentMode bool
	// Logger for output (defaults to standard logger if nil)
	Logger *log.Logger
}

// New creates a new PHP middleware instance with the provided options
func New(opts ...Option) (*Middleware, error) {
	// Default configuration
	m := &Middleware{
		routes:          make(map[string]string),
		developmentMode: true,
		logger:          log.New(os.Stdout, "[frango] ", log.LstdFlags),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// If sourceDir is empty, create a temp directory
	var absSourceDir string
	var err error

	if m.sourceDir == "" {
		absSourceDir, err = os.MkdirTemp("", "frango-middleware")
		if err != nil {
			return nil, fmt.Errorf("error creating temporary directory: %w", err)
		}
	} else {
		// Resolve source directory using the path resolution function
		absSourceDir, err = ResolveDirectory(m.sourceDir)
		if err != nil {
			return nil, fmt.Errorf("error resolving source directory: %w", err)
		}
	}

	// Create temporary directory for environments
	tempDir, err := os.MkdirTemp("", "frango-environments")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}

	// Update middleware with resolved paths
	m.sourceDir = absSourceDir
	m.tempDir = tempDir

	// Create environment cache
	m.envCache = NewEnvironmentCache(absSourceDir, tempDir, m.logger, m.developmentMode)

	// Clean any stored routes that might have query strings (defensive coding)
	for pattern, phpFile := range m.routes {
		if queryIndex := strings.Index(phpFile, "?"); queryIndex != -1 {
			m.logger.Printf("WARNING: Query string in phpFile path at initialization: %s", phpFile)
			m.routes[pattern] = phpFile[:queryIndex]
		}
	}

	return m, nil
}

// ServeHTTP implements the http.Handler interface
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Initialize if needed
	if !m.initialized {
		m.initLock.Lock()
		if !m.initialized { // Double-check after acquiring lock
			if err := m.initialize(r.Context()); err != nil {
				m.logger.Printf("Error initializing PHP environment: %v", err)
				http.Error(w, "PHP initialization error", http.StatusInternalServerError)
				m.initLock.Unlock()
				return
			}
			m.initialized = true
		}
		m.initLock.Unlock()
	}

	path := r.URL.Path

	// Check for method-specific routes first
	methodKey := r.Method + ":" + path
	if phpFile, found := m.routes[methodKey]; found {
		m.servePHPFile(path, phpFile, w, r)
		return
	}

	// Check if this is a registered route
	if phpFile, found := m.routes[path]; found {
		m.servePHPFile(path, phpFile, w, r)
		return
	}

	// Check with trailing slash
	if !strings.HasSuffix(path, "/") {
		if phpFile, found := m.routes[path+"/"]; found {
			m.servePHPFile(path+"/", phpFile, w, r)
			return
		}
	}

	// Check for .php extension version
	if !strings.HasSuffix(path, ".php") {
		if phpFile, found := m.routes[path+".php"]; found {
			m.servePHPFile(path+".php", phpFile, w, r)
			return
		}
	}

	// Special case for root path
	if path == "/" {
		if phpFile, found := m.routes["/"]; found {
			m.servePHPFile("/", phpFile, w, r)
			return
		}
		indexPath := filepath.Join(m.sourceDir, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			m.servePHPFile("/", indexPath, w, r)
			return
		}
	}

	// Check for direct PHP file access
	phpPath := filepath.Join(m.sourceDir, strings.TrimPrefix(path, "/"))
	if _, err := os.Stat(phpPath); err == nil {
		if strings.HasSuffix(phpPath, ".php") {
			m.servePHPFile(path, phpPath, w, r)
			return
		} else {
			// Serve static file
			http.ServeFile(w, r, phpPath)
			return
		}
	}

	// Check for PHP file with .php extension added
	if !strings.HasSuffix(phpPath, ".php") {
		phpPathWithExt := phpPath + ".php"
		if _, err := os.Stat(phpPathWithExt); err == nil {
			m.servePHPFile(path, phpPathWithExt, w, r)
			return
		}
	}

	// Directory check - look for index.php
	dirPath := path
	if !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}

	dirPhpPath := filepath.Join(m.sourceDir, strings.TrimPrefix(dirPath, "/"))
	if stat, err := os.Stat(dirPhpPath); err == nil && stat.IsDir() {
		indexPath := filepath.Join(dirPhpPath, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			m.servePHPFile(dirPath, indexPath, w, r)
			return
		}
	}

	// Not found
	http.NotFound(w, r)
}

// initialize initializes the PHP environment with context
func (m *Middleware) initialize(ctx context.Context) error {
	// Create a background context if nil is provided
	if ctx == nil {
		ctx = context.Background()
	}

	// Check if context is canceled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Initialize FrankenPHP
	if err := frankenphp.Init(); err != nil {
		return fmt.Errorf("error initializing FrankenPHP: %w", err)
	}

	return nil
}

// Shutdown cleans up resources
func (m *Middleware) Shutdown() {
	if m.initialized {
		frankenphp.Shutdown()
		m.initialized = false
	}

	// Clean up all environments
	m.envCache.Cleanup()

	// Remove the temp directory
	os.RemoveAll(m.tempDir)
}

// Handle registers a PHP file to serve at a specific path
func (m *Middleware) Handle(pattern string, phpFile string) {
	// Check if this is a method-specific pattern (contains a space)
	if strings.Contains(pattern, " ") {
		m.handleMethod(pattern, phpFile)
		return
	}

	// Standard path handling
	m.HandlePHP(pattern, phpFile)
}

// HandlePHP maps a URL pattern to a PHP file
func (m *Middleware) HandlePHP(pattern string, phpFile string) {
	// Ensure URL path starts with a slash
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// Strip any query string from the PHP file path
	if queryIndex := strings.Index(phpFile, "?"); queryIndex != -1 {
		phpFile = phpFile[:queryIndex]
	}

	// If the PHP file is not an absolute path, make it relative to source dir
	if !filepath.IsAbs(phpFile) {
		phpFile = filepath.Join(m.sourceDir, phpFile)
	}

	// Store the mapping
	m.routes[pattern] = phpFile

	// Pre-create the environment for this path
	_, err := m.envCache.GetEnvironment(pattern, phpFile)
	if err != nil {
		m.logger.Printf("Warning: Failed to pre-create environment for %s: %v", pattern, err)
	}

	m.logger.Printf("Registered PHP handler: %s -> %s", pattern, phpFile)
}

// HandleDir registers all PHP files in a directory under a URL prefix
func (m *Middleware) HandleDir(prefix string, dirPath string) error {
	// Ensure URL prefix starts with a slash
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	// If trailing slash, ensure it's consistent
	if prefix != "/" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// If the directory is not an absolute path, make it relative to source dir
	if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(m.sourceDir, dirPath)
	}

	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("error accessing directory %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dirPath)
	}

	// Walk directory and register all PHP files
	count := 0
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process PHP files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".php") {
			// Calculate URL path
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return fmt.Errorf("error calculating relative path: %w", err)
			}

			// Convert Windows path separators to URL separators
			relPath = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

			// Create URL path - ensure prefix ends with slash for proper joining
			urlPath := prefix
			if prefix != "/" && !strings.HasSuffix(prefix, "/") {
				urlPath = prefix + "/"
			}
			urlPath += relPath

			// Register the path with .php extension
			m.HandlePHP(urlPath, path)

			// Also register without .php extension for clean URLs
			if strings.HasSuffix(urlPath, ".php") {
				cleanPath := strings.TrimSuffix(urlPath, ".php")
				m.HandlePHP(cleanPath, path)

				// For index.php files, also register the directory path
				if filepath.Base(relPath) == "index.php" {
					dirPath := filepath.Dir(urlPath)
					if dirPath != "/" {
						// Ensure the directory path ends with a slash
						if !strings.HasSuffix(dirPath, "/") {
							dirPath += "/"
						}
						m.HandlePHP(dirPath, path)
					}
				}
			}

			count++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	m.logger.Printf("Registered %d PHP files from directory %s under %s", count, dirPath, prefix)
	return nil
}

// AddFromEmbed adds a PHP file from an embed.FS
func (m *Middleware) AddFromEmbed(urlPath string, fs embed.FS, fsPath string) string {
	// Read the file from the embed.FS
	content, err := fs.ReadFile(fsPath)
	if err != nil {
		m.logger.Printf("Error reading embedded file %s: %v", fsPath, err)
		return ""
	}

	// Ensure path starts with slash
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// Ensure path ends with .php for filesystem purposes
	filePath := urlPath
	if !strings.HasSuffix(filePath, ".php") {
		filePath = filePath + ".php"
	}

	// Create the target path
	targetPath := filepath.Join(m.sourceDir, strings.TrimPrefix(filePath, "/"))

	// Create directory structure
	if targetDir := filepath.Dir(targetPath); targetDir != "" {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			m.logger.Printf("Warning: Failed to create directory for %s: %v", filePath, err)
			return ""
		}
	}

	// Write file to disk
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		m.logger.Printf("Warning: Failed to write file %s: %v", filePath, err)
		return ""
	}

	// Register the URL path
	m.HandlePHP(urlPath, targetPath)

	// Also register clean path (without .php)
	if strings.HasSuffix(urlPath, ".php") {
		cleanPath := strings.TrimSuffix(urlPath, ".php")
		if cleanPath != "" && cleanPath != "/" {
			m.HandlePHP(cleanPath, targetPath)
		}
	}

	m.logger.Printf("Added PHP file from embed at %s", targetPath)
	return targetPath
}

// handleMethod handles a route with HTTP method and potential path parameters
func (m *Middleware) handleMethod(pattern string, phpFilePath string) {
	// Extract method and path from pattern
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) != 2 {
		m.logger.Printf("Invalid pattern format: %s. Expected format: 'METHOD /path'", pattern)
		return
	}

	method := parts[0]
	path := parts[1]

	// Register the endpoint with a special internal key format
	internalKey := method + ":" + path
	m.routes[internalKey] = phpFilePath

	m.logger.Printf("Registered %s endpoint: %s -> %s", method, path, phpFilePath)
}

// Wrap wraps another http.Handler to create middleware chain
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a PHP request that we should handle
		if m.shouldHandlePHP(r) {
			m.ServeHTTP(w, r)
			return
		}

		// Otherwise pass to the next handler
		next.ServeHTTP(w, r)
	})
}

// shouldHandlePHP determines if we should handle this request as PHP
func (m *Middleware) shouldHandlePHP(r *http.Request) bool {
	path := r.URL.Path

	// Check for method-specific routes first
	methodKey := r.Method + ":" + path
	if _, exists := m.routes[methodKey]; exists {
		return true
	}

	// Check standard routes
	if _, exists := m.routes[path]; exists {
		return true
	}

	// Check with trailing slash if the path doesn't have one
	if !strings.HasSuffix(path, "/") {
		if _, exists := m.routes[path+"/"]; exists {
			return true
		}
	}

	// Also check for path with .php extension
	if !strings.HasSuffix(path, ".php") {
		if _, exists := m.routes[path+".php"]; exists {
			return true
		}
	}

	// Check for explicit PHP files
	phpPath := filepath.Join(m.sourceDir, strings.TrimPrefix(path, "/"))
	if _, err := os.Stat(phpPath); err == nil && strings.HasSuffix(phpPath, ".php") {
		return true
	}

	// If path doesn't end with .php, check if a .php version exists
	if !strings.HasSuffix(phpPath, ".php") {
		phpPathWithExt := phpPath + ".php"
		if _, err := os.Stat(phpPathWithExt); err == nil {
			return true
		}
	}

	// Check for directory index.php
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	dirPath := filepath.Join(m.sourceDir, strings.TrimPrefix(path, "/"))
	if stat, err := os.Stat(dirPath); err == nil && stat.IsDir() {
		indexPath := filepath.Join(dirPath, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			return true
		}
	}

	// Special case for root path
	if path == "/" {
		if _, exists := m.routes["/"]; exists {
			return true
		}
		indexPath := filepath.Join(m.sourceDir, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			return true
		}
	}

	return false
}

// RenderData represents data to pass to a PHP template for rendering
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}

// Global map to store render functions for HandleRender, protected by a mutex
var (
	renderHandlers      = make(map[string]RenderData)
	renderHandlersMutex sync.RWMutex
)

// HandleRender registers a PHP file to be rendered with dynamic data
func (m *Middleware) HandleRender(pattern string, phpFile string, renderFn RenderData) {
	// Ensure path has a leading slash
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// Build full path to the PHP file if not absolute
	phpFilePath := phpFile
	if !filepath.IsAbs(phpFile) {
		phpFilePath = filepath.Join(m.sourceDir, phpFile)
	}

	// Verify the PHP file exists before registering
	fileInfo, err := os.Stat(phpFilePath)
	if err != nil {
		m.logger.Printf("Error accessing PHP file %s: %v", phpFilePath, err)
		return
	}

	if fileInfo.IsDir() {
		m.logger.Printf("PHP file path is a directory: %s", phpFilePath)
		return
	}

	// Store the render function in the global map
	renderHandlersMutex.Lock()
	renderHandlers[pattern] = renderFn
	renderHandlersMutex.Unlock()

	// Register this route to point to the PHP file
	m.routes[pattern] = phpFilePath

	m.logger.Printf("Registered render endpoint: %s -> %s", pattern, phpFilePath)
}

// servePHPFile serves a PHP file, checking if it needs special render handling
func (m *Middleware) servePHPFile(urlPath string, sourcePath string, w http.ResponseWriter, r *http.Request) {
	// Check if this is a render path with a render function
	renderHandlersMutex.RLock()
	renderFn, isRenderPath := renderHandlers[urlPath]
	renderHandlersMutex.RUnlock()

	// Initialize path parameters
	pathParams := make(map[string]string)

	// If this is a render path, get the data from the render function
	if isRenderPath {
		m.logger.Printf("Found render handler for path: %s", urlPath)

		// Call the render function to get data
		data := renderFn(w, r)

		// Add a render flag
		pathParams["RENDER"] = "true"

		// Debug the render data
		m.logger.Printf("Render data keys: %v", getMapKeys(data))

		// Convert the data to environment variables
		for key, value := range data {
			jsonData, err := json.Marshal(value)
			if err != nil {
				m.logger.Printf("Error marshaling data for %s: %v", key, err)
				continue
			}

			// Log the JSON data for debugging
			m.logger.Printf("Render data for %s: %s", key, string(jsonData))

			// Add variables with different prefixes for compatibility
			frVarKey := "frango_VAR_" + key
			pathParams[frVarKey] = string(jsonData)
			pathParams["PATH_PARAM_"+strings.ToUpper(key)] = string(jsonData)
		}
	}

	// Serve the PHP file with the appropriate parameters
	m.servePHPFileWithPathParams(urlPath, sourcePath, pathParams, w, r)
}

// getMapKeys is a helper function to get the keys of a map for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// servePHPFileWithPathParams serves a PHP file with path parameters
func (m *Middleware) servePHPFileWithPathParams(urlPath string, sourcePath string, pathParams map[string]string, w http.ResponseWriter, r *http.Request) {
	// Strip any query string from the source path - put this early
	originalSourcePath := sourcePath
	if queryIndex := strings.Index(sourcePath, "?"); queryIndex != -1 {
		m.logger.Printf("WARNING: Query string detected in sourcePath: %s", sourcePath)
		sourcePath = sourcePath[:queryIndex]
		m.logger.Printf("Stripped to: %s", sourcePath)
	}

	// Get or create environment for this endpoint
	env, err := m.envCache.GetEnvironment(urlPath, sourcePath)
	if err != nil {
		m.logger.Printf("Error setting up environment for %s: %v", urlPath, err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Calculate the path to the original PHP file relative to the source directory
	relPath, err := filepath.Rel(m.sourceDir, sourcePath)
	if err != nil {
		m.logger.Printf("Error calculating relative path (for %s -> %s): %v", sourcePath, m.sourceDir, err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Calculate the path to the PHP file in the environment
	phpFilePath := filepath.Join(env.TempPath, relPath)

	// Debug the paths
	m.logger.Printf("Original sourcePath: %s", originalSourcePath)
	m.logger.Printf("Cleaned sourcePath: %s", sourcePath)
	m.logger.Printf("relPath: %s", relPath)
	m.logger.Printf("phpFilePath to look for: %s", phpFilePath)

	// Ensure this is actually pointing to a file, not a directory
	fileInfo, err := os.Stat(phpFilePath)
	if err != nil {
		// If file doesn't exist, log and try to rebuild
		m.logger.Printf("Error accessing PHP file %s: %v", phpFilePath, err)

		// If the file doesn't exist but the environment does, try to rebuild it
		if os.IsNotExist(err) {
			m.logger.Printf("Trying to rebuild environment for %s", urlPath)
			if err := m.envCache.mirrorFilesToEnvironment(env); err != nil {
				m.logger.Printf("Error rebuilding environment: %v", err)
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}

			// Check again after rebuilding
			fileInfo, err = os.Stat(phpFilePath)
			if err != nil {
				m.logger.Printf("File still not found after rebuilding: %s", phpFilePath)
				http.NotFound(w, r)
				return
			}
		} else {
			http.NotFound(w, r)
			return
		}
	}

	// Double check we're not trying to execute a directory
	if fileInfo.IsDir() {
		m.logger.Printf("ERROR: Path is a directory, not a PHP file: %s", phpFilePath)

		// Try appending index.php if it's a directory
		indexPath := filepath.Join(phpFilePath, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			m.logger.Printf("Found index.php in directory, using: %s", indexPath)
			phpFilePath = indexPath
		} else {
			m.logger.Printf("No index.php found in directory: %s", phpFilePath)
			http.Error(w, "Server error - trying to execute directory as PHP", http.StatusInternalServerError)
			return
		}
	}

	// *** CRITICAL: PROPERLY SETUP FRANKENPHP REQUEST ***
	//
	// FrankenPHP works by setting the document root and letting it construct the
	// script path automatically. The document root should be the parent directory
	// of the PHP file, not the environment root.

	// Calculate the document root (parent directory of the PHP file)
	documentRoot := filepath.Dir(phpFilePath)

	// Calculate the script name (basename of the PHP file)
	scriptName := "/" + filepath.Base(phpFilePath)

	m.logger.Printf("Running PHP with DocumentRoot=%s, ScriptName=%s, URL=%s", documentRoot, scriptName, r.URL.String())

	// Setup environment variables
	phpEnv := map[string]string{
		// DO NOT set SCRIPT_FILENAME - FrankenPHP does this automatically
		"SCRIPT_NAME":    scriptName,
		"PHP_SELF":       scriptName,
		"DOCUMENT_ROOT":  documentRoot,
		"REQUEST_URI":    r.URL.RequestURI(), // This includes query string
		"REQUEST_METHOD": r.Method,
		"QUERY_STRING":   r.URL.RawQuery,
		"HTTP_HOST":      r.Host,

		// For debugging - remove in production
		"DEBUG_DOCUMENT_ROOT": documentRoot,
		"DEBUG_SCRIPT_NAME":   scriptName,
		"DEBUG_PHP_FILE_PATH": phpFilePath,
		"DEBUG_URL_PATH":      urlPath,
		"DEBUG_SOURCE_PATH":   sourcePath,
		"DEBUG_ENV_ID":        env.ID,
		"DEBUG_QUERY_STRING":  r.URL.RawQuery,
		"DEBUG_REQUEST_URI":   r.URL.RequestURI(),
	}

	// Add path parameters to environment
	if len(pathParams) > 0 {
		// Create a JSON string with all path parameters
		pathParamsJSON, _ := json.Marshal(pathParams)
		phpEnv["PATH_PARAMS"] = string(pathParamsJSON)

		// Debug the pathParams
		m.logger.Printf("Path parameters: %v", pathParams)

		// First, add all path parameters to environment with their original names
		for name, value := range pathParams {
			// For compatibility with both formats
			phpEnv["PATH_PARAM_"+strings.ToUpper(name)] = value

			// Direct inclusion of render variables
			if strings.HasPrefix(name, "frango_VAR_") {
				// This is critical - add the variable directly to $_SERVER
				phpEnv[name] = value
				m.logger.Printf("Added render variable %s to environment", name)
			}
		}
	}

	// Parse query parameters and add them as individual environment variables
	queryParams := r.URL.Query()
	for key, values := range queryParams {
		if len(values) > 0 {
			// Add as direct environment variable for easier access
			phpEnv["QUERY_PARAM_"+strings.ToUpper(key)] = values[0]
		}
	}

	// Add caching configuration
	if !m.developmentMode {
		phpEnv["PHP_PRODUCTION"] = "1"
		phpEnv["PHP_OPCACHE_ENABLE"] = "1"
	} else {
		phpEnv["PHP_FCGI_MAX_REQUESTS"] = "1"
		phpEnv["PHP_OPCACHE_ENABLE"] = "0"
	}

	// Clone the request and set the URL path to the script name
	// This ensures FrankenPHP looks for the right file
	reqClone := r.Clone(r.Context())
	reqClone.URL.Path = scriptName // Make sure we preserve the query string

	// Debug the environment variables
	m.logger.Printf("PHP environment variables: %d variables", len(phpEnv))
	for key, _ := range phpEnv {
		if strings.HasPrefix(key, "frango_VAR_") {
			m.logger.Printf("  %s is set", key)
		}
	}

	// Create FrankenPHP request using the correct document root
	req, err := frankenphp.NewRequestWithContext(
		reqClone,
		frankenphp.WithRequestDocumentRoot(documentRoot, false), // Document root is the environment directory
		frankenphp.WithRequestEnv(phpEnv),                       // Environment includes SCRIPT_FILENAME
	)
	if err != nil {
		m.logger.Printf("Error creating PHP request: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Execute PHP
	if err := frankenphp.ServeHTTP(w, req); err != nil {
		m.logger.Printf("Error executing PHP: %v", err)
		http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Option is a function that configures a Middleware
type Option func(*Middleware)

// WithSourceDir sets the source directory for PHP files
func WithSourceDir(dir string) Option {
	return func(m *Middleware) {
		m.sourceDir = dir
	}
}

// WithDevelopmentMode enables immediate file change detection and disables caching
func WithDevelopmentMode(enabled bool) Option {
	return func(m *Middleware) {
		m.developmentMode = enabled
	}
}

// WithLogger sets a custom logger
func WithLogger(logger *log.Logger) Option {
	return func(m *Middleware) {
		m.logger = logger
	}
}

// ResolveDirectory resolves a directory path, supporting both absolute and relative paths.
// It tries multiple strategies to find the directory:
// 1. Use the path directly if it exists
// 2. If relative, try to find it relative to runtime caller
// 3. If relative, try to find it relative to current working directory
// 4. Falls back to the original path if nothing is found
func ResolveDirectory(path string) (string, error) {
	// If the path is absolute or explicitly relative (starts with ./ or ../)
	if filepath.IsAbs(path) || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("error resolving absolute path: %w", err)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}

		return "", fmt.Errorf("directory not found: %s", absPath)
	}

	// For a bare directory name, try multiple locations directly

	// Try as-is first
	if _, err := os.Stat(path); err == nil {
		absPath, err := filepath.Abs(path)
		if err == nil {
			return absPath, nil
		}
	}

	// Try relative to runtime caller
	_, filename, _, ok := runtime.Caller(1) // Caller of this function
	if ok {
		callerDir := filepath.Dir(filename)
		callerPath := filepath.Join(callerDir, path)
		if _, err := os.Stat(callerPath); err == nil {
			absPath, err := filepath.Abs(callerPath)
			if err == nil {
				return absPath, nil
			}
		}
	}

	// Try relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		cwdPath := filepath.Join(cwd, path)
		if _, err := os.Stat(cwdPath); err == nil {
			absPath, err := filepath.Abs(cwdPath)
			if err == nil {
				return absPath, nil
			}
		}
	}

	// Nothing found, return error
	return "", fmt.Errorf("directory not found: %s", path)
}

// PHPEnvironment represents a complete PHP execution environment
type PHPEnvironment struct {
	// ID is a unique identifier for this environment
	ID string
	// OriginalPath is the path to the original PHP file
	OriginalPath string
	// EndpointPath is the URL path this environment serves
	EndpointPath string
	// TempPath is the path to the temporary directory for this environment
	TempPath string
	// LastUpdated is when this environment was last rebuilt
	LastUpdated time.Time
	// mutex controls concurrent access to this environment
	mutex sync.Mutex
}

// EnvironmentCache manages all PHP execution environments
type EnvironmentCache struct {
	// sourceDir is the source directory containing PHP files
	sourceDir string
	// baseDir is the base directory for all environments
	baseDir string
	// environments maps endpoint paths to their environments
	environments map[string]*PHPEnvironment
	// mutex controls concurrent access to the environments map
	mutex sync.RWMutex
	// logger for output
	logger *log.Logger
	// developmentMode enables immediate detection of file changes
	developmentMode bool
}

// NewEnvironmentCache creates a new environment cache
func NewEnvironmentCache(sourceDir string, baseDir string, logger *log.Logger, developmentMode bool) *EnvironmentCache {
	return &EnvironmentCache{
		sourceDir:       sourceDir,
		baseDir:         baseDir,
		environments:    make(map[string]*PHPEnvironment),
		logger:          logger,
		developmentMode: developmentMode,
	}
}

// GetEnvironment retrieves or creates an environment for an endpoint
func (c *EnvironmentCache) GetEnvironment(endpointPath string, originalPath string) (*PHPEnvironment, error) {
	// Ensure no query strings in paths
	if queryIndex := strings.Index(originalPath, "?"); queryIndex != -1 {
		c.logger.Printf("WARNING: Query string detected in originalPath: %s", originalPath)
		originalPath = originalPath[:queryIndex]
		c.logger.Printf("Stripped to: %s", originalPath)
	}

	c.mutex.RLock()
	env, exists := c.environments[endpointPath]
	c.mutex.RUnlock()

	if exists {
		// Check if environment needs to be updated (in development mode or file changed)
		if c.developmentMode {
			if err := c.updateEnvironmentIfNeeded(env); err != nil {
				return nil, err
			}
		}
		return env, nil
	}

	// Create a new environment
	env, err := c.createEnvironment(endpointPath, originalPath)
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
func (c *EnvironmentCache) createEnvironment(endpointPath string, originalPath string) (*PHPEnvironment, error) {
	// Triple check for query strings
	if queryIndex := strings.Index(originalPath, "?"); queryIndex != -1 {
		c.logger.Printf("WARNING: Query string still detected in originalPath at createEnvironment: %s", originalPath)
		originalPath = originalPath[:queryIndex]
		c.logger.Printf("Stripped to: %s", originalPath)
	}

	// Create a unique ID for this environment
	// Use full path with non-alphanumeric characters replaced to avoid path issues
	id := strings.TrimPrefix(endpointPath, "/")
	if id == "" {
		id = "root"
	} else {
		// Convert path separators and other problematic characters to underscores
		id = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return '_'
		}, id)
	}

	// Add a random suffix to avoid collisions
	randBytes := make([]byte, 4)
	for i := range randBytes {
		randBytes[i] = byte(time.Now().Nanosecond() % 256)
		time.Sleep(time.Nanosecond)
	}
	idSuffix := fmt.Sprintf("_%x", randBytes)
	id = id + idSuffix

	// Create a temporary directory for this environment
	tempPath := filepath.Join(c.baseDir, id)
	if err := os.RemoveAll(tempPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error removing existing environment: %w", err)
	}
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return nil, fmt.Errorf("error creating environment directory: %w", err)
	}

	env := &PHPEnvironment{
		ID:           id,
		OriginalPath: originalPath,
		EndpointPath: endpointPath,
		TempPath:     tempPath,
		LastUpdated:  time.Now(),
	}

	// Mirror all files to the environment
	if err := c.mirrorFilesToEnvironment(env); err != nil {
		os.RemoveAll(tempPath)
		return nil, err
	}

	c.logger.Printf("Created environment for %s at %s", endpointPath, tempPath)
	return env, nil
}

// updateEnvironmentIfNeeded checks if an environment needs to be updated and rebuilds it if necessary
func (c *EnvironmentCache) updateEnvironmentIfNeeded(env *PHPEnvironment) error {
	env.mutex.Lock()
	defer env.mutex.Unlock()

	// Check if the original file has been modified
	fileInfo, err := os.Stat(env.OriginalPath)
	if err != nil {
		return fmt.Errorf("error checking file %s: %w", env.OriginalPath, err)
	}

	// If the file has been modified since the environment was last updated, rebuild it
	if fileInfo.ModTime().After(env.LastUpdated) {
		c.logger.Printf("Rebuilding environment for %s due to file change", env.EndpointPath)
		if err := c.mirrorFilesToEnvironment(env); err != nil {
			return fmt.Errorf("error rebuilding environment: %w", err)
		}
		env.LastUpdated = time.Now()
	}

	return nil
}

// mirrorFilesToEnvironment mirrors all files from the source directory to the environment
func (c *EnvironmentCache) mirrorFilesToEnvironment(env *PHPEnvironment) error {
	// Get the directory containing the original file
	sourceDir := c.sourceDir

	// Mirror all files from the source directory to the environment
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories - we'll create them when we copy files
		if info.IsDir() {
			return nil
		}

		// Calculate the relative path from the source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("error calculating relative path: %w", err)
		}

		// Calculate the target path in the environment
		targetPath := filepath.Join(env.TempPath, relPath)

		// Create the directory for this file
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("error creating directory for %s: %w", targetPath, err)
		}

		// Copy the file
		sourceData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}

		if err := os.WriteFile(targetPath, sourceData, 0644); err != nil {
			return fmt.Errorf("error writing file %s: %w", targetPath, err)
		}

		return nil
	})
}

// Cleanup removes all environments
func (c *EnvironmentCache) Cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, env := range c.environments {
		os.RemoveAll(env.TempPath)
	}

	c.environments = make(map[string]*PHPEnvironment)

	c.logger.Printf("Cleaned up all environments")
}

// Framework-specific adapters

// For Gin returns a handler function for use with Gin
func (m *Middleware) ForGin(pathPrefix string) interface{} {
	return func(c interface{}) {
		// This is a placeholder implementation - in a real implementation
		// we would extract the http.ResponseWriter and *http.Request
		// from the gin.Context and call our ServeHTTP method if the path
		// matches our pathPrefix or is a PHP file
	}
}

// ForEcho returns a middleware function for use with Echo
func (m *Middleware) ForEcho() interface{} {
	return func(next interface{}) interface{} {
		return func(c interface{}) error {
			// This is a placeholder implementation - in a real implementation
			// we would extract the http.ResponseWriter and *http.Request
			// from the echo.Context and call our ServeHTTP method if needed
			return nil
		}
	}
}

// ForChi returns a middleware function for use with Chi router
func (m *Middleware) ForChi() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.shouldHandlePHP(r) {
				m.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SetRenderHandler directly registers a render function for a specific URL path
func (m *Middleware) SetRenderHandler(pattern string, renderFn RenderData) {
	renderHandlersMutex.Lock()
	renderHandlers[pattern] = renderFn
	renderHandlersMutex.Unlock()
	m.logger.Printf("Registered render handler for path: %s", pattern)
}

// HandleEmbedWithRender combines adding an embedded PHP file and registering a render function in a single call
func (m *Middleware) HandleEmbedWithRender(
	urlPath string,
	embedFS embed.FS,
	embedPath string,
	renderFn RenderData,
) string {
	// Add the file from embed
	targetPath := m.AddFromEmbed(urlPath, embedFS, embedPath)

	// Set the render handler for the path
	m.SetRenderHandler(urlPath, renderFn)

	m.logger.Printf("Registered embedded PHP file with render handler at %s", urlPath)

	return targetPath
}
