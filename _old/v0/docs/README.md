# Frango Documentation

Frango is a Go library that makes it easy to integrate PHP code with Go applications using FrankenPHP.

## Overview

Frango allows you to:

- Serve PHP files directly from Go applications
- Register explicit PHP endpoints for precise URL routing control
- Mix PHP with native Go HTTP handlers in the same application
- Embed PHP files directly in your Go binary
- Use PHP as middleware in existing Go applications

## Documentation Index

- [**API Reference**](api-reference.md) - Detailed reference of all Frango functions, types, and methods
- [**Usage Guide**](usage.md) - Practical examples and instructions for common use cases
- [**Advanced Usage**](advanced.md) - Advanced patterns and techniques for complex scenarios

## Quick Start

Here's a simple example to get started:

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

	// Start the server using the PHP middleware directly
	log.Println("Server starting on :8082")
	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Core Concepts

### PHP Middleware

The `Middleware` type is the central component that manages PHP execution environments and serves HTTP requests. You create a middleware with `New()` and configure it using option functions.

### Routing

Frango provides multiple ways to route requests to PHP files:

- **Direct Mapping**: `HandlePHP("/path", "file.php")`
- **Method-Based Routing**: `Handle("GET /users/{id}", "user.php")`
- **Directory Mapping**: `HandleDir("/api", "api")`

### Middleware Integration

You can use Frango as middleware in existing Go applications:

```go
mux := http.NewServeMux()
mux.Handle("/php/", http.StripPrefix("/php", php))
mux.Handle("/api/", php.Wrap(fallbackHandler))
```

### Data Injection

Inject Go variables into PHP templates:

```go
php.HandleRender("/dashboard", "dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Dashboard",
        "user": getCurrentUser(r),
    }
})
```

## Prerequisites

Before using Frango, make sure you have:

- Go 1.21 or later
- PHP 8.2 or later (built with specific flags for FrankenPHP)
- Required PHP extensions (Redis, cURL)

For more details on requirements and setup, see the project's [README](../README.md). 