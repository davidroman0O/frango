# Request and Response Handling API Reference

This document provides a comprehensive reference for Frango's request and response handling API, which enables seamless integration between Go's HTTP handling and PHP's request/response model.

## Table of Contents

- [Overview](#overview)
- [Request Handling](#request-handling)
- [Response Handling](#response-handling)
- [Request Transformation](#request-transformation)
- [Response Transformation](#response-transformation)
- [Middleware Integration](#middleware-integration)
- [Advanced Patterns](#advanced-patterns)

## Overview

Frango acts as a bridge between Go's HTTP handling and PHP's request/response model. It converts Go's `http.Request` objects into PHP superglobals (`$_GET`, `$_POST`, etc.) and transforms PHP's output into Go's `http.ResponseWriter`.

This bidirectional conversion allows PHP scripts to behave as if they were running in a traditional PHP environment while actually being executed within a Go application.

## Request Handling

### Basic Request Forwarding

The most basic way to handle requests in Frango is to forward them directly to a PHP script:

```go
// Forward requests to index.php
mux := http.NewServeMux()
mux.Handle("/", php.For("index.php"))

// Forward requests to a specific script
mux.Handle("/users", php.For("api/users.php"))
```

### Route Parameters

You can capture route parameters and make them available to your PHP scripts:

```go
// Using gorilla/mux for route parameters
router := mux.NewRouter()
router.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    // Frango will automatically extract path parameters from gorilla/mux
    php.For("user_profile.php").ServeHTTP(w, r)
})
```

In PHP, you can access these path parameters through the `$_PATH` superglobal:

```php
<?php
// Get path parameters
$userId = $_PATH['id'] ?? null;

if ($userId === null) {
    http_response_code(400);
    echo json_encode(['error' => 'User ID is required']);
    exit;
}

// Use the user ID
echo "Showing profile for user: " . htmlspecialchars($userId);
```

### Request Method Handling

Frango automatically populates the `$_SERVER['REQUEST_METHOD']` variable in PHP, allowing scripts to handle different HTTP methods:

```php
<?php
switch ($_SERVER['REQUEST_METHOD']) {
    case 'GET':
        // Handle GET request
        break;
    case 'POST':
        // Handle POST request
        break;
    case 'PUT':
        // Handle PUT request
        break;
    case 'DELETE':
        // Handle DELETE request
        break;
    default:
        http_response_code(405);
        echo "Method not allowed";
        break;
}
```

### Query Parameters

Query parameters from the URL are automatically populated in PHP's `$_GET` superglobal:

```go
// Go: Request to /search?q=frango&page=1
mux.Handle("/search", php.For("search.php"))
```

```php
<?php
// PHP: Access query parameters
$query = $_GET['q'] ?? '';       // "frango"
$page = (int)($_GET['page'] ?? 1);     // 1
```

### Form Data

Form submissions (with `application/x-www-form-urlencoded` or `multipart/form-data` content types) are automatically populated in PHP's `$_POST` superglobal:

```php
<?php
// For a form submission with fields "name" and "email"
$name = $_POST['name'] ?? '';
$email = $_POST['email'] ?? '';

// Validate inputs
if (empty($name) || empty($email)) {
    echo "Both name and email are required.";
    exit;
}
```

### File Uploads

File uploads are handled through PHP's `$_FILES` superglobal:

```php
<?php
// For a file upload with field name "document"
if (isset($_FILES['document']) && $_FILES['document']['error'] === UPLOAD_ERR_OK) {
    $uploadedFile = $_FILES['document'];

    $name = $uploadedFile['name'];       // Original filename
    $type = $uploadedFile['type'];       // MIME type
    $tmpName = $uploadedFile['tmp_name']; // Temporary file path
    $error = $uploadedFile['error'];     // Upload error code
    $size = $uploadedFile['size'];       // File size in bytes

    // Verify it's a safe file type
    $allowedTypes = ['image/jpeg', 'image/png', 'application/pdf'];
    if (!in_array($type, $allowedTypes)) {
        echo "File type not allowed.";
        exit;
    }

    // Move the uploaded file to a permanent location
    $newFilename = md5(uniqid() . $name) . '.' . pathinfo($name, PATHINFO_EXTENSION);
    $uploadDir = '/uploads/';
    
    if (move_uploaded_file($tmpName, $uploadDir . $newFilename)) {
        echo "File uploaded successfully.";
    } else {
        echo "Error uploading file.";
    }
}
```

### JSON Request Body

Frango provides several ways to handle JSON request bodies:

1. Raw access through `php://input`:

```php
<?php
// Get the raw POST data
$jsonData = file_get_contents('php://input');

// Parse it as JSON
$data = json_decode($jsonData, true);

if ($data === null && json_last_error() !== JSON_ERROR_NONE) {
    http_response_code(400);
    echo json_encode(['error' => 'Invalid JSON: ' . json_last_error_msg()]);
    exit;
}

// Now you can use the data
$name = $data['name'] ?? '';
$email = $data['email'] ?? '';

// Process the data...
```

2. Using the PHP globals script that handles JSON input automatically:

```php
<?php
// Check if the JSON was parsed automatically (depends on PHP globals script)
if (isset($_JSON)) {
    $name = $_JSON['name'] ?? '';
    $email = $_JSON['email'] ?? '';
} else {
    // Fallback to manual parsing
    $jsonData = file_get_contents('php://input');
    $data = json_decode($jsonData, true);
    // ...
}
```

### Cookies

Cookies from the request are automatically populated in PHP's `$_COOKIE` superglobal:

```php
<?php
// Access cookies
$sessionID = $_COOKIE['session_id'] ?? null;
$theme = $_COOKIE['theme'] ?? 'light';
```

### Headers

Request headers are available in PHP through `getallheaders()` or the `$_SERVER` superglobal:

```php
<?php
// Using getallheaders()
$headers = getallheaders();
$userAgent = $headers['User-Agent'] ?? '';

// Using $_SERVER
$userAgent = $_SERVER['HTTP_USER_AGENT'] ?? '';
$contentType = $_SERVER['CONTENT_TYPE'] ?? '';
```

## Response Handling

### Basic Response

By default, Frango captures the output from the PHP script and sends it as the HTTP response:

```php
<?php
// This content will be sent to the client
echo "Hello, World!";
```

### Setting Status Code

PHP scripts can set the HTTP status code using the `http_response_code()` function:

```php
<?php
// Set a 404 Not Found status
http_response_code(404);
echo "Page not found";
```

### Setting Headers

PHP scripts can set response headers using the `header()` function:

```php
<?php
// Set content type to JSON
header('Content-Type: application/json');

// Set a custom header
header('X-Custom-Header: SomeValue');

// Redirect to another URL
header('Location: /new-page.php');
exit;
```

### JSON Responses

A common pattern for API endpoints is returning JSON responses:

```php
<?php
// Get the user ID from the request
$userId = $_GET['id'] ?? 0;

// Set the content type to JSON
header('Content-Type: application/json');

// Check if the user ID is valid
if ($userId <= 0) {
    http_response_code(400);
    echo json_encode([
        'error' => 'Invalid user ID',
    ]);
    exit;
}

// Get the user data
$user = getUserById($userId);

// Check if the user exists
if ($user === null) {
    http_response_code(404);
    echo json_encode([
        'error' => 'User not found',
    ]);
    exit;
}

// Return the user data as JSON
echo json_encode([
    'user' => $user,
]);
```

### Binary Responses

PHP scripts can send binary data as responses:

```php
<?php
// Generate a PDF
$pdf = generatePDF();

// Set appropriate headers
header('Content-Type: application/pdf');
header('Content-Disposition: attachment; filename="document.pdf"');
header('Content-Length: ' . strlen($pdf));

// Send the PDF data
echo $pdf;
```

### Streaming Responses

For large responses, you might want to stream the data to avoid memory issues:

```php
<?php
// Set the content type
header('Content-Type: text/csv');
header('Content-Disposition: attachment; filename="large-data.csv"');

// Disable output buffering
if (ob_get_level()) ob_end_clean();

// Output CSV header
echo "id,name,email\n";

// Stream rows from the database
$db = new PDO('mysql:host=localhost;dbname=myapp', 'user', 'pass');
$stmt = $db->query("SELECT id, name, email FROM users LIMIT 10000");

while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
    echo $row['id'] . ',' . $row['name'] . ',' . $row['email'] . "\n";
    
    // Flush the output buffer
    flush();
}
```

## Request Transformation

Frango allows you to transform HTTP requests before they reach PHP through request transformation functions.

### Custom Request Transforms

You can provide a custom function to transform the HTTP request:

```go
// Define a request transformer
transformer := func(r *http.Request) *http.Request {
    // Create a clone to avoid modifying the original
    clone := r.Clone(r.Context())
    
    // Add a custom header
    clone.Header.Set("X-Transformed-By", "Frango")
    
    // Add a query parameter
    q := clone.URL.Query()
    q.Set("added_param", "value")
    clone.URL.RawQuery = q.Encode()
    
    return clone
}

// Use the transformer with Frango
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithRequestTransformer(transformer),
)
```

The transformed request will be used to populate the PHP superglobals, so the new header and query parameter will be available in PHP.

### Request Context Augmentation

You can use the request context to pass additional data to PHP:

```go
// Adding data to the request context
router.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    // Get path parameters from the router
    vars := mux.Vars(r)
    
    // Create a context with additional data
    ctx := context.WithValue(r.Context(), "tenant_id", "acme-corp")
    
    // Use the new context
    r = r.WithContext(ctx)
    
    // Handle the request with Frango
    php.For("user_profile.php").ServeHTTP(w, r)
})
```

By default, Frango doesn't access this context data. To make it available in PHP, you need to extend the PHP globals script or use a request transformer.

## Response Transformation

Similar to request transformation, Frango allows you to transform the response from PHP before it's sent to the client.

### Custom Response Transforms

You can provide a custom function to transform the response:

```go
// Define a response transformer
transformer := func(resp *frango.Response) *frango.Response {
    // Add a security header to all responses
    resp.Headers["X-Content-Type-Options"] = "nosniff"
    resp.Headers["X-Frame-Options"] = "DENY"
    resp.Headers["Content-Security-Policy"] = "default-src 'self'"
    
    // Modify the response body if needed
    // This example adds a timestamp comment to HTML responses
    if strings.HasPrefix(resp.Headers["Content-Type"], "text/html") {
        timestamp := fmt.Sprintf("<!-- Generated at %s -->", time.Now().Format(time.RFC3339))
        resp.Body = append(resp.Body, []byte(timestamp)...)
    }
    
    return resp
}

// Use the transformer with Frango
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithResponseTransformer(transformer),
)
```

### Content Compression

While Frango itself doesn't handle response compression, you can use standard Go middleware like `compress/gzip` if needed:

```go
// Create a gzip middleware
gzipMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check if the client accepts gzip encoding
        if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            // Create a gzip writer
            gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
            if err != nil {
                next.ServeHTTP(w, r)
                return
            }
            defer gz.Close()
            
            // Set the Content-Encoding header
            w.Header().Set("Content-Encoding", "gzip")
            
            // Create a response writer that writes to the gzip writer
            gzw := gzipResponseWriter{
                ResponseWriter: w,
                Writer:         gz,
            }
            
            // Call the next handler with the gzip response writer
            next.ServeHTTP(gzw, r)
            return
        }
        
        // Call the next handler normally
        next.ServeHTTP(w, r)
    })
}

// Apply the middleware
http.Handle("/", gzipMiddleware(php.For("index.php")))
```

## Middleware Integration

Frango is designed to work well with Go's HTTP middleware ecosystem. It can be used with standard middleware patterns and popular middleware libraries.

### Standard HTTP Middleware

You can use Frango with standard Go HTTP middleware:

```go
// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Call the next handler
        next.ServeHTTP(w, r)
        
        // Log the request
        log.Printf(
            "%s %s %s",
            r.Method,
            r.RequestURI,
            time.Since(start),
        )
    })
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        // Handle preflight requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        // Call the next handler
        next.ServeHTTP(w, r)
    })
}

// Apply the middleware
handler := loggingMiddleware(corsMiddleware(php.For("index.php")))
http.Handle("/", handler)
```

### Using with Gorilla Mux

Frango works well with Gorilla Mux for routing:

```go
// Create a new router
router := mux.NewRouter()

// Add routes
router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    php.For("index.php").ServeHTTP(w, r)
}).Methods("GET")

router.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
    php.For("users/list.php").ServeHTTP(w, r)
}).Methods("GET")

router.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    php.For("users/profile.php").ServeHTTP(w, r)
}).Methods("GET")

router.HandleFunc("/api/{resource}", func(w http.ResponseWriter, r *http.Request) {
    php.For("api/handle.php").ServeHTTP(w, r)
}).Methods("GET", "POST", "PUT", "DELETE")

// Use the router
http.Handle("/", router)
```

### Using with Chi Router

Frango also works well with the Chi router:

```go
// Create a new router
router := chi.NewRouter()

// Add middleware
router.Use(middleware.Logger)
router.Use(middleware.Recoverer)

// Add routes
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
    php.For("index.php").ServeHTTP(w, r)
})

router.Route("/users", func(r chi.Router) {
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        php.For("users/list.php").ServeHTTP(w, r)
    })
    
    r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
        php.For("users/profile.php").ServeHTTP(w, r)
    })
})

router.Route("/api/{resource}", func(r chi.Router) {
    r.Use(middleware.BasicAuth("API", map[string]string{"user": "pass"}))
    
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        php.For("api/handle.php").ServeHTTP(w, r)
    })
    
    r.Post("/", func(w http.ResponseWriter, r *http.Request) {
        php.For("api/handle.php").ServeHTTP(w, r)
    })
})

// Use the router
http.Handle("/", router)
```

## Advanced Patterns

### Sharing Data Between Go and PHP

You can use the `Render` method to pass data from Go to PHP:

```go
// Pass data to a PHP script
http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
    // Get the user ID from the request
    userID := r.URL.Query().Get("id")
    
    // Get the user data from the database
    user, err := getUserFromDatabase(userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }
    
    // Render the PHP script with the user data
    php.Render("user_profile.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
        return map[string]interface{}{
            "user": user,
        }
    }).ServeHTTP(w, r)
})
```

In PHP, the data is available as a global variable:

```php
<?php
// Access the user data
global $user;

// Use the user data
echo "<h1>User Profile: " . htmlspecialchars($user['name']) . "</h1>";
echo "<p>Email: " . htmlspecialchars($user['email']) . "</p>";
echo "<p>Role: " . htmlspecialchars($user['role']) . "</p>";
```

### Request Branching with VFS

You can create a branch of the virtual file system for each request, allowing request-specific customizations:

```go
// Create a handler that branches the VFS for each request
func branchingHandler(php *frango.Middleware, scriptPath string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Create a branch of the VFS for this request
        vfs := php.NewVFS()
        if vfs == nil {
            http.Error(w, "Failed to create VFS", http.StatusInternalServerError)
            return
        }
        defer vfs.Cleanup()
        
        // Customize the VFS for this request
        tenantID := r.URL.Query().Get("tenant")
        if tenantID != "" {
            // Create a tenant-specific configuration file
            configContent := []byte("<?php\n$TENANT_ID = '" + tenantID + "';\n")
            err := vfs.CreateVirtualFile("/tenant_config.php", configContent)
            if err != nil {
                http.Error(w, "Failed to create config", http.StatusInternalServerError)
                return
            }
        }
        
        // Use the customized VFS for this request
        php.ForVFS(vfs, scriptPath).ServeHTTP(w, r)
    })
}

// Use the branching handler
http.Handle("/app", branchingHandler(php, "app.php"))
```

### Dynamic Script Selection

You can dynamically select which PHP script to execute based on the request:

```go
// Create a handler that selects the script based on the request
func dynamicScriptHandler(php *frango.Middleware) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get the path from the request
        path := strings.TrimPrefix(r.URL.Path, "/")
        if path == "" {
            path = "index"
        }
        
        // Map the path to a PHP script
        scriptPath := "/scripts/" + path + ".php"
        
        // Check if the script exists
        vfs, err := php.getRootVFS()
        if err != nil {
            http.Error(w, "VFS error", http.StatusInternalServerError)
            return
        }
        
        if !vfs.FileExists(scriptPath) {
            // Try a fallback script
            scriptPath = "/scripts/404.php"
            if !vfs.FileExists(scriptPath) {
                http.Error(w, "Not Found", http.StatusNotFound)
                return
            }
        }
        
        // Execute the selected script
        php.For(scriptPath).ServeHTTP(w, r)
    })
}

// Use the dynamic script handler
http.Handle("/", dynamicScriptHandler(php))
```

### Content Negotiation

You can use content negotiation to serve different formats based on the `Accept` header:

```go
// Create a handler that supports content negotiation
func contentNegotiationHandler(php *frango.Middleware, data interface{}) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get the Accept header
        accept := r.Header.Get("Accept")
        
        // Determine the best format
        if strings.Contains(accept, "application/json") {
            // Serve JSON directly from Go
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(data)
            return
        }
        
        if strings.Contains(accept, "text/xml") || strings.Contains(accept, "application/xml") {
            // Serve XML using a PHP script
            php.Render("xml_template.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
                return map[string]interface{}{
                    "data": data,
                }
            }).ServeHTTP(w, r)
            return
        }
        
        // Default to HTML
        php.Render("html_template.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
            return map[string]interface{}{
                "data": data,
            }
        }).ServeHTTP(w, r)
    })
}

// Use the content negotiation handler
http.Handle("/api/resource", contentNegotiationHandler(php, resource))
```

## Conclusion

Frango's request and response handling API provides a powerful bridge between Go's HTTP infrastructure and PHP's processing model. By understanding and effectively using this API, you can:

1. Forward HTTP requests from Go to PHP scripts
2. Pass data and parameters from Go to PHP
3. Transform PHP's output into HTTP responses
4. Implement middleware chains for request/response processing
5. Create advanced patterns like content negotiation and API versioning

This bidirectional conversion allows you to leverage the strengths of both Go and PHP in a single application while maintaining a clean separation of concerns. 