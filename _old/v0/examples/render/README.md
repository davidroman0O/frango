# PHP Template Rendering Example

This example demonstrates how to use Frango's `HandleRender` function to inject dynamic data from Go into PHP templates.

## Features

- Passing dynamic data from Go to PHP templates
- Structured data injection including nested objects and arrays
- Automatic JSON serialization of Go data structures
- Real-time template updates with dynamic data

## Directory Structure

```
render/
  ├── main.go           # Go code with render handler
  └── php/              # PHP files
      └── template.php  # PHP template that receives variables from Go
```

## How it Works

The example demonstrates the data flow between Go and PHP:

1. A Go handler function returns a map of variables
2. Frango converts these variables to JSON and passes them to PHP
3. Inside the PHP template, variables are accessible using helper functions
4. The PHP template renders dynamic HTML using the variables from Go

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/render
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
// Create PHP middleware
php, err := frango.New(
    frango.WithSourceDir(phpDir),
    frango.WithDevelopmentMode(true),
)

// Register a render handler that provides dynamic data
php.HandleRender("/", "template.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    now := time.Now()

    // Return variables to inject into the PHP template
    return map[string]interface{}{
        "title": "Go PHP Render Example - " + now.Format(time.RFC1123),
        "user": map[string]interface{}{
            "name":  "John Doe",
            "email": "john@example.com",
            "role":  "Administrator",
        },
        "items": []map[string]interface{}{
            {
                "name":        "Product 1",
                "description": "This is the first product",
                "price":       19.99,
            },
            {
                "name":        "Product 2",
                "description": "This is the second product",
                "price":       29.99,
            },
        },
    }
})

// Start the server with the PHP middleware
http.ListenAndServe(":8082", php)
```

In the PHP template, access the variables like this:

```php
<?php
// Include helper functions for variable access
include_once 'render_helper.php';

// Get variables from Go
$title = go_var('title', 'Default Title');
$user = go_var('user', []);
$items = go_var('items', []);
?>

<h1><?= htmlspecialchars($title) ?></h1>

<!-- Accessing nested objects -->
<div class="user-info">
    <p>Name: <?= htmlspecialchars($user['name']) ?></p>
    <p>Email: <?= htmlspecialchars($user['email']) ?></p>
</div>

<!-- Iterating through arrays -->
<ul class="items">
    <?php foreach ($items as $item): ?>
        <li>
            <strong><?= htmlspecialchars($item['name']) ?></strong>
            <p><?= htmlspecialchars($item['description']) ?></p>
            <p>Price: $<?= number_format($item['price'], 2) ?></p>
        </li>
    <?php endforeach; ?>
</ul>
```

This pattern allows you to keep your business logic in Go while leveraging PHP for templating and presentation. 