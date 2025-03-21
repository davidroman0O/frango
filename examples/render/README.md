# PHP Template Rendering Example

This example demonstrates how to use Go-PHP's `HandleRender` function to inject dynamic data from Go into PHP templates.

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
2. Go-PHP converts these variables to JSON and passes them to PHP
3. Inside the PHP template, variables are accessible using the `$gophp->var('name')` syntax
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
// Register a render handler for the root path
server.HandleRender("/", "template.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
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
```

In the PHP template, access the variables like this:

```php
<h1><?= $gophp->var('title') ?></h1>

<!-- Accessing nested objects -->
<div class="user-info">
    <p>Name: <?= $gophp->var('user')['name'] ?></p>
    <p>Email: <?= $gophp->var('user')['email'] ?></p>
</div>
```

This pattern allows you to keep your business logic in Go while leveraging PHP for templating and presentation. 