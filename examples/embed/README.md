# Embedded PHP Files Example

This example demonstrates how to embed PHP files directly into your Go binary using Go's `embed` package, eliminating the need for separate PHP files at runtime.

## Features

- Embedding PHP files directly into Go binary
- No need for external PHP files at runtime
- Adding embedded PHP files to the server
- Registering endpoints for embedded files
- Clean URLs for better user experience

## How it Works

The example uses Go's `embed` package to include PHP files in the compiled binary:

1. PHP files are embedded using the `//go:embed` directive
2. `AddPHPFromEmbed` extracts the files to a temporary directory
3. `HandlePHP` registers URL patterns for the extracted files
4. Multiple URL patterns can point to the same PHP file

## Directory Structure

```
embed/
  ├── main.go           # Go code with embedded PHP files
  └── php/              # Original PHP files (embedded in binary)
      ├── index.php     # Main index page
      └── api/          # API endpoints
          ├── user.php  # User API
          └── items.php # Items API
```

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/embed
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
// Embed the PHP files directly
//go:embed php/index.php
var indexPhp embed.FS

//go:embed php/api/user.php
var userPhp embed.FS

//go:embed php/api/items.php
var itemsPhp embed.FS

// Add the PHP files from embedded filesystem - simple and direct
// Create files without registering endpoints
indexPath := server.AddPHPFromEmbed("/index.php", indexPhp, "php/index.php")
userPath := server.AddPHPFromEmbed("/api/user.php", userPhp, "php/api/user.php")
itemsPath := server.AddPHPFromEmbed("/api/items.php", itemsPhp, "php/api/items.php")

// Now explicitly register the endpoints
server.HandlePHP("/", indexPath)          // Root path
server.HandlePHP("/index", indexPath)     // Without .php extension
server.HandlePHP("/index.php", indexPath) // With .php extension
```

This approach allows you to distribute a single binary containing both your Go application and PHP files, making deployment simpler. 