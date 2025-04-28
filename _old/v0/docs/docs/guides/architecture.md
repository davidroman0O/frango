# Frango Architecture Overview

This guide explains Frango's architecture, core components, and the design principles that underpin the library.

## Design Philosophy

Frango is built on several key design principles:

1. **Seamless Integration**: Provide a natural bridge between Go and PHP without forcing developers to learn new paradigms
2. **Standard Interfaces**: Follow Go's standard HTTP interfaces to integrate easily with existing Go applications
3. **Performance Focus**: Minimize overhead in the PHP-Go communication layer
4. **Developer Experience**: Create intuitive APIs with sensible defaults and comprehensive debugging tools
5. **Security First**: Implement secure defaults and protections against common vulnerabilities
6. **Flexible Infrastructure**: Support multiple file sources, configuration options, and deployment patterns

## High-Level Architecture

At a high level, Frango consists of these main components:

```
+---------------------+
|   Go Application    |
+----------+----------+
           |
+----------v----------+     +-----------------+
|  Frango Middleware  +---->+ Virtual         |
|                     |     | File System     |
+----------+----------+     +-----------------+
           |
+----------v----------+
|    FrankenPHP       |
|    (PHP Runtime)    |
+----------+----------+
           |
+----------v----------+
|    PHP Scripts      |
+---------------------+
```

Let's explore each component in detail:

## Core Components

### 1. Middleware

The central component is the `Middleware` struct, which implements Go's `http.Handler` interface. This allows Frango to seamlessly integrate into standard Go HTTP servers.

Key features of the middleware:
- Route mapping between HTTP paths and PHP scripts
- Request parsing and transformation
- Response handling
- Lifecycle management for the PHP runtime

```go
// Example middleware initialization
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDevelopmentMode(true),
)

// Use in standard http server
http.Handle("/", php.For("/index.php"))
```

The middleware provides methods to handle specific routes:
- `For(scriptPath)`: Maps a route to a PHP script
- `Render(scriptPath, renderFn)`: Maps a route to a PHP script with additional data
- `ForVFS(vfs, scriptPath)`: Uses a specific VFS instance for a route

### 2. Virtual File System (VFS)

The VFS provides an abstraction layer for PHP files, allowing them to come from various sources:

```
+---------------------+
|    Virtual File     |
|      System         |
+---------------------+
          |
          +------------------------+
          |                        |
+---------v----------+ +-----------v--------+
|   Source Files     | |   Embedded Files   |
| (From filesystem)  | | (In Go binary)     |
+--------------------+ +--------------------+
          |                        |
          +------------------------+
                      |
            +---------v----------+
            |   Virtual Files    |
            | (Generated in-mem) |
            +--------------------+
```

The VFS handles:
- File access and caching
- File monitoring for changes (in development mode)
- Path resolution and security
- Branching for isolated environments

Key interfaces:
- `AddSourceFile/Directory`: Adds files from the filesystem
- `AddEmbeddedFile/Directory`: Adds files from embedded resources
- `CreateVirtualFile`: Creates files dynamically in memory
- `Branch()`: Creates a copy of the VFS that inherits from the parent

### 3. PHP Runtime Integration (FrankenPHP)

Frango uses FrankenPHP to execute PHP code within a Go application. The integration layer handles:
- PHP environment setup
- Passing request data to PHP
- Capturing PHP output
- Error handling
- Resource cleanup

This integration is transparent to users, who can focus on writing PHP code without worrying about the underlying mechanisms.

### 4. Request/Response Flow

The flow of an HTTP request through Frango:

```
    HTTP Request
         │
         ▼
┌─────────────────┐
│  Go HTTP Server │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Frango Middleware│
└────────┬────────┘
         │
         ▼
┌─────────────────┐    ┌─────────────────┐
│ Extract Request │───▶│ Prepare PHP     │
│ Data            │    │ Environment      │
└────────┬────────┘    └────────┬────────┘
         │                      │
         ▼                      ▼
┌─────────────────┐    ┌─────────────────┐
│ Resolve PHP     │───▶│ Execute PHP     │
│ Script Path     │    │ Script          │
└────────┬────────┘    └────────┬────────┘
         │                      │
         ▼                      ▼
┌─────────────────┐    ┌─────────────────┐
│ Process PHP     │───▶│ Return HTTP     │
│ Output          │    │ Response        │
└─────────────────┘    └─────────────────┘
```

## Data Flow

### Request Data Flow

Frango transforms HTTP request data into PHP environment variables:

1. **Query Parameters**:
   - URL query parameters are extracted and prefixed with `PHP_QUERY_`
   - These are mapped to `$_GET` in PHP

2. **Form Data**:
   - POST form data is extracted and prefixed with `PHP_FORM_`
   - These are mapped to `$_POST` in PHP

3. **JSON Data**:
   - JSON request bodies are parsed and prefixed with `PHP_JSON_`
   - These are mapped to `$_JSON` in PHP

4. **Path Parameters**:
   - URL path parameters from patterns like `/users/{id}` are extracted
   - These are mapped to `$_PATH` in PHP

5. **Headers**:
   - HTTP headers are prefixed with `PHP_HEADER_`
   - These are accessible via `$_SERVER` in PHP

6. **File Uploads**:
   - File uploads are processed and mapped to `$_FILES` in PHP

### Response Data Flow

PHP output becomes the HTTP response:

1. PHP script is executed
2. Output is captured
3. HTTP status code, headers, and body are extracted
4. Response is sent back to the client

## Key Abstractions

### 1. Options Pattern

Frango uses the functional options pattern for configuration:

```go
php, err := frango.New(
    frango.WithSourceDir("./templates"),
    frango.WithDevelopmentMode(true),
    frango.WithErrorDisplay(true),
)
```

This makes the API flexible and extensible without breaking changes.

### 2. VFS Branching

The VFS branching system allows creating isolated environments:

```
           ┌───────────┐
           │ Root VFS  │
           └─────┬─────┘
                 │
     ┌───────────┴───────────┐
     │                       │
┌────▼────┐            ┌─────▼────┐
│Branch A │            │Branch B  │
└────┬────┘            └─────┬────┘
     │                       │
┌────▼────┐            ┌─────▼────┐
│Branch A1│            │Branch B1 │
└─────────┘            └──────────┘
```

Each branch inherits from its parent but can modify files without affecting the parent.

### 3. Template Rendering

Frango provides a data passing mechanism to render PHP with dynamic data:

```go
http.Handle("/user/profile", php.Render("/profile.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "user": getUser(r),
        "settings": getUserSettings(r),
    }
}))
```

The returned data becomes PHP variables in the script.

## Architecture Layers

Frango follows a layered architecture:

1. **Public API Layer**:
   - The methods exposed to users
   - Simple, consistent interfaces for common operations

2. **Middleware Layer**:
   - HTTP request/response handling
   - Routing and path resolution

3. **Execution Layer**:
   - PHP script execution
   - Environment preparation
   - Output processing

4. **Storage Layer**:
   - VFS implementation
   - File operations and caching
   - Branching mechanics

5. **Runtime Layer**:
   - FrankenPHP integration
   - PHP initialization and shutdown
   - Resource management

## Internal File Mapping

Frango maps file paths in several stages:

1. **Route Path**: The URL path in the HTTP request
2. **Virtual Path**: The path within the VFS
3. **Disk Path**: The actual file location on disk

Example:
- Route path: `/users/profile`
- Virtual path: `/users/profile.php`
- Disk path: `/tmp/frango-123456/vfs-789012/users/profile.php`

## Lifecycle Management

Frango manages several lifecycles:

1. **Application Lifecycle**:
   - `New()`: Initialize Frango
   - `Shutdown()`: Clean up resources

2. **Request Lifecycle**:
   - Request received
   - Data extracted
   - PHP executed
   - Response sent

3. **VFS Lifecycle**:
   - Creation (root or branch)
   - File operations
   - Monitoring (in development mode)
   - Cleanup

4. **PHP Runtime Lifecycle**:
   - FrankenPHP initialization
   - Script execution
   - Output capture
   - Runtime cleanup

## Important Design Decisions

### 1. Using FrankenPHP

Frango uses FrankenPHP as the PHP runtime because:
- It's specifically designed for embedding PHP in Go
- It provides efficient PHP execution
- It handles concurrency properly in Go's environment

### 2. VFS Architecture

The VFS design allows:
- Flexible file sourcing without filesystem dependencies
- Efficient file monitoring for development
- Secure file access with virtual paths
- Isolation between different parts of the application

### 3. Environment Variable Mapping

PHP superglobals are populated via environment variables because:
- It's the most reliable way to pass data between Go and PHP
- It works with FrankenPHP's execution model
- It's consistent across different PHP versions and configurations

### 4. Error Handling

Frango provides comprehensive error handling:
- Helpful error messages for development
- Custom error handlers for production
- Separate error paths for different failure modes
- Logging of PHP errors in Go's context

## Performance Considerations

Frango is designed for performance:

1. **Caching**:
   - VFS caches file content and metadata
   - Path resolution results are cached
   - File hashes are stored to detect changes efficiently

2. **Minimized Copies**:
   - Data is passed by reference when possible
   - Large request bodies are read once

3. **Concurrency Control**:
   - Appropriate locks protect shared resources
   - Read operations use read locks for concurrency

4. **Resource Management**:
   - Temporary files are cleaned up
   - Memory usage is monitored
   - FrankenPHP resources are properly released

## Security Architecture

Security is built into Frango's design:

1. **Path Sanitization**:
   - All paths are normalized
   - Directory traversal attacks are prevented
   - Virtual paths provide isolation

2. **Input Validation**:
   - Request data is validated before processing
   - Path parameters are checked against patterns
   - File operations include bounds checking

3. **Default Protections**:
   - Direct access to PHP files is blocked by default
   - Files are executed in isolated environments
   - Symlink following is disabled

## Extensibility Points

Frango is designed to be extended:

1. **Custom VFS Implementations**:
   - Create custom VFS sources
   - Implement specialized file operations
   - Extend branching mechanics

2. **Middleware Hooks**:
   - Preprocess request data
   - Postprocess PHP output
   - Customize error handling

3. **Runtime Customization**:
   - Configure PHP settings
   - Add custom PHP libraries
   - Extend PHP superglobals

## Implementation Details

### Concurrency Model

Frango uses Go's concurrency primitives:
- Mutexes protect shared resources
- Each request is processed independently
- File watching uses goroutines and channels

### Error Propagation

Errors are propagated through several layers:
1. FrankenPHP errors are captured and converted to Go errors
2. VFS errors include context about file operations
3. HTTP errors are returned with appropriate status codes

### Memory Management

Memory is managed carefully:
- Large file content is streamed when possible
- Temporary files are used for large operations
- Resources are cleaned up via defer statements and finalizers

## Conclusion

Frango's architecture is designed to provide a seamless experience for integrating PHP into Go applications. By understanding the core components and design decisions, you can leverage Frango's full potential for your hybrid applications.

The middleware approach, VFS abstraction, and careful data flow management allow you to write PHP code that feels natural while running within a high-performance Go environment.

## Further Reading

- [VFS Tutorial](../tutorials/virtual-filesystem.md) - Deeper dive into the Virtual File System
- [Request/Response API Reference](../api-reference/request-response.md) - Details on HTTP handling
- [Performance Guide](./performance.md) - Optimizing Frango applications 