# JSON Processing in Frango

This tutorial explains how to work with JSON data in Frango applications, enabling seamless JSON communication between Go and PHP.

## Table of Contents

- [Overview](#overview)
- [Setting Up JSON Processing](#setting-up-json-processing)
- [Receiving JSON in PHP](#receiving-json-in-php)
- [Sending JSON from PHP](#sending-json-from-php)
- [Working with JSON in Go](#working-with-json-in-go)
- [Bidirectional JSON Communication](#bidirectional-json-communication)
- [Advanced Techniques](#advanced-techniques)
- [Debugging JSON Issues](#debugging-json-issues)
- [Best Practices](#best-practices)

## Overview

JSON (JavaScript Object Notation) is a lightweight data interchange format that's easy for humans to read and write and easy for machines to parse and generate. In Frango applications, JSON is often used for:

- Building APIs and microservices
- Frontend-backend communication
- Configuration management
- Data exchange between systems

Frango makes it easy to process JSON data in both directions:
- Incoming JSON requests are automatically parsed and made available to PHP scripts
- PHP scripts can easily generate JSON responses that are sent back to clients

## Setting Up JSON Processing

### Project Structure

For this tutorial, we'll use the following project structure:

```
json-api/
├── main.go           # Go server application
└── php-files/        # PHP files directory
    ├── api/
    │   ├── receive.php  # Example for receiving JSON
    │   └── send.php     # Example for sending JSON
    └── includes/
        └── helpers.php  # Helper functions
```

### Go Server Setup

First, let's create a basic Go server with Frango integration for handling JSON:

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango"
)

func main() {
	// Initialize Frango
	php, err := frango.New(
		frango.WithSourceDir("./php-files"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error initializing Frango: %v", err)
	}
	defer php.Shutdown()

	// Create the router
	mux := http.NewServeMux()

	// JSON endpoints
	mux.Handle("POST /api/receive", php.For("api/receive.php"))
	mux.Handle("GET /api/send", php.For("api/send.php"))

	// Start the server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Receiving JSON in PHP

When a client sends a JSON request to a PHP endpoint in Frango, the JSON data is automatically parsed and made available through the `$_JSON` superglobal variable (a Frango-specific feature).

### Example: Receiving JSON

Create a file at `php-files/api/receive.php`:

```php
<?php
// Set the response content type to JSON
header('Content-Type: application/json');

// Check if we received JSON data
if (!isset($_JSON)) {
    // No JSON data found - the request didn't have application/json content type
    // or the body was empty
    http_response_code(400);
    echo json_encode([
        'success' => false,
        'message' => 'No JSON data received'
    ]);
    exit;
}

// Access the JSON data via the $_JSON superglobal
$name = $_JSON['name'] ?? 'Guest';
$email = $_JSON['email'] ?? '';
$preferences = $_JSON['preferences'] ?? [];

// Process the data
$response = [
    'success' => true,
    'message' => 'Data received successfully',
    'received' => [
        'name' => $name,
        'email' => $email,
        'preferences' => $preferences
    ],
    'timestamp' => date('Y-m-d H:i:s')
];

// Send the JSON response
echo json_encode($response);
```

### Testing the Endpoint

You can test the JSON receiving endpoint with curl:

```bash
curl -X POST http://localhost:8080/api/receive \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","preferences":["dark-mode","notifications"]}'
```

## Sending JSON from PHP

PHP can easily generate and send JSON responses.

### Example: Sending JSON

Create a file at `php-files/api/send.php`:

```php
<?php
// Set the response content type to JSON
header('Content-Type: application/json');

// Create some example data
$data = [
    'users' => [
        [
            'id' => 1,
            'name' => 'John Doe',
            'email' => 'john@example.com',
            'active' => true
        ],
        [
            'id' => 2,
            'name' => 'Jane Smith',
            'email' => 'jane@example.com',
            'active' => true
        ],
        [
            'id' => 3,
            'name' => 'Bob Johnson',
            'email' => 'bob@example.com',
            'active' => false
        ]
    ],
    'metadata' => [
        'count' => 3,
        'page' => 1,
        'generated_at' => date('Y-m-d H:i:s')
    ]
];

// Parse any query parameters
$prettyPrint = isset($_GET['pretty']) && $_GET['pretty'] === 'true';

// Output the JSON (with optional pretty printing)
if ($prettyPrint) {
    echo json_encode($data, JSON_PRETTY_PRINT);
} else {
    echo json_encode($data);
}
```

### Testing the Endpoint

You can test the JSON sending endpoint with curl:

```bash
# Regular output
curl -X GET http://localhost:8080/api/send

# Pretty-printed output
curl -X GET "http://localhost:8080/api/send?pretty=true"
```

## Working with JSON in Go

You can process JSON data in Go before passing it to PHP or after receiving it from PHP.

### Preprocessing JSON Before PHP

```go
// This function handles preprocessing JSON before sending to PHP
func preprocessJSON(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Only intercept POST requests with JSON content type
        if r.Method == "POST" && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
            // Read the request body
            body, err := io.ReadAll(r.Body)
            if err != nil {
                http.Error(w, "Error reading request body", http.StatusBadRequest)
                return
            }
            r.Body.Close()
            
            // Parse the JSON
            var data map[string]interface{}
            if err := json.Unmarshal(body, &data); err != nil {
                http.Error(w, "Invalid JSON", http.StatusBadRequest)
                return
            }
            
            // Add additional fields
            data["processed_by"] = "go"
            data["timestamp"] = time.Now().Format(time.RFC3339)
            
            // Create a new body with the modified JSON
            modifiedBody, _ := json.Marshal(data)
            r.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
            r.ContentLength = int64(len(modifiedBody))
        }
        
        next.ServeHTTP(w, r)
    })
}

// Register the handler with preprocessing
mux.Handle("POST /api/process", preprocessJSON(php.For("api/receive.php")))
```

### Working with JSON Responses in Go

```go
// Create a handler function
func handleJSONResponse(w http.ResponseWriter, r *http.Request) {
    // Create a response recorder
    recorder := httptest.NewRecorder()
    
    // Execute the PHP handler
    php.For("api/send.php").ServeHTTP(recorder, r)
    
    // Get the response
    response := recorder.Result()
    body, _ := io.ReadAll(response.Body)
    response.Body.Close()
    
    // Check if it's JSON
    if response.Header.Get("Content-Type") == "application/json" {
        // Parse the JSON
        var data map[string]interface{}
        if err := json.Unmarshal(body, &data); err != nil {
            // Pass through the response if it can't be parsed
            copyResponseToWriter(response, body, w)
            return
        }
        
        // Modify the JSON if needed
        if metadata, ok := data["metadata"].(map[string]interface{}); ok {
            metadata["processed_by"] = "go"
            data["metadata"] = metadata
        }
        
        // Write the modified response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(response.StatusCode)
        json.NewEncoder(w).Encode(data)
    } else {
        // Pass through the response as-is
        copyResponseToWriter(response, body, w)
    }
}

// Helper to copy response details
func copyResponseToWriter(resp *http.Response, body []byte, w http.ResponseWriter) {
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    w.WriteHeader(resp.StatusCode)
    w.Write(body)
}

// Register the handler
mux.HandleFunc("GET /api/processed", handleJSONResponse)
```

## Bidirectional JSON Communication

In more complex scenarios, you might need to pass JSON data from Go to PHP and then receive a JSON response back.

### Creating Helper Functions

First, let's create some PHP helper functions in `php-files/includes/helpers.php`:

```php
<?php
/**
 * Helper functions for JSON processing
 */

/**
 * Send a JSON response with the given data and status code
 */
function sendJsonResponse($data, $statusCode = 200) {
    http_response_code($statusCode);
    header('Content-Type: application/json');
    echo json_encode($data);
    exit;
}

/**
 * Get JSON data from the request body
 * Frango automatically parses JSON into $_JSON, but this is a fallback
 */
function getJsonInput() {
    if (isset($_JSON)) {
        return $_JSON;
    }
    
    $input = file_get_contents('php://input');
    if (empty($input)) {
        return null;
    }
    
    return json_decode($input, true);
}
```

### Implementing a Bidirectional API

Next, create a PHP file that will handle a bidirectional API request:

```php
<?php
// Include the helpers
require_once __DIR__ . '/../includes/helpers.php';

// Get the JSON input
$data = isset($_JSON) ? $_JSON : getJsonInput();

if (!$data) {
    sendJsonResponse([
        'success' => false,
        'message' => 'No input data provided'
    ], 400);
}

// Process the input
$command = $data['command'] ?? '';
$params = $data['params'] ?? [];

// Execute the command
$result = [];
switch ($command) {
    case 'echo':
        $result = $params;
        break;
        
    case 'calculate':
        $operation = $params['operation'] ?? '';
        $a = $params['a'] ?? 0;
        $b = $params['b'] ?? 0;
        
        switch ($operation) {
            case 'add':
                $result = ['result' => $a + $b];
                break;
            case 'subtract':
                $result = ['result' => $a - $b];
                break;
            case 'multiply':
                $result = ['result' => $a * $b];
                break;
            case 'divide':
                if ($b == 0) {
                    sendJsonResponse([
                        'success' => false,
                        'message' => 'Division by zero'
                    ], 400);
                }
                $result = ['result' => $a / $b];
                break;
            default:
                sendJsonResponse([
                    'success' => false,
                    'message' => 'Unknown operation: ' . $operation
                ], 400);
        }
        break;
        
    default:
        sendJsonResponse([
            'success' => false,
            'message' => 'Unknown command: ' . $command
        ], 400);
}

// Send the response
sendJsonResponse([
    'success' => true,
    'command' => $command,
    'result' => $result,
    'processed_at' => date('Y-m-d H:i:s')
]);
```

## Advanced Techniques

### Handling Complex JSON Structures

PHP's associative arrays map directly to JSON objects, but sometimes you might need to work with more complex structures.

#### Nested Objects and Arrays

```php
<?php
// Create a complex nested structure
$response = [
    'users' => [
        [
            'id' => 1,
            'name' => 'John Doe',
            'contact' => [
                'email' => 'john@example.com',
                'phone' => '555-1234'
            ],
            'roles' => ['admin', 'editor']
        ],
        [
            'id' => 2,
            'name' => 'Jane Smith',
            'contact' => [
                'email' => 'jane@example.com',
                'phone' => '555-5678'
            ],
            'roles' => ['user']
        ]
    ],
    'pagination' => [
        'page' => 1,
        'per_page' => 10,
        'total' => 2
    ]
];

header('Content-Type: application/json');
echo json_encode($response);
```

#### Handling JSON Encoding Options

PHP provides several options for JSON encoding:

```php
<?php
$data = [
    'name' => 'John Doe',
    'url' => 'https://example.com/users/john',
    'tags' => ['php', 'json', 'api']
];

// Basic encoding
$basic = json_encode($data);

// Pretty-printed JSON
$pretty = json_encode($data, JSON_PRETTY_PRINT);

// Escape characters (HTML-safe)
$escaped = json_encode($data, JSON_HEX_TAG | JSON_HEX_AMP | JSON_HEX_APOS | JSON_HEX_QUOT);

// Include non-UTF8 characters
$nonUtf8 = json_encode($data, JSON_UNESCAPED_UNICODE);

header('Content-Type: application/json');
echo json_encode([
    'basic' => $basic,
    'pretty' => $pretty,
    'escaped' => $escaped,
    'nonUtf8' => $nonUtf8
]);
```

### JSON Schema Validation

For more robust applications, you might want to validate JSON against a schema:

```php
<?php
function validateJson($data, $schema) {
    // This is a very simplified validator
    // In a real app, use a library like justinrainbow/json-schema
    
    foreach ($schema as $field => $rules) {
        // Check required fields
        if (isset($rules['required']) && $rules['required']) {
            if (!isset($data[$field])) {
                return [false, "Field '$field' is required"];
            }
        }
        
        // Skip validation if field is not present and not required
        if (!isset($data[$field])) {
            continue;
        }
        
        // Validate type
        if (isset($rules['type'])) {
            $value = $data[$field];
            $type = $rules['type'];
            
            $valid = false;
            switch ($type) {
                case 'string':
                    $valid = is_string($value);
                    break;
                case 'number':
                    $valid = is_numeric($value);
                    break;
                case 'boolean':
                    $valid = is_bool($value);
                    break;
                case 'array':
                    $valid = is_array($value);
                    break;
                case 'object':
                    $valid = is_array($value) && array_keys($value) !== range(0, count($value) - 1);
                    break;
            }
            
            if (!$valid) {
                return [false, "Field '$field' must be of type $type"];
            }
        }
    }
    
    return [true, "Validation successful"];
}

// Example usage
$schema = [
    'name' => ['type' => 'string', 'required' => true],
    'age' => ['type' => 'number', 'required' => true],
    'email' => ['type' => 'string', 'required' => true],
    'preferences' => ['type' => 'array', 'required' => false]
];

$data = $_JSON ?? [];
list($valid, $message) = validateJson($data, $schema);

if (!$valid) {
    header('Content-Type: application/json');
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => $message]);
    exit;
}

// Process valid data
// ...
```

## Debugging JSON Issues

### Common JSON Problems

1. **Invalid JSON Format**: Make sure the JSON data is properly formatted.
2. **Encoding Issues**: Ensure proper character encoding (UTF-8 is recommended).
3. **Missing Content Type**: Set the Content-Type header to "application/json".
4. **PHP Errors in JSON**: Any PHP errors/warnings will break the JSON output.

### Debugging Tips

#### Log Raw Input and Output

```php
<?php
// Log the raw input
$input = file_get_contents('php://input');
error_log('Raw input: ' . $input);

// Log the parsed input
error_log('Parsed JSON: ' . print_r($_JSON, true));

// Prepare response
$response = ['success' => true];

// Log the response before sending
error_log('Response: ' . json_encode($response));

// Send response
header('Content-Type: application/json');
echo json_encode($response);
```

#### Use JSON Pretty Print for Debugging

```php
<?php
// For debugging, use JSON_PRETTY_PRINT
if ($_GET['debug'] ?? false) {
    header('Content-Type: application/json');
    echo json_encode($data, JSON_PRETTY_PRINT);
    exit;
}

// Otherwise use compact JSON
header('Content-Type: application/json');
echo json_encode($data);
```

## Best Practices

### Security Considerations

1. **Input Validation**: Always validate and sanitize JSON input.
2. **Avoid Sensitive Data**: Be careful about what information you expose in JSON responses.
3. **Rate Limiting**: Implement rate limiting for API endpoints.
4. **Error Handling**: Return proper error messages without exposing internal details.

### Performance Optimization

1. **Minimize JSON Size**: Keep JSON responses compact by including only necessary data.
2. **Compression**: Enable HTTP compression for large responses.
3. **Caching**: Implement appropriate caching headers for JSON endpoints.

### API Design

1. **Consistent Structure**: Maintain a consistent structure across all JSON responses.
2. **Use HTTP Status Codes**: Return appropriate HTTP status codes with your JSON responses.
3. **Versioning**: Consider versioning your API for long-term compatibility.

### Example: A Well-Structured JSON API Response

```php
<?php
// Create a well-structured API response
function apiResponse($success, $data = null, $message = null, $statusCode = 200) {
    http_response_code($statusCode);
    header('Content-Type: application/json');
    
    $response = [
        'success' => $success,
        'timestamp' => date('c')
    ];
    
    if ($message !== null) {
        $response['message'] = $message;
    }
    
    if ($data !== null) {
        $response['data'] = $data;
    }
    
    echo json_encode($response);
    exit;
}

// Example usage
try {
    // Process request...
    $result = [/* ... */];
    
    // Return success response
    apiResponse(true, $result);
} catch (Exception $e) {
    // Return error response
    apiResponse(false, null, $e->getMessage(), 500);
}
```

By following these patterns and best practices, you can effectively work with JSON data in your Frango applications, creating robust APIs that leverage the strengths of both Go and PHP. 