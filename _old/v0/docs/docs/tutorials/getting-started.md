# Getting Started with Frango

This guide will walk you through the process of setting up Frango and creating your first hybrid Go-PHP application.

## Prerequisites

Before you begin, make sure you have:

- Go 1.16 or higher installed
- PHP 7.4 or higher installed
- Basic knowledge of Go and PHP

## Installation

Frango relies on FrankenPHP, a PHP runtime for Go. You'll need to install both Frango and FrankenPHP.

### Step 1: Install FrankenPHP

```bash
go get github.com/dunglas/frankenphp
```

### Step 2: Install Frango

```bash
go get github.com/davidroman0O/frango/v1
```

## Basic Setup

Let's create a simple web application that uses Frango to integrate PHP with Go.

### Step 1: Create Project Structure

Create a new directory for your project:

```bash
mkdir myapp
cd myapp
```

Initialize a Go module:

```bash
go mod init myapp
```

Create the following directory structure:

```
myapp/
├── main.go          # Go application entry point
├── templates/
│   └── hello.php    # PHP template file
```

### Step 2: Create a PHP Template

Create a simple PHP template in `templates/hello.php`:

```php
<!DOCTYPE html>
<html>
<head>
    <title>Hello from Frango</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .greeting {
            background-color: #f0f0f0;
            padding: 20px;
            border-radius: 5px;
        }
    </style>
</head>
<body>
    <h1>Hello from Frango!</h1>
    
    <div class="greeting">
        <p>Current time: <?= date('Y-m-d H:i:s') ?></p>
        <p>PHP version: <?= phpversion() ?></p>
        
        <?php if (isset($name)): ?>
            <p>Hello, <strong><?= htmlspecialchars($name) ?></strong>!</p>
        <?php else: ?>
            <p>Welcome, guest!</p>
        <?php endif; ?>
    </div>
    
    <h2>Request Information</h2>
    <ul>
        <li>URL Path: <?= $_SERVER['REQUEST_URI'] ?></li>
        <li>HTTP Method: <?= $_SERVER['REQUEST_METHOD'] ?></li>
        <li>Query Parameters: <?= count($_GET) ?> parameters</li>
    </ul>
    
    <?php if (count($_GET) > 0): ?>
        <h3>Query Parameters:</h3>
        <ul>
            <?php foreach ($_GET as $key => $value): ?>
                <li><strong><?= htmlspecialchars($key) ?>:</strong> <?= htmlspecialchars($value) ?></li>
            <?php endforeach; ?>
        </ul>
    <?php endif; ?>
</body>
</html>
```

### Step 3: Create the Go Application

Create the main Go application in `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Initialize Frango middleware with source directory for PHP files
	php, err := frango.New(
		frango.WithSourceDir("./templates"),
		frango.WithDevelopmentMode(true),
	)

	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Define routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hello", http.StatusFound)
	})

	// Simple route with PHP template
	http.Handle("/hello", php.For("/hello.php"))

	// Route with path parameter
	http.Handle("/hello/", php.For("/hello.php"))

	// Route with named parameter and data passing
	http.Handle("/greet/", php.Render("/hello.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		// Extract name from URL path
		name := r.URL.Path[len("/greet/"):]
		if name == "" {
			name = "World"
		}

		// Return data to be passed to the PHP template
		return map[string]interface{}{
			"name": name,
		}
	}))

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on http://localhost%s", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, nil))
}
```

### Step 4: Run the Application

Build and run your application:

```bash
go mod tidy
go run main.go
```

Visit `http://localhost:8080` in your browser. You should be redirected to `/hello` and see the PHP template rendered.

Try these different routes:
- `http://localhost:8080/hello?name=John` - Passes a query parameter
- `http://localhost:8080/greet/Alice` - Passes a path parameter

## Understanding What's Happening

Let's break down what's happening in this example:

1. **Middleware Initialization**:
   ```go
   php, err := frango.New(
       frango.WithSourceDir("./templates"),
       frango.WithDevelopmentMode(true),
   )
   ```
   This creates a new Frango middleware instance with the templates directory as the source for PHP files and enables development mode for hot reloading.

2. **Route Definition**:
   ```go
   http.Handle("/hello", php.For("/hello.php"))
   ```
   This maps the `/hello` URL path to the `hello.php` PHP script. When a request comes to this path, Frango will execute the PHP script and return its output.

3. **Data Passing**:
   ```go
   http.Handle("/greet/", php.Render("/hello.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
       name := r.URL.Path[len("/greet/"):]
       return map[string]interface{}{
           "name": name,
       }
   }))
   ```
   The `Render` method allows you to pass data from Go to the PHP script. The data is made available as variables in the PHP context.

4. **PHP Execution**:
   When a request is made, Frango:
   - Locates the PHP file in the VFS (Virtual File System)
   - Prepares the PHP environment with request data
   - Executes the PHP code using FrankenPHP
   - Returns the output as the HTTP response

## Next Steps

Now that you have a basic Frango application running, you might want to:

1. Learn about [Form Handling](./form-handling.md) with PHP and Go
2. Explore [Virtual Filesystem](./virtual-filesystem.md) features for embedding PHP files
3. Understand [Path Parameters](./path-parameters.md) for RESTful route patterns
4. Set up a more complex application with [JSON Processing](./json-processing.md)

For a deeper understanding of Frango's architecture and capabilities, see the [Architecture Overview](../guides/architecture.md). 