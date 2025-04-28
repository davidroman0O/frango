# Core Integration Points

This guide covers the key methods and features for integrating PHP with Go using Frango.

## For() Method

The `For()` method creates an HTTP handler that executes a specific PHP script.

```go
// Create a handler for a specific PHP script
userHandler := php.For("users/{id}.php")

// Register with HTTP server with Go 1.22+ pattern matching
// Note: path parameter in both URL pattern and PHP script path
mux.Handle("GET /users/{id}", userHandler)
```

The `For()` method supports path parameters using curly braces syntax:
- `/users/{id}.php` - Extracts the `id` parameter
- `/posts/{category}/{slug}.php` - Extracts both parameters

These parameters are accessible in PHP via the `$_PATH` superglobal:

```php
$userId = $_PATH['id'];
$category = $_PATH['category'];
$slug = $_PATH['slug'];
```

## Render() Method

The `Render()` method passes data from Go to PHP templates. This is useful for injecting dynamic data:

```go
// Create a handler that renders a template with data
dashboardHandler := php.Render("dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "username": "john_doe",
        "items": []string{"Item 1", "Item 2", "Item 3"},
        "count": 42,
        "user": map[string]interface{}{
            "id": 123,
            "email": "john@example.com",
        },
    }
})

// Register with HTTP server
mux.Handle("/dashboard", dashboardHandler)
```

In your PHP file, the data is accessible as direct variables and through the `$_TEMPLATE` superglobal:

```php
<!-- Direct variable access -->
<h1>Welcome, <?= htmlspecialchars($username) ?></h1>

<!-- Access nested data -->
<p>Your email: <?= htmlspecialchars($user['email']) ?></p>

<!-- Iterate over arrays -->
<ul>
    <?php foreach ($items as $item): ?>
        <li><?= htmlspecialchars($item) ?></li>
    <?php endforeach; ?>
</ul>

<!-- Access via $_TEMPLATE superglobal -->
<p>Item count: <?= $_TEMPLATE['count'] ?></p>
```

## NewVFS() Method

The Virtual File System (VFS) manages PHP files from various sources. You can create a new VFS instance:

```go
// Create a new VFS
vfs := php.NewVFS()
defer vfs.Cleanup() // Important: always clean up when done

// Add a file from the filesystem
vfs.AddSourceFile("/path/to/local/file.php", "/virtual/path/file.php")

// Add a directory of PHP files
vfs.AddSourceDirectory("/local/php/files", "/virtual")

// Create a PHP file programmatically
content := []byte("<?php echo 'Generated at runtime'; ?>")
vfs.CreateVirtualFile("/dynamic/script.php", content)
```

For embedding PHP files in your Go binary:

```go
//go:embed php/*.php
var embeddedFiles embed.FS

// Add embedded files to VFS
vfs.AddEmbeddedDirectory(embeddedFiles, "php", "/")
```

## ExecutePHP() Method

For direct execution of PHP scripts:

```go
// Execute a PHP script directly
func handleCustomRequest(w http.ResponseWriter, r *http.Request) {
    // Get or create a VFS
    vfs := php.NewVFS()
    defer vfs.Cleanup()
    
    // Create dynamic PHP script based on request
    script := "<?php\n"
    script += "header('Content-Type: application/json');\n"
    script += "echo json_encode(['timestamp' => time(), 'method' => '" + r.Method + "']);\n"
    
    // Add to VFS
    vfs.CreateVirtualFile("/dynamic/script.php", []byte(script))
    
    // Execute the PHP script
    php.ExecutePHP("/dynamic/script.php", vfs, nil, w, r)
}
```

With data injection:

```go
// Execute PHP with data
func handleTemplatedRequest(w http.ResponseWriter, r *http.Request) {
    vfs := php.NewVFS()
    defer vfs.Cleanup()
    
    // Add template file to VFS
    vfs.AddSourceFile("./templates/user.php", "/template.php")
    
    // Define render data function
    renderData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
        return map[string]interface{}{
            "username": "jane_doe",
            "isAdmin": true,
        }
    }
    
    // Execute with data
    php.ExecutePHP("/template.php", vfs, renderData, w, r)
}
```

## VFS Branching

The VFS supports branching for isolation between requests. This allows you to have shared base files while keeping request-specific files separate:

```
                    ┌───────────────────┐
                    │    Parent VFS     │
                    │ ┌───────────────┐ │
                    │ │ /lib/header.php│ │
                    │ │ /lib/footer.php│ │
                    │ │ /lib/utils.php │ │
                    │ └───────────────┘ │
                    └──────┬─────┬──────┘
                           │     │
            ┌──────────────┘     └───────────────┐
            ▼                                    ▼
┌────────────────────────┐            ┌────────────────────────┐
│     Request 1 VFS      │            │     Request 2 VFS      │
│ ┌────────────────────┐ │            │ ┌────────────────────┐ │
│ │ Parent files:      │ │            │ │ Parent files:      │ │
│ │ /lib/header.php    │ │            │ │ /lib/header.php    │ │
│ │ /lib/footer.php    │ │            │ │ /lib/footer.php    │ │
│ │ /lib/utils.php     │ │            │ │ /lib/utils.php     │ │
│ └────────────────────┘ │            │ └────────────────────┘ │
│ ┌────────────────────┐ │            │ ┌────────────────────┐ │
│ │ Request-specific:  │ │            │ │ Request-specific:  │ │
│ │ /page1.php         │ │            │ │ /page2.php         │ │
│ │ /user/123.php      │ │            │ │ /admin/settings.php│ │
│ └────────────────────┘ │            │ └────────────────────┘ │
└────────────────────────┘            └────────────────────────┘
```

Files in the parent VFS are accessible to all branches, but each branch can have its own files that are isolated from other branches. This is especially useful in concurrent environments where you need to process multiple requests simultaneously.

```go
// Create parent VFS with common files
parentVFS := php.NewVFS()
parentVFS.AddSourceDirectory("./common", "/lib")

// Create a branch for a specific request
requestVFS := parentVFS.Branch()
defer requestVFS.Cleanup()

// Add request-specific files
requestVFS.CreateVirtualFile("/page.php", []byte("<?php include '/lib/header.php'; ?>"))

// Execute using the branch
php.ExecutePHP("/page.php", requestVFS, nil, w, r)
```

## Configuration Options

Frango provides several configuration options:

```go
php, err := frango.New(
    // Set source directory for PHP files
    frango.WithSourceDir("./php"),
    
    // Enable development mode (auto-reloading of changed files)
    frango.WithDevelopmentMode(true),
    
    // Configure temporary directory
    frango.WithTempDir("/tmp/frango"),
    
    // Set custom logger
    frango.WithLogger(customLogger),
    
    // Control whether direct .php URLs are blocked
    frango.WithDirectPHPURLsBlocking(true),
    
    // Set custom error handler
    frango.WithErrorHandler("/error-handler.php"),
    
    // Control error display
    frango.WithErrorDisplay(true),
)
``` 