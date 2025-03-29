# Router Integration Example

This example demonstrates how to use Frango with Go's HTTP router to create a full-featured web application combining both PHP and Go handlers.

## Features

- Method-based routing (GET, POST, PUT, DELETE)
- Path parameters in routes
- In-memory data store for testing
- Mixing Go API handlers with PHP templates
- Different routing patterns for different endpoints

## Directory Structure

```
web/
  ├── index.php           # Main dashboard
  ├── api/
  │   ├── users.php       # Users list page
  │   ├── user.php        # User detail page
  │   ├── items.php       # Items list page
  │   └── item.php        # Item detail page
```

## How It Works

The example uses Go 1.22's pattern-based routing to handle different HTTP methods and URL patterns:

1. Creates an in-memory store for sample data
2. Registers Go API endpoints for CRUD operations
3. Registers PHP files for the UI components
4. Uses path parameters for dynamic routes

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/router
```

Then open your browser to `http://localhost:8082/`

## Key Routing Concepts

### Separate PHP and Go Routes

```go
// Create separate muxes for PHP and Go
phpMux := http.NewServeMux()
apiMux := http.NewServeMux()

// Register PHP paths to be handled by the PHP middleware
phpMux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
    php.ServeHTTP(w, r)
})
phpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    php.ServeHTTP(w, r)
})

// Register API paths with the Go handlers
apiMux.HandleFunc("GET /api/users", getUsersHandler)
apiMux.HandleFunc("POST /api/users", createUserHandler)
apiMux.HandleFunc("GET /api/users/{id}", getUserHandler)

// Register the PHP files with the middleware
php.HandlePHP("/users", "api/users.php")
php.HandlePHP("/users/{id}", "api/user.php")
php.HandlePHP("/", "index.php")

// Create a parent router that combines both
mainMux := http.NewServeMux()
mainMux.Handle("/api/", apiMux)
mainMux.Handle("/", phpMux)

// Start the server with the combined router
http.ListenAndServe(":8082", mainMux)
```

This pattern allows for clean separation of PHP and Go handlers while still having them work together in a unified application. 