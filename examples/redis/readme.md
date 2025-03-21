# GoPHP Redis Example

This example demonstrates how to use Redis with PHP in a Go application using the go-php library. It uses a pure PHP Redis client implementation to avoid extension compatibility issues.

## Prerequisites

1. PHP 8.2+ built with ZTS (thread safety) enabled as described in the main README
2. Redis server installed and running on localhost:6379

## About This Example

This example includes a pure PHP Redis client implementation (`SimpleRedis.php`) that doesn't require the Redis extension. SimpleRedis was created to work around compatibility issues when trying to compile the Redis extension with a custom PHP build.

The example demonstrates:
- Basic Redis key-value operations
- Using Redis as a counter
- Storing and retrieving data
- Building a REST API with Redis as the backend storage

## Installation

```bash
# Install Redis on macOS
brew install redis
brew services start redis

# Install Redis on Linux
sudo apt-get install redis-server
sudo systemctl start redis-server
```

## Running the Example

```bash
# From the project root directory
CGO_CFLAGS=$(php-config --includes) CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" go run -tags=nowatcher ./examples/redis
```

Then open your browser and navigate to:

- `http://localhost:8082/` - Redis status page with statistics
- `http://localhost:8082/api/redis` - Redis REST API example

## Project Structure

```
examples/redis/
├── main.go                 # Go application entry point
├── www/                    # PHP files directory
│   ├── SimpleRedis.php     # Pure PHP Redis client implementation
│   ├── index.php           # Status page showing Redis connection
│   └── api.php             # REST API using Redis for storage
└── README.md               # This file
```

## Included Examples

### 1. Redis Status Page (index.php)

The main page displays:
- PHP information
- Redis connection status
- Page view counter using Redis
- List of Redis keys used by the application
- Redis server information

### 2. REST API (api.php)

A REST API that uses Redis as a data store, supporting:
- GET: Retrieve all items or a single item by ID
- POST: Create a new item
- PUT: Update an existing item
- DELETE: Delete an item

## API Usage Examples

```bash
# Get all items
curl http://localhost:8082/api/redis

# Create a new item
curl -X POST -H "Content-Type: application/json" -d '{"name":"Test Item","value":42}' http://localhost:8082/api/redis

# Get a specific item (replace 67dcc157eae70 with the actual ID)
curl http://localhost:8082/api/redis?id=67dcc157eae70

# Update an item (replace 67dcc157eae70 with the actual ID)
curl -X PUT -H "Content-Type: application/json" -d '{"name":"Updated Item","value":99}' http://localhost:8082/api/redis?id=67dcc157eae70

# Delete an item (replace 67dcc157eae70 with the actual ID)
curl -X DELETE http://localhost:8082/api/redis?id=67dcc157eae70
```

## About SimpleRedis

The included `SimpleRedis.php` is a minimal Redis client implementation in pure PHP that:

1. Uses direct socket connections to the Redis server
2. Implements the Redis protocol (RESP) for communication
3. Provides basic Redis commands:
   - `set` - Store a value
   - `get` - Retrieve a value
   - `incr` - Increment a counter
   - `exists` - Check if a key exists
   - `keys` - Get matching keys
   - `del` - Delete a key
   - `info` - Get Redis server information

This approach doesn't require the PHP Redis extension and works with any PHP build.

## Troubleshooting

If you encounter any issues:

1. Ensure Redis server is running:
2. 
```bash
redis-cli ping
```
It should return `PONG`.

1. Check connection settings in SimpleRedis.php if your Redis server is not on localhost:6379.
2. If you see "Failed to read from socket" errors, make sure the Redis server is running and accessible.
