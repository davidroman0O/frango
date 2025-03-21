# Redis Example PHP Files

This directory contains the PHP files for the Redis example:

## SimpleRedis.php

A pure PHP Redis client implementation that doesn't require the Redis extension. This client communicates directly with the Redis server using sockets and implements the Redis protocol (RESP).

### Key Features:
- Direct socket communication with Redis server
- Support for basic Redis commands
- Error handling for connection issues
- Protocol parsing for Redis responses

## index.php

The main status page for the Redis example. It displays:
- Connection status to Redis
- A page view counter (incrementing on each visit)
- Last visit timestamp
- List of Redis keys in use
- Redis server information

## api.php

A REST API that uses Redis as a data store. It supports:
- GET requests to retrieve all items or a specific item by ID
- POST requests to create new items
- PUT requests to update existing items
- DELETE requests to remove items

All data is stored as JSON in Redis with keys prefixed with `gophp_api_`. 