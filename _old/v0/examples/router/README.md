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

embedded-php/
  ├── dashboard.php       # Embedded dashboard template
  └── utils.php           # Shared PHP utility library

## How It Works

The example uses Go 1.22's pattern-based routing to handle different HTTP methods and URL patterns:

1. Creates an in-memory store for sample data
2. Registers Go API endpoints for CRUD operations
3. Registers PHP files for the UI components
4. Uses path parameters for dynamic routes
5. Demonstrates embedded PHP templates with dynamic data

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/router
```

Then open your browser to `http://localhost:8082/`

## Key Routing Concepts

### Embedded PHP Templates

The example also demonstrates how to embed PHP templates directly in your Go binary and serve them with dynamic data:

```go
// Embed the PHP dashboard template
//go:embed embedded-php/dashboard.php
var dashboardTemplate embed.FS

// In main function:
// Create a render function that provides dynamic data
dashboardRenderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Dashboard",
        "user": map[string]interface{}{
            "name": "Admin User",
            "role": "Administrator",
        },
        "items": items,  // data from memory store
        "stats": stats,  // calculated statistics
    }
}

// Register the embedded template with the new intuitive method
php.HandleEmbedWithRender("/dashboard", dashboardTemplate, "embedded-php/dashboard.php", dashboardRenderFn)

// Make sure the router knows about this path
combinedMux.Handle("/dashboard", phpMux)
```

This pattern allows you to include PHP templates directly in your binary while still providing them with dynamic data from your Go application.

### Shared PHP Libraries

The example also demonstrates how to embed shared PHP libraries/utility files that can be included in any PHP template:

```go
// Embed the PHP utility library
//go:embed embedded-php/utils.php
var utilsLibrary embed.FS

// In main function:
// Add the utility library so it can be included from any PHP page
php.AddEmbeddedLibrary(utilsLibrary, "embedded-php/utils.php", "/lib/utils.php")
```

Then in PHP templates, you can include this library:

```php
<?php
// Include the utility library
include_once($_SERVER['DOCUMENT_ROOT'] . '/lib/utils.php');

// Now use functions from the library
$formattedPrice = format_currency(19.99);
$truncatedText = truncate($description, 50);
?>
```

This lets you maintain shared PHP functionality in a single place and use it across all your templates.

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