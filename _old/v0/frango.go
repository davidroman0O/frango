package frango

import (
	"bytes"
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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dunglas/frankenphp"
)

// --- PHP Utility Scripts ---

// pathUtilityScript is the PHP code that defines the $_PATH superglobal
// and related utilities that make route parameters more accessible.
const pathUtilityScript = `<?php
/**
 * Frango utility script that defines the $_PATH superglobal
 * 
 * This is automatically included in PHP environments to provide
 * a clean interface for accessing path parameters.
 */

// Initialize the $_PATH superglobal
global $_PATH;
$_PATH = [];

// Scan $_SERVER for path parameters (old format for backward compatibility)
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'FRANGO_PARAM_') === 0) {
        $paramName = substr($key, 13); // Remove 'FRANGO_PARAM_' prefix
        $_PATH[$paramName] = $value;
    }
}

// Scan for the new format with serialized path parameters
if (isset($_SERVER['FRANGO_PATH_PARAMS_JSON']) && !empty($_SERVER['FRANGO_PATH_PARAMS_JSON'])) {
    $pathParams = json_decode($_SERVER['FRANGO_PATH_PARAMS_JSON'], true);
    if (is_array($pathParams)) {
        $_PATH = array_merge($_PATH, $pathParams);
    }
}

// Define a helper function to get path segments as an array
function path_segments() {
    $segments = [];
    
    // Extract from FRANGO_URL_SEGMENT_ variables
    $count = isset($_SERVER['FRANGO_URL_SEGMENT_COUNT']) ? (int)$_SERVER['FRANGO_URL_SEGMENT_COUNT'] : 0;
    
    for ($i = 0; $i < $count; $i++) {
        $key = 'FRANGO_URL_SEGMENT_' . $i;
        if (isset($_SERVER[$key])) {
            $segments[] = $_SERVER[$key];
        }
    }
    
    return $segments;
}

// Make segments available as $_PATH_SEGMENTS
global $_PATH_SEGMENTS;
$_PATH_SEGMENTS = path_segments();
`

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

// RequestData contains all relevant information extracted from an HTTP request
type RequestData struct {
	Method       string
	FullURL      string
	Path         string
	RemoteAddr   string
	Headers      http.Header
	QueryParams  url.Values
	PathSegments []string // URL path split by "/"
	JSONBody     map[string]interface{}
	FormData     url.Values
}

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

// For returns an http.Handler that executes a PHP script.
// scriptPath can be relative to the SourceDir or an absolute path.
// The pattern is automatically extracted from the request.
func (m *Middleware) For(scriptPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve script path immediately if relative
		absScriptPath := m.resolveScriptPath(scriptPath)

		// Block direct PHP access in URLs if enabled
		if m.blockDirectPHPURLs && strings.HasSuffix(strings.ToLower(r.URL.Path), ".php") {
			// Get registered pattern from context if available
			registeredUrlPattern := r.URL.Path
			patternKey := php12PatternContextKey(r.Context())
			if patternKey != "" {
				// Extract the pattern part without method
				if parts := strings.SplitN(patternKey, " ", 2); len(parts) > 1 {
					registeredUrlPattern = parts[1]
				} else {
					registeredUrlPattern = patternKey
				}
			}

			// Special case: If this is explicitly the script we're serving, allow it
			baseScript := filepath.Base(scriptPath)
			if registeredUrlPattern == "/"+baseScript {
				m.logger.Printf("Allowing explicitly registered PHP path: %s", registeredUrlPattern)
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

		// Extract pattern from context for path parameter extraction
		registeredPattern := r.URL.Path // Default fallback

		// Get the actual route pattern from the request's context if available
		if patternKey := php12PatternContextKey(r.Context()); patternKey != "" {
			registeredPattern = patternKey // Use the full pattern from context
			m.logger.Printf("Using pattern from context: %s", registeredPattern)

			// Extract parameters from pattern and URL path
			requestPath := r.URL.Path
			pathParams := extractPathParams(registeredPattern, requestPath)
			if pathParams != nil && len(pathParams) > 0 {
				// Log the extracted parameters
				m.logger.Printf("Extracted path parameters: %v", pathParams)

				// Add to environment variables
				paramsJSON, _ := json.Marshal(pathParams)
				// These will be picked up by path_globals.php
				os.Setenv("FRANGO_PATH_PARAMS_JSON", string(paramsJSON))
				for key, value := range pathParams {
					os.Setenv("FRANGO_PARAM_"+key, value)
				}
			}
		} else {
			m.logger.Printf("No pattern found in context, using URL path: %s", registeredPattern)
		}

		// Execute PHP with the appropriate registered pattern for parameter extraction
		m.executePHP(absScriptPath, nil, w, r)

		// Clean up environment variables
		if patternKey := php12PatternContextKey(r.Context()); patternKey != "" {
			pathParams := extractPathParams(patternKey, r.URL.Path)
			if pathParams != nil && len(pathParams) > 0 {
				os.Unsetenv("FRANGO_PATH_PARAMS_JSON")
				for key := range pathParams {
					os.Unsetenv("FRANGO_PARAM_" + key)
				}
			}
		}
	})
}

// Render returns an http.Handler that executes a PHP script with data.
// scriptPath can be relative to the SourceDir or an absolute path.
// The pattern is automatically extracted from the request.
func (m *Middleware) Render(scriptPath string, renderFn RenderData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve script path immediately if relative
		absScriptPath := m.resolveScriptPath(scriptPath)

		// Initialization check
		if !m.ensureInitialized(r.Context()) {
			http.Error(w, "PHP initialization error", http.StatusInternalServerError)
			return
		}

		// Extract pattern from context for path parameter extraction
		registeredPattern := r.URL.Path // Default fallback

		// Get the actual route pattern from the request's context if available
		if patternKey := php12PatternContextKey(r.Context()); patternKey != "" {
			registeredPattern = patternKey // Use the full pattern from context
			m.logger.Printf("Using pattern from context: %s", registeredPattern)
		} else {
			m.logger.Printf("No pattern found in context, using URL path: %s", registeredPattern)
		}

		// Execute PHP with render data and the appropriate pattern for parameter extraction
		m.executePHP(absScriptPath, renderFn, w, r)
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
		handler := frangoInstance.For(scriptPathForHandler)
		routes = append(routes, FileSystemRoute{Method: method, Pattern: patternPath, Handler: handler, ScriptPath: path})
		frangoInstance.logger.Printf("Mapped FS Route: [%s] %s -> %s", method, patternPath, path)

		// --- Generate Implicit Routes (if enabled and method allows) ---
		// Only generate clean/index for GET or ANY method routes
		if method == "" || method == http.MethodGet {
			if generateClean && strings.HasSuffix(patternPath, ".php") {
				cleanPattern := strings.TrimSuffix(patternPath, ".php")
				if cleanPattern != urlPrefix || len(cleanPattern) > 0 { // Avoid root conflict
					cleanHandler := frangoInstance.For(scriptPathForHandler)
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
					dirHandler := frangoInstance.For(scriptPathForHandler)
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

// --- Virtual Filesystem Types ---

// VirtualFS represents a virtual filesystem container for PHP files
type VirtualFS struct {
	name              string
	sourceMappings    map[string]string // Virtual path -> source path
	reverseSource     map[string]string // Source path -> virtual path
	embedMappings     map[string]string // Virtual path -> embed temp path
	baseTempPath      string            // Base temp dir for this VFS
	sourceHashes      map[string]string // Source path -> content hash
	middleware        *Middleware
	mutex             sync.RWMutex
	invalidated       bool              // Whether this VFS needs refresh
	invalidatedPaths  map[string]bool   // Specific paths that need refresh
	watchTicker       *time.Ticker      // Ticker for file watching
	watchStop         chan bool         // Channel to stop watching
	fileOrigins       map[string]string // Virtual path -> origin type ("source", "embed", "virtual")
	virtualFiles      map[string][]byte // Virtual path -> content for virtual files
	virtualFileHashes map[string]string // Virtual path -> hash for virtual files
}

// NewFS creates a new virtual filesystem container
func (m *Middleware) NewFS() *VirtualFS {
	vfs := &VirtualFS{
		name:              generateUniqueID(),
		sourceMappings:    make(map[string]string),
		reverseSource:     make(map[string]string),
		embedMappings:     make(map[string]string),
		sourceHashes:      make(map[string]string),
		invalidatedPaths:  make(map[string]bool),
		fileOrigins:       make(map[string]string),
		virtualFiles:      make(map[string][]byte),
		virtualFileHashes: make(map[string]string),
		watchStop:         make(chan bool),
		middleware:        m,
	}

	// Create base temp dir for this VFS
	tempPath, err := os.MkdirTemp(m.tempDir, "vfs-"+vfs.name+"-")
	if err != nil {
		m.logger.Printf("Warning: Failed to create VFS temp dir: %v", err)
		tempPath = filepath.Join(m.tempDir, "vfs-"+vfs.name)
		os.MkdirAll(tempPath, 0755)
	}
	vfs.baseTempPath = tempPath

	return vfs
}

// AddSourceFile adds a single file from the filesystem to the VFS
func (v *VirtualFS) AddSourceFile(sourcePath string, virtualPath string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

	// Get absolute path for the source file
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("error resolving absolute path for '%s': %w", sourcePath, err)
	}

	// Verify file exists
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("error accessing file '%s': %w", absPath, err)
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("source path '%s' is a directory, expected a file", absPath)
	}

	// Calculate initial hash
	hash, err := calculateFileHash(absPath)
	if err != nil {
		return fmt.Errorf("error calculating hash for '%s': %w", absPath, err)
	}

	// Store mappings
	v.sourceMappings[virtualPath] = absPath
	v.reverseSource[absPath] = virtualPath
	v.sourceHashes[absPath] = hash
	v.fileOrigins[virtualPath] = "source"

	v.middleware.logger.Printf("Added source file mapping: %s -> %s (hash: %s)", virtualPath, absPath, hash[:8])

	// Ensure file watching in development mode (if not already running)
	if v.middleware.developmentMode && v.watchTicker == nil {
		go v.watchSourceFiles()
	}

	return nil
}

// AddSourceDirectory adds all files from a source directory to the VFS
// The pathPattern can contain glob patterns (e.g., "./php/dashboard/*")
// The virtualPrefix is the base path to mount these files in the VFS
func (v *VirtualFS) AddSourceDirectory(pathPattern string, virtualPrefix string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual prefix
	virtualPrefix = filepath.Clean("/" + strings.TrimPrefix(virtualPrefix, "/"))

	// Expand the glob pattern
	matches, err := filepath.Glob(pathPattern)
	if err != nil {
		return fmt.Errorf("error expanding glob pattern '%s': %w", pathPattern, err)
	}

	for _, match := range matches {
		absPath, err := filepath.Abs(match)
		if err != nil {
			v.middleware.logger.Printf("Warning: Could not resolve absolute path for '%s': %v", match, err)
			continue
		}

		fileInfo, err := os.Stat(absPath)
		if err != nil {
			v.middleware.logger.Printf("Warning: Could not stat '%s': %v", absPath, err)
			continue
		}

		if fileInfo.IsDir() {
			// Process the directory recursively
			err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					relPath, err := filepath.Rel(absPath, path)
					if err != nil {
						return nil // Skip file with error
					}

					virtualPath := filepath.Join(virtualPrefix, relPath)
					sourcePath := path

					// Calculate initial hash
					hash, _ := calculateFileHash(sourcePath)

					// Store mappings
					v.sourceMappings[virtualPath] = sourcePath
					v.reverseSource[sourcePath] = virtualPath
					v.sourceHashes[sourcePath] = hash
					v.fileOrigins[virtualPath] = "source"

					v.middleware.logger.Printf("Added source file mapping: %s -> %s (hash: %s)", virtualPath, sourcePath, hash[:8])
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("error walking directory '%s': %w", absPath, err)
			}
		} else {
			// Single file
			baseName := filepath.Base(absPath)
			virtualPath := filepath.Join(virtualPrefix, baseName)
			sourcePath := absPath

			// Calculate initial hash
			hash, _ := calculateFileHash(sourcePath)

			// Store mappings
			v.sourceMappings[virtualPath] = sourcePath
			v.reverseSource[sourcePath] = virtualPath
			v.sourceHashes[sourcePath] = hash
			v.fileOrigins[virtualPath] = "source"

			v.middleware.logger.Printf("Added source file mapping: %s -> %s (hash: %s)", virtualPath, sourcePath, hash[:8])
		}
	}

	// Schedule file watching in development mode (if not already running)
	if v.middleware.developmentMode && v.watchTicker == nil {
		go v.watchSourceFiles()
	}

	return nil
}

// AddEmbeddedFile adds a single file from an embed.FS to the VFS
func (v *VirtualFS) AddEmbeddedFile(embedFS embed.FS, fsPath string, virtualPath string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

	// Read the content from the embedded filesystem
	content, err := embedFS.ReadFile(fsPath)
	if err != nil {
		return fmt.Errorf("error reading embedded file '%s': %w", fsPath, err)
	}

	// Create target directory in VFS temp space
	targetDir := filepath.Dir(filepath.Join(v.baseTempPath, virtualPath))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("error creating directory for embedded file '%s': %w", targetDir, err)
	}

	// Write to temp path
	tempPath := filepath.Join(v.baseTempPath, virtualPath)
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing embedded file to '%s': %w", tempPath, err)
	}

	// Store mapping
	v.embedMappings[virtualPath] = tempPath
	v.fileOrigins[virtualPath] = "embed"
	v.middleware.logger.Printf("Added embedded file mapping: %s -> %s", virtualPath, tempPath)

	return nil
}

// AddEmbeddedDirectory adds an entire directory from an embed.FS to the VFS
func (v *VirtualFS) AddEmbeddedDirectory(embedFS embed.FS, fsPath string, virtualPrefix string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual prefix
	virtualPrefix = filepath.Clean("/" + strings.TrimPrefix(virtualPrefix, "/"))

	// List the directory contents
	entries, err := embedFS.ReadDir(fsPath)
	if err != nil {
		return fmt.Errorf("error reading embedded directory '%s': %w", fsPath, err)
	}

	// Process each entry
	for _, entry := range entries {
		entryPath := filepath.Join(fsPath, entry.Name())
		virtualEntryPath := filepath.Join(virtualPrefix, entry.Name())

		if entry.IsDir() {
			// Recursively process subdirectory
			if err := v.AddEmbeddedDirectory(embedFS, entryPath, virtualEntryPath); err != nil {
				return err
			}
		} else {
			// Process file
			content, err := embedFS.ReadFile(entryPath)
			if err != nil {
				v.middleware.logger.Printf("Warning: Could not read embedded file '%s': %v", entryPath, err)
				continue
			}

			// Create target directory in VFS temp space
			targetDir := filepath.Dir(filepath.Join(v.baseTempPath, virtualEntryPath))
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				v.middleware.logger.Printf("Warning: Could not create directory for embedded file '%s': %v", targetDir, err)
				continue
			}

			// Write to temp path
			tempPath := filepath.Join(v.baseTempPath, virtualEntryPath)
			if err := os.WriteFile(tempPath, content, 0644); err != nil {
				v.middleware.logger.Printf("Warning: Could not write embedded file to '%s': %v", tempPath, err)
				continue
			}

			// Store mapping
			v.embedMappings[virtualEntryPath] = tempPath
			v.fileOrigins[virtualEntryPath] = "embed"
			v.middleware.logger.Printf("Added embedded file mapping: %s -> %s", virtualEntryPath, tempPath)
		}
	}

	return nil
}

// CreateVirtualFile creates a file directly in the virtual filesystem with provided content
func (v *VirtualFS) CreateVirtualFile(virtualPath string, content []byte) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

	// Create target directory in VFS temp space
	targetDir := filepath.Dir(filepath.Join(v.baseTempPath, virtualPath))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("error creating directory for virtual file '%s': %w", targetDir, err)
	}

	// Write to temp path
	tempPath := filepath.Join(v.baseTempPath, virtualPath)
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing virtual file to '%s': %w", tempPath, err)
	}

	// Store the content and mapping
	v.virtualFiles[virtualPath] = content
	v.embedMappings[virtualPath] = tempPath // Use embed mappings for write access
	v.fileOrigins[virtualPath] = "virtual"

	// Calculate hash of content
	h := sha256.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))
	v.virtualFileHashes[virtualPath] = hash

	v.middleware.logger.Printf("Created virtual file: %s (hash: %s)", virtualPath, hash[:8])

	return nil
}

// CopyFile copies a file from one virtual path to another within the VFS
func (v *VirtualFS) CopyFile(srcVirtualPath, destVirtualPath string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize paths
	srcVirtualPath = filepath.Clean("/" + strings.TrimPrefix(srcVirtualPath, "/"))
	destVirtualPath = filepath.Clean("/" + strings.TrimPrefix(destVirtualPath, "/"))

	// Resolve the actual source path
	var content []byte
	var err error

	// Check the source type
	originType, exists := v.fileOrigins[srcVirtualPath]
	if !exists {
		return fmt.Errorf("source file not found in VFS: %s", srcVirtualPath)
	}

	switch originType {
	case "source":
		// Read from filesystem
		sourcePath := v.sourceMappings[srcVirtualPath]
		content, err = os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("error reading source file '%s': %w", sourcePath, err)
		}
	case "embed", "virtual":
		// For embedded or virtual files, get from the temp location
		if tempPath, ok := v.embedMappings[srcVirtualPath]; ok {
			content, err = os.ReadFile(tempPath)
			if err != nil {
				return fmt.Errorf("error reading embedded/virtual file '%s': %w", tempPath, err)
			}
		} else if originType == "virtual" {
			// Get from in-memory content for virtual files
			content = v.virtualFiles[srcVirtualPath]
		} else {
			return fmt.Errorf("embedded file mapping not found: %s", srcVirtualPath)
		}
	default:
		return fmt.Errorf("unknown file origin type for %s: %s", srcVirtualPath, originType)
	}

	// Create target directory in VFS temp space
	targetDir := filepath.Dir(filepath.Join(v.baseTempPath, destVirtualPath))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("error creating directory for destination '%s': %w", targetDir, err)
	}

	// Write to destination temp path
	tempPath := filepath.Join(v.baseTempPath, destVirtualPath)
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("error writing file to '%s': %w", tempPath, err)
	}

	// Store as a virtual file
	v.virtualFiles[destVirtualPath] = content
	v.embedMappings[destVirtualPath] = tempPath
	v.fileOrigins[destVirtualPath] = "virtual"

	// Calculate hash
	h := sha256.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))
	v.virtualFileHashes[destVirtualPath] = hash

	v.middleware.logger.Printf("Copied file: %s -> %s (hash: %s)", srcVirtualPath, destVirtualPath, hash[:8])

	return nil
}

// MoveFile moves a file from one virtual path to another within the VFS
func (v *VirtualFS) MoveFile(srcVirtualPath, destVirtualPath string) error {
	// First copy the file
	if err := v.CopyFile(srcVirtualPath, destVirtualPath); err != nil {
		return err
	}

	// Then delete the source
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Don't actually delete source files from disk
	originType := v.fileOrigins[srcVirtualPath]

	// Remove mappings
	delete(v.embedMappings, srcVirtualPath)
	delete(v.virtualFiles, srcVirtualPath)
	delete(v.virtualFileHashes, srcVirtualPath)
	delete(v.fileOrigins, srcVirtualPath)

	// For source files, only remove the virtual mapping, not the actual file
	if originType == "source" {
		sourcePath := v.sourceMappings[srcVirtualPath]
		delete(v.sourceMappings, srcVirtualPath)
		delete(v.reverseSource, sourcePath)
		// We keep the sourceHashes entry for monitoring changes
	}

	v.middleware.logger.Printf("Moved file: %s -> %s", srcVirtualPath, destVirtualPath)
	return nil
}

// DeleteFile removes a file from the VFS
func (v *VirtualFS) DeleteFile(virtualPath string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Normalize virtual path
	virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

	// Check if file exists in VFS
	originType, exists := v.fileOrigins[virtualPath]
	if !exists {
		return fmt.Errorf("file not found in VFS: %s", virtualPath)
	}

	// Remove mappings based on origin type
	if originType == "source" {
		sourcePath := v.sourceMappings[virtualPath]
		delete(v.sourceMappings, virtualPath)
		delete(v.reverseSource, sourcePath)
		// We don't delete the source file from disk
	} else if originType == "embed" || originType == "virtual" {
		if tempPath, ok := v.embedMappings[virtualPath]; ok {
			// Try to remove the temp file but don't error if it fails
			_ = os.Remove(tempPath)
			delete(v.embedMappings, virtualPath)
		}
		delete(v.virtualFiles, virtualPath)
		delete(v.virtualFileHashes, virtualPath)
	}

	delete(v.fileOrigins, virtualPath)
	v.middleware.logger.Printf("Deleted file from VFS: %s", virtualPath)
	return nil
}

// ListFiles returns a list of all files in the VFS
func (v *VirtualFS) ListFiles() []string {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	files := make([]string, 0, len(v.fileOrigins))
	for path := range v.fileOrigins {
		files = append(files, path)
	}
	return files
}

// GetFileContent reads the content of a file from the VFS
func (v *VirtualFS) GetFileContent(virtualPath string) ([]byte, error) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Normalize virtual path
	virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

	// Check file origin type
	originType, exists := v.fileOrigins[virtualPath]
	if !exists {
		return nil, fmt.Errorf("file not found in VFS: %s", virtualPath)
	}

	// Get content based on origin type
	switch originType {
	case "source":
		sourcePath := v.sourceMappings[virtualPath]
		return os.ReadFile(sourcePath)
	case "embed", "virtual":
		// For virtual files, use in-memory content if available
		if originType == "virtual" && len(v.virtualFiles[virtualPath]) > 0 {
			return v.virtualFiles[virtualPath], nil
		}
		// Otherwise, read from temp path
		if tempPath, ok := v.embedMappings[virtualPath]; ok {
			return os.ReadFile(tempPath)
		}
		return nil, fmt.Errorf("error resolving file path: %s", virtualPath)
	default:
		return nil, fmt.Errorf("unknown file origin type: %s", originType)
	}
}

// --- Internal methods ---

// resolvePath translates a virtual path to its actual filesystem path
func (v *VirtualFS) resolvePath(virtualPath string) string {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Check origin type to prioritize correctly
	originType, exists := v.fileOrigins[virtualPath]
	if !exists {
		return ""
	}

	// Based on the origin type, get the appropriate path
	switch originType {
	case "source":
		return v.sourceMappings[virtualPath]
	case "embed", "virtual":
		return v.embedMappings[virtualPath]
	default:
		return ""
	}
}

// watchSourceFiles periodically checks source files for changes
func (v *VirtualFS) watchSourceFiles() {
	// Stop existing watcher if any
	if v.watchTicker != nil {
		v.watchTicker.Stop()
	}

	v.watchTicker = time.NewTicker(500 * time.Millisecond)
	go func() {
		for {
			select {
			case <-v.watchTicker.C:
				v.checkFileChanges()
			case <-v.watchStop:
				v.watchTicker.Stop()
				return
			}
		}
	}()
}

// StopWatching stops the file watching goroutine
func (v *VirtualFS) StopWatching() {
	if v.watchTicker != nil {
		v.watchStop <- true
		v.watchTicker = nil
	}
}

// checkFileChanges checks if any source files have changed
func (v *VirtualFS) checkFileChanges() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Check source files
	for sourcePath, oldHash := range v.sourceHashes {
		// Skip if file doesn't exist
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		// Calculate new hash
		newHash, err := calculateFileHash(sourcePath)
		if err != nil {
			v.middleware.logger.Printf("Warning: Could not calculate hash for '%s': %v", sourcePath, err)
			continue
		}

		// Check if hash changed
		if newHash != oldHash {
			virtualPath := v.reverseSource[sourcePath]
			v.middleware.logger.Printf("Source file changed: %s (virtual: %s)", sourcePath, virtualPath)
			v.middleware.logger.Printf("  Hash: %s -> %s", oldHash[:8], newHash[:8])

			// Update hash
			v.sourceHashes[sourcePath] = newHash

			// Mark path as invalidated
			v.invalidatedPaths[virtualPath] = true
			v.invalidated = true
		}
	}

	// Check virtual files (for external modifications to temp files)
	for virtualPath, oldHash := range v.virtualFileHashes {
		if tempPath, ok := v.embedMappings[virtualPath]; ok {
			// Skip if file doesn't exist
			if _, err := os.Stat(tempPath); os.IsNotExist(err) {
				continue
			}

			// Calculate new hash
			newHash, err := calculateFileHash(tempPath)
			if err != nil {
				continue
			}

			// Check if hash changed
			if newHash != oldHash {
				v.middleware.logger.Printf("Virtual file changed on disk: %s", virtualPath)
				v.middleware.logger.Printf("  Hash: %s -> %s", oldHash[:8], newHash[:8])

				// Read the new content
				content, err := os.ReadFile(tempPath)
				if err == nil {
					// Update in-memory content
					v.virtualFiles[virtualPath] = content
					v.virtualFileHashes[virtualPath] = newHash
				}

				// Mark path as invalidated
				v.invalidatedPaths[virtualPath] = true
				v.invalidated = true
			}
		}
	}
}

// refreshIfNeeded ensures the PHP environment is updated if files changed
func (v *VirtualFS) refreshIfNeeded(virtualPath string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Check if this specific path was invalidated
	if v.invalidatedPaths[virtualPath] {
		v.middleware.logger.Printf("Refreshing environment for path: %s", virtualPath)
		delete(v.invalidatedPaths, virtualPath)

		// Force environment refresh for this path by invalidating any cache
		originType := v.fileOrigins[virtualPath]
		if originType == "source" {
			// For source files, invalidate by source path
			if sourcePath, ok := v.sourceMappings[virtualPath]; ok {
				v.middleware.logger.Printf("Invalidating cache for source file: %s", sourcePath)
				// Find any environments using this path and invalidate them
				for _, env := range v.middleware.envCache.environments {
					if env.OriginalPath == sourcePath {
						// Force update by clearing its hash
						env.mutex.Lock()
						env.OriginalFileHash = ""
						env.mutex.Unlock()
						break
					}
				}
			}
		} else if originType == "embed" || originType == "virtual" {
			// For embed/virtual files, invalidate by temp path
			if tempPath, ok := v.embedMappings[virtualPath]; ok {
				v.middleware.logger.Printf("Invalidating cache for embedded/virtual file: %s", tempPath)
				// Find any environments using this path and invalidate them
				for _, env := range v.middleware.envCache.environments {
					if env.OriginalPath == tempPath {
						// Force update by clearing its hash
						env.mutex.Lock()
						env.OriginalFileHash = ""
						env.mutex.Unlock()
						break
					}
				}
			}
		}
	}
}

// For returns an http.Handler that executes a PHP script from the VFS
func (v *VirtualFS) For(virtualPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for file changes if needed
		if v.middleware.developmentMode {
			v.refreshIfNeeded(virtualPath)
		}

		// Normalize virtual path
		virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

		// Resolve the actual path
		actualPath := v.resolvePath(virtualPath)
		if actualPath == "" {
			v.middleware.logger.Printf("Error: Virtual path not found in VFS: %s", virtualPath)
			http.NotFound(w, r)
			return
		}

		// Initialization check
		if !v.middleware.ensureInitialized(r.Context()) {
			http.Error(w, "PHP initialization error", http.StatusInternalServerError)
			return
		}

		// Execute PHP
		v.middleware.executePHP(actualPath, nil, w, r)
	})
}

// Render returns an http.Handler that executes a PHP script with data
func (v *VirtualFS) Render(virtualPath string, renderFn RenderData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for file changes if needed
		if v.middleware.developmentMode {
			v.refreshIfNeeded(virtualPath)
		}

		// Normalize virtual path
		virtualPath = filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))

		// Resolve the actual path
		actualPath := v.resolvePath(virtualPath)
		if actualPath == "" {
			v.middleware.logger.Printf("Error: Virtual path not found in VFS: %s", virtualPath)
			http.NotFound(w, r)
			return
		}

		// Initialization check
		if !v.middleware.ensureInitialized(r.Context()) {
			http.Error(w, "PHP initialization error", http.StatusInternalServerError)
			return
		}

		// Execute PHP with render data
		v.middleware.executePHP(actualPath, renderFn, w, r)
	})
}

// generateUniqueID creates a unique identifier for VFS instances
func generateUniqueID() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:8]
}

// --- Internal Methods (Middleware Core) ---

// resolveScriptPath ensures the script path is absolute.
// If relative, it's joined with the SourceDir.
func (m *Middleware) resolveScriptPath(scriptPath string) string {
	if !filepath.IsAbs(scriptPath) {
		// Assume relative to SourceDir
		return filepath.Join(m.sourceDir, scriptPath)
	}
	return scriptPath // Already absolute
}

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

// executePHP handles the core logic of preparing the environment and executing a PHP script.
// Takes the absolute path to the PHP script to execute.
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

// php12PatternContextKey extracts the pattern from Go 1.22 ServeMux context
// This is a helper function to extract the pattern from the context in Go 1.22+
func php12PatternContextKey(ctx context.Context) string {
	// Try several known context keys for Go 1.22 ServeMux
	for _, key := range []interface{}{"pattern", "http.pattern", phpContextKey("pattern"), phpContextKey("http.pattern")} {
		if val, ok := ctx.Value(key).(string); ok && val != "" {
			return val
		}
	}

	// Try a more exhaustive approach - inspect context for any pattern-like keys
	type ctxKey struct{}
	contextDump := fmt.Sprintf("%+v", ctx.Value(ctxKey{}))
	if strings.Contains(contextDump, "pattern") {
		log.Printf("Context contains pattern key but unable to extract: %s", contextDump)
	}

	return ""
}

// ExtractRequestData pulls all relevant data from an HTTP request
func ExtractRequestData(r *http.Request) *RequestData {
	// Parse form and multipart form data
	r.ParseForm()
	r.ParseMultipartForm(32 << 20) // 32MB max

	// Get path segments
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	// Try to parse JSON body if content type indicates JSON
	var jsonBody map[string]interface{}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		// Save the body so it can still be read later
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			// Restore the body for other handlers
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Attempt to decode as JSON
			_ = json.Unmarshal(bodyBytes, &jsonBody)
		}
	}

	// Build the complete request data
	return &RequestData{
		Method:       r.Method,
		FullURL:      r.URL.String(),
		Path:         r.URL.Path,
		RemoteAddr:   r.RemoteAddr,
		Headers:      r.Header,
		QueryParams:  r.URL.Query(),
		PathSegments: segments,
		JSONBody:     jsonBody,
		FormData:     r.Form,
	}
}
