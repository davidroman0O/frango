# Go-PHP Router Example

This example demonstrates how to use Go-PHP with Go 1.22's new method-based routing pattern.

## Features

- Method-based routing (`GET`, `POST`, `PUT`, `DELETE`)
- Path parameters (`/users/{id}`)
- Mix of PHP endpoints and native Go handlers
- Interactive web UI to test all endpoints

## File Structure

```
router/
├── main.go              // The Go server implementation with method-based routes
├── web/                 // PHP web directory
│   ├── index.php        // Interactive UI for testing the API
│   └── api/             // API endpoints
│       ├── users_list.php       // GET /users
│       ├── users_create.php     // POST /users
│       ├── user_detail.php      // GET /users/{id}
│       ├── user_update.php      // PUT /users/{id}
│       ├── user_delete.php      // DELETE /users/{id}
│       └── items.php            // Standard endpoint for all methods
```

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/router
```

Then visit http://localhost:8082 in your browser.

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code Patterns

### Go Method-Based Routing (Go 1.22+)

```go
// Create a method-based router
mux := server.CreateMethodRouter()

// Register PHP endpoints with method constraints
server.RegisterPHPEndpoint("GET /users", "api/users_list.php")
server.RegisterPHPEndpoint("GET /users/{id}", "api/user_detail.php")

// Add native Go handlers with path parameters
mux.HandleFunc("GET /api/user/{id}/posts", func(w http.ResponseWriter, r *http.Request) {
    userID := r.PathValue("id") // Access path parameters
    // ...
})
```

### Accessing Path Parameters in PHP

```php
// Extract user ID from the URL path
$path = $_SERVER['REQUEST_URI'];
$pathParts = explode('/', trim($path, '/'));
$userId = $pathParts[count($pathParts) - 1]; // Get the last part
``` 