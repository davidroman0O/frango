# Frango PHP Implementation Plan with FrankenPHP Integration

This document outlines a comprehensive implementation plan for enhancing the Frango PHP library to work seamlessly with FrankenPHP, focusing on correct PHP behavior and optimal performance.

## Table of Contents

1. [Introduction and Key Issues](#introduction-and-key-issues)
2. [Core Script Execution with FrankenPHP](#core-script-execution-with-frankenphp)
3. [PHP Globals and Superglobals](#php-globals-and-superglobals)
4. [VFS and Path Management](#vfs-and-path-management)
5. [Composer and Autoloader Support](#composer-and-autoloader-support)
6. [FrankenPHP Worker Mode](#frankenphp-worker-mode)
7. [Performance Optimizations](#performance-optimizations)
8. [Testing Strategy](#testing-strategy)
9. [Implementation Schedule](#implementation-schedule)

## Introduction and Key Issues

The current implementation of Frango has several issues that prevent it from fully leveraging FrankenPHP's capabilities and meeting standard PHP developer expectations:

1. **Wrapper-Based Script Execution**
   - PHP scripts are executed through wrapper scripts which causes incorrect `__FILE__` and `__DIR__` values
   - The wrapper approach breaks autoloading and relative includes
   - FrankenPHP has better ways to prepend code than creating wrappers

2. **Non-Standard Superglobals**
   - Path parameters are stored in a custom `$_PATH` superglobal instead of `$_GET` as expected
   - FrankenPHP expects standard PHP superglobals for compatibility

3. **Directory and Path Issues**
   - Physical vs. logical path structure discrepancies affect autoloading
   - Incorrect working directory handling affects include paths
   - FrankenPHP provides mechanisms (`chdir()` via `auto_prepend_text`) for this

4. **Thread Safety Concerns**
   - FrankenPHP uses threads for concurrency rather than processes (unlike traditional PHP-FPM)
   - Current VFS implementation isn't optimized for thread-safety

5. **Performance Opportunities**
   - No caching of path resolutions and directory listings
   - No support for FrankenPHP's worker mode for persistent PHP execution
   - Redundant operations that could be pre-computed 

## Core Script Execution with FrankenPHP

The most fundamental issue with the current implementation is the use of wrapper scripts to execute PHP files. This approach leads to incorrect `__FILE__` and `__DIR__` values, breaks autoloading, and creates inconsistent directory contexts for includes.

### 2.1. Replace Wrapper-Based Execution

**Problem:** 
- Magic constants like `__FILE__` and `__DIR__` point to the wrapper, not the original script
- Relative includes (`include 'utils.php'`) resolve against the wrapper directory
- Composer autoloading breaks because it relies on correct file paths

**Solution:** Replace the wrapper approach with direct execution and use FrankenPHP's `auto_prepend_file` directive.

```go
// Replace createPhpWrapper with a simpler, direct execution approach
func preparePhpExecution(vfs *VFS, phpFilePath, virtualPath string, globalsFile string) (string, error) {
    // Just return the actual PHP file path - no wrapper needed
    // We'll use FrankenPHP's auto_prepend_file to include globals instead
    return phpFilePath, nil
}
```

**Explanation:** 
- This removes the wrapper creation entirely and simply returns the actual PHP file path
- Instead of creating a wrapper script, we use FrankenPHP's `auto_prepend_file` directive
- This allows the PHP engine to execute the target script directly, preserving the correct file context

### 2.2. Leverage FrankenPHP Environment Variables

**Problem:** The current `ExecutePHP` function doesn't take advantage of FrankenPHP's environment handling.

**Solution:** Update `buildPhpEnvironment` to use FrankenPHP-specific features:

```go
func buildPhpEnvironment(m *Middleware, phpFilePath, scriptPath, scriptDir, documentRoot string,
    requestData *RequestData, virtualPath, vfsName string, envData map[string]string) map[string]string {

    // Base PHP environment with correct path information
    phpEnv := map[string]string{
        "SCRIPT_FILENAME": phpFilePath,          // Physical path needed for execution
        "FRANGO_LOGICAL_FILENAME": scriptPath,   // Logical path for __FILE__ emulation
        "SCRIPT_NAME": scriptPath,               // Web-accessible path
        "PHP_SELF": scriptPath,                  // Match SCRIPT_NAME
        "DOCUMENT_ROOT": documentRoot,           // Must be the parent directory of the script
    }

    // Add environment for PHP globals via auto_prepend
    globalsPath := filepath.Join(vfs.tempDir, "/_frango_php_globals.php")
    phpEnv["auto_prepend_file"] = globalsPath

    // CRITICAL: Set the directory context via chdir in auto_prepend
    // This ensures relative includes work correctly
    chdirScript := fmt.Sprintf("<?php chdir('%s'); ?>", 
        strings.ReplaceAll(scriptDir, "'", "\\'"))
    phpEnv["auto_prepend_text"] = chdirScript
    
    return phpEnv
}
```

**Explanation:**
- We use `auto_prepend_file` to include our globals script before executing the target script
- We use `auto_prepend_text` to inject a `chdir()` call that sets the correct working directory context
- This ensures relative includes work correctly by resolving against the logical directory

## PHP Globals and Superglobals

The current implementation uses non-standard superglobals (`$_PATH`) instead of following PHP conventions. FrankenPHP expects standard PHP superglobals like `$_GET`, `$_POST`, etc.

### 3.1. Magic Constants Handling

**Problem:** 
- PHP's magic constants like `__FILE__` and `__DIR__` resolve to wrapper script paths, not logical paths
- FrankenPHP's direct execution model requires a way to override these constants

**Solution:** Implement a mechanism to override magic constants to use logical paths.

```php
// ----- MAGIC CONSTANTS HANDLING -----

// If logical filename is set, override __FILE__ and __DIR__
if (isset($_SERVER['FRANGO_LOGICAL_FILENAME'])) {
    $GLOBALS['__FRANGO_LOGICAL_FILE__'] = $_SERVER['FRANGO_LOGICAL_FILENAME'];
    $GLOBALS['__FRANGO_LOGICAL_DIR__'] = dirname($_SERVER['FRANGO_LOGICAL_FILENAME']);
    
    // Define a class to override magic constants via stream wrapper
    class FrangoFileMagic {
        public static function getFile() {
            // Return logical path from globals
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
    
    // Redefine __FILE__ and __DIR__ as functions where possible
    if (!defined('__FILE__')) {
        function __FILE__() {
            return \FrangoFileMagic::getFile();
        }
    }
}
```

**Explanation:**
- This mechanism tries to override PHP's built-in `__FILE__` and `__DIR__` constants to return logical paths
- We store the logical path in `$_SERVER['FRANGO_LOGICAL_FILENAME']` which FrankenPHP will make available
- We use a class to handle the overriding logic, which integrates well with FrankenPHP's execution model
- While this approach isn't perfect (PHP constants can't be truly overridden), it works for most use cases

### 3.2. Integrate Path Parameters with $_GET

**Problem:** 
- Path parameters currently go into a custom `$_PATH` superglobal
- PHP developers expect route parameters to be in `$_GET`
- FrankenPHP populates `$_GET` with query parameters

**Solution:** Merge path parameters into `$_GET` for compatibility with PHP standards.

```php
// Initialize $_PATH from pre-computed JSON 
$_PATH = json_decode($_SERVER['_PATH'] ?? '{}', true);
$GLOBALS['_PATH'] = $_PATH;

// Initialize $_GET from pre-computed JSON
$_GET = json_decode($_SERVER['_GET'] ?? '{}', true);

// CRITICAL: Merge path parameters into $_GET for PHP compatibility
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
- Path parameters take precedence over query parameters (if both exist)
- This follows PHP standards and ensures compatibility with existing code
- FrankenPHP expects `$_GET` to contain all parameters, whether from query string or route

### 3.3. Form Data Handling for FrankenPHP

**Problem:** 
- FrankenPHP expects standard form handling
- Our implementation might not properly parse nested form arrays

**Solution:** Update form data handling for PHP standard compatibility.

```php
// Properly parse form data from JSON into $_POST
$_POST_RAW = json_decode($_SERVER['_POST'] ?? '{}', true);
$_POST = [];

// Handle form arrays properly (foo[] syntax)
foreach ($_POST_RAW as $key => $value) {
    if (is_array($value) && count($value) === 1 && !isset($value[0])) {
        // Simple value array from Go - unwrap it
        $_POST[$key] = current($value);
    } else {
        $_POST[$key] = $value;
    }
    
    // Handle nested form arrays by parsing keys with [] notation
    if (strpos($key, '[') !== false) {
        parseArrayParameter($_POST, $key, $value);
    }
}

$GLOBALS['_POST'] = $_POST;
```

**Explanation:**
- This implementation ensures form data is properly parsed
- It handles PHP's form array syntax (e.g., `foo[]`, `foo[bar]`)
- It unwraps single-value arrays that come from Go's representation of form data
- This ensures compatibility with PHP code that uses form arrays
- FrankenPHP expects form data to follow PHP standards

## VFS and Path Management

The current VFS implementation doesn't properly maintain a mapping between logical paths (web-accessible paths) and physical paths (actual filesystem paths). This is especially important for FrankenPHP which has specific expectations for file paths.

### 4.1. Thread-Safe VFS Implementation

**Problem:** 
- FrankenPHP uses threads rather than processes for concurrency
- Current VFS implementation isn't optimized for thread-safety
- Potential race conditions during path resolution

**Solution:** Enhance the VFS structure with proper thread-safety mechanisms.

```go
type VFS struct {
    // Existing fields...
    
    // New fields for thread-safety
    pathMutex      sync.RWMutex     // For path resolution operations
    cacheMutex     sync.RWMutex     // For cache operations
    fileMutex      sync.RWMutex     // For file content operations
    
    // New fields for path mapping and caching
    logicalPaths   map[string]string     // Physical path -> logical path mapping
    pathCache      map[string]string     // Virtual path -> resolved path cache
    dirCache       map[string]*DirInfo   // Directory path -> directory info cache
}

// Update ResolvePath with better locking mechanism
func (v *VFS) ResolvePath(virtualPath string) (string, error) {
    // Check cache with read lock first
    v.cacheMutex.RLock()
    if cachedPath, ok := v.pathCache[virtualPath]; ok && !v.developMode {
        v.cacheMutex.RUnlock()
        return cachedPath, nil
    }
    v.cacheMutex.RUnlock()
    
    // Resolution needs read lock on paths
    v.pathMutex.RLock()
    defer v.pathMutex.RUnlock()
    
    // ... path resolution logic ...
    
    // Update cache with write lock
    if resolvedPath != "" && !v.developMode {
        v.cacheMutex.Lock()
        v.pathCache[virtualPath] = resolvedPath
        v.cacheMutex.Unlock()
    }
    
    return resolvedPath, nil
}
```

**Explanation:**
- We add separate mutex locks for different operations to reduce contention
- We use read-write locks for better performance during concurrent access
- This is particularly important in FrankenPHP's threaded environment
- The fine-grained locking improves performance for high-concurrency scenarios

### 4.2. Logical Path Tracking

**Problem:** 
- No way to determine the logical path corresponding to a physical path
- FrankenPHP needs to report correct file paths for magic constants

**Solution:** Add methods to track and retrieve logical paths.

```go
// Track logical paths corresponding to physical paths
func (v *VFS) TrackLogicalPath(physicalPath, logicalPath string) {
    v.cacheMutex.Lock()
    defer v.cacheMutex.Unlock()
    
    v.logicalPaths[physicalPath] = logicalPath
}

// Get logical path for a physical path
func (v *VFS) GetLogicalPath(physicalPath string) string {
    v.cacheMutex.RLock()
    defer v.cacheMutex.RUnlock()
    
    if logicalPath, ok := v.logicalPaths[physicalPath]; ok {
        return logicalPath
    }
    
    // Check parent VFS if available
    if v.parent != nil {
        return v.parent.GetLogicalPath(physicalPath)
    }
    
    // Default to physical path if not found
    return physicalPath
}
```

**Explanation:**
- We add methods to track and retrieve logical paths corresponding to physical paths
- This allows PHP scripts to reference their logical paths (e.g., for `__FILE__`)
- FrankenPHP can use these to report the correct paths to PHP scripts
- We maintain consistent file path behavior across the library

### 4.3. Directory Structure Caching

**Problem:** 
- Directory listings are performed repeatedly for includes/requires
- FrankenPHP's performance is hindered by redundant filesystem operations

**Solution:** Cache directory listings for faster include path resolution.

```go
// Directory information for include path resolution
type DirInfo struct {
    IncludePaths []string          // Priority-ordered include paths
    Files        map[string]string  // basename -> full path
    LastUpdated  time.Time         // For invalidation
}

// Get directory info (cached)
func (v *VFS) GetDirInfo(dirPath string) (*DirInfo, error) {
    normalizedPath := normalizePath(dirPath)
    
    // Check cache first
    v.cacheMutex.RLock()
    if info, ok := v.dirCache[normalizedPath]; ok && !v.developMode {
        v.cacheMutex.RUnlock()
        return info, nil
    }
    v.cacheMutex.RUnlock()
    
    // Build directory info
    info := &DirInfo{
        IncludePaths: []string{normalizedPath},
        Files:        make(map[string]string),
        LastUpdated:  time.Now(),
    }
    
    // ... scan directory and build info ...
    
    // Cache the result
    if !v.developMode {
        v.cacheMutex.Lock()
        v.dirCache[normalizedPath] = info
        v.cacheMutex.Unlock()
    }
    
    return info, nil
}
```

**Explanation:**
- We cache directory listings to dramatically speed up include path resolution
- This is extremely important for FrankenPHP performance with autoloading
- The cache is skipped in development mode to ensure changes are detected
- Each directory's information includes its files and potential include paths

## Composer and Autoloader Support

Composer is the standard package manager for PHP, and proper autoloader support is critical for modern PHP applications, especially when running in FrankenPHP.

### 5.1. Composer Integration

**Problem:**
- Current implementation doesn't properly support Composer's autoloading
- FrankenPHP applications often use Composer for dependencies
- Incorrect file paths break Composer's autoloader

**Solution:** Add explicit support for Composer vendor directories.

```go
// Ensure Composer support in VFS
func (v *VFS) EnsureComposerSupport(sourceDir string) error {
    // Check for vendor directory
    vendorDir := filepath.Join(sourceDir, "vendor")
    if _, err := os.Stat(vendorDir); err == nil {
        // Add entire vendor directory to VFS
        if err := v.AddSourceDirectory(vendorDir, "/vendor"); err != nil {
            return fmt.Errorf("failed to add vendor directory: %w", err)
        }
    }
    return nil
}
```

**Explanation:**
- We add the entire Composer vendor directory to the VFS
- This preserves the directory structure expected by Composer
- FrankenPHP can now correctly load vendor files
- This ensures `require 'vendor/autoload.php'` works as expected

### 5.2. PSR-4 Autoloader Support

**Problem:**
- PHP applications often use PSR-4 autoloading for class loading
- FrankenPHP needs correct paths for autoloading to work

**Solution:** Add support for PSR-4 autoloading.

```go
// Add PSR-4 namespace support
func (m *Middleware) AddPSR4Namespace(namespace string, basePath string) {
    // Add trailing backslash to namespace if not present
    if !strings.HasSuffix(namespace, "\\") {
        namespace += "\\"
    }
    
    // Add mapping
    m.psr4Namespaces = append(m.psr4Namespaces, PSR4Namespace{
        Namespace: namespace,
        BasePath:  normalizePath(basePath),
    })
}
```

**Explanation:**
- We add support for PSR-4 namespace mapping
- Developers can register namespace-to-directory mappings
- This works alongside Composer or as a standalone solution
- FrankenPHP can use this to resolve class file paths correctly

## FrankenPHP Worker Mode

One of FrankenPHP's most significant features is its worker mode, which allows PHP applications to stay in memory between requests for dramatic performance improvements.

### 6.1. Worker Mode Configuration

**Problem:**
- Current implementation only supports classic mode (separate PHP processes per request)
- FrankenPHP's worker mode can provide 3-5x performance improvement
- No integration with `frankenphp_handle_request()` function

**Solution:** Add configuration options for FrankenPHP's worker mode.

```go
// Worker configuration for FrankenPHP
type WorkerConfig struct {
    Enabled       bool
    MaxRequests   int
    BootstrapFile string
}

// Add worker mode option
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
- We add configuration options for FrankenPHP's worker mode
- The `MaxRequests` setting allows workers to restart after processing a number of requests
- This is similar to PHP-FPM's behavior and helps prevent memory leaks
- The `BootstrapFile` allows loading application code once at startup

### 6.2. Worker Script Template

**Problem:**
- Need to generate a PHP worker script that uses FrankenPHP's worker mode
- Must handle garbage collection and request limits

**Solution:** Create a template for the worker script.

```php
// Worker script template for FrankenPHP
$workerTemplate = `<?php
// Worker mode bootstrap for Frango/FrankenPHP
ignore_user_abort(true);

// Store original environment
$workerServer = $_SERVER;

// Load bootstrap file if specified
if (file_exists('{{.BootstrapFile}}')) {
    require_once '{{.BootstrapFile}}';
}

// Request handler function
$handler = function() use ($workerServer) {
    // All standard PHP superglobals are set for each request
    $uri = $_SERVER['REQUEST_URI'];
    
    // Your application logic goes here
    // Include files, handle routing, etc.
};

// Request handling loop with FrankenPHP
$requestCount = 0;
$maxRequests = {{.MaxRequests}};

while ($requestCount < $maxRequests && frankenphp_handle_request($handler)) {
    $requestCount++;
    
    // Run garbage collection periodically
    if ($requestCount % 100 === 0) {
        gc_collect_cycles();
    }
}
`;
```

**Explanation:**
- The worker script uses FrankenPHP's `frankenphp_handle_request()` function
- It handles multiple requests in a single PHP process
- We include garbage collection to prevent memory leaks
- The worker respects configured request limits to prevent memory issues

## Performance Optimizations

FrankenPHP is designed for high performance, and our implementation should leverage various optimizations to maximize speed.

### 7.1. Path Resolution Caching

**Problem:** Path resolution is performed repeatedly, causing filesystem operation overhead.

**Solution:** Cache path resolution results.

**Explanation:**
- Cache resolved paths to avoid redundant filesystem operations
- Skip cache in development mode to ensure changes are detected
- Significantly improves performance for repeated path resolutions

### 7.2. Object Pooling

**Problem:** Request data structures are created and garbage collected for each request.

**Solution:** Use object pooling for request data.

**Explanation:**
- Use sync.Pool to reuse request data structures
- Reduce garbage collection pressure by reusing memory
- Benefit from reduced memory churn in high-traffic scenarios

### 7.3. Pre-allocation of Maps

**Problem:** Environment maps are created with default capacity and resize repeatedly.

**Solution:** Pre-allocate maps with appropriate capacity.

**Explanation:**
- Pre-allocate environment maps with appropriate initial capacity
- Avoid expensive map resizing operations during request handling
- Pre-initialize common values to avoid redundant operations

## Testing Strategy

A comprehensive testing approach is essential to ensure our implementation works correctly with FrankenPHP.

### 8.1. Unit Tests for Path Resolution

**Problem:** Need to verify correct path resolution behavior.

**Solution:** Create unit tests for path mapping and resolution.

**Explanation:**
- Test VFS path resolution with cached and non-cached paths
- Verify logical path tracking for correct magic constants
- Ensure thread safety with concurrent path resolution

### 8.2. Integration Tests with FrankenPHP

**Problem:** Need to verify integration with FrankenPHP.

**Solution:** Create integration tests using FrankenPHP.

**Explanation:**
- Test magic constant resolution in FrankenPHP environment
- Verify autoloading behavior works correctly
- Test worker mode request handling

### 8.3. Performance Benchmarks

**Problem:** Need to measure performance improvements.

**Solution:** Create benchmarks to measure key metrics.

**Explanation:**
- Compare performance with and without optimizations
- Measure request processing time in worker mode vs. classic mode
- Profile memory usage and garbage collection pressure

## Implementation Schedule

The implementation will be divided into four phases:

### Phase 1: Core Execution and Globals (Week 1)
- Replace wrapper-based execution
- Implement FrankenPHP environment integration
- Update superglobals handling

### Phase 2: VFS and Path Management (Week 2)
- Enhance VFS with thread-safety
- Implement path caching
- Add logical path tracking

### Phase 3: Autoloading and Composer (Week 3)
- Add Composer support
- Implement PSR-4 autoloader
- Set up proper include path handling

### Phase 4: Worker Mode and Optimization (Week 4)
- Implement FrankenPHP worker mode support
- Add performance optimizations
- Create comprehensive test suite 