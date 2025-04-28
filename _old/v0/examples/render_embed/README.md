# Frango Render Embed Example

This example demonstrates how to combine embedded PHP templates with dynamic data from Go using Frango middleware.

## What This Example Demonstrates

1. Embedding PHP files in your Go binary
2. Using embedded PHP templates with dynamic data
3. Passing variables from Go to PHP
4. Accessing those variables in PHP templates

## Files

- `main.go` - Main Go application that embeds PHP files and sets up the middleware
- `php/dashboard.php` - PHP template for rendering a dashboard

## How It Works

The example embeds a PHP template directly into the Go binary. When the application starts:

1. The PHP template is embedded using Go's `//go:embed` directive
2. Frango middleware extracts the embedded template to a temporary location
3. A render function is registered to provide dynamic data
4. When a request arrives, the middleware:
   - Executes the render function to get data
   - Passes the data to the PHP template
   - Renders the complete page with dynamic content

## Key Code

```go
package main

import (
	"embed"
	"log"
	"net/http"
	"time"
	"github.com/davidroman0O/frango"
)

// Embed the dashboard PHP template
//go:embed php/dashboard.php
var dashboardTemplate embed.FS

func main() {
	// Create a new PHP middleware
	php, err := frango.New(
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Create render function to provide dynamic data
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		// Return variables to inject into the PHP template
		return map[string]interface{}{
			"title": "Dashboard - " + time.Now().Format(time.RFC1123),
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"role":  "Administrator",
			},
			"items": []map[string]interface{}{
				{
					"id": 1,
					"name": "Product A",
					"price": 19.99,
				},
				{
					"id": 2,
					"name": "Product B",
					"price": 29.99,
				},
			},
		}
	}

	// SIMPLIFIED VERSION - Use the intuitive HandleEmbedWithRender method
	php.HandleEmbedWithRender("/dashboard", dashboardTemplate, "php/dashboard.php", renderFn)
	
	// Also make it available at the root for convenience
	php.HandleEmbedWithRender("/", dashboardTemplate, "php/dashboard.php", renderFn)

	// Start the server with the PHP middleware
	log.Println("Server starting on :8082")
	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

### Comparison with Old Way

Previously, you had to use multiple method calls for what is conceptually a single operation:

```go
// OLD WAY - Multiple steps
// 1. First add the file from embed
targetPath := php.AddFromEmbed("/dashboard", dashboardTemplate, "php/dashboard.php")

// 2. Then register the render handler separately
php.SetRenderHandler("/dashboard", renderFn)
```

The new `HandleEmbedWithRender` method simplifies this common pattern into a single, intuitive call:

```go 
// NEW WAY - Single intuitive call
php.HandleEmbedWithRender("/dashboard", dashboardTemplate, "php/dashboard.php", renderFn)
```

This significantly improves code readability and maintenance.

In PHP, the variables are accessed using helper functions:

```php
<?php
// Get data from Go
$title = go_var('title', 'Dashboard');
$user = go_var('user', []);
$items = go_var('items', []);

// Use the data
echo "<h1>{$title}</h1>";
echo "<p>Welcome, {$user['name']}!</p>";

// Render items list
foreach ($items as $item) {
    echo "<div>{$item['name']}: \${$item['price']}</div>";
}
?>
```

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/render_embed
```

Then open http://localhost:8082/ in your browser.

> **Important**: The `nowatcher` build tag is required to make this example work properly. 