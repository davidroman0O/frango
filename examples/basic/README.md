# Basic PHP Integration Example

This example demonstrates how to integrate PHP with Go using Go-PHP.

## Features

- Basic PHP file serving
- Custom URL routing for PHP files
- Mixing Go handlers with PHP files
- Clean URLs without .php extensions
- Multiple URL paths for the same PHP file

## Directory Structure

```
web/
  ├── index.php       # Main index page
  ├── about.php       # About page
  ├── api/
  │   ├── user.php    # User API endpoint
  │   └── items.php   # Items API endpoint
```

## How it Works

The example shows the most common usage patterns:

1. Setting up a Go-PHP server with a source directory
2. Registering PHP files with specific URL patterns
3. Adding Go handlers alongside PHP endpoints
4. Creating clean URLs without .php extensions

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/basic
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
// Standard endpoints
server.HandlePHP("/api/user", "api/user.php")
server.HandlePHP("/api/items", "api/items.php")

// You can map the same PHP file to multiple URL paths
server.HandlePHP("/api/users", "api/user.php") // Alias for the same file

// You can register URLs with or without .php extension
server.HandlePHP("/about", "about.php")     // Clean URL without .php
server.HandlePHP("/about.php", "about.php") // Traditional URL with .php

// Create clean URLs for index pages
server.HandlePHP("/", "index.php") // Root maps to index.php

// Register a custom Go handler
server.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `"}`))
})
``` 