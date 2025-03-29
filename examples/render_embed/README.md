# Frango Render Embed Example

This example demonstrates how to use the `HandleRenderEmbed` function to render PHP templates with embedded files.

## What This Example Demonstrates

1. Embedding PHP files in your Go binary
2. Using `HandleRenderEmbed` to render PHP templates with dynamic data
3. Passing variables from Go to PHP
4. Accessing those variables in PHP templates

## Files

- `main.go` - Main Go application that embeds PHP files and sets up the server
- `php/dashboard.php` - PHP template for rendering a dashboard
- `php/render_helper.php` - Helper functions for accessing variables in PHP

## How It Works

The example embeds both a PHP template and a helper library directly into the Go binary. When the application starts:

1. The PHP template and helper are embedded using Go's `//go:embed` directive
2. The server registers the embedded helper file so it can be accessed
3. `HandleRenderEmbed` extracts the embedded dashboard template and registers it with a render function
4. The render function dynamically generates data when a request is made
5. The PHP template uses helper functions to access the variables passed from Go

## Key Code

```go
// Embed PHP files
//go:embed php/dashboard.php
var dashboardTemplate embed.FS

//go:embed php/render_helper.php
var helperTemplate embed.FS

// ...

// Register the render_helper.php file first
server.AddPHPFromEmbed("/render_helper.php", helperTemplate, "php/render_helper.php")

// Register the dashboard template with HandleRenderEmbed
server.HandleRenderEmbed("/", dashboardTemplate, "php/dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    // Generate data...
    
    // Return variables to inject
    return map[string]interface{}{
        "title": "Dashboard - Embedded PHP Rendering",
        "user": map[string]interface{}{
            "name":  "John Doe",
            "email": "john@example.com",
            "role":  "Administrator",
        },
        "items": items,
        "stats": stats,
    }
})
```

In PHP, the variables are accessed using the helper functions:

```php
// Get data from Go
$title = go_var('title', 'Dashboard');
$user = go_var('user', []);
$items = go_var('items', []);
$stats = go_var('stats', []);
```

## Running the Example

```bash
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/render_embed
```

Then open http://localhost:8082/ in your browser.

## Production Mode

Run with the `-prod` flag to enable production mode:

```bash
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/render_embed -prod
``` 