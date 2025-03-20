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
# Make sure you have built PHP correctly for FrankenPHP (see main README)
cd examples/router

# Run with php-config dynamic flags
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher .

# Or with explicit flags if php-config doesn't work
CGO_CFLAGS="-I/usr/local/include/php -I/usr/local/include/php/main -I/usr/local/include/php/Zend -I/usr/local/include/php/TSRM -I/usr/local/include/php/ext" CGO_LDFLAGS="-L/usr/local/lib -lphp" go run -tags=nowatcher .
```

Then visit http://localhost:8082 in your browser.

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