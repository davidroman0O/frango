# Technical Architecture

## System Components

### 1. FrankenPHP Integration Layer
- Direct integration with FrankenPHP for PHP execution
- Replaces wrapper-based execution with FrankenPHP's native mechanisms
- Uses `auto_prepend_file` and `auto_prepend_text` directives for environment setup
- Supports worker mode for persistent PHP processes

### 2. Virtual File System (VFS)
- Thread-safe implementation with fine-grained locking
- Supports multiple file sources (filesystem, embedded, virtual)
- Maintains mappings between logical and physical paths
- Includes caching mechanisms for path resolution and directory listings
- Handles file operations (create, read, update, delete, copy, move)

### 3. Request Processing Pipeline
- Extracts and normalizes HTTP request data
- Maps request data to PHP superglobals
- Processes path parameters, query parameters, form data, and JSON bodies
- Handles file uploads and multipart form data

### 4. PHP Environment Manager
- Configures PHP execution environment
- Sets up correct working directory and include paths
- Ensures proper magic constant resolution
- Handles session and cookie management

### 5. Middleware Integration
- Provides HTTP handlers for serving PHP scripts
- Supports integration with Go's HTTP middleware patterns
- Includes render support for template data

## Data Models

### 1. VFS Data Structures
```go
type VFS struct {
    // Core mappings
    sourceMappings     map[string]string     // Virtual path -> source path
    embedMappings      map[string]string     // Virtual path -> embed temp path
    virtualFiles       map[string][]byte     // Virtual path -> content
    fileOrigins        map[string]FileOrigin // Virtual path -> origin type
    
    // Enhanced mappings for path tracking
    logicalPaths       map[string]string     // Physical path -> logical path
    pathCache          map[string]string     // Virtual path -> resolved path
    dirCache           map[string]*DirInfo   // Directory path -> directory info
    
    // Thread safety
    pathMutex          sync.RWMutex          // For path operations
    fileMutex          sync.RWMutex          // For file content operations
    cacheMutex         sync.RWMutex          // For cache operations
    
    // Configuration
    tempDir            string                // Base temp directory
    developMode        bool                  // Whether development mode is enabled
}

type DirInfo struct {
    Files              map[string]string     // basename -> full path
    IncludePaths       []string              // Priority-ordered include paths
    LastUpdated        time.Time             // For cache invalidation
}
```

### 2. Request Data Model
```go
type RequestData struct {
    Method             string
    FullURL            string
    Path               string
    RemoteAddr         string
    Headers            http.Header
    QueryParams        map[string][]string
    PathSegments       []string
    PathParams         map[string]string
    JSONBody           map[string]interface{}
    FormData           map[string][]string
    FileUploads        map[string][]*multipart.FileHeader
    RawBody            []byte
}
```

### 3. PHP Environment Configuration
```go
type PHPEnvironment struct {
    DocumentRoot       string
    ScriptPath         string
    ScriptDir          string
    WorkingDir         string
    RequestURI         string
    RequestMethod      string
    ServerVars         map[string]string
    EnvVars            map[string]string
    AutoPrependFile    string
    AutoPrependText    string
    ErrorReporting     string
    DisplayErrors      bool
}

type WorkerConfig struct {
    Enabled            bool
    MaxRequests        int
    BootstrapFile      string
    GCInterval         int
}
```

## APIs and Integrations

### 1. Core API
```go
// Middleware creation and configuration
func New(opts ...Option) (*Middleware, error)
func WithSourceDir(dir string) Option
func WithTempDir(dir string) Option
func WithDevelopmentMode(enabled bool) Option
func WithErrorHandler(phpErrorHandlerPath string) Option
func WithErrorDisplay(display bool) Option
func WithWorkerMode(enabled bool, maxRequests int, bootstrapFile string) Option

// VFS operations
func (m *Middleware) NewVFS() *VFS
func (v *VFS) AddSourceFile(sourcePath, virtualPath string) error
func (v *VFS) AddSourceDirectory(sourceDir, virtualPrefix string) error
func (v *VFS) AddEmbeddedFile(embedFS embed.FS, fsPath, virtualPath string) error
func (v *VFS) AddEmbeddedDirectory(embedFS embed.FS, fsPath, virtualPrefix string) error
func (v *VFS) CreateVirtualFile(virtualPath string, content []byte) error
func (v *VFS) CopyFile(srcVirtualPath, destVirtualPath string) error
func (v *VFS) MoveFile(srcVirtualPath, destVirtualPath string) error
func (v *VFS) DeleteFile(virtualPath string) error
func (v *VFS) GetFileContent(virtualPath string) ([]byte, error)
func (v *VFS) FileExists(virtualPath string) bool
func (v *VFS) ResolvePath(virtualPath string) (string, error)
func (v *VFS) Branch() *VFS

// HTTP handlers
func (m *Middleware) For(phpScriptPath string) http.Handler
func (m *Middleware) ForVFS(vfs *VFS, scriptPath string) http.Handler
func (m *Middleware) Render(scriptPath string, renderFn RenderData) http.Handler
func (m *Middleware) RenderVFS(vfs *VFS, scriptPath string, renderFn RenderData) http.Handler

// PSR-4 autoloader support
func (m *Middleware) AddPSR4Namespace(namespace string, basePath string)

// Direct execution
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request)
```

### 2. FrankenPHP Integration
- Uses `frankenphp.NewRequestWithContext` for request creation
- Uses `frankenphp.ServeHTTP` for PHP execution
- Uses `frankenphp.Init` for initialization
- Uses `frankenphp.Shutdown` for cleanup
- Supports `frankenphp_handle_request()` for worker mode

### 3. Standard Library Integrations
- `net/http` for HTTP handling
- `sync` for concurrency primitives
- `embed` for embedded filesystem support
- `path/filepath` for path manipulation
- `os` for filesystem operations

## Infrastructure Requirements

### 1. Development Environment
- Go 1.16+
- PHP 8.0+
- FrankenPHP compiled and available

### 2. Production Environment
- Linux, macOS, or Windows OS
- Sufficient memory for PHP execution
- Fast storage for VFS operations
- Multi-core CPU for concurrent request handling

### 3. Dependencies
- FrankenPHP library
- Standard Go libraries
- No external services required

# Development Roadmap

## Phase 1: Core Execution Enhancements

### 1.1 Direct Script Execution
- Replace wrapper-based execution with direct execution
- Implement environment setup using FrankenPHP directives
- Update `ExecutePHP` function to use new approach

### 1.2 PHP Globals Handling
- Rewrite PHP globals initialization script
- Add standard PHP superglobal implementation
- Ensure correct merging of path parameters into `$_GET`
- Implement proper form data normalization

### 1.3 Magic Constants Support
- Add mechanism to track logical file paths
- Implement override approach for `__FILE__` and `__DIR__` constants
- Update request creation to pass logical path information

## Phase 2: Thread-Safe VFS and Path Management

### 2.1 VFS Concurrency Improvements
- Refactor VFS with fine-grained locking mechanisms
- Separate mutex locks for different operations
- Implement proper read-write lock usage patterns

### 2.2 Path Resolution and Caching
- Add path resolution caching system
- Implement directory structure caching
- Add logical path tracking for physical paths
- Create efficient cache invalidation for development mode

### 2.3 Special Path Handling
- Improve handling of parameterized paths
- Support paths with special characters
- Implement consistent normalization rules

## Phase 3: Composer and Autoloading Support

### 3.1 Enhanced Include Path Handling
- Ensure correct working directory for includes
- Implement proper include path resolution
- Support relative and absolute includes correctly

### 3.2 Composer Integration
- Add automatic Composer vendor directory detection
- Support loading from vendor directory in VFS
- Ensure Composer autoload.php works correctly

### 3.3 PSR-4 Autoloader Support
- Implement PSR-4 namespace mapping
- Add API for registering custom namespaces
- Ensure compatibility with Composer's autoloader

## Phase 4: Worker Mode and Performance Optimization

### 4.1 FrankenPHP Worker Mode
- Implement worker mode configuration
- Create worker script template
- Add support for bootstrapping application code
- Implement request handling loop with FrankenPHP

### 4.2 Memory and Resource Optimization
- Implement object pooling for request data
- Pre-allocate maps with appropriate capacity
- Add proper garbage collection triggers
- Prevent memory leaks in long-running processes

### 4.3 Benchmark and Optimization
- Create performance benchmarking suite
- Identify and optimize bottlenecks
- Implement additional caching where beneficial
- Fine-tune locking strategies for better concurrency

## Phase 5: Error Handling and API Refinement

### 5.1 Enhanced Error Handling
- Improve PHP error detection
- Support configurable error reporting levels
- Add global and per-request error handlers
- Implement better stack trace presentation

### 5.2 API Refinement
- Streamline API for common use cases
- Add convenience methods for typical operations
- Improve documentation and examples
- Maintain backward compatibility where possible

### 5.3 Testing and Validation
- Create comprehensive test suite
- Test with popular PHP frameworks
- Validate across supported platforms
- Measure performance improvements

# Logical Dependency Chain

## Foundation (Must Be Built First)
1. **Direct Script Execution**: The foundation of everything else, replacing wrapper-based execution.
   - Must implement FrankenPHP directive usage
   - Must handle environment setup correctly
   - Must ensure proper working directory

2. **Thread-Safe VFS**: Core infrastructure needed for all file operations.
   - Must implement fine-grained locking
   - Must ensure thread safety for all operations
   - Must maintain backward compatibility with existing VFS functions

3. **PHP Globals Handling**: Essential for PHP script correctness.
   - Must correctly initialize all PHP superglobals
   - Must handle request data mapping properly
   - Must ensure compatibility with PHP expectations

## Building Up (Quick Path to Usable System)
4. **Path Resolution and Tracking**: Critical for correct PHP behavior.
   - Logical path tracking for magic constants
   - Directory structure caching
   - Proper include path handling

5. **Basic Error Handling**: Needed for debugging and usability.
   - Basic PHP error detection
   - Simple error reporting
   - Support for custom error handler scripts

6. **Simple API Functions**: Get to a usable state quickly.
   - HTTP handlers for common use cases
   - Simplified VFS operations
   - Basic middleware functionality

## Enhancement (Atomic Features to Build Upon)
7. **Composer and Autoloading**: Important for modern PHP apps.
   - Detection and integration of vendor directories
   - Support for PSR-0 and PSR-4 autoloading
   - Namespace mapping functionality

8. **Worker Mode**: Significant performance enhancement.
   - Worker configuration options
   - Request handling loop
   - Memory management and GC

9. **Performance Optimizations**: Incremental improvements.
   - Path resolution caching
   - Object pooling
   - Map pre-allocation
   - Lock contention reduction

## Polish and Refinement
10. **Advanced Error Handling**: Better developer experience.
    - Detailed error reporting
    - Stack trace formatting
    - Error type categorization

11. **API Refinement**: Improved usability.
    - Consistent naming conventions
    - Comprehensive documentation
    - Additional convenience methods

12. **Testing and Validation**: Ensure quality.
    - Framework compatibility tests
    - Performance benchmarking
    - Platform-specific testing

# Risks and Mitigations

## Technical Challenges

### 1. Magic Constants Resolution
**Risk**: PHP magic constants (`__FILE__`, `__DIR__`) may not work correctly with direct execution.

**Mitigation**:
- Implement logical path tracking and mapping
- Use FrankenPHP's environment variables to pass logical paths
- Create a PHP mechanism to override magic constants where possible
- Test extensively with different PHP code patterns

### 2. Thread Safety
**Risk**: Concurrent PHP execution may cause race conditions and data corruption in the VFS.

**Mitigation**:
- Use fine-grained locking with read-write mutexes
- Separate locks for different types of operations
- Implement lock hierarchy to prevent deadlocks
- Test with high concurrency loads

### 3. Composer Compatibility
**Risk**: Complex Composer autoloading may not work correctly with the VFS.

**Mitigation**:
- Ensure complete vendor directory structure in VFS
- Maintain correct file paths for Composer's autoloader
- Support all autoloading mechanisms (PSR-0, PSR-4, classmap)
- Test with popular Composer packages

## MVP Definition

### 1. Minimum Viable Product Scope
**Risk**: Trying to implement too many features at once may delay useful releases.

**Mitigation**:
- Focus first on direct script execution and thread-safe VFS
- Prioritize PHP compatibility and correct behavior
- Defer advanced optimizations to later phases
- Release incremental improvements

### 2. Backward Compatibility
**Risk**: Breaking changes may make upgrading difficult for existing users.

**Mitigation**:
- Maintain compatibility with existing API where possible
- Provide clear migration paths for breaking changes
- Create compatibility layers for transition
- Document all changes thoroughly

### 3. Performance vs. Correctness Trade-offs
**Risk**: Optimizing for performance may affect correctness or compatibility.

**Mitigation**:
- Prioritize correctness over performance initially
- Add performance optimizations incrementally
- Benchmark before and after each optimization
- Test thoroughly for regressions

## Resource Constraints

### 1. Testing Complexity
**Risk**: Comprehensive testing across platforms and PHP versions is resource-intensive.

**Mitigation**:
- Create automated test suites
- Use CI/CD pipelines for cross-platform testing
- Focus testing on key compatibility areas
- Leverage community testing for edge cases

### 2. FrankenPHP Dependency
**Risk**: Changes in FrankenPHP may affect Frango's implementation.

**Mitigation**:
- Create abstraction layer for FrankenPHP integration
- Monitor FrankenPHP development and adapt proactively
- Maintain good relationship with FrankenPHP maintainers
- Plan for compatibility with multiple FrankenPHP versions

### 3. Documentation and Examples
**Risk**: Complex features may be difficult to document and explain.

**Mitigation**:
- Create comprehensive documentation with examples
- Provide sample applications demonstrating key features
- Create video tutorials for complex workflows
- Support community Q&A channels

# Appendix

## A. Technical Specifications

### A.1 PHP Execution Process

```
┌────────────────┐       ┌────────────────┐       ┌────────────────┐
│  HTTP Request  │───────│ Go HTTP Handler │───────│  Request Data  │
└────────────────┘       └────────────────┘       └────────────────┘
                                 │                          │
                                 ▼                          ▼
┌────────────────┐       ┌────────────────┐       ┌────────────────┐
│   PHP Output   │◄──────│ FrankenPHP Run │◄──────│ PHP Environment │
└────────────────┘       └────────────────┘       └────────────────┘
        │                                                  ▲
        ▼                                                  │
┌────────────────┐                              ┌────────────────┐
│ HTTP Response  │                              │  VFS Resolver  │
└────────────────┘                              └────────────────┘
```

1. HTTP request is received by Go HTTP handler
2. Request data is extracted and normalized
3. PHP environment is configured (globals, server vars, etc.)
4. VFS resolves the PHP script path to a physical file
5. FrankenPHP executes the PHP script
6. PHP output is captured and returned as HTTP response

### A.2 VFS Path Resolution Algorithm

```
function ResolvePath(virtualPath):
    // Normalize path
    path = normalizePath(virtualPath)
    
    // Check cache first (with read lock)
    if path in pathCache and not developMode:
        return pathCache[path]
    
    // Check this VFS for the path (with read lock)
    origin = fileOrigins[path]
    if origin exists:
        if origin == OriginSource:
            return sourceMappings[path]
        else if origin == OriginEmbed or origin == OriginVirtual:
            return embedMappings[path]
    
    // Check parent VFS if exists
    if parent exists:
        parentPath = parent.ResolvePath(path)
        if parentPath exists:
            // Remember this is inherited
            inheritedPaths[path] = true
            
            // Cache the path (with write lock)
            if not developMode:
                pathCache[path] = parentPath
                
            return parentPath
    
    // Path not found
    return error("Path not found")
```

### A.3 PHP Globals Initialization

```php
// Initialize $_PATH from pre-computed JSON
$_PATH = json_decode($_SERVER['_PATH'] ?? '{}', true);
$GLOBALS['_PATH'] = $_PATH;

// Initialize $_GET from pre-computed JSON
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

// Merge path parameters into $_GET
foreach ($_PATH as $key => $value) {
    $_GET[$key] = $value;
}
$GLOBALS['_GET'] = $_GET;

// Initialize other superglobals similarly...
```

## B. Research Findings

### B.1 PHP Path Handling Expectations

1. **Magic Constants**:
   - `__FILE__`: Absolute path to the current script
   - `__DIR__`: Directory containing the current script
   - Both resolve based on the actual file being executed

2. **Include Behavior**:
   - `include 'file.php'`: Looks in current directory first, then include_path
   - `include './file.php'`: Explicitly looks in current directory
   - `include '../file.php'`: Goes up one directory
   - `include '/file.php'`: Absolute path from filesystem root
   - `include __DIR__ . '/file.php'`: Absolute path from script directory

3. **Working Directory**:
   - `getcwd()`: Returns the current working directory
   - Relative paths are resolved against the current working directory
   - `chdir()` changes the working directory for the current request

### B.2 FrankenPHP Capabilities

1. **Worker Mode**:
   - Uses `frankenphp_handle_request()` for handling multiple requests in one process
   - Significant performance improvement over classic mode
   - Memory must be carefully managed to prevent leaks

2. **PHP Directives**:
   - `auto_prepend_file`: Path to a PHP file to include before the main script
   - `auto_prepend_text`: PHP code to execute before the main script
   - `auto_append_file`: Path to a PHP file to include after the main script
   - `auto_append_text`: PHP code to execute after the main script

3. **Environment Variables**:
   - Can set any PHP environment variable through the Go API
   - Environment is isolated between concurrent requests
   - Special variables like `SCRIPT_FILENAME` affect PHP behavior 