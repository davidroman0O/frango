# Quick Start Guide

This guide will help you set up Frango and create a simple PHP-enabled Go application.

## Installation

Install Frango using Go modules:

```bash
go get github.com/davidroman0O/frango
```

## Basic Setup

Here's a minimal example to get started with Frango (Go 1.22+ required):

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create a new middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./php"), // Where your PHP files are located
		frango.WithDevelopmentMode(true), // Enable development mode for auto-reloading
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown() // Clean up resources when done

	// Create a simple HTTP mux
	mux := http.NewServeMux()

	// Map routes to PHP scripts
	mux.Handle("/", php.For("index.php"))
	mux.Handle("/about", php.For("about.php"))
	
	// The parameter is used in both the URL pattern
	mux.Handle("GET /users/{id}", php.For("users/{id}.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Create Your First PHP Script

Create a directory named `php` in your project and add the file `index.php`:

```php
<!DOCTYPE html>
<html>
<head>
    <title>Frango Example</title>
</head>
<body>
    <h1>Hello from PHP!</h1>
    <p>The current time is: <?= date('Y-m-d H:i:s') ?></p>
</body>
</html>
```

## Handling Path Parameters

Create a file `php/users/{id}.php` to demonstrate path parameter handling:

```php
<?php

// Access path parameters through the $_PATH superglobal
$userId = $_PATH['id'] ?? 'unknown';

?>
<!DOCTYPE html>
<html>
<head>
    <title>User Profile</title>
</head>
<body>
    <h1>User Profile</h1>
    <p>You are viewing user ID: <?= htmlspecialchars($userId) ?></p>
    
    <p>Available path data:</p>
    <pre><?php var_dump($_PATH); ?></pre>
</body>
</html>
```

## Project Structure

A typical Frango project structure looks like this:

```
myproject/
├── main.go         # Your Go application
├── php/            # PHP source files
│   ├── index.php
│   ├── about.php
│   └── users/
│       └── {id}.php    # Note the parameter in the actual filename
├── go.mod
└── go.sum
```

## Running Your Application

Run your Go application:

```bash
go run main.go
```

Visit http://localhost:8080 in your browser to see your PHP script in action.

For more complex examples and advanced usage, check out the [Core Integration](core-integration.md) guide. 