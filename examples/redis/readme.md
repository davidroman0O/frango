# Redis Integration Example

This example demonstrates how to use Redis with PHP in a Go application using Frango middleware.

## Features

- PHP with Redis integration
- Session handling with Redis
- Counter with Redis persistence
- Using Redis as a caching layer

## Prerequisites

To run this example, you'll need:

1. Go 1.21 or later
2. PHP 8.2 with Redis extension
3. FrankenPHP compiled with Redis support
4. Redis server running locally

You can check if your PHP has Redis support by running:

```bash
php -m | grep redis
```

## Setting up Redis

Make sure you have Redis installed and running on your system. Most package managers will have Redis available:

```bash
# Debian/Ubuntu
sudo apt install redis-server

# macOS (using Homebrew)
brew install redis
brew services start redis

# Check Redis status
redis-cli ping
```

## Directory Structure

```
redis/
  ├── main.go          # Go application with Frango middleware
  └── www/             # PHP files directory
      ├── index.php    # Main page with Redis usage examples
      ├── counter.php  # Redis counter example
      └── session.php  # Redis session example
```

## How it Works

1. The Go application starts the Redis-enabled PHP middleware
2. PHP files use Redis to store and retrieve data
3. Redis provides persistence and faster data access compared to file-based solutions

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/redis
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
package main

import (
	"log"
	"net/http"
	"github.com/davidroman0O/frango"
)

func main() {
	// Find the web directory with PHP files
	webDir, err := frango.ResolveDirectory("www")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}
	log.Printf("Using web directory: %s", webDir)

	// Create the PHP middleware
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Register all PHP files in the directory
	php.HandleDir("/", "")

	// Start the server
	log.Println("Redis Example starting on :8082")
	log.Println("Open http://localhost:8082/ in your browser")
	if err := http.ListenAndServe(":8082", php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## PHP Redis Usage

Here's a simple example of using Redis in a PHP file:

```php
<?php
// Connect to Redis
$redis = new Redis();
$redis->connect('127.0.0.1', 6379);

// Store a value
$redis->set('example_key', 'Hello from Redis!');

// Retrieve the value
$value = $redis->get('example_key');
echo $value;
?>
```

## Redis Features in PHP

### Key-Value Storage

```php
// Store and retrieve values
$redis->set('user:1:name', 'John Doe');
$name = $redis->get('user:1:name');
```

### Counters

```php
// Increment counters
$redis->incr('page_views');
$views = $redis->get('page_views');
```

### Hash Tables

```php
// Store structured data
$redis->hSet('user:1', 'name', 'John Doe');
$redis->hSet('user:1', 'email', 'john@example.com');

// Get a single field
$name = $redis->hGet('user:1', 'name');

// Get all fields
$user = $redis->hGetAll('user:1');
```

### Session Handling

```php
// In php.ini or with ini_set
ini_set('session.save_handler', 'redis');
ini_set('session.save_path', 'tcp://127.0.0.1:6379');

// Then use sessions normally
session_start();
$_SESSION['user_id'] = 123;
```

### Caching

```php
// Cache expensive operations
$cacheKey = 'expensive_data';
$data = $redis->get($cacheKey);

if (!$data) {
    // Data not in cache, generate it
    $data = generateExpensiveData();
    
    // Store in Redis with expiration (30 seconds)
    $redis->setex($cacheKey, 30, serialize($data));
} else {
    // Data found in cache
    $data = unserialize($data);
}
```
