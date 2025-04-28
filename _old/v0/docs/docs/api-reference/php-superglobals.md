# PHP Superglobals API Reference

This document provides a comprehensive reference for PHP superglobals in Frango, explaining how these built-in arrays are populated and can be used within PHP scripts running in a Frango environment.

## Table of Contents

- [Overview](#overview)
- [$_SERVER](#_server)
- [$_GET](#_get)
- [$_POST](#_post)
- [$_FILES](#_files)
- [$_COOKIE](#_cookie)
- [$_SESSION](#_session)
- [$_ENV](#_env)
- [$_REQUEST](#_request)
- [$_JSON](#_json)
- [Custom Variables](#custom-variables)
- [Best Practices](#best-practices)

## Overview

PHP superglobals are predefined arrays that are available in all scopes throughout a script. In a traditional PHP environment, these arrays are populated by the PHP runtime based on the HTTP request. In Frango, these superglobals are populated based on the Go `http.Request` object and any additional data you provide.

Frango ensures that PHP scripts running within its environment have access to the same superglobals they would in a standard PHP environment, allowing existing PHP code to work with minimal modification.

The following superglobals are available in Frango:

| Superglobal | Description |
|-------------|-------------|
| $_SERVER | Server and execution environment information |
| $_GET | HTTP GET variables |
| $_POST | HTTP POST variables |
| $_FILES | HTTP file upload variables |
| $_COOKIE | HTTP cookies |
| $_SESSION | Session variables (requires session management) |
| $_ENV | Environment variables |
| $_REQUEST | HTTP request variables (combined GET, POST, and COOKIE) |
| $_PATH | Path parameters from URL patterns (Frango-specific) |
| $_JSON | JSON request body (Frango-specific, not standard PHP) |

## $_SERVER

The `$_SERVER` superglobal contains information about the server and execution environment. Frango populates this array based on the Go `http.Request` and server environment.

### Key Server Variables

| Key | Description | Example |
|-----|-------------|---------|
| REQUEST_METHOD | The HTTP request method | "GET", "POST", "PUT", "DELETE" |
| REQUEST_URI | The URI which was given to access the page | "/path/to/page.php?query=value" |
| QUERY_STRING | The query string (if any) | "query=value&param=data" |
| HTTP_HOST | The Host header from the request | "example.com" |
| HTTP_USER_AGENT | The User-Agent header | "Mozilla/5.0 ..." |
| HTTP_REFERER | The Referer header (if available) | "https://example.com/previous-page" |
| HTTP_ACCEPT | The Accept header | "text/html,application/xhtml+xml,..." |
| REMOTE_ADDR | The IP address of the client | "192.168.1.1" |
| SERVER_PROTOCOL | The protocol and version | "HTTP/1.1" |
| SERVER_NAME | The server name | "example.com" |
| SERVER_PORT | The server port | "80" |
| DOCUMENT_ROOT | The document root directory | "/var/www/html" |
| SCRIPT_FILENAME | The absolute path to the script | "/var/www/html/index.php" |
| SCRIPT_NAME | The relative path to the script | "/index.php" |
| PHP_SELF | The filename of the script | "/index.php" |
| REQUEST_TIME | The timestamp of the request | 1609459200 |
| REQUEST_TIME_FLOAT | The timestamp with microseconds | 1609459200.123456 |
| PATH_INFO | Additional path information | "/extra/path" |
| PATH_PARAMS | JSON-encoded path parameters (Frango-specific) | {"id":"123","action":"view"} |

### Accessing Server Variables

```php
<?php
// Get the request method
$method = $_SERVER['REQUEST_METHOD'];

// Get the client's IP address
$ip = $_SERVER['REMOTE_ADDR'];

// Get the User-Agent
$userAgent = $_SERVER['HTTP_USER_AGENT'];

// Check if a header exists
$referer = isset($_SERVER['HTTP_REFERER']) ? $_SERVER['HTTP_REFERER'] : '';

// Get the current URL
$protocol = isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? 'https' : 'http';
$host = $_SERVER['HTTP_HOST'];
$uri = $_SERVER['REQUEST_URI'];
$currentUrl = $protocol . '://' . $host . $uri;
```

### HTTP Headers in $_SERVER

All HTTP request headers are available in `$_SERVER` with the `HTTP_` prefix. The header names are converted to uppercase, and hyphens are replaced with underscores.

For example:
- `User-Agent` header becomes `$_SERVER['HTTP_USER_AGENT']`
- `X-Requested-With` header becomes `$_SERVER['HTTP_X_REQUESTED_WITH']`
- `Content-Type` header becomes `$_SERVER['CONTENT_TYPE']` (special case)
- `Content-Length` header becomes `$_SERVER['CONTENT_LENGTH']` (special case)

```php
<?php
// Check for AJAX request
$isAjax = isset($_SERVER['HTTP_X_REQUESTED_WITH']) && 
          $_SERVER['HTTP_X_REQUESTED_WITH'] === 'XMLHttpRequest';

// Get content type
$contentType = $_SERVER['CONTENT_TYPE'] ?? '';
```

### Custom Server Variables

Frango allows you to set custom `$_SERVER` variables when rendering a PHP script:

```go
// Set custom server variables
php.Render("/script.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "server": map[string]string{
            "APP_ENV": "production",
            "CUSTOM_HEADER": "custom-value",
        },
    }
}).ServeHTTP(w, r)
```

In PHP, these will be available in `$_SERVER`:

```php
<?php
$appEnv = $_SERVER['APP_ENV']; // "production"
$customHeader = $_SERVER['CUSTOM_HEADER']; // "custom-value"
```

## $_GET

The `$_GET` superglobal contains variables passed to the script via URL parameters (query string). Frango populates this array from the query parameters in the Go `http.Request`.

### Accessing GET Variables

```php
<?php
// For a URL like /search.php?q=frango&page=1

// Get a specific parameter
$query = $_GET['q']; // "frango"
$page = $_GET['page']; // "1"

// Check if a parameter exists
if (isset($_GET['filter'])) {
    $filter = $_GET['filter'];
} else {
    $filter = 'default';
}

// Get a parameter with a default value
$sort = $_GET['sort'] ?? 'name';

// Loop through all GET parameters
foreach ($_GET as $key => $value) {
    echo "$key: $value\n";
}
```

### Array Parameters

Query parameters can be arrays by using square brackets in the parameter names:

```
/search.php?filters[category]=books&filters[price]=10-20
```

In PHP, these will be available as nested arrays:

```php
<?php
$category = $_GET['filters']['category']; // "books"
$price = $_GET['filters']['price']; // "10-20"
```

## $_POST

The `$_POST` superglobal contains variables passed to the script via HTTP POST method. Frango populates this array from the form data in the Go `http.Request`.

### Accessing POST Variables

```php
<?php
// For a form submitted with fields "name" and "email"

// Get specific fields
$name = $_POST['name'];
$email = $_POST['email'];

// Check if a field exists
if (isset($_POST['phone'])) {
    $phone = $_POST['phone'];
}

// Get a field with a default value
$age = $_POST['age'] ?? '';

// Loop through all POST fields
foreach ($_POST as $key => $value) {
    echo "$key: $value\n";
}
```

### Array Fields

Form fields can be arrays by using square brackets in the field names:

```html
<input type="checkbox" name="interests[]" value="sports">
<input type="checkbox" name="interests[]" value="music">
<input type="checkbox" name="interests[]" value="books">
```

In PHP, these will be available as arrays:

```php
<?php
$interests = $_POST['interests']; // ["sports", "music", "books"]

foreach ($interests as $interest) {
    echo $interest . "\n";
}
```

### JSON POST Data

When a request has a `Content-Type: application/json` header, Frango automatically parses the JSON body and populates a special `$_JSON` variable. The original JSON data is still available via `php://input`.

```php
<?php
// For a JSON POST request with {"name":"John","email":"john@example.com"}

// Access JSON data (if using a custom PHP globals script that supports $_JSON)
if (isset($_JSON)) {
    $name = $_JSON['name']; // "John"
    $email = $_JSON['email']; // "john@example.com"
} else {
    // Fallback to manual JSON parsing
    $jsonData = file_get_contents('php://input');
    $data = json_decode($jsonData, true);
    
    $name = $data['name'] ?? '';
    $email = $data['email'] ?? '';
}
```

## $_FILES

The `$_FILES` superglobal contains information about files uploaded via HTTP POST method. Frango populates this array from the multipart form data in the Go `http.Request`.

### File Upload Structure

Each uploaded file in `$_FILES` has the following structure:

```php
$_FILES['field_name'] = [
    'name' => 'filename.jpg',           // Original filename
    'type' => 'image/jpeg',             // MIME type
    'tmp_name' => '/tmp/phpXXXXXX',     // Temporary file path
    'error' => 0,                       // Upload error code (0 = no error)
    'size' => 12345                     // File size in bytes
];
```

### Accessing Uploaded Files

```php
<?php
// For a form with <input type="file" name="document">

// Check if a file was uploaded
if (isset($_FILES['document']) && $_FILES['document']['error'] === UPLOAD_ERR_OK) {
    // Get file details
    $fileName = $_FILES['document']['name'];
    $fileType = $_FILES['document']['type'];
    $fileSize = $_FILES['document']['size'];
    $fileTmpPath = $_FILES['document']['tmp_name'];
    
    // Move the uploaded file to a permanent location
    $uploadDir = '/uploads/';
    $targetPath = $uploadDir . basename($fileName);
    
    if (move_uploaded_file($fileTmpPath, $targetPath)) {
        echo "File uploaded successfully to $targetPath";
    } else {
        echo "Error moving uploaded file";
    }
} else {
    // Get the error message if upload failed
    $errorCode = $_FILES['document']['error'] ?? UPLOAD_ERR_NO_FILE;
    $errorMessages = [
        UPLOAD_ERR_INI_SIZE => 'The uploaded file exceeds the upload_max_filesize directive in php.ini',
        UPLOAD_ERR_FORM_SIZE => 'The uploaded file exceeds the MAX_FILE_SIZE directive in the HTML form',
        UPLOAD_ERR_PARTIAL => 'The uploaded file was only partially uploaded',
        UPLOAD_ERR_NO_FILE => 'No file was uploaded',
        UPLOAD_ERR_NO_TMP_DIR => 'Missing a temporary folder',
        UPLOAD_ERR_CANT_WRITE => 'Failed to write file to disk',
        UPLOAD_ERR_EXTENSION => 'A PHP extension stopped the file upload',
    ];
    echo "Upload error: " . ($errorMessages[$errorCode] ?? 'Unknown error');
}
```

### Multiple Files Upload

When multiple files are uploaded with the same field name using the array syntax (`name="documents[]"`), `$_FILES` will have a structure like this:

```php
$_FILES['documents'] = [
    'name' => [
        0 => 'first.jpg',
        1 => 'second.png',
        2 => 'third.pdf'
    ],
    'type' => [
        0 => 'image/jpeg',
        1 => 'image/png',
        2 => 'application/pdf'
    ],
    'tmp_name' => [
        0 => '/tmp/phpA1B2C3',
        1 => '/tmp/phpD4E5F6',
        2 => '/tmp/phpG7H8I9'
    ],
    'error' => [
        0 => 0,
        1 => 0,
        2 => 0
    ],
    'size' => [
        0 => 12345,
        1 => 23456,
        2 => 34567
    ]
];
```

Accessing multiple uploaded files:

```php
<?php
if (isset($_FILES['documents'])) {
    $fileCount = count($_FILES['documents']['name']);
    
    for ($i = 0; $i < $fileCount; $i++) {
        if ($_FILES['documents']['error'][$i] === UPLOAD_ERR_OK) {
            $fileName = $_FILES['documents']['name'][$i];
            $fileTmpPath = $_FILES['documents']['tmp_name'][$i];
            
            // Process each file...
            move_uploaded_file($fileTmpPath, "/uploads/" . basename($fileName));
        }
    }
}
```

## $_COOKIE

The `$_COOKIE` superglobal contains variables passed to the script via HTTP cookies. Frango populates this array from the cookies in the Go `http.Request`.

### Accessing Cookies

```php
<?php
// Get a specific cookie
$sessionId = $_COOKIE['session_id'] ?? '';
$theme = $_COOKIE['theme'] ?? 'light';

// Check if a cookie exists
if (isset($_COOKIE['user_preferences'])) {
    $preferences = json_decode($_COOKIE['user_preferences'], true);
}

// Loop through all cookies
foreach ($_COOKIE as $name => $value) {
    echo "$name: $value\n";
}
```

### Setting Cookies

You can set cookies in PHP scripts running in Frango using the `setcookie()` function or by setting the `Set-Cookie` header:

```php
<?php
// Set a simple cookie
setcookie('user', 'john', time() + 3600, '/', '', false, true);

// Set a cookie with all options
setcookie(
    'preferences',           // name
    json_encode(['theme' => 'dark']), // value
    [
        'expires' => time() + 86400,   // expiry time (1 day)
        'path' => '/',                // path
        'domain' => '',               // domain
        'secure' => true,             // only sent over HTTPS
        'httponly' => true,           // not accessible via JavaScript
        'samesite' => 'Strict'        // CSRF protection
    ]
);

// Alternative: Set cookie using header
header('Set-Cookie: analytics=true; Max-Age=3600; Path=/; HttpOnly');
```

## $_SESSION

The `$_SESSION` superglobal provides a way to preserve data across subsequent HTTP requests. Frango supports PHP sessions if properly initialized.

### Starting a Session

Before using the `$_SESSION` array, you need to start the session:

```php
<?php
// Start the session
session_start();

// Now $_SESSION is available
$_SESSION['user_id'] = 123;
$_SESSION['username'] = 'john';
```

### Accessing Session Data

```php
<?php
// Start the session
session_start();

// Get session data
$userId = $_SESSION['user_id'] ?? null;
$username = $_SESSION['username'] ?? 'Guest';

// Check if a session variable exists
if (isset($_SESSION['login_time'])) {
    $loginTime = $_SESSION['login_time'];
}

// Store new data in the session
$_SESSION['last_activity'] = time();
```

### Destroying a Session

```php
<?php
// Start the session
session_start();

// Unset all session variables
$_SESSION = [];

// If it's desired to kill the session, also delete the session cookie
if (ini_get("session.use_cookies")) {
    $params = session_get_cookie_params();
    setcookie(
        session_name(),
        '',
        time() - 42000,
        $params["path"],
        $params["domain"],
        $params["secure"],
        $params["httponly"]
    );
}

// Finally, destroy the session
session_destroy();
```

## $_ENV

The `$_ENV` superglobal contains environment variables. In Frango, these are populated from the Go environment and any additional variables you provide.

### Accessing Environment Variables

```php
<?php
// Get specific environment variables
$appEnv = $_ENV['APP_ENV'] ?? 'development';
$dbHost = $_ENV['DB_HOST'] ?? 'localhost';

// Check if an environment variable exists
if (isset($_ENV['API_KEY'])) {
    $apiKey = $_ENV['API_KEY'];
}

// Loop through all environment variables
foreach ($_ENV as $name => $value) {
    echo "$name: $value\n";
}
```

### Environment Variables in Go

You can provide environment variables to PHP scripts in Frango:

```go
// Set environment variables for PHP scripts
os.Setenv("APP_ENV", "production")
os.Setenv("DB_HOST", "db.example.com")

// These will be available in PHP as $_ENV['APP_ENV'] and $_ENV['DB_HOST']
```

### Worker Environment Variables

When using FrankenPHP's worker mode, be aware that environment variables loaded at worker startup may be different from those available during each request. If you need to preserve specific environment variables, capture them at startup.

## $_REQUEST

The `$_REQUEST` superglobal is an array containing the contents of `$_GET`, `$_POST`, and `$_COOKIE`. If the same key exists in multiple sources, the precedence is determined by the `variables_order` PHP configuration directive (typically `EGPCS`, meaning `$_COOKIE` overwrites `$_POST` which overwrites `$_GET`).

### Accessing Request Variables

```php
<?php
// Get a parameter from any source (GET, POST, or COOKIE)
$id = $_REQUEST['id'] ?? null;
$name = $_REQUEST['name'] ?? '';

// Check if a parameter exists in any source
if (isset($_REQUEST['action'])) {
    $action = $_REQUEST['action'];
}

// Loop through all request parameters
foreach ($_REQUEST as $key => $value) {
    echo "$key: $value\n";
}
```

## $_JSON

The `$_JSON` superglobal is specific to Frango and contains the decoded JSON request body. This is not a standard PHP superglobal but is provided by Frango for convenience.

### Accessing JSON Request Data

```php
<?php
// For a request with Content-Type: application/json
// And body: {"name": "John", "email": "john@example.com"}

// Check if JSON data exists
if (isset($_JSON)) {
    // Get specific fields
    $name = $_JSON['name'] ?? '';
    $email = $_JSON['email'] ?? '';
    
    // Check if a field exists
    if (isset($_JSON['phone'])) {
        $phone = $_JSON['phone'];
    }
    
    // Access nested properties
    $address = $_JSON['address']['city'] ?? '';
    
    // Loop through all JSON data
    foreach ($_JSON as $key => $value) {
        echo "$key: " . json_encode($value) . "\n";
    }
} else {
    // Not a JSON request or no JSON body
    echo "No JSON data available";
    
    // Alternative: manually parse the JSON
    $input = file_get_contents('php://input');
    if (!empty($input)) {
        $data = json_decode($input, true);
        if (json_last_error() === JSON_ERROR_NONE) {
            // Use $data...
        }
    }
}
```

### Using $_JSON with Different HTTP Methods

The `$_JSON` superglobal works with any HTTP method that has a JSON request body:

```php
<?php
// Check the request method
$method = $_SERVER['REQUEST_METHOD'];

switch ($method) {
    case 'GET':
        // GET requests typically don't have a body
        // Use $_GET for query parameters
        break;
        
    case 'POST':
        // Create a new resource
        if (isset($_JSON)) {
            // Create resource with $_JSON data
        }
        break;
        
    case 'PUT':
        // Update a resource
        if (isset($_JSON)) {
            $id = $_GET['id'] ?? 0;
            // Update resource $id with $_JSON data
        }
        break;
        
    case 'PATCH':
        // Partially update a resource
        if (isset($_JSON)) {
            $id = $_GET['id'] ?? 0;
            // Partially update resource $id with $_JSON data
        }
        break;
        
    case 'DELETE':
        // Delete requests might have a body with additional instructions
        if (isset($_JSON)) {
            // Use $_JSON data for deletion logic
        }
        break;
}
```

## Custom Variables

Frango allows you to pass custom variables to PHP scripts using the `Render` method:

```go
// Pass custom data to a PHP script
php.Render("/template.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "user": map[string]interface{}{
            "id":       123,
            "username": "john",
            "email":    "john@example.com",
        },
        "products": []map[string]interface{}{
            {"id": 1, "name": "Product 1", "price": 19.99},
            {"id": 2, "name": "Product 2", "price": 29.99},
        },
        "settings": map[string]interface{}{
            "showPrices": true,
            "currency":   "USD",
        },
    }
}).ServeHTTP(w, r)
```

In the PHP script, these variables are available as global variables:

```php
<?php
// Access custom variables
global $user, $products, $settings;

// Use the variables
echo "Welcome, " . htmlspecialchars($user['username']) . "!";

if ($settings['showPrices']) {
    foreach ($products as $product) {
        echo "<div class='product'>";
        echo "<h3>" . htmlspecialchars($product['name']) . "</h3>";
        echo "<p>Price: " . htmlspecialchars($settings['currency']) . " " . number_format($product['price'], 2) . "</p>";
        echo "</div>";
    }
}
```

## Path Parameters

Frango adds support for URL path parameters with a special `$_PATH` superglobal when using routers like gorilla/mux. These are automatically extracted from the context and made available to PHP.

```go
// In Go, using gorilla/mux
router := mux.NewRouter()
router.HandleFunc("/users/{id}/profile", php.For("/user_profile.php").ServeHTTP)
```

```php
<?php
// In PHP, access the path parameter
$userId = $_PATH['id'] ?? null;

if ($userId === null) {
    echo "User ID not provided";
    exit;
}

echo "Showing profile for user ID: " . htmlspecialchars($userId);
```

## Best Practices

### Security Considerations

Always validate and sanitize input from superglobals:

```php
<?php
// Validate ID is numeric
$id = isset($_GET['id']) ? (int)$_GET['id'] : 0;
if ($id <= 0) {
    echo "Invalid ID";
    exit;
}

// Sanitize user input before output
$name = htmlspecialchars($_POST['name'] ?? '', ENT_QUOTES);
echo "Hello, $name";
```

### Default Values

Use the null coalescing operator (`??`) or `isset()` checks to provide default values:

```php
<?php
// With null coalescing operator (PHP 7+)
$page = $_GET['page'] ?? 1;
$perPage = $_GET['per_page'] ?? 20;

// With isset()
$page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
$perPage = isset($_GET['per_page']) ? (int)$_GET['per_page'] : 20;
```

### Handling Arrays

Be careful when dealing with potentially missing nested arrays:

```php
<?php
// Check for nested arrays safely
$category = '';
if (isset($_GET['filters']) && is_array($_GET['filters']) && isset($_GET['filters']['category'])) {
    $category = $_GET['filters']['category'];
}

// Alternative using null coalescing
$category = $_GET['filters']['category'] ?? '';
```

### Debugging Superglobals

For debugging, you can dump the contents of superglobals:

```php
<?php
// Only in development environments
if ($_SERVER['APP_ENV'] === 'development') {
    echo "<h3>GET Variables:</h3>";
    echo "<pre>" . print_r($_GET, true) . "</pre>";
    
    echo "<h3>POST Variables:</h3>";
    echo "<pre>" . print_r($_POST, true) . "</pre>";
    
    echo "<h3>SERVER Variables:</h3>";
    echo "<pre>" . print_r($_SERVER, true) . "</pre>";
}
```

### Using Superglobals in Functions

Superglobals are available in any scope, but it's better to pass them as parameters:

```php
<?php
// Not recommended - using superglobals directly in functions
function getUserId() {
    return $_GET['user_id'] ?? 0;
}

// Better - pass as parameters
function getUserId($params) {
    return $params['user_id'] ?? 0;
}
$userId = getUserId($_GET);
```

### Working with JSON Requests

For API endpoints that accept JSON:

```php
<?php
// Check content type
$contentType = $_SERVER['CONTENT_TYPE'] ?? '';
if (strpos($contentType, 'application/json') !== false) {
    // Read raw JSON body
    $jsonData = file_get_contents('php://input');
    $data = json_decode($jsonData, true);
    
    if ($data === null && json_last_error() !== JSON_ERROR_NONE) {
        // Invalid JSON
        http_response_code(400);
        echo json_encode(['error' => 'Invalid JSON']);
        exit;
    }
    
    // Process JSON data
    $name = $data['name'] ?? '';
    // ...
}
```

## Conclusion

Frango implements PHP superglobals to provide a familiar environment for PHP scripts running within a Go application. By understanding how each superglobal is populated and used, you can effectively build applications that leverage both the power of Go and the flexibility of PHP.

The ability to pass custom variables from Go to PHP scripts further enhances this integration, allowing for seamless data exchange between the two languages. Combined with proper validation and security practices, Frango's implementation of superglobals provides a robust foundation for building hybrid Go-PHP applications. 