# PHP Access Patterns

This guide covers how to access HTTP request data and other information from your PHP scripts when running with Frango.

## URL Parameter Access

### Path Parameters

Path parameters are extracted from URL patterns with curly braces and made available through the `$_PATH` superglobal:

```php
<?php
// For a route like /users/{id}/profiles/{profile_id}
$userId = $_PATH['id'];
$profileId = $_PATH['profile_id'];

// Check if a parameter exists
if (isset($_PATH['optional_param'])) {
    // ...
}

// With default value
$category = $_PATH['category'] ?? 'default';
```

Helper functions are also available:

```php
<?php
// Get a path parameter with optional default
$userId = path_param('id', 'guest');

// Check if parameter exists
if (has_path_param('id')) {
    // ...
}
```

### Path Segments

You can access individual URL path segments using the `$_PATH_SEGMENTS` array:

```php
<?php
// For a URL like /products/electronics/phones
// $_PATH_SEGMENTS = ['products', 'electronics', 'phones']

$section = $_PATH_SEGMENTS[0]; // 'products'
$category = $_PATH_SEGMENTS[1]; // 'electronics'
$subcategory = $_PATH_SEGMENTS[2]; // 'phones'

// Get the number of segments
$segmentCount = count($_PATH_SEGMENTS);
```

## Form Data Handling

### GET Parameters

Access query string parameters through the standard `$_GET` superglobal:

```php
<?php
// For a URL like /search?q=keyword&page=2
$query = $_GET['q'] ?? '';
$page = (int)($_GET['page'] ?? 1);

// Iterate over all parameters
foreach ($_GET as $key => $value) {
    echo "$key: $value\n";
}
```

### POST Data

Access POST form data through the standard `$_POST` superglobal:

```php
<?php
// For a form submission with fields 'username' and 'email'
$username = $_POST['username'] ?? '';
$email = $_POST['email'] ?? '';

// Iterate over all POST data
foreach ($_POST as $key => $value) {
    echo "$key: $value\n";
}
```

### Unified Form Access

Frango provides a special `$_FORM` superglobal that contains form data regardless of the HTTP method used:

```php
<?php
// Works for both GET and POST requests
$username = $_FORM['username'] ?? '';
$email = $_FORM['email'] ?? '';

// Iterate over all form data
foreach ($_FORM as $key => $value) {
    echo "$key: $value\n";
}
```

## JSON Data Processing

### Receiving JSON

For requests with JSON bodies, use the `$_JSON` superglobal:

```php
<?php
// For a request with JSON like {"user":{"name":"John","role":"admin"}}
$userName = $_JSON['user']['name'] ?? '';
$userRole = $_JSON['user']['role'] ?? '';

// Check if a key exists
if (isset($_JSON['settings'])) {
    $settings = $_JSON['settings'];
}

// Get the entire JSON data
$allData = $_JSON;
```

### Sending JSON Responses

To send JSON responses:

```php
<?php
// Set content type
header('Content-Type: application/json');

// Create response data
$response = [
    'status' => 'success',
    'data' => [
        'id' => 123,
        'name' => 'Product Name',
        'price' => 19.99
    ]
];

// Output as JSON
echo json_encode($response);
```

## Request Headers

Access HTTP request headers:

```php
<?php
// Get all headers
$headers = getallheaders();

// Access specific headers
$contentType = $headers['Content-Type'] ?? '';
$authorization = $headers['Authorization'] ?? '';

// Case-insensitive access
foreach ($headers as $name => $value) {
    if (strtolower($name) === 'user-agent') {
        $userAgent = $value;
        break;
    }
}
```

## Setting Response Headers

Set HTTP response headers:

```php
<?php
// Set content type
header('Content-Type: application/json');

// Set custom headers
header('X-API-Version: 1.0');
header('Cache-Control: no-cache');

// Set response code
http_response_code(201); // Created
```

## Accessing Template Variables

When using the `Render()` method, access passed variables:

```php
<?php
// Direct variable access - variables passed from Go are directly accessible
echo $username;
echo $user['email'];

// Access via $_TEMPLATE superglobal
echo $_TEMPLATE['username'];
echo $_TEMPLATE['user']['email'];

// Check if a variable exists
if (isset($_TEMPLATE['optional'])) {
    // ...
}
```

## Server Information

Access server and request information:

```php
<?php
// Request method
$method = $_SERVER['REQUEST_METHOD'];

// Request URI
$uri = $_SERVER['REQUEST_URI'];

// Query string
$queryString = $_SERVER['QUERY_STRING'];

// Client IP
$clientIp = $_SERVER['REMOTE_ADDR'];
```

## Additional Helper Variables

Frango provides some additional helper variables:

```php
<?php
// Current URL path
$currentPath = $_URL;

// Complete current URL (including query string)
$fullUrl = $_CURRENT_URL;

// Direct access to query parameters (same as $_GET)
$queryParams = $_QUERY;
``` 