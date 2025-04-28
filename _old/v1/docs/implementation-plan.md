# Frango PHP Implementation Plan

This document outlines a comprehensive implementation plan to address the issues in the Frango PHP library, particularly focusing on providing a more native-like PHP experience that aligns with standard PHP developer expectations.

## Table of Contents

1. [Introduction and Key Issues](#introduction-and-key-issues)
2. [Core Script Execution Changes](#core-script-execution-changes)
3. [PHP Globals and Superglobals](#php-globals-and-superglobals)
4. [VFS and Path Management](#vfs-and-path-management)
5. [Magic Constants and Include Paths](#magic-constants-and-include-paths)
6. [Composer and Autoloader Support](#composer-and-autoloader-support)
7. [Performance Optimizations](#performance-optimizations)
8. [Testing Plan](#testing-plan)
9. [Implementation Schedule](#implementation-schedule)

## Introduction and Key Issues

The current implementation of Frango has several issues that prevent it from meeting standard PHP developer expectations:

1. **Wrapper-Based Script Execution**
   - PHP scripts are executed through wrapper scripts which causes incorrect `__FILE__` and `__DIR__` values
   - The wrapper approach breaks autoloading and relative includes
   - PHP constants point to wrapper files instead of original script files

2. **Non-Standard Superglobals**
   - Path parameters are stored in a custom `$_PATH` superglobal instead of `$_GET` as expected
   - Some standard PHP superglobals may not behave as expected

3. **Directory and Path Issues**
   - Physical vs. logical path structure discrepancies affect autoloading
   - Incorrect working directory handling affects include paths

4. **Performance Issues**
   - Redundant file operations and path resolutions
   - Lack of caching for repeated operations

## Core Script Execution Changes

The most fundamental issue with the current implementation is the use of wrapper scripts to execute PHP files. This approach leads to incorrect `__FILE__` and `__DIR__` values, breaks autoloading, and creates inconsistent directory contexts for includes.

### 1.1. Replace Wrapper-Based Execution

**Problem:** The current approach creates a wrapper PHP file that includes the target file, which causes several issues:
- Magic constants like `__FILE__` and `__DIR__` point to the wrapper, not the original script
- Relative includes (`include 'utils.php'`) resolve against the wrapper directory
- Composer autoloading breaks because it relies on correct file paths

**Solution:** Replace the wrapper approach with direct execution and use FrankenPHP's `auto_prepend_file` to include our globals initialization.

**File:** `execute.go`

```go
// Replace createPhpWrapper with a simpler, direct execution approach
func preparePhpExecution(vfs *VFS, phpFilePath, virtualPath string, globalsFile string) (string, error) {
    // Create target directory in VFS if needed
    targetDir := filepath.Dir(filepath.Join(vfs.tempDir, virtualPath))
    if err := os.MkdirAll(targetDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create directory structure: %w", err)
    }
    
    // Just return the actual PHP file path - no wrapper needed
    // We'll use auto_prepend_file to include globals instead
    return phpFilePath, nil
}
```

**Explanation:** 
- This removes the wrapper creation entirely and simply returns the actual PHP file path
- Instead of creating a wrapper script, we'll use FrankenPHP's `auto_prepend_file` directive to include our globals script
- This allows the PHP engine to execute the target script directly, preserving the correct file context

### 1.2. Update PHP Execution Environment

**Problem:** The current `ExecutePHP` function uses wrapper scripts and doesn't properly track logical paths vs. physical paths.

**Solution:** Update the `ExecutePHP` function to:
1. Use logical paths for PHP constants
2. Set the correct directory context via `chdir()`
3. Remove wrapper script creation entirely

**File:** `execute.go`

```go
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
    const (
        emptyJSON      = "{}"
        emptyJSONArray = "[]"
    )

    m.logger.Printf("========== EXECUTING PHP SCRIPT ==========")
    m.logger.Printf("ExecutePHP: Executing script '%s' with VFS %s", scriptPath, vfs.name)
    m.logger.Printf("ExecutePHP: HTTP Request %s %s", r.Method, r.URL.String())

    // Ensure the VFS has the PHP globals script installed
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

    // 3. Get logical script information (for __FILE__, __DIR__, etc.)
    scriptDir := filepath.Dir(scriptPath)
    logicalScriptPath := scriptPath // The path as it should appear to PHP

    // 4. Set up document root and script name
    documentRoot := filepath.Dir(phpFilePath)  // Physical path for FrankenPHP
    scriptName := "/" + filepath.Base(phpFilePath) // Physical filename

    m.logger.Printf("Executing PHP script: '%s' (virtual: '%s')", phpFilePath, scriptPath)
    m.logger.Printf("DocumentRoot='%s', ScriptName='%s', LogicalPath='%s'",
        documentRoot, scriptName, logicalScriptPath)

    // 5. Verify the PHP file exists
    if _, err := os.Stat(phpFilePath); err != nil {
        http.Error(w, "Server error: Failed to locate PHP file", http.StatusInternalServerError)
        return
    }

    // 6. Set up PHP environment variables with correct paths
    phpEnv := buildPhpEnvironment(m, phpFilePath, scriptPath, scriptDir, documentRoot, 
        requestData, scriptPath, vfs.name, envData)

    // 7. Execute the PHP script directly (no wrapper)
    if err := executePhpRequest(m, w, r, documentRoot, scriptName, phpEnv); err != nil {
        http.Error(w, fmt.Sprintf("PHP execution error: %v", err), http.StatusInternalServerError)
        return
    }

    m.logger.Printf("PHP execution completed successfully for '%s'", scriptPath)
    m.logger.Printf("========== PHP EXECUTION COMPLETE ==========")
}
```

**Explanation:**
- We keep track of both the logical path (what PHP developers expect) and the physical path (the actual file location in the temp directory)
- We use the logical path for PHP's environment variables that are exposed to developers
- We use the physical path for FrankenPHP execution
- No wrappers are created - we execute the PHP file directly

### 1.3. Set Correct Environment Variables

**Problem:** Current environment variables don't properly support logical paths or set the correct script context.

**Solution:** Update `buildPhpEnvironment` to:
1. Set proper path information in environment variables
2. Use `auto_prepend_file` for globals script
3. Use `auto_prepend_text` for directory context (`chdir()`)

**File:** `execute.go`

```go
func buildPhpEnvironment(m *Middleware, phpFilePath, scriptPath, scriptDir, documentRoot string,
    requestData *RequestData, virtualPath, vfsName string, envData map[string]string) map[string]string {

    // Base PHP environment with correct path information
    phpEnv := map[string]string{
        // CRITICAL: Set paths for PHP to logical paths, not temp paths
        "SCRIPT_FILENAME": phpFilePath,          // Physical path needed for execution
        "FRANGO_LOGICAL_FILENAME": scriptPath,   // Logical path for __FILE__ emulation
        "SCRIPT_NAME": scriptPath,               // Web-accessible path
        "PHP_SELF": scriptPath,                  // Match SCRIPT_NAME
        "DOCUMENT_ROOT": documentRoot,           // Must be the parent directory of the script
        "REQUEST_URI": requestData.FullURL,      // Use the same full URL
        "REQUEST_METHOD": requestData.Method,
        "QUERY_STRING": extractQueryString(requestData.FullURL),
        "HTTP_HOST": requestData.Headers.Get("Host"),
        "REMOTE_ADDR": extractHostOnly(requestData.RemoteAddr),
    }

    // Add extracted data from request
    for key, value := range envData {
        phpEnv[key] = value
    }

    // Add environment for PHP globals via auto_prepend
    globalsPath := filepath.Join(vfs.tempDir, "/_frango_php_globals.php")
    phpEnv["auto_prepend_file"] = globalsPath

    // CRITICAL: Set the directory context via chdir in auto_prepend
    // This ensures relative includes work correctly
    chdirScript := fmt.Sprintf("<?php chdir('%s'); ?>", 
        strings.ReplaceAll(scriptDir, "'", "\\'"))
    phpEnv["auto_prepend_text"] = chdirScript

    // Set PHP configuration options based on development mode
    if m.developmentMode {
        phpEnv["display_errors"] = "1"
        phpEnv["display_startup_errors"] = "1"
        phpEnv["error_reporting"] = "E_ALL"
    } else {
        phpEnv["display_errors"] = "0"
        phpEnv["display_startup_errors"] = "0"
        // Still log errors in production
        phpEnv["log_errors"] = "1"
        phpEnv["error_log"] = "/dev/stderr"
    }

    // Security and performance settings
    phpEnv["max_execution_time"] = "30"
    phpEnv["memory_limit"] = "128M"
    
    return phpEnv
}
```

**Explanation:**
- We use `FRANGO_LOGICAL_FILENAME` to store the logical path, which our PHP globals script will use to emulate correct `__FILE__` and `__DIR__` constants
- We use `auto_prepend_file` to include our globals script before executing the target script
- We use `auto_prepend_text` to inject a `chdir()` call that sets the correct working directory context
- This ensures relative includes (`include 'utils.php'`) work correctly by resolving against the logical directory

## PHP Globals and Superglobals

The current implementation uses non-standard superglobals (`$_PATH`) instead of following PHP conventions. Additionally, magic constants don't resolve correctly.

### 2.1. Enhance PHP Globals Script for Magic Constants

**Problem:** PHP's magic constants like `__FILE__` and `__DIR__` resolve to wrapper script paths, not logical paths.

**Solution:** Add a mechanism to override magic constants to use logical paths instead.

**File:** `php_globals.go`

```php
// ----- MAGIC CONSTANTS HANDLING -----

// If logical filename is set, override __FILE__ and __DIR__
if (isset($_SERVER['FRANGO_LOGICAL_FILENAME'])) {
    $GLOBALS['__FRANGO_LOGICAL_FILE__'] = $_SERVER['FRANGO_LOGICAL_FILENAME'];
    $GLOBALS['__FRANGO_LOGICAL_DIR__'] = dirname($_SERVER['FRANGO_LOGICAL_FILENAME']);
    
    // Register autoload function to handle magic constants
    spl_autoload_register(function($class) {
        // We don't actually load anything here
        // This is just to ensure our class_exists below doesn't trigger
        // other autoloaders for our internal FrangoFileMagic class
        return false;
    }, true, true);
    
    // Only define this class once
    if (!class_exists('\\FrangoFileMagic', false)) {
        // Define a class to override magic constants via stream wrapper
        class FrangoFileMagic {
            public static function getFile() {
                $trace = debug_backtrace(DEBUG_BACKTRACE_IGNORE_ARGS, 2);
                if (isset($trace[1]['file']) && $trace[1]['file'] === __FILE__) {
                    // If called from our wrapper, return the wrapper path
                    return __FILE__;
                }
                // Otherwise return logical path
                if (isset($GLOBALS['__FRANGO_LOGICAL_FILE__'])) {
                    return $GLOBALS['__FRANGO_LOGICAL_FILE__'];
                }
                $trace = debug_backtrace(DEBUG_BACKTRACE_IGNORE_ARGS, 1);
                return $trace[0]['file'];
            }
            
            public static function getDir() {
                if (isset($GLOBALS['__FRANGO_LOGICAL_DIR__'])) {
                    return $GLOBALS['__FRANGO_LOGICAL_DIR__'];
                }
                $trace = debug_backtrace(DEBUG_BACKTRACE_IGNORE_ARGS, 1);
                return dirname($trace[0]['file']);
            }
        }
    }
    
    // Redefine __FILE__ and __DIR__ as functions where possible
    if (!defined('__FILE__')) {
        function __FILE__() {
            return \FrangoFileMagic::getFile();
        }
    }
    
    if (!defined('__DIR__')) {
        function __DIR__() {
            return \FrangoFileMagic::getDir();
        }
    }
}
```

**Explanation:**
- This code attempts to override PHP's built-in `__FILE__` and `__DIR__` constants to return logical paths
- It uses a class to store the logic for resolving file paths
- It uses `debug_backtrace()` to determine the caller's context
- While this approach isn't perfect (PHP constants can't be truly overridden), it will work for most use cases
- For deeper integration, we would need to modify the PHP engine itself

### 2.2. Integrate Path Parameters with $_GET

**Problem:** Currently, path parameters go into a custom `$_PATH` superglobal, but PHP developers expect them in `$_GET`.

**Solution:** Modify the PHP globals script to merge path parameters into `$_GET`.

**File:** `php_globals.go`

```php
// Initialize $_PATH from pre-computed JSON
$_PATH = json_decode($_SERVER['_PATH'] ?? '{}', true);
$GLOBALS['_PATH'] = $_PATH;

// Initialize $_GET from pre-computed JSON with array item normalization
$_GET_RAW = json_decode($_SERVER['_GET'] ?? '{}', true);
$_GET = [];
foreach ($_GET_RAW as $key => $value) {
    // Convert array with single item to string (PHP behavior)
    if (is_array($value) && count($value) === 1) {
        $_GET[$key] = $value[0];
    } else {
        $_GET[$key] = $value;
    }
}

// CRITICAL: Merge path parameters into $_GET for compatibility with PHP standards
// Path parameters take precedence over query string parameters
foreach ($_PATH as $key => $value) {
    $_GET[$key] = $value;
}
$GLOBALS['_GET'] = $_GET;

// Update $_REQUEST to include the merged $_GET
$_REQUEST = array_merge($_GET, $_POST, $_COOKIE ?? []);
```

**Explanation:**
- We keep the custom `$_PATH` superglobal for backward compatibility
- We add all path parameters to the standard `$_GET` superglobal
- We use path parameters with higher precedence than query parameters (if both exist)
- This follows the expectations of PHP developers while maintaining backward compatibility

### 2.3. Update Path Parameter Extraction in Go

**Problem:** Path parameters are only stored in `$_PATH` and not in `$_GET`.

**Solution:** Update the `extractPathParameters` function to integrate path parameters with query parameters.

**File:** `execute.go`

```go
// Modify extractPathParameters to integrate with $_GET
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

        if pathParams != nil && len(pathParams) > 0 {
            // Store in $_PATH
            marshalToEnv(pathParams, "_PATH", "{}", envData)
            
            // CRITICAL: Also merge path parameters into query parameters
            // This ensures they appear in $_GET for PHP
            for key, value := range pathParams {
                requestData.QueryParams[key] = []string{value}
            }
            
            // Re-marshal the updated query parameters
            marshalToEnv(requestData.QueryParams, "_GET", "{}", envData)
            
            m.logger.Printf("Extracted path parameters merged into $_GET: %v", pathParams)
        } else {
            envData["_PATH"] = "{}"
        }
    } else {
        envData["_PATH"] = "{}"
        m.logger.Printf("No pattern available, using URL path without parameter extraction: %s", requestData.Path)
    }
}
```

**Explanation:**
- We extract path parameters from the URL pattern as before
- In addition to storing them in `$_PATH`, we also add them to `requestData.QueryParams`
- This ensures they'll be passed to PHP in both `$_PATH` and `$_GET`
- Each parameter is represented as a single-item array because that's how Go represents URL parameters
- When marshaled to PHP, the single-item arrays will be converted to strings by our normalization code

## VFS and Path Management

The current VFS implementation doesn't maintain a mapping between logical paths (web-accessible paths) and physical paths (actual filesystem paths), which leads to issues with magic constants and includes.

### 3.1. Enhance VFS Structure for Path Tracking

**Problem:** The VFS doesn't track logical paths, leading to issues with `__FILE__`, `__DIR__`, and includes.

**Solution:** Enhance the VFS structure to track logical paths corresponding to physical paths.

**File:** `vfs.go`

```go
// Add fields to track logical paths in VFS
type VFS struct {
    name            string                // Unique identifier for this VFS
    parent          *VFS                  // Parent VFS (if this is a branch)
    sourceMappings  map[string]string     // Virtual path -> source path (for files on disk)
    embedMappings   map[string]string     // Virtual path -> embed temp path (for embedded files)
    virtualFiles    map[string][]byte     // Virtual path -> content (for in-memory files)
    fileOrigins     map[string]FileOrigin // Virtual path -> origin type
    fileHashes      map[string]FileHash   // Path -> hash info (for change detection)
    tempDir         string                // Base temp directory for this VFS
    mutex           sync.RWMutex          // For thread safety
    
    // New fields for improved path management
    logicalPaths    map[string]string     // Physical path -> logical path mapping
    pathCache       map[string]string     // Cache for path resolution
    dirCache        map[string]*DirInfo   // Cache for directory information
    
    // Other existing fields...
}

// Directory information for include path resolution
type DirInfo struct {
    IncludePaths []string       // Priority-ordered include paths
    Files        map[string]string // basename -> full path
}
```

**Explanation:**
- `logicalPaths` maintains a mapping from physical paths (temp directory) to logical paths (web URLs)
- `pathCache` caches path resolution results for performance
- `dirCache` caches directory information to speed up include path resolution
- These data structures will help maintain the correct context for PHP scripts

### 3.2. Implement Logical Path Tracking

**Problem:** Currently, there's no way to determine the logical path corresponding to a physical path.

**Solution:** Add methods to track and retrieve logical paths.

**File:** `vfs.go`

```go
// Initialize the new fields in NewVFS
func NewVFS(tempDir string, logger *log.Logger, developMode bool) (*VFS, error) {
    // Existing initialization...
    
    v := &VFS{
        // Existing fields...
        logicalPaths:    make(map[string]string),
        pathCache:       make(map[string]string),
        dirCache:        make(map[string]*DirInfo),
    }
    
    // Rest of initialization...
    return v, nil
}

// Add a function to track logical paths
func (v *VFS) TrackLogicalPath(physicalPath, logicalPath string) {
    v.mutex.Lock()
    defer v.mutex.Unlock()
    
    v.logicalPaths[physicalPath] = logicalPath
}

// Get the logical path for a physical path
func (v *VFS) GetLogicalPath(physicalPath string) string {
    v.mutex.RLock()
    defer v.mutex.RUnlock()
    
    if logicalPath, ok := v.logicalPaths[physicalPath]; ok {
        return logicalPath
    }
    
    // Try parent if not found and we have one
    if v.parent != nil {
        return v.parent.GetLogicalPath(physicalPath)
    }
    
    // Default to the physical path if no logical path is found
    return physicalPath
}
```

**Explanation:**
- `TrackLogicalPath` stores the mapping from physical path to logical path
- `GetLogicalPath` retrieves the logical path for a physical path
- If no logical path is found, it checks the parent VFS (for branched VFS instances)
- This allows us to resolve logical paths at runtime for PHP constants

### 3.3. Enhanced Path Resolution with Caching

**Problem:** Paths are resolved repeatedly, leading to unnecessary filesystem operations.

**Solution:** Implement caching for path resolution.

**File:** `vfs.go`

```go
// Add path resolution cache
func (v *VFS) ResolvePath(virtualPath string) (string, error) {
    // Normalize path
    virtualPath = normalizePath(virtualPath)
    
    // Check cache first in non-development mode
    if !v.developMode {
        v.mutex.RLock()
        if cachedPath, ok := v.pathCache[virtualPath]; ok {
            v.mutex.RUnlock()
            return cachedPath, nil
        }
        v.mutex.RUnlock()
    }
    
    v.mutex.RLock()
    
    // If in development mode, check for changes first
    if v.developMode {
        // Release lock before checking for changes
        v.mutex.RUnlock()
        v.checkFileChanges(virtualPath)
        v.mutex.RLock()
    }
    
    // Check this VFS first
    origin, exists := v.fileOrigins[virtualPath]
    if exists {
        // Check for virtual "tombstone" files
        if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
            v.mutex.RUnlock()
            return "", fmt.Errorf("file not found in VFS: %s (shadowed)", virtualPath)
        }
        
        // Resolve based on origin type
        var resolvedPath string
        switch origin {
        case OriginSource:
            resolvedPath = v.sourceMappings[virtualPath]
        case OriginEmbed, OriginVirtual:
            resolvedPath = v.embedMappings[virtualPath]
        }
        
        // Cache the result
        if resolvedPath != "" && !v.developMode {
            v.mutex.Lock()
            v.pathCache[virtualPath] = resolvedPath
            v.mutex.Unlock()
        }
        
        return resolvedPath, nil
    }
    
    // If not found in this VFS, check parent (if exists)
    if v.parent != nil {
        // Release our lock before calling parent
        v.mutex.RUnlock()
        resolvedPath, err := v.parent.ResolvePath(virtualPath)
        
        if err == nil {
            // Successfully resolved in parent - track this path as inherited
            v.mutex.Lock()
            v.inheritedPaths[virtualPath] = true
            // Also track logical path
            v.logicalPaths[resolvedPath] = virtualPath
            v.mutex.Unlock()
        }
        
        return resolvedPath, err
    }
    
    v.mutex.RUnlock()
    return "", fmt.Errorf("file not found in VFS: %s", virtualPath)
}
```

**Explanation:**
- This adds a cache for path resolution to avoid redundant filesystem operations
- In development mode, we skip the cache to ensure changes are detected
- This significantly improves performance for repeated path resolutions
- Path resolution is a common operation during PHP execution, especially with autoloading
- The cache is thread-safe with proper mutex locking

## Composer and Autoloader Support

Composer and other autoloading mechanisms rely on correct file paths and directory structures. Our changes need to ensure these work correctly.

### 4.1. Add Explicit Composer Support

**Problem:** Composer autoloading breaks because it relies on specific directory structures and file paths.

**Solution:** Add specific support for Composer vendor directories.

**File:** `vfs.go`

```go
// Add function to ensure composer autoloading works
func (v *VFS) EnsureComposerSupport(sourceDir string) error {
    // Check for vendor directory
    vendorDir := filepath.Join(sourceDir, "vendor")
    if _, err := os.Stat(vendorDir); err == nil {
        // Add the entire vendor directory to the VFS
        if err := v.AddSourceDirectory(vendorDir, "/vendor"); err != nil {
            return fmt.Errorf("failed to add vendor directory: %w", err)
        }
        
        // Check for composer autoload file
        autoloadFile := filepath.Join(vendorDir, "autoload.php")
        if _, err := os.Stat(autoloadFile); err == nil {
            v.logger.Printf("Found Composer autoload file: %s", autoloadFile)
        }
    }
    
    return nil
}
```

**Explanation:**
- This function checks for a Composer vendor directory and adds it to the VFS
- It preserves the directory structure expected by Composer
- This ensures that `require 'vendor/autoload.php'` works correctly
- By adding it to the VFS, we ensure the files are available to PHP scripts
- Composer's autoloader relies on the correct directory structure and file paths

### 4.2. Update Middleware Initialization

**Problem:** Composer support isn't automatically enabled.

**Solution:** Update the middleware initialization to check for and enable Composer support.

**File:** `frango.go`

```go
// Update New function to check for Composer support
func New(opts ...Option) (*Middleware, error) {
    // Existing initialization...
    
    // If sourceDir is specified, check for Composer support
    if m.sourceDir != "" && m.rootVFS != nil {
        if err := m.rootVFS.EnsureComposerSupport(m.sourceDir); err != nil {
            m.logger.Printf("Warning: Failed to enable Composer support: %v", err)
        } else {
            m.logger.Printf("Composer support enabled")
        }
    }
    
    return m, nil
}
```

**Explanation:**
- This checks for a Composer setup during middleware initialization
- If found, it automatically adds the vendor directory to the VFS
- This makes Composer autoloading work out of the box
- This is transparent to PHP developers - they just use Composer as they normally would
- The middleware detects Composer automatically without requiring additional configuration

### 4.3. Add Option for Composer Support

**Problem:** No explicit way to enable/disable Composer support.

**Solution:** Add a middleware option for controlling Composer support.

**File:** `options.go`

```go
// Add option for Composer support
func WithComposerSupport(enabled bool) Option {
    return func(m *Middleware) {
        m.enableComposer = enabled
    }
}
```

**File:** `frango.go`

```go
// Update Middleware struct
type Middleware struct {
    // Existing fields...
    enableComposer    bool        // Whether to enable Composer support
}
```

**Explanation:**
- This adds an explicit option for enabling/disabling Composer support
- By default, it's enabled if a vendor directory is found
- This gives developers control over whether Composer is enabled
- Some applications might not use Composer or might have custom autoloading mechanisms
- The option allows flexibility for different project structures

### 4.4. Add PSR-4 Autoloader Support

**Problem:** PHP applications often use PSR-4 autoloading, which requires specific directory structures.

**Solution:** Add explicit support for PSR-4 autoloading patterns.

**File:** `autoload.go`

```go
// Add support for PSR-4 autoloading
type PSR4Namespace struct {
    Namespace  string
    BasePath   string
}

// Add PSR4 namespaces to the middleware
func (m *Middleware) AddPSR4Namespace(namespace string, basePath string) {
    if m.psr4Namespaces == nil {
        m.psr4Namespaces = make([]PSR4Namespace, 0)
    }
    
    // Add trailing backslash to namespace if not present
    if !strings.HasSuffix(namespace, "\\") {
        namespace += "\\"
    }
    
    // Normalize base path
    basePath = normalizePath(basePath)
    
    m.psr4Namespaces = append(m.psr4Namespaces, PSR4Namespace{
        Namespace: namespace,
        BasePath:  basePath,
    })
    
    m.logger.Printf("Added PSR-4 namespace mapping: %s -> %s", namespace, basePath)
}

// Register autoloaders with PHP
func (m *Middleware) registerAutoloaders(w http.ResponseWriter, r *http.Request) error {
    // If there are no PSR-4 namespaces, don't do anything
    if len(m.psr4Namespaces) == 0 {
        return nil
    }
    
    // Create autoloader PHP code
    var autoloaderCode strings.Builder
    
    autoloaderCode.WriteString("<?php\n")
    autoloaderCode.WriteString("// Generated PSR-4 autoloader\n")
    autoloaderCode.WriteString("spl_autoload_register(function($class) {\n")
    
    // Add each namespace mapping
    for _, ns := range m.psr4Namespaces {
        autoloaderCode.WriteString(fmt.Sprintf("    // %s -> %s\n", ns.Namespace, ns.BasePath))
        autoloaderCode.WriteString(fmt.Sprintf("    if (strpos($class, '%s') === 0) {\n", ns.Namespace))
        autoloaderCode.WriteString(fmt.Sprintf("        $path = '%s' . str_replace('\\\\', '/', substr($class, %d)) . '.php';\n", 
            ns.BasePath, len(ns.Namespace)))
        autoloaderCode.WriteString("        if (file_exists($path)) {\n")
        autoloaderCode.WriteString("            require_once $path;\n")
        autoloaderCode.WriteString("            return true;\n")
        autoloaderCode.WriteString("        }\n")
        autoloaderCode.WriteString("    }\n")
    }
    
    autoloaderCode.WriteString("    return false;\n")
    autoloaderCode.WriteString("});\n")
    
    // Add the autoloader code to the VFS
    autoloaderPath := "/_frango_autoloader.php"
    if err := m.rootVFS.CreateVirtualFile(autoloaderPath, []byte(autoloaderCode.String())); err != nil {
        return fmt.Errorf("failed to create autoloader: %w", err)
    }
    
    // Include the autoloader in PHP's auto_prepend_file
    phpPath, err := m.rootVFS.ResolvePath(autoloaderPath)
    if err != nil {
        return fmt.Errorf("failed to resolve autoloader path: %w", err)
    }
    
    // Set the autoloader to be included before the script
    // This will be merged with the existing auto_prepend_file
    m.additionalPrependFiles = append(m.additionalPrependFiles, phpPath)
    
    return nil
}
```

**Explanation:**
- We add support for PSR-4 namespace mapping (the standard used by Composer)
- Developers can explicitly register namespace-to-directory mappings
- We generate a PSR-4 autoloader that gets included with every PHP script
- This ensures classes are automatically loaded when referenced
- This works alongside Composer's autoloader or as a standalone solution

## Performance Optimizations

The current implementation has performance issues due to redundant file operations and path resolutions. We can optimize these for better performance.

### 5.1. Path Resolution Caching

**Problem:** Paths are resolved repeatedly, leading to unnecessary filesystem operations.

**Solution:** Implement caching for path resolution.

**File:** `vfs.go`

```go
// Add path resolution cache
func (v *VFS) ResolvePath(virtualPath string) (string, error) {
    // Normalize path
    virtualPath = normalizePath(virtualPath)
    
    // Check cache first in non-development mode
    if !v.developMode {
        v.mutex.RLock()
        if cachedPath, ok := v.pathCache[virtualPath]; ok {
            v.mutex.RUnlock()
            return cachedPath, nil
        }
        v.mutex.RUnlock()
    }
    
    v.mutex.RLock()
    
    // If in development mode, check for changes first
    if v.developMode {
        // Release lock before checking for changes
        v.mutex.RUnlock()
        v.checkFileChanges(virtualPath)
        v.mutex.RLock()
    }
    
    // Check this VFS first
    origin, exists := v.fileOrigins[virtualPath]
    if exists {
        // Check for virtual "tombstone" files
        if origin == OriginVirtual && v.virtualFiles[virtualPath] == nil {
            v.mutex.RUnlock()
            return "", fmt.Errorf("file not found in VFS: %s (shadowed)", virtualPath)
        }
        
        // Resolve based on origin type
        var resolvedPath string
        switch origin {
        case OriginSource:
            resolvedPath = v.sourceMappings[virtualPath]
        case OriginEmbed, OriginVirtual:
            resolvedPath = v.embedMappings[virtualPath]
        }
        
        // Cache the result
        if resolvedPath != "" && !v.developMode {
            v.mutex.Lock()
            v.pathCache[virtualPath] = resolvedPath
            v.mutex.Unlock()
        }
        
        return resolvedPath, nil
    }
    
    // If not found in this VFS, check parent (if exists)
    if v.parent != nil {
        // Release our lock before calling parent
        v.mutex.RUnlock()
        resolvedPath, err := v.parent.ResolvePath(virtualPath)
        
        if err == nil {
            // Successfully resolved in parent - track this path as inherited
            v.mutex.Lock()
            v.inheritedPaths[virtualPath] = true
            // Also track logical path
            v.logicalPaths[resolvedPath] = virtualPath
            v.mutex.Unlock()
        }
        
        return resolvedPath, err
    }
    
    v.mutex.RUnlock()
    return "", fmt.Errorf("file not found in VFS: %s", virtualPath)
}
```

**Explanation:**
- This adds a cache for path resolution to avoid redundant filesystem operations
- In development mode, we skip the cache to ensure changes are detected
- This significantly improves performance for repeated path resolutions
- Path resolution is a common operation during PHP execution, especially with autoloading
- The cache is thread-safe with proper mutex locking

### 5.2. Request Data Pooling

**Problem:** Request data structures are created and filled repeatedly, leading to GC pressure.

**Solution:** Use a sync.Pool to reuse request data structures.

**File:** `execute.go`

```go
// Create a pool for RequestData objects
var requestDataPool = sync.Pool{
    New: func() interface{} {
        return &RequestData{
            QueryParams:  make(map[string][]string),
            FormData:     make(map[string][]string),
            JSONBody:     make(map[string]interface{}),
            FileUploads:  make(map[string][]*multipart.FileHeader),
        }
    },
}

// Get RequestData from pool
func getRequestData() *RequestData {
    return requestDataPool.Get().(*RequestData)
}

// Return RequestData to pool
func releaseRequestData(rd *RequestData) {
    // Clear maps for reuse
    for k := range rd.QueryParams {
        delete(rd.QueryParams, k)
    }
    for k := range rd.FormData {
        delete(rd.FormData, k)
    }
    for k := range rd.JSONBody {
        delete(rd.JSONBody, k)
    }
    for k := range rd.FileUploads {
        delete(rd.FileUploads, k)
    }
    
    // Reset fields
    rd.Method = ""
    rd.FullURL = ""
    rd.Path = ""
    rd.RemoteAddr = ""
    rd.Headers = nil
    rd.PathSegments = rd.PathSegments[:0]
    
    // Return to pool
    requestDataPool.Put(rd)
}

// Update extractRequestData to use the pool
func extractRequestData(r *http.Request) *RequestData {
    rd := getRequestData()
    
    // Fill with request data
    // ... [existing code] ...
    
    return rd
}

// Update ExecutePHP to release the request data
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request) {
    // ... [existing code] ...
    
    // Get request data from pool
    requestData := extractRequestData(r)
    defer releaseRequestData(requestData)
    
    // ... [rest of the function] ...
}
```

**Explanation:**
- This uses a `sync.Pool` to reuse `RequestData` structures across requests
- This reduces garbage collection pressure by reusing memory
- The pool automatically creates new instances when needed
- We clear all maps and reset all fields when returning to the pool
- This is particularly beneficial in high-traffic scenarios
- Object pooling is a common optimization in Go web servers

### 5.3. Pre-compiled Path Patterns

**Problem:** Path patterns are compiled repeatedly, leading to unnecessary computation.

**Solution:** Cache compiled regular expressions for path patterns.

**File:** `execute.go`

```go
// Cache for compiled path patterns
var (
    pathPatternCache     = make(map[string]*regexp.Regexp)
    pathPatternCacheLock sync.RWMutex
)

// Get or compile pattern
func getCompiledPattern(pattern string) *regexp.Regexp {
    pathPatternCacheLock.RLock()
    re, exists := pathPatternCache[pattern]
    pathPatternCacheLock.RUnlock()
    
    if !exists {
        // Convert pattern to regex
        regexPattern := convertPatternToRegex(pattern)
        
        // Compile and cache
        re = regexp.MustCompile(regexPattern)
        
        pathPatternCacheLock.Lock()
        pathPatternCache[pattern] = re
        pathPatternCacheLock.Unlock()
    }
    
    return re
}

// Update extractPathParams to use cached patterns
func extractPathParams(pattern, path string) map[string]string {
    // Fast path for non-parameterized patterns
    if !strings.Contains(pattern, "{") {
        if pattern == path {
            return map[string]string{} // Match but no parameters
        }
        return nil // No match
    }
    
    // Get compiled pattern
    re := getCompiledPattern(pattern)
    
    // ... [rest of function using the compiled regex] ...
}
```

**Explanation:**
- This adds caching for compiled regular expressions
- Path patterns are only compiled once and then reused
- This improves performance for repeatedly used path patterns (common in web applications)
- The cache is thread-safe using a read-write mutex
- Regex compilation is expensive, especially for complex patterns

### 5.4. Directory Listing Caching

**Problem:** Directory listings are performed repeatedly, which is slow.

**Solution:** Cache directory listings for faster include path resolution.

**File:** `vfs.go`

```go
// Cache directory structure information
type DirInfo struct {
    IncludePaths []string     // Prioritized include paths
    Files        map[string]string // filename -> full path mapping
    LastUpdated  time.Time    // Last time this cache was updated
}

// Get directory info (cached)
func (v *VFS) GetDirInfo(dirPath string) (*DirInfo, error) {
    normalizedPath := normalizePath(dirPath)
    
    // Check cache first (in non-development mode)
    if !v.developMode {
        v.mutex.RLock()
        if cache, ok := v.dirCache[normalizedPath]; ok {
            v.mutex.RUnlock()
            return cache, nil
        }
        v.mutex.RUnlock()
    }
    
    // Build directory info
    info := &DirInfo{
        IncludePaths: []string{normalizedPath},
        Files:        make(map[string]string),
        LastUpdated:  time.Now(),
    }
    
    // Add parent directory if not root
    if normalizedPath != "/" {
        info.IncludePaths = append(info.IncludePaths, filepath.Dir(normalizedPath))
    }
    
    // List files in directory
    v.mutex.RLock()
    for path, origin := range v.fileOrigins {
        if filepath.Dir(path) == normalizedPath {
            // File is in this directory
            filename := filepath.Base(path)
            var resolvedPath string
            
            switch origin {
            case OriginSource:
                resolvedPath = v.sourceMappings[path]
            case OriginEmbed, OriginVirtual:
                resolvedPath = v.embedMappings[path]
            }
            
            if resolvedPath != "" {
                info.Files[filename] = path // Store virtual path
            }
        }
    }
    v.mutex.RUnlock()
    
    // Cache the result
    if !v.developMode {
        v.mutex.Lock()
        v.dirCache[normalizedPath] = info
        v.mutex.Unlock()
    }
    
    return info, nil
}
```

**Explanation:**
- This caches directory listings to speed up include path resolution
- This is particularly useful for include statements and autoloaders
- The cache is skipped in development mode to ensure changes are detected
- The cache includes file mappings and include paths
- This significantly improves performance for PHP applications that use many includes

### 5.5. Pre-allocation of Environment Data

**Problem:** Environment data maps are created and filled repeatedly, leading to inefficient memory usage.

**Solution:** Pre-allocate environment data with appropriate capacity.

**File:** `execute.go`

```go
// Pre-allocate environment data
func prepareEnvironmentData(m *Middleware, requestData *RequestData, scriptPath string, 
    r *http.Request, renderFn RenderData, w http.ResponseWriter) map[string]string {
    
    // Pre-allocate with typical capacity
    envData := make(map[string]string, 32)
    
    // Pre-fill with common values that don't depend on request
    envData["FRANGO_VERSION"] = "1.0.0"
    envData["FRANGO_DEVELOPMENT_MODE"] = strconv.FormatBool(m.developmentMode)
    
    // Extract path parameters - fills envData["_PATH"]
    extractPathParameters(m, requestData, scriptPath, r, envData)
    
    // Add path segments
    marshalToEnv(requestData.PathSegments, "_PATH_SEGMENTS", "[]", envData)
    envData["_PATH_SEGMENT_COUNT"] = strconv.Itoa(len(requestData.PathSegments))
    
    // Add query parameters
    marshalToEnv(requestData.QueryParams, "_GET", "{}", envData)
    
    // Add form data if present
    if len(requestData.FormData) > 0 {
        marshalToEnv(requestData.FormData, "_POST", "{}", envData)
        marshalToEnv(requestData.FormData, "_FORM", "{}", envData)
    } else {
        envData["_POST"] = "{}"
        envData["_FORM"] = "{}"
    }
    
    // Add JSON body if present
    if len(requestData.JSONBody) > 0 {
        marshalToEnv(requestData.JSONBody, "_JSON", "{}", envData)
    } else {
        envData["_JSON"] = "{}"
    }
    
    // Add template variables
    addTemplateVariables(m, renderFn, w, r, envData)
    
    return envData
}
```

**Explanation:**
- We pre-allocate the environment data map with a capacity that matches typical usage
- We pre-fill common values that don't depend on the request
- We use default empty JSON objects for optional data
- This reduces map resizing operations during request handling
- Proper capacity planning can significantly improve memory usage patterns

## Testing Plan

To ensure the changes work correctly, we need a comprehensive testing plan.

### 6.1. Magic Constants Tests

**Problem:** Need to verify `__FILE__` and `__DIR__` work correctly.

**Solution:** Create dedicated tests for magic constants.

**File:** `magic_constants_test.go`

```go
func TestMagicConstants(t *testing.T) {
    // Create test PHP file that outputs magic constants
    testPHP := `<?php
    header('Content-Type: application/json');
    echo json_encode([
        'FILE' => __FILE__,
        'DIR' => __DIR__,
        'LINE' => __LINE__,
        'SCRIPT_FILENAME' => $_SERVER['SCRIPT_FILENAME'],
        'SCRIPT_NAME' => $_SERVER['SCRIPT_NAME'],
        'PHP_SELF' => $_SERVER['PHP_SELF']
    ]);
    ?>`
    
    // Setup test environment
    env := SetupTest(t, map[string]string{
        "magic_constants.php": testPHP,
    })
    defer CleanupTest(env)
    
    // Execute the PHP script
    status, _, body := ExecutePHP(t, env, "/magic_constants.php", 
        httptest.NewRequest("GET", "/magic_constants.php", nil), nil)
    
    // Parse the JSON response
    result := ParseJSON(t, body)
    
    // Verify magic constants are correct
    file := result["FILE"].(string)
    if !strings.Contains(file, "/magic_constants.php") {
        t.Errorf("__FILE__ should contain logical path, got: %s", file)
    }
    
    dir := result["DIR"].(string)
    expectedDir := filepath.Dir("/magic_constants.php")
    if dir != expectedDir {
        t.Errorf("__DIR__ should be %s, got: %s", expectedDir, dir)
    }
}
```

**Explanation:**
- This tests that magic constants like `__FILE__` and `__DIR__` contain the logical path
- It verifies that `$_SERVER` variables are set correctly
- This ensures our magic constant handling mechanism works correctly

### 6.2. Path Parameter Tests

**Problem:** Need to verify path parameters are correctly merged into `$_GET`.

**Solution:** Create tests for path parameter handling.

**File:** `path_parameters_test.go`

```go
func TestPathParametersInGET(t *testing.T) {
    // Create test PHP file that outputs path parameters and GET variables
    testPHP := `<?php
    header('Content-Type: application/json');
    echo json_encode([
        'PATH' => $_PATH, 
        'GET' => $_GET,
        'REQUEST' => $_REQUEST
    ]);
    ?>`
    
    // Setup test environment
    env := SetupTest(t, map[string]string{
        "users/{id}.php": testPHP,
    })
    defer CleanupTest(env)
    
    // Create request with both path parameter and query parameter
    req := httptest.NewRequest("GET", "/users/123?name=test", nil)
    req.Pattern = "/users/{id}"
    
    // Execute request
    status, _, body := ExecutePHP(t, env, "/users/{id}.php", req, nil)
    
    // Parse JSON response
    result := ParseJSON(t, body)
    
    // Verify path parameter is in both $_PATH and $_GET
    path, ok := result["PATH"].(map[string]interface{})
    if !ok || path["id"] != "123" {
        t.Errorf("Expected $_PATH['id'] = '123', got: %v", path)
    }
    
    get, ok := result["GET"].(map[string]interface{})
    if !ok || get["id"] != "123" || get["name"] != "test" {
        t.Errorf("Expected $_GET to contain both path and query params, got: %v", get)
    }
}
```

**Explanation:**
- This tests that path parameters are correctly merged into `$_GET`
- It verifies that both path parameters and query parameters work together
- It ensures that PHP developers can access path parameters via standard superglobals

### 6.3. Include and Autoload Tests

**Problem:** Need to verify includes and autoloading work correctly.

**Solution:** Create tests for include paths and autoloading.

**File:** `include_test.go`

```go
func TestRelativeIncludes(t *testing.T) {
    // Create main PHP file
    mainPHP := `<?php
    // Include a file using a relative path
    include_once('lib/functions.php');
    
    // Use function from included file
    echo getGreeting();
    ?>`
    
    // Create functions file
    functionsPHP := `<?php
    function getGreeting() {
        return "Hello from " . __FILE__;
    }
    ?>`
    
    // Setup test environment
    env := SetupTest(t, map[string]string{
        "main.php": mainPHP,
        "lib/functions.php": functionsPHP,
    })
    defer CleanupTest(env)
    
    // Execute the PHP script
    status, _, body := ExecutePHP(t, env, "/main.php", 
        httptest.NewRequest("GET", "/main.php", nil), nil)
    
    // Verify output contains the correct __FILE__ path
    if !strings.Contains(body, "/lib/functions.php") {
        t.Errorf("Expected output to contain '/lib/functions.php', got: %s", body)
    }
}
```

**Explanation:**
- This tests that relative includes work correctly
- It verifies that `__FILE__` in included files contains the correct path
- It ensures that the directory context is set correctly for includes

### 6.4. Autoloader Test

**Problem:** Need to verify PSR-4 autoloading works correctly.

**Solution:** Create dedicated tests for autoloading.

**File:** `autoload_test.go`

```go
func TestAutoloaderBehavior(t *testing.T) {
    // Setup test with PSR-4 style autoloading
    mainPHP := `<?php
    // Try to use an autoloaded class
    $user = new App\\Models\\User(123, "John");
    echo json_encode($user->toArray());
    ?>`
    
    // Define the User class
    userClassPHP := `<?php
    namespace App\\Models;
    
    class User {
        private $id;
        private $name;
        
        public function __construct($id, $name) {
            $this->id = $id;
            $this->name = $name;
        }
        
        public function toArray() {
            return [
                'id' => $this->id,
                'name' => $this->name,
                'class_file' => __FILE__
            ];
        }
    }
    ?>`
    
    // Setup test environment
    env := SetupTest(t, map[string]string{
        "index.php": mainPHP,
        "src/Models/User.php": userClassPHP,
    })
    defer CleanupTest(env)
    
    // Configure PSR-4 autoloading
    middleware := env.Middleware.(*Middleware)
    middleware.AddPSR4Namespace("App\\", "/src")
    
    // Execute the PHP script
    status, _, body := ExecutePHP(t, env, "/index.php", 
        httptest.NewRequest("GET", "/index.php", nil), nil)
    
    // Parse the JSON response
    result := ParseJSON(t, body)
    
    // Verify class was properly autoloaded
    id, ok := result["id"].(float64)
    if !ok || id != 123 {
        t.Errorf("User ID should be 123, got: %v", id)
    }
    
    name, ok := result["name"].(string)
    if !ok || name != "John" {
        t.Errorf("User name should be 'John', got: %v", name)
    }
    
    // Check that __FILE__ in autoloaded class is correct
    classFile, ok := result["class_file"].(string)
    if !ok || !strings.HasSuffix(classFile, "/src/Models/User.php") {
        t.Errorf("Class file should end with /src/Models/User.php, got: %v", classFile)
    }
}
```

**Explanation:**
- This tests PSR-4 autoloading which is crucial for modern PHP applications
- It verifies that namespaced classes can be autoloaded correctly
- It checks that magic constants like `__FILE__` work in autoloaded classes
- The test registers a namespace mapping (like Composer would do)
- We verify both the functionality of the class and the correctness of file paths

### 6.5. End-to-End Application Test

**Problem:** Need to verify all components work together in a realistic scenario.

**Solution:** Create an end-to-end test of a small PHP application.

**File:** `e2e_test.go`

```go
func TestEndToEndApplication(t *testing.T) {
    // Set up a simple PHP application with router, controller, template
    routerPHP := `<?php
    // Simple router
    $path = $_SERVER['REQUEST_URI'];
    $segments = explode('/', trim($path, '/'));
    
    if (empty($segments[0])) {
        include 'controllers/home.php';
    } else {
        $controller = $segments[0];
        $action = $segments[1] ?? 'index';
        $id = $segments[2] ?? null;
        
        if (file_exists("controllers/{$controller}.php")) {
            $_PATH['controller'] = $controller;
            $_PATH['action'] = $action;
            $_PATH['id'] = $id;
            include "controllers/{$controller}.php";
        } else {
            header("HTTP/1.1 404 Not Found");
            echo "404 - Not Found";
        }
    }
    ?>`
    
    // Create controller
    controllerPHP := `<?php
    // User controller
    $userID = $_PATH['id'] ?? 0;
    
    // Use same ID from $_GET (should match $_PATH)
    $getID = $_GET['id'] ?? 'not-set';
    
    echo "User ID from PATH: {$userID}, from GET: {$getID}";
    ?>`
    
    // Setup test environment
    env := SetupTest(t, map[string]string{
        "index.php": routerPHP,
        "controllers/users.php": controllerPHP,
    })
    defer CleanupTest(env)
    
    // Execute request
    req := httptest.NewRequest("GET", "/users/view/123", nil)
    status, _, body := ExecutePHP(t, env, "/index.php", req, nil)
    
    // Verify output contains correct data
    expectedOutput := "User ID from PATH: 123, from GET: 123"
    if !strings.Contains(body, expectedOutput) {
        t.Errorf("Expected output '%s', got: %s", expectedOutput, body)
    }
}
```

**Explanation:**
- This tests a complete PHP application with router and controller
- It verifies that path parameters work correctly within the application
- It ensures that includes and logical paths work together properly
- This simulates a real-world PHP application structure
- It validates that our PHP execution matches expected behavior in a realistic scenario

## FrankenPHP-Specific Optimizations

Based on the FrankenPHP documentation, we should incorporate these additional optimizations to better align with FrankenPHP's capabilities and ensure maximum compatibility.

### 7.1. Better Leverage FrankenPHP Features

**Problem:** Our current approach doesn't fully utilize FrankenPHP's built-in features.

**Solution:** Better use FrankenPHP directives and capabilities.

**File:** `execute.go`

```go
func buildPhpEnvironment(m *Middleware, phpFilePath, scriptPath, scriptDir, documentRoot string,
    requestData *RequestData, virtualPath, vfsName string, envData map[string]string) map[string]string {

    // Base PHP environment with correct path information
    phpEnv := map[string]string{
        // CRITICAL: Set paths for PHP to logical paths, not temp paths
        "SCRIPT_FILENAME": phpFilePath,          // Physical path needed for execution
        "FRANGO_LOGICAL_FILENAME": scriptPath,   // Logical path for __FILE__ emulation
        "SCRIPT_NAME": scriptPath,               // Web-accessible path
        "PHP_SELF": scriptPath,                  // Match SCRIPT_NAME
        "DOCUMENT_ROOT": documentRoot,           // Must be the parent directory of the script
        "REQUEST_URI": requestData.FullURL,      // Use the same full URL
        "REQUEST_METHOD": requestData.Method,
        "QUERY_STRING": extractQueryString(requestData.FullURL),
        "HTTP_HOST": requestData.Headers.Get("Host"),
        "REMOTE_ADDR": extractHostOnly(requestData.RemoteAddr),
    }

    // Add extracted data from request
    for key, value := range envData {
        phpEnv[key] = value
    }

    // Add environment for PHP globals via auto_prepend
    globalsPath := filepath.Join(vfs.tempDir, "/_frango_php_globals.php")
    phpEnv["auto_prepend_file"] = globalsPath

    // CRITICAL: Set the directory context via chdir in auto_prepend
    // This ensures relative includes work correctly
    chdirScript := fmt.Sprintf("<?php chdir('%s'); ?>", 
        strings.ReplaceAll(scriptDir, "'", "\\'"))
    phpEnv["auto_prepend_text"] = chdirScript

    // Set PHP configuration options based on development mode
    if m.developmentMode {
        phpEnv["display_errors"] = "1"
        phpEnv["display_startup_errors"] = "1"
        phpEnv["error_reporting"] = "E_ALL"
    } else {
        phpEnv["display_errors"] = "0"
        phpEnv["display_startup_errors"] = "0"
        // Still log errors in production
        phpEnv["log_errors"] = "1"
        phpEnv["error_log"] = "/dev/stderr"
    }

    // Security and performance settings
    phpEnv["max_execution_time"] = "30"
    phpEnv["memory_limit"] = "128M"
    
    return phpEnv
}
```

**Explanation:**
- FrankenPHP's documentation emphasizes its use of standard PHP superglobals and environment variables
- We optimize our integration by using FrankenPHP's directives like `auto_prepend_file` and `auto_prepend_text`
- We ensure proper header mapping to match FrankenPHP's expectations
- This approach aligns with FrankenPHP's design, which is to maintain PHP's standard behaviors

### 7.2. Worker Mode Support

**Problem:** Our implementation doesn't account for FrankenPHP's worker mode, which can provide significant performance benefits.

**Solution:** Add support for worker mode configuration.

**File:** `worker.go`

```go
// Add worker mode support
type WorkerConfig struct {
    Enabled       bool
    MaxRequests   int
    BootstrapFile string
}

// Update Middleware to support worker configuration
func (m *Middleware) SetWorkerMode(config WorkerConfig) {
    m.workerConfig = config
    
    if config.Enabled {
        m.logger.Printf("FrankenPHP worker mode enabled, max requests: %d", config.MaxRequests)
    }
}

// Generate worker script if worker mode is enabled
func (m *Middleware) generateWorkerScript(vfs *VFS) error {
    if !m.workerConfig.Enabled {
        return nil
    }
    
    // Create worker script template
    workerTemplate := `<?php
// Worker mode bootstrap for Frango
ignore_user_abort(true);

// Store original server environment
$workerServer = $_SERVER;

// Load bootstrap file if specified
if (file_exists('{{.BootstrapFile}}')) {
    require_once '{{.BootstrapFile}}';
}

// Request handler function
$handler = function() use ($workerServer) {
    // Your application code here
    // All standard PHP superglobals ($_GET, $_POST, etc.) are set for each request
    
    // Example router
    $uri = $_SERVER['REQUEST_URI'];
    
    // Basic routing
    if (preg_match('#^/([^/]+)(?:/([^/]+))?(?:/(.*))?$#', $uri, $matches)) {
        $controller = $matches[1] ?? 'home';
        $action = $matches[2] ?? 'index';
        $params = $matches[3] ?? '';
        
        // Include controller file if exists
        $controllerFile = "controllers/{$controller}.php";
        if (file_exists($controllerFile)) {
            include $controllerFile;
        } else {
            header("HTTP/1.1 404 Not Found");
            echo "Not Found: $uri";
        }
    } else {
        // Default handler
        echo "Welcome to FrangoPhp worker mode!";
    }
};

// Request handling loop
$requestCount = 0;
$maxRequests = {{.MaxRequests}};

while ($requestCount < $maxRequests && frankenphp_handle_request($handler)) {
    $requestCount++;
    
    // Run garbage collection occasionally
    if ($requestCount % 100 === 0) {
        gc_collect_cycles();
    }
}

// Worker finished
error_log("Worker handled $requestCount requests before exiting");
`
    
    // Replace template variables
    tmpl, err := template.New("worker").Parse(workerTemplate)
    if err != nil {
        return fmt.Errorf("failed to parse worker template: %w", err)
    }
    
    data := struct {
        BootstrapFile string
        MaxRequests   int
    }{
        BootstrapFile: m.workerConfig.BootstrapFile,
        MaxRequests:   m.workerConfig.MaxRequests,
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return fmt.Errorf("failed to execute worker template: %w", err)
    }
    
    // Create the worker script in the VFS
    workerPath := "/_frango_worker.php"
    if err := vfs.CreateVirtualFile(workerPath, buf.Bytes()); err != nil {
        return fmt.Errorf("failed to create worker script: %w", err)
    }
    
    m.workerScriptPath = workerPath
    m.logger.Printf("Created worker script at %s", workerPath)
    
    return nil
}

// Add a worker mode option
func WithWorkerMode(enabled bool, maxRequests int, bootstrapFile string) Option {
    return func(m *Middleware) {
        m.workerConfig = WorkerConfig{
            Enabled:       enabled,
            MaxRequests:   maxRequests,
            BootstrapFile: bootstrapFile,
        }
    }
}
```

**Explanation:**
- FrankenPHP's worker mode is a key feature that allows PHP to stay in memory between requests
- This implementation adds support for generating a worker script that uses `frankenphp_handle_request()` for handling requests
- The worker script includes garbage collection to prevent memory leaks
- The `maxRequests` setting allows workers to restart after a certain number of requests, similar to php-fpm's behavior
- This aligns with FrankenPHP's recommendation to use worker mode for better performance

### 7.3. Thread Safety Enhancements

**Problem:** FrankenPHP uses threads for concurrency, but our implementation doesn't fully account for thread-safety requirements.

**Solution:** Enhance thread safety throughout the codebase.

**File:** `vfs.go`

```go
// Add more granular locking for better thread safety
type VFS struct {
    // Existing fields...
    
    // Separate locks for different operations
    pathMutex      sync.RWMutex     // For path resolution operations
    cacheMutex     sync.RWMutex     // For cache operations
    fileMutex      sync.RWMutex     // For file content operations
}

// Update ResolvePath with better locking
func (v *VFS) ResolvePath(virtualPath string) (string, error) {
    // Normalize path
    virtualPath = normalizePath(virtualPath)
    
    // Check cache first in non-development mode
    if !v.developMode {
        v.cacheMutex.RLock()
        if cachedPath, ok := v.pathCache[virtualPath]; ok {
            v.cacheMutex.RUnlock()
            return cachedPath, nil
        }
        v.cacheMutex.RUnlock()
    }
    
    // More fine-grained locking for the actual resolution
    v.pathMutex.RLock()
    defer v.pathMutex.RUnlock()
    
    // Rest of resolution logic...
    
    // Cache result with separate lock
    if resolvedPath != "" && !v.developMode {
        v.cacheMutex.Lock()
        v.pathCache[virtualPath] = resolvedPath
        v.cacheMutex.Unlock()
    }
    
    return resolvedPath, nil
}

// Thread-safe file content access
func (v *VFS) GetFileContent(virtualPath string) ([]byte, error) {
    v.fileMutex.RLock()
    defer v.fileMutex.RUnlock()
    
    // File content retrieval logic...
    
    return content, nil
}
```

**Explanation:**
- FrankenPHP uses threads rather than processes, making thread safety critical
- This implementation adds more granular locks to improve thread safety
- Different lock types are used for different operations to reduce contention
- This helps prevent issues when multiple threads access the VFS concurrently
- The documentation notes that thread-unsafe extensions can cause problems in FrankenPHP

### 7.4. Environment Variables Handling

**Problem:** FrankenPHP makes environment variables available in `$_SERVER`, but our implementation doesn't account for this properly.

**Solution:** Properly handle environment variables in the PHP context.

**File:** `execute.go`

```go
// Update environment variables handling
func buildPhpEnvironment(m *Middleware, phpFilePath, scriptPath, scriptDir, documentRoot string,
    requestData *RequestData, virtualPath, vfsName string, envData map[string]string) map[string]string {

    // Base PHP environment...
    phpEnv := map[string]string{
        // Existing settings...
    }
    
    // CRITICAL: Add system environment variables to $_SERVER
    // FrankenPHP makes env vars available in $_SERVER by default
    for _, envVar := range os.Environ() {
        if pair := strings.SplitN(envVar, "=", 2); len(pair) == 2 {
            key, value := pair[0], pair[1]
            // Only add if not already set by request data
            if _, exists := phpEnv[key]; !exists {
                phpEnv[key] = value
            }
        }
    }
    
    // Rest of function...
    
    return phpEnv
}
```

**Explanation:**
- The FrankenPHP documentation indicates that environment variables are available in `$_SERVER`
- This implementation ensures system environment variables are properly passed to PHP
- We maintain priority for request-specific variables over system variables
- This aligns with how FrankenPHP makes environment variables accessible to PHP scripts

### 7.5. Document Root and Directory Handling

**Problem:** Directory context is crucial for includes and autoloading, but our handling doesn't match FrankenPHP's expectations.

**Solution:** Implement FrankenPHP-style document root and directory handling.

**File:** `execute.go`

```go
// Update document root handling
func resolveScriptPath(m *Middleware, vfs *VFS, scriptPath, requestPath string) (string, error) {
    // Existing resolution logic...
    
    // Ensure document root emulation matches FrankenPHP's
    phpFilePath, err := vfs.ResolvePath(scriptPath)
    if err != nil {
        return "", err
    }
    
    // Track both physical and logical paths
    vfs.TrackLogicalPath(phpFilePath, scriptPath)
    
    // Set document root in global map for consistent access
    m.mutex.Lock()
    m.documentRoots[scriptPath] = filepath.Dir(scriptPath)
    m.mutex.Unlock()
    
    return phpFilePath, nil
}

// Add function to get document root for a script
func (m *Middleware) GetDocumentRoot(scriptPath string) string {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    if docRoot, exists := m.documentRoots[scriptPath]; exists {
        return docRoot
    }
    
    // Default to script directory
    return filepath.Dir(scriptPath)
}
```

**Explanation:**
- FrankenPHP properly sets document root for correct include path resolution
- This implementation ensures that document root handling matches FrankenPHP's behavior
- We track document roots for each script to ensure consistent behavior across requests
- This is critical for proper functioning of relative includes and autoloading

### 7.6. Request Parameter Handling

**Problem:** Our current implementation doesn't account for FrankenPHP's specific handling of request data.

**Solution:** Align our parameter handling with FrankenPHP's behavior.

**File:** `php_globals.go`

```php
// Update parameter handling in PHP globals script

// Initialize $_GET with improved normalization (FrankenPHP behavior)
$_GET_RAW = json_decode($_SERVER['_GET'] ?? '{}', true);
$_GET = [];
foreach ($_GET_RAW as $key => $value) {
    // Handle nested arrays properly (important for form arrays like foo[])
    if (is_array($value)) {
        if (count($value) === 1 && !isset($value[0]) && !is_array(current($value))) {
            // Simple single-value array from Go - unwrap it
            $_GET[$key] = current($value);
        } else {
            // Preserve arrays that appear to be intentional
            $_GET[$key] = $value;
        }
    } else {
        $_GET[$key] = $value;
    }
}

// Properly parse nested form data for $_POST (FrankenPHP behavior)
$_POST_RAW = json_decode($_SERVER['_POST'] ?? '{}', true);
$_POST = [];
foreach ($_POST_RAW as $key => $value) {
    // Handle form arrays (foo[] syntax)
    if (is_array($value)) {
        if (count($value) === 1 && !isset($value[0]) && !is_array(current($value))) {
            $_POST[$key] = current($value);
        } else {
            $_POST[$key] = $value;
        }
    } else {
        $_POST[$key] = $value;
    }
}

// Handle nested form arrays with PHP's expected behavior
function parseArrayParameters(&$result, $key, $value) {
    if (preg_match('/^([^[]+)(\[.*)/', $key, $matches)) {
        $mainKey = $matches[1];
        $subKeys = $matches[2];
        
        if (!isset($result[$mainKey]) || !is_array($result[$mainKey])) {
            $result[$mainKey] = [];
        }
        
        // Parse remaining brackets
        $subKeyParts = [];
        $pattern = '/\[([^\]]*)\]/';
        preg_match_all($pattern, $subKeys, $matches);
        
        if (!empty($matches[1])) {
            $current = &$result[$mainKey];
            foreach ($matches[1] as $subKey) {
                if ($subKey === '') {
                    // Empty brackets [] indicate appending to array
                    $current[] = [];
                    $keys = array_keys($current);
                    $lastKey = end($keys);
                    $current = &$current[$lastKey];
                } else {
                    // Named key
                    if (!isset($current[$subKey]) || !is_array($current[$subKey])) {
                        $current[$subKey] = [];
                    }
                    $current = &$current[$subKey];
                }
            }
            $current = $value;
        }
    } else {
        $result[$key] = $value;
    }
}
```

**Explanation:**
- FrankenPHP uses standard PHP superglobals, so we need to ensure our handling matches
- This implementation provides better handling of nested arrays in form data
- It implements proper parsing for PHP's form array syntax (e.g., `foo[]`, `foo[bar]`)
- This ensures compatibility with existing PHP code that expects standard behavior

## Updated Implementation Schedule

Based on these FrankenPHP-specific optimizations, we should update our implementation schedule:

### Phase 1: Core Changes (Week 1)
- Replace wrapper-based execution
- Update PHP globals script for magic constants
- Update path parameter handling to integrate with `$_GET`
- **Add: FrankenPHP environment integration**

### Phase 2: Path Management (Week 2)
- Enhance VFS for logical path tracking
- Implement path resolution caching
- Add directory context management
- **Add: Thread-safety enhancements**

### Phase 3: Composer and Autoloader Support (Week 3)
- Add Composer support features
- Implement PSR-4 autoloader
- **Add: FrankenPHP worker mode support**
- **Add: Document root and include path handling**

### Phase 4: Testing and Performance (Week 4)
- Implement comprehensive test suite
- Add performance optimizations
- **Add: FrankenPHP compatibility tests**
- Conduct benchmarks and fine-tune