# Middleware API Reference

This document provides a comprehensive reference for the Frango middleware API, detailing all public methods, their parameters, return values, and usage examples.

## Table of Contents

- [Initialization](#initialization)
  - [New](#new)
  - [Configuration Options](#configuration-options)
- [Routing Methods](#routing-methods)
  - [For](#for)
  - [Render](#render)
  - [ForVFS](#forvfs)
  - [RenderVFS](#rendervfs)
- [VFS Management](#vfs-management)
  - [NewVFS](#newvfs)
  - [SourceDir](#sourcedir)
  - [TempDir](#tempdir)
- [File Management](#file-management)
  - [AddSourceFile](#addsourcefile)
  - [AddSourceDirectory](#addsourcedirectory)
  - [AddEmbeddedFile](#addembeddedfile)
  - [AddEmbeddedDirectory](#addembeddeddirectory)
  - [AddEmbeddedLibrary](#addembeddedlibrary)
  - [CreateVirtualFile](#createvirtualfile)
  - [CopyFile](#copyfile)
  - [MoveFile](#movefile)
  - [DeleteFile](#deletefile)
- [PHP Execution](#php-execution)
  - [ExecutePHP](#executephp)
- [Lifecycle Management](#lifecycle-management)
  - [Shutdown](#shutdown)

## Initialization

### New

Creates a new Frango middleware instance with the specified options.

**Signature:**
```go
func New(opts ...Option) (*Middleware, error)
```

**Parameters:**
- `opts` (variadic `Option`): Configuration options for the middleware

**Returns:**
- `*Middleware`: A pointer to the initialized middleware
- `error`: An error if initialization fails

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDevelopmentMode(true),
)
if err != nil {
    log.Fatalf("Failed to initialize Frango: %v", err)
}
defer php.Shutdown()
```

### Configuration Options

These functions create options that can be passed to `New()`.

#### WithSourceDir

Sets the source directory for PHP files.

```go
func WithSourceDir(dir string) Option
```

**Example:**
```go
frango.WithSourceDir("./php-files")
```

#### WithTempDir

Sets the temporary directory for PHP files and VFS storage.

```go
func WithTempDir(dir string) Option
```

**Example:**
```go
frango.WithTempDir("/tmp/my-app")
```

#### WithDevelopmentMode

Enables real-time file change detection and disables caching.

```go
func WithDevelopmentMode(enabled bool) Option
```

**Example:**
```go
frango.WithDevelopmentMode(true)  // For development
frango.WithDevelopmentMode(false) // For production
```

#### WithLogger

Sets a custom logger.

```go
func WithLogger(logger *log.Logger) Option
```

**Example:**
```go
customLogger := log.New(os.Stdout, "[PHP] ", log.LstdFlags)
frango.WithLogger(customLogger)
```

#### WithDirectPHPURLsBlocking

Controls whether direct PHP file access in URLs should be blocked.

```go
func WithDirectPHPURLsBlocking(block bool) Option
```

**Example:**
```go
frango.WithDirectPHPURLsBlocking(true)  // Block direct access (default)
```

#### WithErrorHandler

Sets a custom PHP error handler script path.

```go
func WithErrorHandler(phpErrorHandlerPath string) Option
```

**Example:**
```go
frango.WithErrorHandler("/error_handler.php")
```

#### WithErrorDisplay

Controls whether PHP errors are displayed in the output.

```go
func WithErrorDisplay(display bool) Option
```

**Example:**
```go
frango.WithErrorDisplay(true)  // Show errors (default in development mode)
frango.WithErrorDisplay(false) // Hide errors (default in production mode)
```

## Routing Methods

### For

Returns an HTTP handler for a specific PHP script.

**Signature:**
```go
func (m *Middleware) For(scriptPath string) http.Handler
```

**Parameters:**
- `scriptPath` (string): Path to the PHP script in the VFS

**Returns:**
- `http.Handler`: An HTTP handler that executes the specified PHP script

**Example:**
```go
http.Handle("/", php.For("/index.php"))
http.Handle("/users/", php.For("/users/index.php"))
http.Handle("/users/{id}", php.For("/users/profile.php"))
```

### Render

Returns an HTTP handler that renders a PHP script with data from the renderFn.

**Signature:**
```go
func (m *Middleware) Render(scriptPath string, renderFn RenderData) http.Handler
```

**Parameters:**
- `scriptPath` (string): Path to the PHP script in the VFS
- `renderFn` (RenderData): Function that returns template data for the script

**Returns:**
- `http.Handler`: An HTTP handler that renders the script with the provided data

**Example:**
```go
http.Handle("/profile", php.Render("/profile.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "user": getUser(r),
        "stats": getUserStats(r),
    }
}))
```

### ForVFS

Returns an HTTP handler for a specific PHP script in a specific VFS.

**Signature:**
```go
func (m *Middleware) ForVFS(vfs *VFS, scriptPath string) http.Handler
```

**Parameters:**
- `vfs` (*VFS): Pointer to the VFS to use
- `scriptPath` (string): Path to the PHP script in the VFS

**Returns:**
- `http.Handler`: An HTTP handler that executes the script using the specified VFS

**Example:**
```go
// Create a branch VFS
tenantVFS := php.NewVFS()

// Add tenant-specific files
tenantVFS.CreateVirtualFile("/tenant.php", []byte("<?php echo 'Tenant View'; ?>"))

// Use the branch VFS for a specific route
http.Handle("/tenant", php.ForVFS(tenantVFS, "/tenant.php"))
```

### RenderVFS

Returns an HTTP handler that renders a PHP script in a specific VFS with data from renderFn.

**Signature:**
```go
func (m *Middleware) RenderVFS(vfs *VFS, scriptPath string, renderFn RenderData) http.Handler
```

**Parameters:**
- `vfs` (*VFS): Pointer to the VFS to use
- `scriptPath` (string): Path to the PHP script in the VFS
- `renderFn` (RenderData): Function that returns template data for the script

**Returns:**
- `http.Handler`: An HTTP handler that renders the script with the provided data using the specified VFS

**Example:**
```go
// Create custom VFS for admin section
adminVFS := php.NewVFS()
adminVFS.AddSourceDirectory("./admin-templates", "/")

// Render admin page with data
http.Handle("/admin/dashboard", php.RenderVFS(adminVFS, "/dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "stats": getAdminStats(),
        "users": getRecentUsers(),
    }
}))
```

## VFS Management

### NewVFS

Creates a new virtual filesystem instance.

**Signature:**
```go
func (m *Middleware) NewVFS() *VFS
```

**Returns:**
- `*VFS`: A pointer to the new VFS instance

**Example:**
```go
// Create a new VFS
vfs := php.NewVFS()

// Add files to the VFS
vfs.CreateVirtualFile("/hello.php", []byte("<?php echo 'Hello, world!'; ?>"))

// Use the VFS in a route
http.Handle("/hello", php.ForVFS(vfs, "/hello.php"))
```

### SourceDir

Returns the configured source directory.

**Signature:**
```go
func (m *Middleware) SourceDir() string
```

**Returns:**
- `string`: The configured source directory path

**Example:**
```go
sourceDir := php.SourceDir()
log.Printf("Using source directory: %s", sourceDir)
```

### TempDir

Returns the temporary directory used by the middleware.

**Signature:**
```go
func (m *Middleware) TempDir() string
```

**Returns:**
- `string`: The temporary directory path

**Example:**
```go
tempDir := php.TempDir()
log.Printf("Using temporary directory: %s", tempDir)
```

## File Management

### AddSourceFile

Adds a file from the filesystem to the VFS.

**Signature:**
```go
func (m *Middleware) AddSourceFile(sourcePath string, virtualPath string) error
```

**Parameters:**
- `sourcePath` (string): Path to the file on disk
- `virtualPath` (string): Virtual path to map the file to in the VFS

**Returns:**
- `error`: An error if adding the file fails

**Example:**
```go
err := php.AddSourceFile("/path/to/config.php", "/config.php")
if err != nil {
    log.Printf("Failed to add source file: %v", err)
}
```

### AddSourceDirectory

Adds all files from a directory to the VFS.

**Signature:**
```go
func (m *Middleware) AddSourceDirectory(sourceDir string, virtualPrefix string) error
```

**Parameters:**
- `sourceDir` (string): Path to the directory on disk
- `virtualPrefix` (string): Virtual path prefix to map the directory to in the VFS

**Returns:**
- `error`: An error if adding the directory fails

**Example:**
```go
err := php.AddSourceDirectory("/path/to/templates", "/templates")
if err != nil {
    log.Printf("Failed to add source directory: %v", err)
}
```

### AddEmbeddedFile

Adds a single file from an embed.FS to the VFS.

**Signature:**
```go
func (m *Middleware) AddEmbeddedFile(embedFS embed.FS, fsPath string, virtualPath string) error
```

**Parameters:**
- `embedFS` (embed.FS): Embedded filesystem
- `fsPath` (string): Path to the file in the embedded filesystem
- `virtualPath` (string): Virtual path to map the file to in the VFS

**Returns:**
- `error`: An error if adding the file fails

**Example:**
```go
//go:embed templates/index.php
var templates embed.FS

err := php.AddEmbeddedFile(templates, "templates/index.php", "/index.php")
if err != nil {
    log.Printf("Failed to add embedded file: %v", err)
}
```

### AddEmbeddedDirectory

Adds a directory from an embed.FS to the VFS.

**Signature:**
```go
func (m *Middleware) AddEmbeddedDirectory(embedFS embed.FS, fsPath string, virtualPrefix string) error
```

**Parameters:**
- `embedFS` (embed.FS): Embedded filesystem
- `fsPath` (string): Path to the directory in the embedded filesystem
- `virtualPrefix` (string): Virtual path prefix to map the directory to in the VFS

**Returns:**
- `error`: An error if adding the directory fails

**Example:**
```go
//go:embed templates/*
var templates embed.FS

err := php.AddEmbeddedDirectory(templates, "templates", "/")
if err != nil {
    log.Printf("Failed to add embedded directory: %v", err)
}
```

### AddEmbeddedLibrary

Adds an embedded file to the VFS and returns its disk path.

**Signature:**
```go
func (m *Middleware) AddEmbeddedLibrary(embedFS embed.FS, fsPath string, targetLibraryPath string) (string, error)
```

**Parameters:**
- `embedFS` (embed.FS): Embedded filesystem
- `fsPath` (string): Path to the file in the embedded filesystem
- `targetLibraryPath` (string): Target path for the library in the VFS

**Returns:**
- `string`: The disk path to the library file
- `error`: An error if adding the library fails

**Example:**
```go
//go:embed libs/validator.php
var libs embed.FS

libraryPath, err := php.AddEmbeddedLibrary(libs, "libs/validator.php", "/libs/validator.php")
if err != nil {
    log.Printf("Failed to add embedded library: %v", err)
}
log.Printf("Library available at disk path: %s", libraryPath)
```

### CreateVirtualFile

Creates a file directly in the VFS.

**Signature:**
```go
func (m *Middleware) CreateVirtualFile(virtualPath string, content []byte) error
```

**Parameters:**
- `virtualPath` (string): Virtual path for the file in the VFS
- `content` ([]byte): Content of the file

**Returns:**
- `error`: An error if creating the file fails

**Example:**
```go
content := []byte(`<?php
// Dynamically generated PHP file
echo "Generated at " . date('Y-m-d H:i:s');
?>`)

err := php.CreateVirtualFile("/dynamic.php", content)
if err != nil {
    log.Printf("Failed to create virtual file: %v", err)
}
```

### CopyFile

Copies a file within the VFS.

**Signature:**
```go
func (m *Middleware) CopyFile(srcVirtualPath, destVirtualPath string) error
```

**Parameters:**
- `srcVirtualPath` (string): Source path in the VFS
- `destVirtualPath` (string): Destination path in the VFS

**Returns:**
- `error`: An error if copying the file fails

**Example:**
```go
err := php.CopyFile("/templates/original.php", "/templates/copy.php")
if err != nil {
    log.Printf("Failed to copy file: %v", err)
}
```

### MoveFile

Moves a file within the VFS.

**Signature:**
```go
func (m *Middleware) MoveFile(srcVirtualPath, destVirtualPath string) error
```

**Parameters:**
- `srcVirtualPath` (string): Source path in the VFS
- `destVirtualPath` (string): Destination path in the VFS

**Returns:**
- `error`: An error if moving the file fails

**Example:**
```go
err := php.MoveFile("/old-location.php", "/new-location.php")
if err != nil {
    log.Printf("Failed to move file: %v", err)
}
```

### DeleteFile

Deletes a file from the VFS.

**Signature:**
```go
func (m *Middleware) DeleteFile(virtualPath string) error
```

**Parameters:**
- `virtualPath` (string): Path to the file in the VFS

**Returns:**
- `error`: An error if deleting the file fails

**Example:**
```go
err := php.DeleteFile("/temporary.php")
if err != nil {
    log.Printf("Failed to delete file: %v", err)
}
```

## PHP Execution

### ExecutePHP

Handles execution of a PHP script through the VFS.

**Signature:**
```go
func (m *Middleware) ExecutePHP(scriptPath string, vfs *VFS, renderFn RenderData, w http.ResponseWriter, r *http.Request)
```

**Parameters:**
- `scriptPath` (string): Path to the PHP script in the VFS
- `vfs` (*VFS): Pointer to the VFS to use
- `renderFn` (RenderData): Function that returns template data for the script
- `w` (http.ResponseWriter): HTTP response writer
- `r` (*http.Request): HTTP request

**Example:**
```go
// This is typically used internally by the middleware
// but can be called manually for custom handling
php.ExecutePHP("/custom.php", nil, nil, w, r)
```

## Lifecycle Management

### Shutdown

Cleans up resources used by the middleware.

**Signature:**
```go
func (m *Middleware) Shutdown()
```

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
)
if err != nil {
    log.Fatalf("Failed to initialize Frango: %v", err)
}

// Use the middleware...

// When done, clean up resources
defer php.Shutdown()
```

## Type Definitions

### RenderData

Function type that provides template data to a PHP script.

**Signature:**
```go
type RenderData func(w http.ResponseWriter, r *http.Request) map[string]interface{}
```

**Example:**
```go
renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "user": getCurrentUser(r),
        "items": getItems(),
        "timestamp": time.Now().Unix(),
    }
}

http.Handle("/dashboard", php.Render("/dashboard.php", renderFn))
```

## Common Patterns

### Sharing Data Between Route Handlers

```go
// Create a shared VFS
sharedVFS := php.NewVFS()

// Add shared data
sharedData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "siteTitle": "My Application",
        "version": "1.0.0",
    }
}

// Use in multiple routes
http.Handle("/page1", php.RenderVFS(sharedVFS, "/page1.php", sharedData))
http.Handle("/page2", php.RenderVFS(sharedVFS, "/page2.php", sharedData))
```

### Dynamic Template Generation

```go
// Generate PHP file with custom template
func generateTemplate(title string, content string) []byte {
    return []byte(fmt.Sprintf(`<?php
    $title = "%s";
    $content = "%s";
    include "/templates/layout.php";
    ?>`, title, content))
}

// Create dynamic page handler
http.HandleFunc("/dynamic-page/", func(w http.ResponseWriter, r *http.Request) {
    // Extract page name from URL
    pageName := strings.TrimPrefix(r.URL.Path, "/dynamic-page/")
    
    // Create a branch VFS for this request
    pageVFS := php.NewVFS()
    
    // Generate template for this page
    pageContent := generateTemplate(
        "Page: " + pageName,
        "This is the content for " + pageName,
    )
    
    // Add to the VFS
    pageVFS.CreateVirtualFile("/page.php", pageContent)
    
    // Render using this VFS
    php.ForVFS(pageVFS, "/page.php").ServeHTTP(w, r)
})
``` 