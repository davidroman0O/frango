package frango

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// pathGlobalsScript contains the code to initialize $_PATH superglobal
const pathGlobalsScript = `<?php
// Initialize $_PATH superglobal for path parameters
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available
    $pathParamsJson = $_SERVER['FRANGO_PATH_PARAMS_JSON'] ?? '{}';
    $decodedParams = json_decode($pathParamsJson, true);
    if (is_array($decodedParams)) {
        $_PATH = $decodedParams;
    }
    
    // Also add any FRANGO_PARAM_ variables from $_SERVER for backward compatibility
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'FRANGO_PARAM_') === 0) {
            $paramName = substr($key, strlen('FRANGO_PARAM_'));
            if (!isset($_PATH[$paramName])) {
                $_PATH[$paramName] = $value;
            }
        }
    }
}

// Initialize $_PATH_SEGMENTS superglobal for URL segments
if (!isset($_PATH_SEGMENTS)) {
    $_PATH_SEGMENTS = [];
    
    // Get segment count
    $segmentCount = intval($_SERVER['FRANGO_URL_SEGMENT_COUNT'] ?? 0);
    
    // Add segments to array
    for ($i = 0; $i < $segmentCount; $i++) {
        $segmentKey = "FRANGO_URL_SEGMENT_$i";
        if (isset($_SERVER[$segmentKey])) {
            $_PATH_SEGMENTS[] = $_SERVER[$segmentKey];
        }
    }
}

// Helper function to get path segments
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}
`

// MiddlewareRouter implements http.Handler and acts as a middleware
// for routing PHP requests to the appropriate handlers.
type MiddlewareRouter struct {
	php        *Middleware
	fs         *VirtualFS
	logger     *log.Logger
	next       http.Handler
	routes     map[string]string // pattern -> virtualPath
	routesMu   sync.RWMutex
	indexFiles []string
}

// NewMiddlewareRouter creates a new middleware router with the given options
func NewMiddlewareRouter(php *Middleware, next http.Handler) *MiddlewareRouter {
	return &MiddlewareRouter{
		php:        php,
		fs:         php.NewFS(),
		logger:     php.logger,
		next:       next,
		routes:     make(map[string]string),
		indexFiles: []string{"index.php"},
	}
}

// AddSourceDirectory adds a directory of PHP files to the router
func (r *MiddlewareRouter) AddSourceDirectory(sourceDir, urlPrefix string) error {
	err := r.fs.AddSourceDirectory(sourceDir, "/")
	if err != nil {
		return fmt.Errorf("error adding source directory: %w", err)
	}

	return r.mapFileSystemRoutes(urlPrefix)
}

// AddSourceFile adds a single PHP file to the router
func (r *MiddlewareRouter) AddSourceFile(sourceFile, urlPath string) error {
	virtualPath := "/" + strings.TrimPrefix(urlPath, "/")
	err := r.fs.AddSourceFile(sourceFile, virtualPath)
	if err != nil {
		return fmt.Errorf("error adding source file: %w", err)
	}

	r.routesMu.Lock()
	r.routes[urlPath] = virtualPath
	r.routesMu.Unlock()

	return nil
}

// AddRoute registers a route pattern (can include path parameters like {id})
// to be served by the given PHP file
func (r *MiddlewareRouter) AddRoute(pattern string, phpFilePath string) error {
	// Normalize paths
	pattern = "/" + strings.TrimPrefix(pattern, "/")
	virtualPath := "/" + strings.TrimPrefix(phpFilePath, "/")

	// Check if PHP file exists
	if r.fs.For(virtualPath) == nil {
		return fmt.Errorf("PHP file %s not found in virtual filesystem", phpFilePath)
	}

	// Register the route
	r.routesMu.Lock()
	r.routes[pattern] = virtualPath
	r.routesMu.Unlock()

	r.logger.Printf("Added parameterized route: %s => %s", pattern, virtualPath)
	return nil
}

// AddEmbeddedDirectory adds an embedded directory to the router
func (r *MiddlewareRouter) AddEmbeddedDirectory(embedFS embed.FS, fsPath, urlPrefix string) error {
	err := r.fs.AddEmbeddedDirectory(embedFS, fsPath, "/")
	if err != nil {
		return fmt.Errorf("error adding embedded directory: %w", err)
	}

	return r.mapFileSystemRoutes(urlPrefix)
}

// ServeHTTP implements http.Handler
func (r *MiddlewareRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	urlPath := req.URL.Path

	// Check for root path
	if urlPath == "/" {
		if handler := r.phpHandlerForPath("/index.php"); handler != nil {
			r.logger.Printf("Handling root path with index.php")
			handler.ServeHTTP(w, req)
			return
		}
	}

	// Check for mapped routes
	r.routesMu.RLock()
	virtualPath, exists := r.routes[urlPath]
	r.routesMu.RUnlock()

	if exists {
		handler := r.phpHandlerForPath(virtualPath)
		if handler != nil {
			r.logger.Printf("Handling route %s with PHP file %s", urlPath, virtualPath)
			handler.ServeHTTP(w, req)
			return
		}
	}

	// Check for parameterized routes
	params, paramVirtualPath := r.matchParameterizedRoute(urlPath)
	if paramVirtualPath != "" {
		// Add path parameters to the request context
		ctx := req.Context()

		// Create environment variables for path parameters
		envVars := make(map[string]string)
		for name, value := range params {
			envVars["FRANGO_PARAM_"+name] = value
		}

		// Add JSON form of parameters
		if len(params) > 0 {
			jsonParams, err := json.Marshal(params)
			if err == nil {
				envVars["FRANGO_PATH_PARAMS_JSON"] = string(jsonParams)
			}

			// Create PHP code for path initialization
			phpCode := `<?php
// Initialize $_PATH with parameters directly
$_PATH = ` + phpArrayFromMap(params) + `;

// Make it globally available
$GLOBALS['_PATH'] = $_PATH;
?>`
			// Use data URI for auto_prepend_file (this is executed before the main script)
			envVars["PHP_AUTO_PREPEND_FILE"] = "data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte(phpCode))
		}

		// Create a context with the path parameters
		ctx = context.WithValue(ctx, phpContextKey("path_params"), params)
		ctx = context.WithValue(ctx, phpContextKey("env_vars"), envVars)

		// Get handler for the PHP file
		handler := r.phpHandlerForPath(paramVirtualPath)
		if handler != nil {
			r.logger.Printf("Handling parameterized route %s with PHP file %s (params: %v)", urlPath, paramVirtualPath, params)
			handler.ServeHTTP(w, req.WithContext(ctx))
			return
		}
	}

	// Check if this is a directory path that might map to an index file
	indexPath := filepath.Join(urlPath, "index.php")
	normalizedIndexPath := "/" + strings.TrimPrefix(indexPath, "/")

	r.routesMu.RLock()
	_, indexExists := r.routes[normalizedIndexPath]
	r.routesMu.RUnlock()

	if indexExists {
		handler := r.phpHandlerForPath(normalizedIndexPath)
		if handler != nil {
			r.logger.Printf("Handling directory path %s with index file %s", urlPath, normalizedIndexPath)
			handler.ServeHTTP(w, req)
			return
		}
	}

	// If we got here, no PHP route was found, pass to next handler
	if r.next != nil {
		r.logger.Printf("No PHP route found for %s, passing to next handler", urlPath)
		r.next.ServeHTTP(w, req)
	} else {
		r.logger.Printf("No PHP route found for %s and no next handler, returning 404", urlPath)
		http.NotFound(w, req)
	}
}

// phpHandlerForPath returns a handler for the given PHP file path
func (r *MiddlewareRouter) phpHandlerForPath(virtualPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get the original handler from VFS
		origHandler := r.fs.For(virtualPath)
		if origHandler == nil {
			r.logger.Printf("No PHP handler found for %s", virtualPath)
			http.NotFound(w, req)
			return
		}

		// Check if the request has path parameters
		var params map[string]string
		if ctx := req.Context(); ctx != nil {
			if p, ok := ctx.Value(phpContextKey("path_params")).(map[string]string); ok && len(p) > 0 {
				params = p

				// Set environment variables directly
				// These will be picked up by the PHP script itself
				for name, value := range params {
					os.Setenv("FRANGO_PARAM_"+name, value)
				}

				// Set JSON form of parameters
				jsonParams, err := json.Marshal(params)
				if err == nil {
					os.Setenv("FRANGO_PATH_PARAMS_JSON", string(jsonParams))
				}

				// Create a direct PHP variable initialization
				r.logger.Printf("Setting path parameters via environment: %v", params)
			}
		}

		// Call the original handler
		origHandler.ServeHTTP(w, req)

		// Clean up environment variables if needed
		if params != nil {
			for name := range params {
				os.Unsetenv("FRANGO_PARAM_" + name)
			}
			os.Unsetenv("FRANGO_PATH_PARAMS_JSON")
		}
	})
}

// phpArrayFromMap converts a Go map to PHP array syntax
func phpArrayFromMap(m map[string]string) string {
	if len(m) == 0 {
		return "[]"
	}

	var parts []string
	for k, v := range m {
		// Escape the key and value for PHP
		k = strings.ReplaceAll(k, "'", "\\'")
		v = strings.ReplaceAll(v, "'", "\\'")
		parts = append(parts, fmt.Sprintf("'%s' => '%s'", k, v))
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

// mapFileSystemRoutes scans the VirtualFS and maps files to URL routes
func (r *MiddlewareRouter) mapFileSystemRoutes(urlPrefix string) error {
	files := r.fs.ListFiles()
	r.logger.Printf("Mapping routes for %d files with URL prefix: %s", len(files), urlPrefix)

	for _, virtualPath := range files {
		// Skip non-PHP files
		if !strings.HasSuffix(virtualPath, ".php") {
			continue
		}

		// Calculate URL path
		routePath := r.calculateRoutePath(virtualPath, urlPrefix)

		// Store route mapping
		r.routesMu.Lock()
		r.routes[routePath] = virtualPath
		r.routesMu.Unlock()

		r.logger.Printf("Mapped route: %s => %s", routePath, virtualPath)
	}

	return nil
}

// calculateRoutePath converts a VirtualFS path to a URL route path
func (r *MiddlewareRouter) calculateRoutePath(virtualPath, urlPrefix string) string {
	// Handle index files specially
	fileName := filepath.Base(virtualPath)
	for _, indexFile := range r.indexFiles {
		if fileName == indexFile {
			// For index.php, use the directory path
			dirPath := filepath.Dir(virtualPath)
			if dirPath == "." || dirPath == "/" {
				// Root index.php
				urlPath := "/"
				if urlPrefix != "" && urlPrefix != "/" {
					urlPath = "/" + strings.Trim(urlPrefix, "/")
				}
				return urlPath
			} else {
				// Directory index.php
				urlPath := dirPath
				if urlPrefix != "" && urlPrefix != "/" {
					urlPath = filepath.Join("/"+strings.Trim(urlPrefix, "/"), strings.TrimPrefix(dirPath, "/"))
				}
				return "/" + strings.TrimPrefix(urlPath, "/")
			}
		}
	}

	// Regular PHP file
	urlPath := strings.TrimSuffix(virtualPath, ".php")
	if urlPrefix != "" && urlPrefix != "/" {
		urlPath = filepath.Join("/"+strings.Trim(urlPrefix, "/"), strings.TrimPrefix(urlPath, "/"))
	}

	return "/" + strings.TrimPrefix(urlPath, "/")
}

// matchParameterizedRoute tries to match a URL path to a parameterized route pattern
// Returns the extracted parameters and the matched virtual path
func (r *MiddlewareRouter) matchParameterizedRoute(urlPath string) (map[string]string, string) {
	r.routesMu.RLock()
	defer r.routesMu.RUnlock()

	// Try direct first-level match without path parameters
	urlSegments := strings.Split(strings.Trim(urlPath, "/"), "/")

	// Try each route
	for pattern, virtualPath := range r.routes {
		patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")

		// Check if the number of segments matches
		if len(patternSegments) != len(urlSegments) {
			continue
		}

		// Try to match segments
		params := make(map[string]string)
		match := true

		for i, patternSegment := range patternSegments {
			urlSegment := urlSegments[i]

			// Check if this is a parameter segment {name}
			if strings.HasPrefix(patternSegment, "{") && strings.HasSuffix(patternSegment, "}") {
				// Extract parameter name
				paramName := patternSegment[1 : len(patternSegment)-1]
				params[paramName] = urlSegment
			} else if patternSegment != urlSegment {
				// Not a parameter and doesn't match exactly
				match = false
				break
			}
		}

		if match {
			return params, virtualPath
		}
	}

	return nil, ""
}
