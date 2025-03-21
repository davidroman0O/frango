# PHP Middleware Example

This example demonstrates how to use Go-PHP as middleware in a standard Go HTTP server, allowing you to integrate PHP functionality with existing Go applications.

## Features

- Using Go-PHP as middleware in a standard Go HTTP server
- Mixing Go handlers, PHP files, and static files
- Path-based routing to different handler types
- Using PHP server for specific URL paths
- Fallback handlers for API endpoints

## Directory Structure

```
middleware/
  ├── web/               # PHP files directory
  │   ├── index.php      # Main index page
  │   └── api/           # API endpoints
  │       ├── user.php   # User API
  │       └── items.php  # Items API
  └── static/            # Static files (created at runtime)
      ├── style.css      # Sample CSS file
      └── image.jpg      # Sample image file
```

## How it Works

The example demonstrates several middleware patterns:

1. Using Go-PHP directly for specific routes (`/api/user`, `/api/items`)
2. Using Go-PHP with path prefix (`/php/`)
3. Using Go-PHP as middleware with fallback for missing files (`/api/`)
4. Mixing with static file serving (`/static/`)
5. Using Go handlers for other paths (`/go/hello`, `/go/time`)

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/middleware
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
// Create a standard HTTP mux
mux := http.NewServeMux()

// Add Go handlers
mux.HandleFunc("/go/hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "<h1>Hello from Go handler!</h1>")
})

// Register PHP endpoints directly with the server
server.HandlePHP("/api/user", "api/user.php")
server.HandlePHP("/api/items", "api/items.php")

// Handle PHP content under /php/ path
mux.Handle("/php/", http.StripPrefix("/php", server))

// Static file handling
fileServer := http.FileServer(http.Dir(staticDir))
mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

// For API paths, check PHP first then fall back to Go
apiHandler := server.AsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // This handler is called when no PHP file handles the request
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"error": "Endpoint not found", "path": "%s", "method": "%s"}`,
        r.URL.Path, r.Method)
}))

mux.Handle("/api/", apiHandler)

// Add a root handler using the PHP server directly
mux.Handle("/", server)
``` 