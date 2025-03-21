# frango - Integrate PHP with Go using FrankenPHP

A Go library that makes it easy to integrate PHP code with Go applications using FrankenPHP.

> It started as a joke... i think it's not a joke anymore!

<p align="center">
  <img src="./docs/frango.png" width="400" height="400" alt="frango Logo">
</p>

⚠️: work in progress

## ⚠️ IMPORTANT: Prerequisites

Before you begin, make sure you have all the prerequisites below. **The library will not work without properly built PHP.**

- Go 1.21 or later
- PHP 8.2 or later **built with specific flags** (see Building PHP section)
- Required PHP extensions:
  - Redis extension (`pecl install redis`)
  - cURL extension
- GCC and other build tools for compiling FrankenPHP

### Building PHP for FrankenPHP on macOS

FrankenPHP requires PHP to be built as a static library with ZTS (thread safety) enabled. The standard PHP installation from Homebrew won't work.

1. Install required dependencies:
```bash
brew install libiconv bison brotli re2c pkg-config
echo 'export PATH="/opt/homebrew/opt/bison/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc  # Reload shell configuration
```

2. Get PHP source and configure it:
```bash
# Get PHP source
cd ~ && mkdir -p php-build && cd php-build
curl -LO https://www.php.net/distributions/php-8.2.20.tar.gz
tar -xzf php-8.2.20.tar.gz
cd php-8.2.20

# Configure with the correct flags for macOS
# Note: We're explicitly configuring with minimal extensions to avoid dependency issues
./configure \
    --enable-embed=static \
    --enable-zts \
    --disable-zend-signals \
    --disable-opcache-jit \
    --enable-static \
    --enable-shared=no \
    --with-iconv=/opt/homebrew/opt/libiconv/ \
    --without-sqlite3 \
    --without-pdo-sqlite \
    --disable-dom \
    --disable-xml \
    --disable-simplexml \
    --disable-xmlreader \
    --disable-xmlwriter \
    --disable-libxml
```

3. Compile and install PHP:
```bash
make -j"$(getconf _NPROCESSORS_ONLN)"
sudo make install
```

4. Verify the PHP build:
```bash
# Check that the static library was created
ls -la /usr/local/lib/libphp.a

# Check php-config output
php-config --includes
php-config --ldflags
php-config --libs

# The output should include paths to the PHP header files and libraries
```

### Alternative: Building PHP with Official FrankenPHP Method

If the above method doesn't work, try using the exact method from the FrankenPHP repository:

1. Clone the FrankenPHP repository and build PHP from there:
```bash
# Clone the repository
git clone https://github.com/dunglas/frankenphp.git
cd frankenphp

# Build PHP using the provided script (this will handle everything for you)
./install.sh

# The script will download, configure and compile PHP with the correct flags
```

### Running the Application

1. Install Go dependencies:
```bash
go mod tidy
```

2. Run the application with the correct CGO flags:
```bash
# Method 1: Using php-config
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/basic

# Method 2: Explicitly setting the paths (try this if Method 1 fails)
CGO_CFLAGS="-I/usr/local/include/php -I/usr/local/include/php/main -I/usr/local/include/php/Zend -I/usr/local/include/php/TSRM -I/usr/local/include/php/ext" CGO_LDFLAGS="-L/usr/local/lib -lphp" go run -tags=nowatcher ./examples/basic
```

## Important Notes About FrankenPHP

Before using this library, be aware of these FrankenPHP characteristics:

- It doesn't provide a built-in global memory store that persists across multiple PHP requests and workers.
- It runs PHP scripts using Caddy and a worker pool model, which means:
  - Each request is processed independently.
  - PHP memory resets after each request (just like traditional FPM).
  - There is no global memory space shared between requests by default.

## Features

- Serve PHP files directly from Go applications
- Register explicit PHP endpoints for precise URL routing control
- Mix PHP with native Go HTTP handlers in the same application
- Automatic directory resolution for your PHP files
- Support for both development and production modes
- Advanced routing including HTTP method-based routing with path parameters (Go 1.22+)
- Efficient caching for production environments
- Embed PHP files directly in your Go binary
- Use as middleware in existing Go HTTP applications

## Installation

```bash
go get github.com/davidroman0O/frango
```

## Quick Start

```go
package main

import (
	"log"
	"github.com/davidroman0O/frango"
)

func main() {
	// Find web directory with automatic resolution
	webDir, err := frango.ResolveDirectory("web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}

	// Create server with functional options
	server, err := frango.NewServer(
		frango.WithSourceDir(webDir),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Start serving PHP files
	if err := server.ListenAndServe(":8082"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Core Concepts

### Directory Resolution

frango automatically finds your web directory:

```go
webDir, err := frango.ResolveDirectory("web")
```

This will try multiple search strategies:
1. Check if the path exists as-is
2. Look relative to the calling code
3. Check relative to the current working directory

### Running in Development vs Production Mode

```go
// Development mode (default)
server, err := frango.NewServer() // Default is development mode

// Production mode
server, err := frango.NewServer(
    frango.WithDevelopmentMode(false),
    frango.WithCacheDuration(300), // Cache responses for 5 minutes
)
```

### Registering Specific Endpoints

```go
// Register specific PHP endpoints
server.HandlePHP("/api/user", "api/user.php")
server.HandlePHP("/api/items", "api/items.php")

// Register clean URLs (without .php extension)
server.HandlePHP("/about", "about.php")

// Map multiple URLs to the same PHP file
server.HandlePHP("/", "index.php")
server.HandlePHP("/home", "index.php")

// Mix with Go handlers
server.HandleFunc("/api/time", myTimeHandler)
```

### Using Method-Based Routing (Go 1.22+)

```go
// Create a method router
mux := server.CreateMethodRouter()

// Register PHP endpoints with method constraints and path parameters
server.Handle("GET /users", "users_list.php")
server.Handle("POST /users", "users_create.php") 
server.Handle("GET /users/{id}", "user_detail.php")

// Start the server with the router
http.ListenAndServe(":8082", mux)
```

### Using the Render Function to Inject Variables

```go
// Register a render handler
server.HandleRender("/dashboard", "dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    // Return a map of variables to inject into the PHP context
    return map[string]interface{}{
        "user": map[string]interface{}{
            "id":    42,
            "name":  "John Doe",
            "email": "john@example.com",
            "role":  "Administrator",
        },
        "stats": map[string]interface{}{
            "visits": 12435,
            "conversions": 532,
            "revenue": 95432.50,
        },
        "items": []map[string]interface{}{
            {"id": 1, "name": "Product A", "price": 19.99},
            {"id": 2, "name": "Product B", "price": 29.99},
            {"id": 3, "name": "Product C", "price": 39.99},
        },
    }
})
```

In your PHP file, you can access these variables using the helper functions:

```php
<?php
// Include the helper functions
include_once 'render_helper.php';

// Get individual variables with defaults
$user = go_var('user', []);
$stats = go_var('stats', []);
$items = go_var('items', []);

// Or get all variables at once
$allVars = go_vars();
?>

<h1>Dashboard for <?= htmlspecialchars($user['name']) ?></h1>

<div class="stats">
    <p>Total Visits: <?= number_format($stats['visits']) ?></p>
    <p>Conversions: <?= number_format($stats['conversions']) ?></p>
    <p>Revenue: $<?= number_format($stats['revenue'], 2) ?></p>
</div>

<h2>Product List</h2>
<ul>
    <?php foreach ($items as $item): ?>
    <li>
        <?= htmlspecialchars($item['name']) ?> - $<?= number_format($item['price'], 2) ?>
    </li>
    <?php endforeach; ?>
</ul>
```

### Embedding PHP Files

```go
package main

import (
	"embed"
	"github.com/davidroman0O/frango"
)

//go:embed php/index.php
var indexPhp embed.FS

//go:embed php/api/user.php
var userPhp embed.FS

func main() {
	server, err := frango.NewServer()
	defer server.Shutdown()

	// Add PHP files from embed.FS
	indexPath := server.AddPHPFromEmbed("/index.php", indexPhp, "php/index.php")
	userPath := server.AddPHPFromEmbed("/api/user.php", userPhp, "php/api/user.php")
	
	// Register endpoints
	server.HandlePHP("/", indexPath)
	server.HandlePHP("/api/user", userPath)
	
	server.ListenAndServe(":8082")
}
```

### Using as Middleware

```go
// Create a standard HTTP mux
mux := http.NewServeMux()

// Add Go handlers
mux.HandleFunc("/go/hello", myGoHandler)

// Use PHP server as middleware
phpServer, _ := frango.NewServer(options)
mux.Handle("/php/", http.StripPrefix("/php", phpServer))

// Start server with the mux
http.ListenAndServe(":8080", mux)
```

## Examples

The library includes several examples to help you get started:

- **Basic**: Simple PHP endpoint serving with automatic directory resolution
- **Middleware**: Using frango as middleware in an existing Go application
- **Embed**: Embedding PHP files directly in your Go binary
- **Router**: Advanced routing with method-based constraints and path parameters

Run the examples with:

```bash
# Basic example
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/basic

# Run in production mode
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/basic -prod

# Router example with advanced routing
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/router
```

<!-- 
## Troubleshooting

### Common errors

1. **Missing header errors**
   - For `fatal error: 'wtr/watcher-c.h' file not found`: Use the `-tags=nowatcher` flag when building or running
   - For `fatal error: 'php_variables.h' file not found`: PHP development headers are missing or not in the include path

2. **Undefined symbols errors**
   ```
   Undefined symbols for architecture arm64:
     "_compiler_globals", referenced from:...
   ```
   This indicates that PHP wasn't built correctly. Solutions include:
   
   a) Make sure PHP is built with exactly these flags:
   ```
   --enable-embed=static --enable-zts --disable-zend-signals --disable-opcache-jit --enable-static --enable-shared=no
   ```
   
   b) Use the install.sh script from the FrankenPHP repository:
   ```
   git clone https://github.com/dunglas/frankenphp.git
   cd frankenphp
   ./install.sh
   ```
   
   c) Try building FrankenPHP as a standalone binary instead of embedding it:
   ```
   git clone https://github.com/dunglas/frankenphp.git
   cd frankenphp
   make
   ```
 I will do more testing
3. **Simplest alternative approach**
   Instead of trying to embed FrankenPHP in your Go application, consider:
   
   a) Using FrankenPHP as a standalone server:
   ```bash
   # Install FrankenPHP with Homebrew
   brew install dunglas/frankenphp/frankenphp
   
   # Run FrankenPHP with your PHP files
   frankenphp run --config Caddyfile
   
   # In a separate terminal, run your Go API server
   go run api/main.go
   ```
   
   b) Use separate PHP-FPM and Go servers with a reverse proxy in front

4. **PHP extension errors**
   Make sure the required PHP extensions are installed:
   ```bash
   pecl install redis
   ``` -->

## License

MIT 