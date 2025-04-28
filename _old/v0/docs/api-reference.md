# Frango API Reference

This document provides a comprehensive reference for the Frango API.

## Middleware Creation and Configuration

### New

```go
func New(opts ...Option) (*Middleware, error)
```

Creates a new PHP middleware instance with the given options. Returns a configured middleware and any error encountered.

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("web"),
    frango.WithDevelopmentMode(true),
)
if err != nil {
    log.Fatalf("Error creating PHP middleware: %v", err)
}
defer php.Shutdown()
```

### Configuration Options

#### WithSourceDir

```go
func WithSourceDir(dir string) Option
```

Sets the source directory for PHP files.

**Example:**
```go
frango.WithSourceDir("web")
```

#### WithDevelopmentMode

```go
func WithDevelopmentMode(enabled bool) Option
```

Enables or disables development mode. In development mode, file changes are detected immediately, and caching is disabled.

**Example:**
```go
frango.WithDevelopmentMode(true) // Enable development mode
frango.WithDevelopmentMode(false) // Enable production mode
```

#### WithLogger

```go
func WithLogger(logger *log.Logger) Option
```

Sets a custom logger.

**Example:**
```go
customLogger := log.New(os.Stdout, "[custom] ", log.LstdFlags)
frango.WithLogger(customLogger)
```

## Middleware Operation

### ServeHTTP

```go
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

Implements the http.Handler interface, allowing the middleware to be used directly as a handler or in middleware chains.

**Example:**
```go
http.Handle("/php/", php) // Direct usage
http.Handle("/api/", php.Wrap(apiHandler)) // Middleware chain
```

### Shutdown

```go
func (m *Middleware) Shutdown()
```

Cleans up resources and shuts down the PHP middleware.

**Example:**
```go
defer php.Shutdown()
```

## PHP Endpoint Registration

### HandlePHP

```go
func (m *Middleware) HandlePHP(pattern string, phpFilePath string)
```

Maps a URL pattern to a PHP file. The pattern is the URL path that will be exposed to clients, and the PHP file path is relative to the source directory.

**Example:**
```go
php.HandlePHP("/api/user", "api/user.php")
php.HandlePHP("/about", "about.php")
php.HandlePHP("/", "index.php")
```

### Handle

```go
func (m *Middleware) Handle(pattern string, phpFilePath string)
```

A flexible handler registration function that supports multiple formats:
1. Classic style: `Handle("/users", "users.php")`
2. Method-specific: `Handle("GET /users", "users_get.php")`
3. With parameters: `Handle("GET /users/{id}", "user_detail.php")`

**Example:**
```go
php.Handle("/api/users", "api/users.php")
php.Handle("GET /api/users/{id}", "api/user_detail.php")
php.Handle("POST /api/users", "api/user_create.php")
```

### HandleDir

```go
func (m *Middleware) HandleDir(prefix string, dirPath string) error
```

Registers all PHP files in a directory under a URL prefix.

**Example:**
```go
if err := php.HandleDir("/pages", "pages"); err != nil {
    log.Fatalf("Error registering pages directory: %v", err)
}
```

### Wrap

```go
func (m *Middleware) Wrap(next http.Handler) http.Handler
```

Creates middleware that tries to handle a request with PHP, and if it doesn't match, passes it to the next handler.

**Example:**
```go
// Create combined middleware
handler := php.Wrap(apiHandler)
http.Handle("/api/", handler)
```

### ShouldHandlePHP

```go
func (m *Middleware) ShouldHandlePHP(r *http.Request) bool
```

Checks if a request should be handled by PHP based on registered routes and file paths.

**Example:**
```go
if php.ShouldHandlePHP(r) {
    php.ServeHTTP(w, r)
    return
}
// Not a PHP request, handle with other logic
```

## Data Injection and Rendering

### RenderData

```go
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}
```

A handler function type that can inject variables into PHP rendering.

### HandleRender

```go
func (m *Middleware) HandleRender(pattern string, phpFile string, renderFn RenderData)
```

Registers a handler that lets you inject variables into a PHP template before rendering.

**Example:**
```go
php.HandleRender("/dashboard", "dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Dashboard",
        "user": map[string]interface{}{
            "name":  "John Doe",
            "email": "john@example.com",
        },
    }
})
```

### SetRenderHandler

```go
func (m *Middleware) SetRenderHandler(pattern string, renderFn RenderData)
```

Directly associates a render function with a specific URL path. Useful for updating an existing handler.

**Example:**
```go
// First register the PHP file
php.HandlePHP("/dashboard", "dashboard.php")

// Then set the render function
php.SetRenderHandler("/dashboard", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Dashboard",
        "data": getData(), // Dynamic data function
    }
})
```

## Embedding PHP Files

### AddFromEmbed

```go
func (m *Middleware) AddFromEmbed(urlPath string, fs embed.FS, fsPath string) string
```

Adds a PHP file from an embedded filesystem and returns the temporary file path.

**Parameters:**
- `urlPath`: URL pattern to serve this file at
- `fs`: Embedded filesystem containing the PHP file
- `fsPath`: Path to the PHP file within the embedded filesystem

**Example:**
```go
//go:embed php/template.php
var templateFS embed.FS

// Add from embedded FS and get the path
targetPath := php.AddFromEmbed("/template", templateFS, "php/template.php")

// To use with a render function
php.SetRenderHandler("/template", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Embedded Template",
        // other data...
    }
})
```

## Path Resolution

### ResolveDirectory

```go
func ResolveDirectory(path string) (string, error)
```

Utility function that tries to find a directory using multiple strategies.

**Example:**
```go
webDir, err := frango.ResolveDirectory("web")
if err != nil {
    log.Fatalf("Error finding web directory: %v", err)
}
log.Printf("Found web directory at: %s", webDir)
``` 