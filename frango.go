// Package frango provides a simple way to integrate PHP code with Go applications
// using FrankenPHP.
package frango

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"embed"

	"github.com/dunglas/frankenphp"
)

// HandlerOptions configures how PHP files are served
type HandlerOptions struct {
	// SourceDir is the directory containing PHP files (empty for embedded files)
	SourceDir string
	// DevelopmentMode enables immediate file change detection and disables caching
	DevelopmentMode bool
	// CheckInterval specifies how often to check for file changes in production mode
	CheckInterval time.Duration
	// CacheDuration specifies browser cache duration in production mode
	CacheDuration time.Duration
	// Logger for output (defaults to standard logger if nil)
	Logger *log.Logger
}

// Server represents a PHP server instance
type Server struct {
	options        HandlerOptions
	sourceDir      string
	tempDir        string
	logger         *log.Logger
	initialized    bool
	endpoints      map[string]string // Maps URL paths to PHP files
	customHandlers map[string]http.HandlerFunc
	embedFS        any               // Optional embedded filesystem
	embedPath      string            // Base path within the embedded filesystem
	embedFiles     map[string]any    // Map of individual embedded files
	envCache       *EnvironmentCache // Environment cache
}

// EmbedOptions provides configuration options for embedded files
type EmbedOptions struct {
	// Path within the embed.FS, if different from virtualPath
	Path string
	// Don't automatically register as PHP endpoint
	NoAutoRegister bool
	// Additional options could be added here in the future
}

// PHPFileOptions configures behavior when adding PHP files
type PHPFileOptions struct {
	// HandleEndpoints determines whether endpoints should be automatically registered
	HandleEndpoints bool
	// HandleCleanPath registers the path without .php extension
	HandleCleanPath bool
	// HandleRoot for index.php files, also register at "/"
	HandleRoot bool
}

// DefaultPHPFileOptions returns default options for adding PHP files
func DefaultPHPFileOptions() PHPFileOptions {
	return PHPFileOptions{
		HandleEndpoints: false, // Don't register automatically by default
		HandleCleanPath: false,
		HandleRoot:      false,
	}
}

// DefaultHandlerOptions returns the default handler options
func DefaultHandlerOptions() HandlerOptions {
	return HandlerOptions{
		SourceDir:       "", // Empty by default
		DevelopmentMode: true,
		CheckInterval:   5 * time.Second,
		CacheDuration:   60, // seconds
		Logger:          nil,
	}
}

// // I need to debate with myself about this one
// // StaticHandlerOptions returns handler options for a directory of static PHP files
// func StaticHandlerOptions(sourceDir string) HandlerOptions {
// 	options := DefaultHandlerOptions()
// 	options.SourceDir = sourceDir
// 	return options
// }

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

// NewServer creates a new PHP server with the given options
func NewServer(opts ...Option) (*Server, error) {
	// Default options
	options := DefaultHandlerOptions()

	// Create server with default options
	server := &Server{
		options:        options,
		endpoints:      make(map[string]string),
		customHandlers: make(map[string]http.HandlerFunc),
		embedFiles:     make(map[string]any),
	}

	// Apply options
	for _, opt := range opts {
		opt(server)
	}

	var absSourceDir string
	var err error

	// If sourceDir is empty, create a temp directory for embedded files
	if server.options.SourceDir == "" {
		absSourceDir, err = os.MkdirTemp("", "frango-server")
		if err != nil {
			return nil, fmt.Errorf("error creating temporary directory: %w", err)
		}
	} else {
		// Resolve source directory using the path resolution function
		absSourceDir, err = ResolveDirectory(server.options.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("error resolving source directory: %w", err)
		}
	}

	// Create temporary directory for environments
	tempDir, err := os.MkdirTemp("", "frango-environments")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}

	// Set up logger if not already set
	if server.logger == nil {
		if server.options.Logger != nil {
			server.logger = server.options.Logger
		} else {
			server.logger = log.New(os.Stdout, "[frango] ", log.LstdFlags)
		}
	}

	// Update server with resolved paths
	server.sourceDir = absSourceDir
	server.tempDir = tempDir

	// Create environment cache
	server.envCache = NewEnvironmentCache(absSourceDir, tempDir, server.logger, server.options.DevelopmentMode)

	return server, nil
}

// initialize initializes the PHP environment with context (private method)
func (s *Server) initialize(ctx context.Context) error {
	if s.initialized {
		return nil
	}

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

	s.initialized = true
	return nil
}

// Shutdown cleans up resources
func (s *Server) Shutdown() {
	if s.initialized {
		frankenphp.Shutdown()
		s.initialized = false
	}

	// Clean up all environments
	s.envCache.Cleanup()

	// Remove the temp directory
	os.RemoveAll(s.tempDir)
}

// HandleFunc registers a custom Go handler function for a specific URL pattern
func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	// Ensure URL path starts with a slash
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	s.customHandlers[pattern] = handler
	s.logger.Printf("Registered handler function for: %s", pattern)
}

// RenderFunc is a handler function type that can inject variables into PHP rendering
type RenderFunc func(w http.ResponseWriter, r *http.Request) map[string]interface{}

// HandleRenderEmbed registers a handler that lets you inject variables into an embedded PHP template before rendering
func (s *Server) HandleRenderEmbed(pattern string, embedFS embed.FS, fsPath string, renderFunc RenderFunc) {
	s.logger.Printf("HandleRenderEmbed: Setting up template at %s from embedded file %s", pattern, fsPath)

	// Ensure pattern starts with a slash
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// Create the target path for extraction
	targetPath := pattern
	if !strings.HasSuffix(targetPath, ".php") {
		targetPath = targetPath + ".php"
	}

	// Remove leading slash for filesystem path
	targetPath = strings.TrimPrefix(targetPath, "/")

	// Full path where the file will be extracted
	fullPath := filepath.Join(s.sourceDir, targetPath)
	s.logger.Printf("HandleRenderEmbed: Will extract to %s", fullPath)

	// Ensure directory exists
	if targetDir := filepath.Dir(fullPath); targetDir != "" {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			s.logger.Printf("Error creating directory for %s: %v", targetPath, err)
			return
		}
	}

	// Extract the file from the embed.FS
	data, err := embedFS.ReadFile(fsPath)
	if err != nil {
		s.logger.Printf("Error reading embedded file %s: %v", fsPath, err)
		return
	}

	// Write the file to the target path
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		s.logger.Printf("Error writing file %s: %v", fullPath, err)
		return
	}

	s.logger.Printf("Extracted embedded file %s to %s", fsPath, fullPath)

	// Register a handler for this pattern
	s.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		// Get variables from the render function
		variables := renderFunc(w, r)
		s.logger.Printf("HandleRenderEmbed: Render function returned %d variables for %s", len(variables), pattern)

		// Create empty path parameters (could be extended later)
		pathParams := make(map[string]string)

		// Get or create environment for this endpoint
		env, err := s.envCache.GetEnvironment(pattern, fullPath)
		if err != nil {
			s.logger.Printf("Error setting up environment for %s: %v", pattern, err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Calculate relative path to the PHP file in the environment
		phpFileInEnv := filepath.Join(env.TempPath, targetPath)

		// Ensure the PHP file is up to date in development mode
		if s.options.DevelopmentMode {
			s.envCache.mirrorFilesToEnvironment(env)
		}

		// Using the same approach as servePHPFileWithPathParams
		// Calculate the document root and script name
		documentRoot := filepath.Dir(phpFileInEnv)
		scriptName := "/" + filepath.Base(phpFileInEnv)

		s.logger.Printf("Running PHP with DocumentRoot=%s, ScriptName=%s", documentRoot, scriptName)

		// Clone the request
		reqClone := r.Clone(r.Context())

		// Set URL path to script name (this is crucial for FrankenPHP to find the file)
		reqClone.URL.Path = scriptName

		// Setup PHP environment variables
		phpEnv := map[string]string{
			// DO NOT set SCRIPT_FILENAME - FrankenPHP does this automatically
			"SCRIPT_NAME":    scriptName,
			"PHP_SELF":       scriptName,
			"DOCUMENT_ROOT":  documentRoot,
			"REQUEST_URI":    r.URL.RequestURI(),
			"REQUEST_METHOD": r.Method,
			"QUERY_STRING":   r.URL.RawQuery,
			"HTTP_HOST":      r.Host,
		}

		// Add debugging variables
		phpEnv["DEBUG_DOCUMENT_ROOT"] = documentRoot
		phpEnv["DEBUG_SCRIPT_NAME"] = scriptName
		phpEnv["DEBUG_PHP_FILE_PATH"] = phpFileInEnv
		phpEnv["DEBUG_URL_PATH"] = pattern
		phpEnv["DEBUG_SOURCE_PATH"] = fullPath
		phpEnv["DEBUG_ENV_ID"] = env.ID

		// Add path parameters
		if len(pathParams) > 0 {
			// Create a JSON string with all path parameters
			pathParamsJSON, _ := json.Marshal(pathParams)
			phpEnv["PATH_PARAMS"] = string(pathParamsJSON)

			// Also add individual path parameters for easier access
			for name, value := range pathParams {
				phpEnv["PATH_PARAM_"+strings.ToUpper(name)] = value
			}
		}

		// Add custom variables from the render function
		for key, value := range variables {
			jsonValue, err := json.Marshal(value)
			if err != nil {
				s.logger.Printf("Error marshaling variable %s: %v", key, err)
				continue
			}

			// Add as environment variable - use both PHP_SELF prefix and frango_VAR prefix
			// Some versions of PHP might need different prefixes
			varName := "frango_VAR_" + key
			phpEnv[varName] = string(jsonValue)
			// Also add with PHP_ prefix as some configurations may need this
			phpEnv["PHP_"+varName] = string(jsonValue)
			// Add directly to the CGI environment as well
			phpEnv["HTTP_"+varName] = string(jsonValue)

			s.logger.Printf("Added variable %s with value of length %d", varName, len(string(jsonValue)))
		}

		// Debug: Add a special marker value to help diagnose where values are accessible
		phpEnv["DEBUG_FRANGO_MARKER"] = "THIS_SHOULD_BE_VISIBLE_IN_PHP"
		phpEnv["HTTP_DEBUG_FRANGO_MARKER"] = "THIS_SHOULD_BE_VISIBLE_IN_PHP"
		phpEnv["PHP_DEBUG_FRANGO_MARKER"] = "THIS_SHOULD_BE_VISIBLE_IN_PHP"

		// Create FrankenPHP request
		req, err := frankenphp.NewRequestWithContext(
			reqClone,
			frankenphp.WithRequestDocumentRoot(documentRoot, false),
			frankenphp.WithRequestEnv(phpEnv),
		)
		if err != nil {
			s.logger.Printf("Error creating PHP request: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Execute PHP
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			s.logger.Printf("Error executing PHP: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// HandleRender registers a handler that lets you inject variables into a PHP template before rendering
func (s *Server) HandleRender(pattern string, phpFile string, renderFunc RenderFunc) {
	// Ensure pattern starts with a slash
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// Verify the PHP file exists
	phpFilePath := phpFile
	if !filepath.IsAbs(phpFile) {
		phpFilePath = filepath.Join(s.sourceDir, phpFile)
	}

	// Check if the file exists
	fileInfo, err := os.Stat(phpFilePath)
	if err != nil {
		s.logger.Printf("Error accessing PHP file %s: %v", phpFilePath, err)
		return
	}

	if fileInfo.IsDir() {
		s.logger.Printf("PHP file path is a directory: %s", phpFilePath)
		return
	}

	// Register endpoint
	s.endpoints[pattern] = phpFilePath
	s.logger.Printf("Registered render endpoint: %s -> %s", pattern, phpFilePath)

	// Register a handler for this pattern
	s.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		// Get variables from the render function
		variables := renderFunc(w, r)

		// Create empty path parameters (could be extended later)
		pathParams := make(map[string]string)

		// Get or create environment for this endpoint
		env, err := s.envCache.GetEnvironment(pattern, phpFilePath)
		if err != nil {
			s.logger.Printf("Error setting up environment for %s: %v", pattern, err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Calculate relative path to the PHP file
		relPath, err := filepath.Rel(s.sourceDir, phpFilePath)
		if err != nil {
			s.logger.Printf("Error calculating relative path: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Calculate the path to the PHP file in the environment
		phpFileInEnv := filepath.Join(env.TempPath, relPath)

		// Ensure the PHP file is up to date in development mode
		if s.options.DevelopmentMode {
			s.envCache.mirrorFilesToEnvironment(env)
		}

		// Using the same approach as servePHPFileWithPathParams
		// Calculate the document root and script name
		documentRoot := filepath.Dir(phpFileInEnv)
		scriptName := "/" + filepath.Base(phpFileInEnv)

		s.logger.Printf("Running PHP with DocumentRoot=%s, ScriptName=%s", documentRoot, scriptName)

		// Clone the request
		reqClone := r.Clone(r.Context())

		// Set URL path to script name (this is crucial for FrankenPHP to find the file)
		reqClone.URL.Path = scriptName

		// Setup PHP environment variables
		phpEnv := map[string]string{
			// DO NOT set SCRIPT_FILENAME - FrankenPHP does this automatically
			"SCRIPT_NAME":    scriptName,
			"PHP_SELF":       scriptName,
			"DOCUMENT_ROOT":  documentRoot,
			"REQUEST_URI":    r.URL.RequestURI(),
			"REQUEST_METHOD": r.Method,
			"QUERY_STRING":   r.URL.RawQuery,
			"HTTP_HOST":      r.Host,
		}

		// Add debugging variables
		phpEnv["DEBUG_DOCUMENT_ROOT"] = documentRoot
		phpEnv["DEBUG_SCRIPT_NAME"] = scriptName
		phpEnv["DEBUG_PHP_FILE_PATH"] = phpFileInEnv
		phpEnv["DEBUG_URL_PATH"] = pattern
		phpEnv["DEBUG_SOURCE_PATH"] = phpFilePath
		phpEnv["DEBUG_ENV_ID"] = env.ID

		// Add path parameters
		if len(pathParams) > 0 {
			// Create a JSON string with all path parameters
			pathParamsJSON, _ := json.Marshal(pathParams)
			phpEnv["PATH_PARAMS"] = string(pathParamsJSON)

			// Also add individual path parameters for easier access
			for name, value := range pathParams {
				phpEnv["PATH_PARAM_"+strings.ToUpper(name)] = value
			}
		}

		// Add custom variables from the render function
		for key, value := range variables {
			jsonValue, err := json.Marshal(value)
			if err != nil {
				s.logger.Printf("Error marshaling variable %s: %v", key, err)
				continue
			}

			// Add as environment variable - use both PHP_SELF prefix and frango_VAR prefix
			// Some versions of PHP might need different prefixes
			varName := "frango_VAR_" + key
			phpEnv[varName] = string(jsonValue)
			// Also add with PHP_ prefix as some configurations may need this
			phpEnv["PHP_"+varName] = string(jsonValue)
			// Add directly to the CGI environment as well
			phpEnv["HTTP_"+varName] = string(jsonValue)

			s.logger.Printf("Added variable %s with value of length %d", varName, len(string(jsonValue)))
		}

		// Debug: Add a special marker value to help diagnose where values are accessible
		phpEnv["DEBUG_FRANGO_MARKER"] = "THIS_SHOULD_BE_VISIBLE_IN_PHP"
		phpEnv["HTTP_DEBUG_FRANGO_MARKER"] = "THIS_SHOULD_BE_VISIBLE_IN_PHP"
		phpEnv["PHP_DEBUG_FRANGO_MARKER"] = "THIS_SHOULD_BE_VISIBLE_IN_PHP"

		// Create FrankenPHP request
		req, err := frankenphp.NewRequestWithContext(
			reqClone,
			frankenphp.WithRequestDocumentRoot(documentRoot, false),
			frankenphp.WithRequestEnv(phpEnv),
		)
		if err != nil {
			s.logger.Printf("Error creating PHP request: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Execute PHP
		if err := frankenphp.ServeHTTP(w, req); err != nil {
			s.logger.Printf("Error executing PHP: %v", err)
			http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// HandleDir registers all PHP files in a directory under a URL prefix
func (s *Server) HandleDir(prefix string, dirPath string) error {
	// Ensure URL prefix starts with a slash
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	// If trailing slash, remove it
	if prefix != "/" && strings.HasSuffix(prefix, "/") {
		prefix = prefix[:len(prefix)-1]
	}

	// If the directory is not an absolute path, make it relative to source dir
	if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(s.sourceDir, dirPath)
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

			// Create URL path
			urlPath := prefix
			if prefix != "/" {
				urlPath = prefix + "/"
			}
			urlPath += relPath

			// Remove .php extension for cleaner URLs (will be added back when needed)
			urlPath = strings.TrimSuffix(urlPath, ".php")

			// Register endpoint using HandlePHP
			s.HandlePHP(urlPath, path)
			count++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	s.logger.Printf("Registered %d PHP files from directory %s under %s", count, dirPath, prefix)
	return nil
}

// findPathInEmbedFS attempts to find a file path in an embed.FS
// This function uses reflection to inspect the embed.FS and find embedded files
// If successful, it returns the path of the first (or only) file found
func findPathInEmbedFS(embedFS any) (string, error) {
	// Get the value of the embed.FS
	val := reflect.ValueOf(embedFS)

	// Try using the ReadDir method (available in Go 1.16+)
	readDirMethod := val.MethodByName("ReadDir")
	if readDirMethod.IsValid() {
		// Try to scan the root directory first
		results := readDirMethod.Call([]reflect.Value{reflect.ValueOf(".")})
		if len(results) == 2 && results[1].IsNil() {
			// Process the directory entries
			dirEntries := results[0].Interface()
			dirEntriesVal := reflect.ValueOf(dirEntries)

			if dirEntriesVal.Kind() == reflect.Slice && dirEntriesVal.Len() > 0 {
				// Find the first PHP file, or if no PHP files, the first file
				var firstFile string
				var firstPHPFile string

				for i := 0; i < dirEntriesVal.Len(); i++ {
					entry := dirEntriesVal.Index(i).Interface()
					entryVal := reflect.ValueOf(entry)

					// Get name and check if it's a file
					nameMethod := entryVal.MethodByName("Name")
					isFileMethod := entryVal.MethodByName("IsDir")

					if nameMethod.IsValid() && isFileMethod.IsValid() {
						nameResults := nameMethod.Call(nil)
						isFileResults := isFileMethod.Call(nil)

						if len(nameResults) == 1 && len(isFileResults) == 1 {
							name := nameResults[0].String()
							isDir := isFileResults[0].Bool()

							if !isDir {
								// Save the first file we find
								if firstFile == "" {
									firstFile = name
								}

								// If it's a PHP file, that's our preference
								if strings.HasSuffix(strings.ToLower(name), ".php") {
									firstPHPFile = name
									break
								}
							}
						}
					}
				}

				// Prefer PHP files, but fall back to any file
				if firstPHPFile != "" {
					return firstPHPFile, nil
				}
				if firstFile != "" {
					return firstFile, nil
				}
			}
		}

		// If root directory didn't work, recursively search subdirectories
		return recursivelyFindPathInEmbedFS(embedFS, val, "")
	}

	// Check if the Open method exists as a fallback
	openMethod := val.MethodByName("Open")
	if !openMethod.IsValid() {
		return "", fmt.Errorf("not a valid embed.FS (no Open method)")
	}

	// As a last resort, try some common filenames
	commonFiles := []string{"index.php", "main.php", "app.php"}

	// Get type name for a hint
	typeName := reflect.TypeOf(embedFS).String()
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) > 0 {
			baseName := parts[len(parts)-1]
			baseName = strings.TrimSuffix(baseName, "FS")
			baseName = strings.TrimSuffix(baseName, "Fs")
			if baseName != "" {
				commonFiles = append([]string{baseName + ".php"}, commonFiles...)
			}
		}
	}

	// Try common filenames
	for _, path := range commonFiles {
		results := openMethod.Call([]reflect.Value{reflect.ValueOf(path)})
		if len(results) == 2 && results[1].IsNil() {
			return path, nil
		}
	}

	return "", fmt.Errorf("couldn't find a file in the embed.FS automatically")
}

// recursivelyFindPathInEmbedFS recursively searches for files in subdirectories
func recursivelyFindPathInEmbedFS(embedFS any, val reflect.Value, prefix string) (string, error) {
	readDirMethod := val.MethodByName("ReadDir")
	if !readDirMethod.IsValid() {
		return "", fmt.Errorf("invalid embed.FS")
	}

	results := readDirMethod.Call([]reflect.Value{reflect.ValueOf(prefix)})
	if len(results) != 2 || !results[1].IsNil() {
		return "", fmt.Errorf("failed to read directory: %s", prefix)
	}

	dirEntries := results[0].Interface()
	dirEntriesVal := reflect.ValueOf(dirEntries)

	if dirEntriesVal.Kind() != reflect.Slice {
		return "", fmt.Errorf("invalid directory entries")
	}

	// First look for PHP files
	for i := 0; i < dirEntriesVal.Len(); i++ {
		entry := dirEntriesVal.Index(i).Interface()
		entryVal := reflect.ValueOf(entry)

		nameMethod := entryVal.MethodByName("Name")
		isFileMethod := entryVal.MethodByName("IsDir")

		if nameMethod.IsValid() && isFileMethod.IsValid() {
			nameResults := nameMethod.Call(nil)
			isFileResults := isFileMethod.Call(nil)

			if len(nameResults) == 1 && len(isFileResults) == 1 {
				name := nameResults[0].String()
				isDir := isFileResults[0].Bool()

				path := name
				if prefix != "" {
					path = prefix + "/" + name
				}

				if !isDir && strings.HasSuffix(strings.ToLower(name), ".php") {
					return path, nil
				}
			}
		}
	}

	// Then check subdirectories
	for i := 0; i < dirEntriesVal.Len(); i++ {
		entry := dirEntriesVal.Index(i).Interface()
		entryVal := reflect.ValueOf(entry)

		nameMethod := entryVal.MethodByName("Name")
		isFileMethod := entryVal.MethodByName("IsDir")

		if nameMethod.IsValid() && isFileMethod.IsValid() {
			nameResults := nameMethod.Call(nil)
			isFileResults := isFileMethod.Call(nil)

			if len(nameResults) == 1 && len(isFileResults) == 1 {
				name := nameResults[0].String()
				isDir := isFileResults[0].Bool()

				if isDir {
					subdir := name
					if prefix != "" {
						subdir = prefix + "/" + name
					}

					if path, err := recursivelyFindPathInEmbedFS(embedFS, val, subdir); err == nil {
						return path, nil
					}
				}
			}
		}
	}

	// If no PHP files, return the first file found
	for i := 0; i < dirEntriesVal.Len(); i++ {
		entry := dirEntriesVal.Index(i).Interface()
		entryVal := reflect.ValueOf(entry)

		nameMethod := entryVal.MethodByName("Name")
		isFileMethod := entryVal.MethodByName("IsDir")

		if nameMethod.IsValid() && isFileMethod.IsValid() {
			nameResults := nameMethod.Call(nil)
			isFileResults := isFileMethod.Call(nil)

			if len(nameResults) == 1 && len(isFileResults) == 1 {
				name := nameResults[0].String()
				isDir := isFileResults[0].Bool()

				path := name
				if prefix != "" {
					path = prefix + "/" + name
				}

				if !isDir {
					return path, nil
				}
			}
		}
	}

	return "", fmt.Errorf("couldn't find a file in the embed.FS automatically")
}

// AddEmbeddedFile adds an individual embedded file
// virtualPath is the virtual path where this file will be accessible
// embedFS is the embed.FS containing the file
// options allows customizing the embedding behavior
// Returns virtual path that was registered
func (s *Server) AddEmbeddedFile(virtualPath string, embedFS any, options ...EmbedOptions) string {
	// Default options
	var opts EmbedOptions
	if len(options) > 0 {
		opts = options[0]
	}

	// If Path is not specified, try to auto-detect it
	embedPath := opts.Path
	if embedPath == "" {
		// First try the virtualPath
		embedPath = virtualPath

		// Then try to auto-detect if it's a single-file embed
		if autoPath, err := findPathInEmbedFS(embedFS); err == nil {
			embedPath = autoPath
			s.logger.Printf("Auto-detected embedded file path: %s", embedPath)
		}
	}

	s.embedFiles[virtualPath] = struct {
		fs   any
		path string
	}{
		fs:   embedFS,
		path: embedPath,
	}
	s.logger.Printf("Added embedded file: %s -> %s", virtualPath, embedPath)

	// If this is a PHP file and auto-registration is not disabled, register it as an endpoint
	if strings.HasSuffix(virtualPath, ".php") && !opts.NoAutoRegister {
		// First, extract the file to the temp directory so it exists when accessed
		targetPath := filepath.Join(s.sourceDir, virtualPath)
		if targetDir := filepath.Dir(targetPath); targetDir != "" {
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				s.logger.Printf("Warning: Failed to create directory for %s: %v", virtualPath, err)
				return virtualPath
			}
		}

		// Extract the file immediately so it exists on disk
		if err := s.getFileFromEmbed(virtualPath, targetPath); err != nil {
			s.logger.Printf("Warning: Failed to extract embedded file %s: %v", virtualPath, err)
			return virtualPath
		}

		// Now register both the full path (.php) and the shortened version (without .php)
		s.HandlePHP(virtualPath, targetPath)

		// Also register the clean version (without .php extension)
		endpointPath := strings.TrimSuffix(virtualPath, ".php")
		if endpointPath == "" {
			endpointPath = "/"
		}

		// Only register clean version if it's different from the original path
		if endpointPath != virtualPath {
			s.HandlePHP(endpointPath, targetPath)
			s.logger.Printf("Registered PHP endpoint: %s -> %s", endpointPath, targetPath)
		}
	}

	return virtualPath
}

// getFileFromEmbed retrieves a file from the embedded filesystem or individual files and extracts it if needed
func (s *Server) getFileFromEmbed(requestPath, targetPath string) error {
	// First check individually embedded files
	for virtualPath, embedInfo := range s.embedFiles {
		embedFS := embedInfo.(struct {
			fs   any
			path string
		}).fs
		embedPath := embedInfo.(struct {
			fs   any
			path string
		}).path

		// Check if this virtual path matches
		if virtualPath == requestPath || (strings.HasPrefix(requestPath, virtualPath) && virtualPath != "/") {
			// Use reflection to access the ReadFile method on the embed.FS
			readFileMethod := reflect.ValueOf(embedFS).MethodByName("ReadFile")
			if !readFileMethod.IsValid() {
				continue
			}

			// Call the ReadFile method
			results := readFileMethod.Call([]reflect.Value{reflect.ValueOf(embedPath)})
			if len(results) != 2 || !results[1].IsNil() {
				continue
			}

			// Get content
			content := results[0].Bytes()

			// Ensure directory exists
			targetDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
			}

			// Write to file
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return fmt.Errorf("error writing file %s: %w", targetPath, err)
			}

			s.logger.Printf("Extracted individually embedded file %s to %s", embedPath, targetPath)
			return nil
		}
	}

	// Fall back to the main embedded filesystem if none of the individual files matched
	if s.embedFS == nil {
		return fmt.Errorf("no embedded filesystem available")
	}

	embedPath := filepath.Join(s.embedPath, strings.TrimPrefix(requestPath, "/"))

	// Ensure directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Use reflection to access the ReadFile method on the embed.FS
	readFileMethod := reflect.ValueOf(s.embedFS).MethodByName("ReadFile")
	if !readFileMethod.IsValid() {
		return fmt.Errorf("embedded filesystem does not have ReadFile method")
	}

	// Call the ReadFile method
	results := readFileMethod.Call([]reflect.Value{reflect.ValueOf(embedPath)})
	if len(results) != 2 {
		return fmt.Errorf("unexpected result from ReadFile")
	}

	// Check for error
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return fmt.Errorf("error reading embedded file %s: %w", embedPath, err)
	}

	// Get content
	content := results[0].Bytes()

	// Write to file
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", targetPath, err)
	}

	s.logger.Printf("Extracted embedded file %s to %s", embedPath, targetPath)
	return nil
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Initialize if needed
	if !s.initialized {
		if err := s.initialize(r.Context()); err != nil {
			s.logger.Printf("Error initializing server: %v", err)
			http.Error(w, "Server initialization error", http.StatusInternalServerError)
			return
		}
	}

	// For root path, automatically serve index.php if it exists
	if r.URL.Path == "/" {
		indexPath := filepath.Join(s.sourceDir, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			s.servePHPFile("/", indexPath, w, r)
			return
		}
	}

	// Check for custom handler
	if handler, exists := s.customHandlers[r.URL.Path]; exists {
		handler(w, r)
		return
	}

	// Check for registered endpoint
	phpFile, found := s.endpoints[r.URL.Path]
	if !found {
		// If no explicit match, try default endpoint for root
		if r.URL.Path == "/" && s.endpoints["/"] == "" {
			// Look for index.php in source directory
			defaultIndex := filepath.Join(s.sourceDir, "index.php")

			// Try to extract from embedded filesystem if it doesn't exist
			if _, err := os.Stat(defaultIndex); os.IsNotExist(err) && (s.embedFS != nil || len(s.embedFiles) > 0) {
				if err := s.getFileFromEmbed("/index.php", defaultIndex); err == nil {
					phpFile = defaultIndex
				}
			} else if err == nil {
				phpFile = defaultIndex
			}
		}

		// Check for static file or directory
		if phpFile == "" {
			staticPath := filepath.Join(s.sourceDir, strings.TrimPrefix(r.URL.Path, "/"))

			// Check if it's a directory
			if stat, err := os.Stat(staticPath); err == nil && stat.IsDir() {
				// Try to serve index.php from this directory
				indexPath := filepath.Join(staticPath, "index.php")
				if _, err := os.Stat(indexPath); err == nil {
					s.servePHPFile(r.URL.Path, indexPath, w, r)
					return
				}
			}

			// Try to extract from embedded filesystem if it doesn't exist
			if _, err := os.Stat(staticPath); os.IsNotExist(err) && (s.embedFS != nil || len(s.embedFiles) > 0) {
				if err := s.getFileFromEmbed(r.URL.Path, staticPath); err == nil {
					// If it's a PHP file, serve it as PHP
					if strings.HasSuffix(staticPath, ".php") {
						phpFile = staticPath
					} else {
						// Serve extracted static file
						http.ServeFile(w, r, staticPath)
						return
					}
				}
			} else if err == nil {
				// File exists in source directory
				if strings.HasSuffix(staticPath, ".php") {
					phpFile = staticPath
				} else {
					// Serve static file
					http.ServeFile(w, r, staticPath)
					return
				}
			}
		}
	}

	// If no PHP file found, return 404
	if phpFile == "" {
		http.NotFound(w, r)
		return
	}

	// Serve PHP file
	s.servePHPFile(r.URL.Path, phpFile, w, r)
}

// servePHPFile serves a PHP file
func (s *Server) servePHPFile(urlPath string, sourcePath string, w http.ResponseWriter, r *http.Request) {
	// Call servePHPFileWithPathParams with empty path parameters
	s.servePHPFileWithPathParams(urlPath, sourcePath, make(map[string]string), w, r)
}

// servePHPFileWithPathParams serves a PHP file with path parameters
func (s *Server) servePHPFileWithPathParams(urlPath string, sourcePath string, pathParams map[string]string, w http.ResponseWriter, r *http.Request) {
	// Get or create environment for this endpoint
	env, err := s.envCache.GetEnvironment(urlPath, sourcePath)
	if err != nil {
		s.logger.Printf("Error setting up environment for %s: %v", urlPath, err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Calculate the path to the original PHP file relative to the source directory
	relPath, err := filepath.Rel(s.sourceDir, sourcePath)
	if err != nil {
		s.logger.Printf("Error calculating relative path: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Calculate the path to the PHP file in the environment
	phpFilePath := filepath.Join(env.TempPath, relPath)

	// Ensure this is actually pointing to a file, not a directory
	fileInfo, err := os.Stat(phpFilePath)
	if err != nil {
		// If file doesn't exist, log and try to rebuild
		s.logger.Printf("Error accessing PHP file %s: %v", phpFilePath, err)

		// If the file doesn't exist but the environment does, try to rebuild it
		if os.IsNotExist(err) {
			s.logger.Printf("Trying to rebuild environment for %s", urlPath)
			if err := s.envCache.mirrorFilesToEnvironment(env); err != nil {
				s.logger.Printf("Error rebuilding environment: %v", err)
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}

			// Check again after rebuilding
			fileInfo, err = os.Stat(phpFilePath)
			if err != nil {
				s.logger.Printf("File still not found after rebuilding: %s", phpFilePath)
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
		s.logger.Printf("ERROR: Path is a directory, not a PHP file: %s", phpFilePath)

		// Try appending index.php if it's a directory
		indexPath := filepath.Join(phpFilePath, "index.php")
		if _, err := os.Stat(indexPath); err == nil {
			s.logger.Printf("Found index.php in directory, using: %s", indexPath)
			phpFilePath = indexPath
		} else {
			s.logger.Printf("No index.php found in directory: %s", phpFilePath)
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

	s.logger.Printf("Running PHP with DocumentRoot=%s, ScriptName=%s", documentRoot, scriptName)

	// Setup environment variables
	phpEnv := map[string]string{
		// DO NOT set SCRIPT_FILENAME - FrankenPHP does this automatically
		"SCRIPT_NAME":    scriptName,
		"PHP_SELF":       scriptName,
		"DOCUMENT_ROOT":  documentRoot,
		"REQUEST_URI":    r.URL.RequestURI(),
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
	}

	// Add path parameters to environment
	if len(pathParams) > 0 {
		// Create a JSON string with all path parameters
		pathParamsJSON, _ := json.Marshal(pathParams)
		phpEnv["PATH_PARAMS"] = string(pathParamsJSON)

		// Also add individual path parameters for easier access
		for name, value := range pathParams {
			phpEnv["PATH_PARAM_"+strings.ToUpper(name)] = value
		}
	}

	// Add caching configuration
	if !s.options.DevelopmentMode {
		phpEnv["PHP_PRODUCTION"] = "1"
		phpEnv["PHP_OPCACHE_ENABLE"] = "1"
	} else {
		phpEnv["PHP_FCGI_MAX_REQUESTS"] = "1"
		phpEnv["PHP_OPCACHE_ENABLE"] = "0"
	}

	// Clone the request and set the URL path to the script name
	// This ensures FrankenPHP looks for the right file
	reqClone := r.Clone(r.Context())
	reqClone.URL.Path = scriptName

	// Create FrankenPHP request using the correct document root
	req, err := frankenphp.NewRequestWithContext(
		reqClone,
		frankenphp.WithRequestDocumentRoot(documentRoot, false), // Document root is the environment directory
		frankenphp.WithRequestEnv(phpEnv),                       // Environment includes SCRIPT_FILENAME
	)
	if err != nil {
		s.logger.Printf("Error creating PHP request: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Execute PHP
	if err := frankenphp.ServeHTTP(w, req); err != nil {
		s.logger.Printf("Error executing PHP: %v", err)
		http.Error(w, "PHP execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// RemoveHandler removes a registered PHP handler
func (s *Server) RemoveHandler(pattern string) {
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	delete(s.endpoints, pattern)
	s.logger.Printf("Removed PHP handler: %s", pattern)
}

// RemoveHandleFunc removes a registered custom handler function
func (s *Server) RemoveHandleFunc(pattern string) {
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	delete(s.customHandlers, pattern)
	s.logger.Printf("Removed handler function: %s", pattern)
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	if err := s.initialize(ctx); err != nil {
		return fmt.Errorf("error initializing server: %w", err)
	}

	s.logger.Printf("PHP server listening on %s", addr)
	s.logger.Printf("Source directory: %s", s.sourceDir)
	s.logger.Printf("Mode: %s", func() string {
		if s.options.DevelopmentMode {
			return "DEVELOPMENT"
		}
		return "PRODUCTION"
	}())

	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return server.ListenAndServe()
}

// WithMiddleware wraps the PHP server with middleware
func (s *Server) WithMiddleware(middleware func(http.Handler) http.Handler) http.Handler {
	return middleware(s)
}

// AsMiddleware returns a middleware function that processes PHP files when appropriate
// and passes other requests to the next handler
func (s *Server) AsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a registered endpoint or if the file exists
		_, registered := s.endpoints[r.URL.Path]

		// Static file check
		relPath := strings.TrimPrefix(r.URL.Path, "/")
		staticPath := filepath.Join(s.sourceDir, relPath)

		// PHP file check (for both static files and directories with index.php)
		phpExists := false

		// Check if the path is a PHP file
		if strings.HasSuffix(staticPath, ".php") {
			if _, err := os.Stat(staticPath); err == nil {
				phpExists = true
			}
		} else {
			// Check if it's a directory with an index.php
			if stat, err := os.Stat(staticPath); err == nil && stat.IsDir() {
				indexPath := filepath.Join(staticPath, "index.php")
				if _, err := os.Stat(indexPath); err == nil {
					phpExists = true
				}
			} else if err == nil && strings.HasSuffix(staticPath, ".php") {
				// It's a PHP file
				phpExists = true
			}
		}

		// If it's registered or exists as a PHP file, handle it with the PHP server
		if registered || phpExists {
			s.ServeHTTP(w, r)
			return
		}

		// Otherwise, pass to the next handler
		if next != nil {
			next.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
}

// SourceDir returns the source directory path
func (s *Server) SourceDir() string {
	return s.sourceDir
}

// EmbeddedFile represents a file to be embedded
type EmbeddedFile struct {
	Data []byte
	Path string
}

// AddPHPFile adds a PHP file directly to the server
// urlPath: the URL path where this file will be accessible
// phpContent: the raw PHP code as bytes
// options: configuration options for how the file should be registered
// Returns the file path where it was extracted
func (s *Server) AddPHPFile(urlPath string, phpContent []byte, options ...PHPFileOptions) string {
	// Apply options
	opts := DefaultPHPFileOptions()
	if len(options) > 0 {
		opts = options[0]
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
	targetPath := filepath.Join(s.sourceDir, filePath)

	// Create directory structure
	if targetDir := filepath.Dir(targetPath); targetDir != "" {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			s.logger.Printf("Warning: Failed to create directory for %s: %v", filePath, err)
			return ""
		}
	}

	// Write file to disk
	if err := os.WriteFile(targetPath, phpContent, 0644); err != nil {
		s.logger.Printf("Warning: Failed to write file %s: %v", filePath, err)
		return ""
	}

	if opts.HandleEndpoints {
		// Register the URL path (possibly with .php)
		s.HandlePHP(urlPath, targetPath)

		// Also register the clean version (without .php extension) if requested
		if opts.HandleCleanPath && strings.HasSuffix(urlPath, ".php") {
			cleanPath := strings.TrimSuffix(urlPath, ".php")
			if cleanPath != urlPath && cleanPath != "" {
				s.HandlePHP(cleanPath, targetPath)
			}
		}

		// Register special case for "index.php" at root if requested
		if opts.HandleRoot &&
			(urlPath == "/index.php" || strings.HasSuffix(urlPath, "/index.php")) {
			// Extract the directory part from the urlPath
			dir := filepath.Dir(urlPath)
			if dir == "/" {
				// Root index.php
				s.HandlePHP("/", targetPath)
			} else {
				// Directory index.php (e.g., /foo/index.php -> /foo/)
				s.HandlePHP(dir, targetPath)
			}
		}
	}

	s.logger.Printf("Added PHP file at %s", targetPath)
	return targetPath
}

// AddPHPFromEmbed extracts a PHP file from an embed.FS and adds it to the server
// urlPath: the URL path where this file will be accessible
// fs: the embed.FS containing the file
// fsPath: the path to the file within the embed.FS
// options: configuration options for how the file should be registered
func (s *Server) AddPHPFromEmbed(urlPath string, fs embed.FS, fsPath string, options ...PHPFileOptions) string {
	// Read the file from the embed.FS
	content, err := fs.ReadFile(fsPath)
	if err != nil {
		s.logger.Printf("Error reading embedded file %s: %v", fsPath, err)
		return ""
	}

	// Add the PHP file using the content
	return s.AddPHPFile(urlPath, content, options...)
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

// CleanupEnvironment removes an environment
func (c *EnvironmentCache) CleanupEnvironment(endpointPath string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	env, exists := c.environments[endpointPath]
	if !exists {
		return
	}

	// Remove the environment directory
	os.RemoveAll(env.TempPath)

	// Remove from the map
	delete(c.environments, endpointPath)

	c.logger.Printf("Cleaned up environment for %s", endpointPath)
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

// HandleWithMethod maps a URL path with specific HTTP method to a PHP file
// Example: server.HandleWithMethod("GET /api/users/{id}", "api/user_get.php")
func (s *Server) HandleWithMethod(pattern string, phpFilePath string) {
	// Extract method and path from pattern
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) != 2 {
		s.logger.Printf("Invalid pattern format: %s. Expected format: 'METHOD /path'", pattern)
		return
	}

	method := parts[0]
	path := parts[1]

	// Ensure path starts with a slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// If the PHP file is not an absolute path, make it relative to source dir
	if !filepath.IsAbs(phpFilePath) {
		phpFilePath = filepath.Join(s.sourceDir, phpFilePath)
	}

	// Create a special handler that checks the HTTP method before processing
	s.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Check if the HTTP method matches
		if r.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract path parameters from the URL pattern
		pathParams := make(map[string]string)

		// Check if pattern contains path parameters (e.g., /users/{id})
		if strings.Contains(path, "{") && strings.Contains(path, "}") {
			// Get parameter names from pattern
			patternParts := strings.Split(path, "/")
			urlParts := strings.Split(r.URL.Path, "/")

			// Matching pattern parts with URL parts
			for i, part := range patternParts {
				if i < len(urlParts) && strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
					// Extract parameter name (remove { and })
					paramName := part[1 : len(part)-1]
					paramValue := urlParts[i]
					pathParams[paramName] = paramValue
				}
			}
		}

		// Serve the PHP file with path parameters
		s.servePHPFileWithPathParams(r.URL.Path, phpFilePath, pathParams, w, r)
	})

	s.logger.Printf("Registered %s endpoint: %s -> %s", method, path, phpFilePath)
}

// Handle is a flexible handler registration function that supports multiple formats:
// 1. Classic style: Handle("/users", "users.php")
// 2. Method-specific: Handle("GET /users", "users_get.php")
// 3. With parameters: Handle("GET /users/{id}", "user_detail.php")
func (s *Server) Handle(pattern string, phpFilePath string) {
	// Check if this is a method-specific pattern (contains a space)
	if strings.Contains(pattern, " ") {
		s.HandleWithMethod(pattern, phpFilePath)
		return
	}

	// Otherwise, register as a standard endpoint (works with all methods)
	s.HandlePHP(pattern, phpFilePath)
}

// CreateMethodRouter creates a router that supports both standard HandleFunc
// and the new Go 1.22+ pattern-based routing, integrating PHP endpoints
func (s *Server) CreateMethodRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Register any existing PHP endpoints with the mux
	for urlPath, phpFile := range s.endpoints {
		// Create a closure to capture the values
		phpFilePath := phpFile
		mux.Handle(urlPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.servePHPFile(urlPath, phpFilePath, w, r)
		}))
	}

	// Add a catch-all handler for dynamic file handling
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First check if there's a custom handler for this path
		if handler, exists := s.customHandlers[r.URL.Path]; exists {
			handler(w, r)
			return
		}

		// Otherwise use the default PHP handler
		s.ServeHTTP(w, r)
	}))

	return mux
}

// RouteBuilder provides a fluent API for defining routes
type RouteBuilder struct {
	server *Server
	mux    *http.ServeMux
}

// NewRouter creates a new router with enhanced pattern matching
func (s *Server) NewRouter() *RouteBuilder {
	return &RouteBuilder{
		server: s,
		mux:    s.CreateMethodRouter(),
	}
}

// GET registers a GET route
func (rb *RouteBuilder) GET(pattern string, handler interface{}) *RouteBuilder {
	rb.registerPatternHandler("GET", pattern, handler)
	return rb
}

// POST registers a POST route
func (rb *RouteBuilder) POST(pattern string, handler interface{}) *RouteBuilder {
	rb.registerPatternHandler("POST", pattern, handler)
	return rb
}

// PUT registers a PUT route
func (rb *RouteBuilder) PUT(pattern string, handler interface{}) *RouteBuilder {
	rb.registerPatternHandler("PUT", pattern, handler)
	return rb
}

// DELETE registers a DELETE route
func (rb *RouteBuilder) DELETE(pattern string, handler interface{}) *RouteBuilder {
	rb.registerPatternHandler("DELETE", pattern, handler)
	return rb
}

// Build returns the configured ServeMux
func (rb *RouteBuilder) Build() *http.ServeMux {
	return rb.mux
}

// ServeHTTP implements the http.Handler interface
func (rb *RouteBuilder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rb.mux.ServeHTTP(w, r)
}

// registerPatternHandler registers a pattern handler, supporting both PHP files and Go handlers
func (rb *RouteBuilder) registerPatternHandler(method string, pattern string, handler interface{}) {
	// Strip leading slash for consistency
	if pattern != "/" && strings.HasSuffix(pattern, "/") {
		pattern = pattern[:len(pattern)-1]
	}

	// Handle different handler types
	switch h := handler.(type) {
	case string:
		// String is interpreted as a PHP file path
		rb.server.Handle(method+" "+pattern, h)
	case http.HandlerFunc:
		// Go handler function
		rb.mux.HandleFunc(method+" "+pattern, h)
	case func(http.ResponseWriter, *http.Request):
		// Go handler function
		rb.mux.HandleFunc(method+" "+pattern, h)
	default:
		rb.server.logger.Printf("Unsupported handler type for %s %s: %T", method, pattern, handler)
	}
}

// Option is a function that configures a Server
type Option func(*Server)

// WithSourceDir sets the source directory for PHP files
func WithSourceDir(dir string) Option {
	return func(s *Server) {
		s.options.SourceDir = dir
	}
}

// WithDevelopmentMode enables immediate file change detection and disables caching
func WithDevelopmentMode(enabled bool) Option {
	return func(s *Server) {
		s.options.DevelopmentMode = enabled
	}
}

// WithCheckInterval sets how often to check for file changes in production mode
func WithCheckInterval(interval time.Duration) Option {
	return func(s *Server) {
		s.options.CheckInterval = interval
	}
}

// WithCacheDuration sets browser cache duration in production mode
func WithCacheDuration(duration time.Duration) Option {
	return func(s *Server) {
		s.options.CacheDuration = duration
	}
}

// WithLogger sets a custom logger
func WithLogger(logger *log.Logger) Option {
	return func(s *Server) {
		s.options.Logger = logger
		s.logger = logger
	}
}

// HandlePHP maps a URL pattern to a PHP file
func (s *Server) HandlePHP(pattern string, phpFilePath string) {
	// Ensure URL path starts with a slash
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// If the PHP file is not an absolute path, make it relative to source dir
	if !filepath.IsAbs(phpFilePath) {
		phpFilePath = filepath.Join(s.sourceDir, phpFilePath)
	}

	// Store the mapping
	s.endpoints[pattern] = phpFilePath

	// Pre-create the environment for this endpoint
	_, err := s.envCache.GetEnvironment(pattern, phpFilePath)
	if err != nil {
		s.logger.Printf("Warning: Failed to pre-create environment for %s: %v", pattern, err)
	}

	s.logger.Printf("Registered PHP handler: %s -> %s", pattern, phpFilePath)
}
