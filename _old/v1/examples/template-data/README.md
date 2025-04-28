# Template Data Example

This example demonstrates how to pass data from Go to PHP templates using Frango's `Render()` method, which allows you to combine Go's data processing capabilities with PHP's templating features.

## What This Example Shows

- Passing different data types from Go to PHP templates
- Accessing template data in PHP using variables and the `$_TEMPLATE` superglobal
- Building complex nested data structures in Go and using them in PHP
- Creating dynamic dashboards with data prepared in Go

## Files

- `main.go`: Go HTTP server that prepares and sends data to PHP templates
- `php/index.php`: Home page with explanation of template data concepts
- `php/basic.php`: Simple example showing basic template data usage
- `php/dashboard.php`: Complex example with a dashboard UI

## Template Data Types

This example demonstrates passing various data types from Go to PHP:

- Scalar values (strings, numbers, booleans)
- Arrays/slices (converted to PHP indexed arrays)
- Maps (converted to PHP associative arrays)
- Structs (converted to PHP associative arrays)
- Nested structures (combinations of the above)

## Running the Example

1. Navigate to this directory
2. Run:
   ```bash
   go run main.go
   ```
3. Open a browser and visit http://localhost:8080
4. Try both the Basic Example and Dashboard Example

## How It Works

Instead of using `php.For()`, this example uses `php.Render()` with a callback function that returns data to be passed to the PHP template:

```go
mux.Handle("/basic", php.Render("basic.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title":        "Basic Template Example",
        "message":      "This data was passed from Go to PHP!",
        // ... more data
    }
}))
```

In PHP, the data is automatically extracted into local variables for easy access:

```php
// Access template data as local variables
<h1><?= htmlspecialchars($title) ?></h1>
<p><?= htmlspecialchars($message) ?></p>

// Or through the $_TEMPLATE superglobal
<pre><?php var_export($_TEMPLATE); ?></pre>
``` 