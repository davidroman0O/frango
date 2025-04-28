# Frango Usage Guide

This document provides detailed usage instructions and examples for common use cases with Frango.

## Basic Usage

### Creating a Simple PHP Middleware

The most basic usage of Frango is to create a middleware that serves PHP files from a directory:

```go
package main

import (
	"log"
	"net/http"
	"github.com/davidroman0O/frango"
)

func main() {
	// Find the web directory with PHP files
	webDir, err := frango.ResolveDirectory("web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}

	// Create a new middleware
	php, err := frango.New(
		frango.WithSourceDir(webDir),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Start the server using the PHP middleware
	log.Println("Server starting on :8082")
	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

This will serve PHP files from the "web" directory. By default, Frango will:
- Look for index.php in the root directory for requests to "/"
- Serve any PHP file requested directly
- Handle static files within the web directory

### Registering Specific PHP Endpoints

For more control over URL routing, you can register specific PHP files:

```go
// Register standard endpoints
php.HandlePHP("/api/user", "api/user.php")
php.HandlePHP("/api/items", "api/items.php")

// Map same PHP file to multiple URLs
php.HandlePHP("/api/users", "api/user.php")

// Create clean URLs (without .php extension)
php.HandlePHP("/about", "about.php")

// Map root URL to index.php
php.HandlePHP("/", "index.php")
```

### Mixing Go and PHP Handlers

You can mix Go HTTP handlers with PHP handlers:

```go
// Create a standard HTTP mux
mux := http.NewServeMux()

// Register a Go handler for an API endpoint
mux.HandleFunc("GET /api/time", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `"}`))
})

// Use PHP middleware as a wrapper
handler := php.Wrap(mux)

// Start the server
http.ListenAndServe(":8082", handler)
```

## Advanced Routing

### Method-Based Routing (Go 1.22+)

Frango works with Go 1.22+ pattern-based routing with HTTP methods:

```go
// Create a standard Go HTTP mux
mux := http.NewServeMux()

// Register standard Go endpoints with method patterns
mux.HandleFunc("GET /api/status", statusHandler)
mux.HandleFunc("GET /api/users", getUsersHandler)
mux.HandleFunc("POST /api/users", createUserHandler)

// Create a PHP mux to direct specific paths to PHP
phpMux := http.NewServeMux()
phpMux.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    php.ServeHTTP(w, r)
})

// Register the PHP files
php.HandlePHP("/users/{id}", "user_detail.php")

// Create a combined router
combinedMux := http.NewServeMux()
combinedMux.Handle("/api/", mux)
combinedMux.Handle("/", phpMux)

// Start the server
if err := http.ListenAndServe(":8082", combinedMux); err != nil {
    log.Fatalf("Server error: %v", err)
}
```

### Using Middleware Chain Approach

For a middleware chain approach:

```go
// Create a root mux for your application
mux := http.NewServeMux()

// Register Go handlers
mux.HandleFunc("GET /api/status", statusHandler)

// Wrap mux with PHP middleware for PHP files
handler := php.Wrap(mux)

// Add PHP endpoints
php.HandlePHP("/users", "users.php")
php.HandlePHP("/about", "about.php")

// Start the server with the handler chain
if err := http.ListenAndServe(":8082", handler); err != nil {
    log.Fatalf("Server error: %v", err)
}
```

### Registering an Entire Directory

You can register all PHP files in a directory under a specific URL prefix:

```go
// Register all PHP files in the "pages" directory under the "/pages" URL prefix
if err := php.HandleDir("/pages", "pages"); err != nil {
    log.Fatalf("Error registering pages directory: %v", err)
}

// Register all PHP files in the "api" directory under the "/api" URL prefix
if err := php.HandleDir("/api", "api"); err != nil {
    log.Fatalf("Error registering API directory: %v", err)
}
```

## Data Injection

### Injecting Variables into PHP Templates

You can inject Go variables into PHP templates using the `HandleRender` function:

```go
php.HandleRender("/dashboard", "dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    // Return variables to inject into the PHP context
    return map[string]interface{}{
        "title": "Dashboard - " + time.Now().Format(time.RFC1123),
        "user": map[string]interface{}{
            "id":    42,
            "name":  "John Doe",
            "email": "john@example.com",
            "role":  "Administrator",
        },
        "items": []map[string]interface{}{
            {"id": 1, "name": "Product A", "price": 19.99},
            {"id": 2, "name": "Product B", "price": 29.99},
            {"id": 3, "name": "Product C", "price": 39.99},
        },
    }
})
```

In your PHP file, access these variables using the helper functions:

```php
<?php
// Include the helper functions
include_once 'render_helper.php';

// Get individual variables with defaults
$user = go_var('user', []);
$items = go_var('items', []);

// Or get all variables at once
$allVars = go_vars();
?>

<h1><?= htmlspecialchars($title) ?></h1>
<p>Welcome, <?= htmlspecialchars($user['name']) ?></p>

<h2>Products</h2>
<ul>
    <?php foreach ($items as $item): ?>
        <li><?= htmlspecialchars($item['name']) ?> - $<?= number_format($item['price'], 2) ?></li>
    <?php endforeach; ?>
</ul>
```

## Embedded PHP Files

### Embedding PHP Files in Your Go Binary

You can embed PHP files directly in your Go binary:

```go
package main

import (
	"embed"
	"log"
	"net/http"
	"github.com/davidroman0O/frango"
)

//go:embed php/index.php
var indexPhp embed.FS

//go:embed php/api/*.php
var apiPhp embed.FS

func main() {
	php, err := frango.New()
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Add PHP files from embed.FS
	indexPath := php.AddFromEmbed("/", indexPhp, "php/index.php")
	
	// Add API files (you'll need to handle each file individually)
	php.AddFromEmbed("/api/users", apiPhp, "php/api/users.php")
	php.AddFromEmbed("/api/items", apiPhp, "php/api/items.php")
	
	// Start the server
	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

### Rendering Embedded PHP Templates with Dynamic Data

You can combine embedding and rendering:

```go
//go:embed templates/dashboard.php
var dashboardTemplate embed.FS

// First add the file from embed
targetPath := php.AddFromEmbed("/dashboard", dashboardTemplate, "templates/dashboard.php")

// Then set up the render handler to inject data
php.SetRenderHandler("/dashboard", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Dashboard - " + time.Now().Format(time.RFC1123),
        "user": getCurrentUser(r), // Function to get current user data
        "stats": getSystemStats(), // Function to get system statistics
    }
})
```

## Framework Integration

### Chi Router

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/davidroman0O/frango"
)

func main() {
    // Create PHP middleware
    php, err := frango.New(
        frango.WithSourceDir("web"),
        frango.WithDevelopmentMode(true),
    )
    if err != nil {
        log.Fatalf("Error creating PHP middleware: %v", err)
    }
    defer php.Shutdown()
    
    // Register PHP routes
    php.HandlePHP("/users", "users.php")
    
    // Create a Chi router
    r := chi.NewRouter()
    
    // Register Go routes
    r.Get("/api/status", statusHandler)
    
    // Mount PHP at a specific path
    r.Mount("/php", php)
    
    // Start the server
    http.ListenAndServe(":8080", r)
}
```

### Gin Framework

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/davidroman0O/frango"
)

func main() {
    // Create PHP middleware
    php, err := frango.New(
        frango.WithSourceDir("web"),
        frango.WithDevelopmentMode(true),
    )
    if err != nil {
        log.Fatalf("Error creating PHP middleware: %v", err)
    }
    defer php.Shutdown()
    
    // Register PHP routes
    php.HandlePHP("/users", "users.php")
    
    // Create a Gin router
    g := gin.Default()
    
    // Register Go routes
    g.GET("/api/status", statusHandler)
    
    // Add PHP middleware to a group
    phpGroup := g.Group("/")
    phpGroup.Use(func(c *gin.Context) {
        req := c.Request
        if php.ShouldHandlePHP(req) {
            php.ServeHTTP(c.Writer, req)
            c.Abort()
            return
        }
        c.Next()
    })
    
    // Start the server
    g.Run(":8080")
}
```

## Caching and Performance

### Development Mode

In development mode, PHP files are reloaded on each request:

```go
php, err := frango.New(
    frango.WithSourceDir("web"),
    frango.WithDevelopmentMode(true), // Enable dev mode
)
```

### Production Mode

For production, disable development mode:

```go
php, err := frango.New(
    frango.WithSourceDir("web"),
    frango.WithDevelopmentMode(false), // Disable dev mode for production
)
```

## Path Resolution

Frango includes a helper to find directories:

```go
// Tries multiple strategies to find a directory
webDir, err := frango.ResolveDirectory("web")
if err != nil {
    log.Fatalf("Error finding web directory: %v", err)
}

// Then use the resolved directory
php, err := frango.New(
    frango.WithSourceDir(webDir),
)
``` 