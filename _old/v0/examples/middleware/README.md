# PHP Middleware Example

This example demonstrates how to use Frango as middleware in a standard Go HTTP server.

## Features

- Using Frango as middleware in standard Go HTTP server
- Multiple middleware patterns for different paths
- Serving static files alongside PHP
- Using middleware with existing Go handlers

## Directory Structure

```
middleware/
  ├── main.go         # Go server with middleware configuration
  ├── static/         # Static files directory
  │   ├── css/        # CSS stylesheets
  │   └── js/         # JavaScript files
  └── web/            # PHP files directory
      ├── index.php   # Main PHP file
      └── about.php   # About page
```

## How it Works

This example shows several ways to use Frango as middleware:

1. **Direct handler** - Using the middleware directly for PHP files
2. **Wrapper middleware** - Using `Wrap()` to fall back to a Go handler
3. **Path-specific middleware** - Applying PHP middleware only to specific URL paths
4. **Prefix stripping** - Serving PHP from a subdirectory with prefix stripping

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/middleware
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
// Create the PHP middleware
php, err := frango.New(
    frango.WithSourceDir("web"),
    frango.WithDevelopmentMode(true),
)

// Create a standard HTTP mux
mux := http.NewServeMux()

// 1. Direct middleware for a specific path prefix
mux.Handle("/php/", http.StripPrefix("/php", php))

// 2. PHP middleware with fallback to next handler
notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("<h1>Not Found</h1><p>The requested page was not found.</p>"))
})
mux.Handle("/api/", php.Wrap(notFoundHandler))

// 3. Standard Go handler without PHP
mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
    // Strip /static/ prefix for filesystem lookup
    http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
})

// 4. PHP middleware directly as the root handler
mux.Handle("/", php)

// Start the server with the mux
http.ListenAndServe(":8082", mux)
```

## Middleware Patterns

### 1. Direct Use as Handler

```go
mux.Handle("/", php)
```

This uses the PHP middleware directly to handle all requests at the root path.

### 2. Middleware with Fallback

```go
mux.Handle("/api/", php.Wrap(fallbackHandler))
```

The PHP middleware tries to handle the request, but if no PHP file is found, it passes the request to the fallback handler.

### 3. Prefix Stripping

```go
mux.Handle("/php/", http.StripPrefix("/php", php))
```

This serves PHP files at the `/php/` path but strips this prefix when looking up the files. For example, a request to `/php/info.php` will look for `info.php` in the source directory. 