# Basic Usage Example

This example demonstrates the simplest way to serve PHP files from a Go HTTP server using Frango.

## What This Example Shows

- Basic Frango middleware setup
- Serving PHP files from a directory
- Mapping URL routes to PHP files
- Development mode configuration

## Files

- `main.go`: Go HTTP server with Frango integration
- `php/index.php`: Simple home page
- `php/info.php`: Page displaying PHP information

## Running the Example

1. Make sure you have Go and PHP installed
2. Navigate to this directory
3. Run:
   ```bash
   go run main.go
   ```
4. Open a browser and visit http://localhost:8080

## How It Works

The Go code sets up a simple HTTP server with two routes:
- `/` - Serves `php/index.php`
- `/info` - Serves `php/info.php`

Frango handles the PHP execution and returns the output as HTTP responses.

## Key Concepts

```go
// Create the middleware with configuration options
php, err := frango.New(
    frango.WithSourceDir("./php"),       // Directory containing PHP files
    frango.WithDevelopmentMode(true),    // Enable development mode
)

// Map URL paths to PHP files
mux.Handle("/", php.For("index.php"))
mux.Handle("/info", php.For("info.php"))
``` 