# PHP Embedding Example

This example demonstrates how to embed PHP files directly into your Go binary, eliminating the need for external PHP files.

## Features

- Embedding PHP files directly into Go binary
- No external PHP files needed
- Same functionality as if serving from disk
- Mixing embedded PHP with Go handlers

## How it Works

1. The PHP files are embedded into the Go binary using Go's `embed` package.
2. When the application starts, it:
   - Extracts the embedded PHP files to memory
   - Registers them with the Frango middleware
   - Serves them like regular PHP files

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/embed
```

Then open your browser to `http://localhost:8082/`

## Key Code

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

//go:embed php/api/data.php
var dataPhp embed.FS

func main() {
	// Create a new PHP middleware (no source directory needed)
	php, err := frango.New(
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Add the embedded PHP files
	php.AddFromEmbed("/", indexPhp, "php/index.php")
	php.AddFromEmbed("/api/data", dataPhp, "php/api/data.php")

	// Create a standard HTTP mux for routing
	mux := http.NewServeMux()

	// Register a Go handler
	mux.HandleFunc("GET /api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"time": "` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// Use the PHP middleware for all requests
	handler := php.Wrap(mux)

	// Start the server
	log.Println("Server starting on :8082")
	log.Println("Open http://localhost:8082/ in your browser")
	if err := http.ListenAndServe(":8082", handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Benefits

- **Simplified Deployment**: Package everything in a single binary
- **Enhanced Security**: No PHP files exposed on disk
- **Versioning**: PHP code is versioned along with Go code
- **Simplified Development**: Changes to PHP code are included in Go builds

## Important Note

When embedding PHP files, be aware that:

1. Changes to PHP files require rebuilding the Go binary
2. Runtime PHP file editing is not possible
3. The embedding process extracts files to a temporary directory at runtime 