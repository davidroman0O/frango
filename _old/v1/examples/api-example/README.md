# API Example

This example demonstrates how to create API endpoints in both Go and PHP, and how they can communicate with each other using Frango.

## What This Example Shows

- Creating JSON API endpoints in Go
- Creating JSON API endpoints in PHP
- Go-to-PHP communication (calling PHP API from Go)
- PHP-to-Go communication (calling Go API from PHP) 
- Path parameter handling in both Go and PHP
- JSON data processing

## Files

- `main.go`: Go HTTP server with API endpoints
- `php/index.php`: Interactive page to test API communication
- `php/api/products.php`: PHP API endpoint for listing products
- `php/api/product-detail.php`: PHP API endpoint for single product details

## API Endpoints

- `/api/php/products` - List all products (PHP implementation)
- `/api/php/products/{id}` - Get a specific product (PHP implementation)
- `/api/go/products` - List all products (Go implementation)
- `/api/go/products/{id}` - Get a specific product (Go implementation)

Additional query parameters:
- `?call_go=true` - When added to PHP endpoints, makes them call the corresponding Go endpoint

## Running the Example

1. Navigate to this directory
2. Run:
   ```bash
   go run main.go
   ```
3. Open a browser and visit http://localhost:8080

## How It Works

1. The Go code sets up API endpoints and serves PHP files
2. PHP files can make requests to Go endpoints using cURL
3. Go code can make requests to PHP endpoints using standard HTTP libraries
4. Both implementations handle JSON data with proper content type headers

## Key Concepts

```go
// Go API endpoint
mux.HandleFunc("GET /api/go/products", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(/* ... */)
})

// PHP API endpoint handled by Frango
mux.Handle("GET /api/php/products/{id}", php.For("api/product-detail.php"))
```

```php
// PHP accessing path parameters
$productId = $_PATH['id'] ?? null;

// PHP making requests to Go
$ch = curl_init();
curl_setopt($ch, CURLOPT_URL, 'http://localhost:8080/api/go/products');
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
$goApiResponse = curl_exec($ch);
curl_close($ch);
``` 