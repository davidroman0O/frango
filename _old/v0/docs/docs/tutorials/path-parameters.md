# Path Parameters in Frango

This tutorial explains how to work with path parameters in Frango applications, allowing you to build flexible and RESTful APIs that extract values from URL paths.

## Table of Contents

- [Overview](#overview)
- [Understanding Path Parameters](#understanding-path-parameters)
- [Setting Up Basic Path Routing](#setting-up-basic-path-routing)
- [Handling Path Parameters in Go](#handling-path-parameters-in-go)
- [Accessing Path Parameters in PHP](#accessing-path-parameters-in-php)
- [Advanced Path Parameter Techniques](#advanced-path-parameter-techniques)
- [Path Parameter Validation](#path-parameter-validation)
- [Common Patterns and Best Practices](#common-patterns-and-best-practices)
- [Debugging Path Parameter Issues](#debugging-path-parameter-issues)

## Overview

Path parameters (sometimes called URL parameters or route parameters) are variable segments in the URL path that capture values from the request URL. They are typically used in RESTful APIs to identify specific resources, such as:

- `/users/123` - where `123` is a user ID
- `/products/electronics/laptops` - where `electronics` and `laptops` are category identifiers
- `/articles/2023/05/my-article-title` - where `2023`, `05`, and `my-article-title` are date parts and a slug

Frango provides ways to work with path parameters on both the Go and PHP sides of your application, giving you flexibility in how you structure your APIs.

## Understanding Path Parameters

In a URL like `/api/users/123/posts/456`, there are several path segments:

1. `/api` - API namespace
2. `/users` - Resource type
3. `/123` - User ID parameter
4. `/posts` - Nested resource type
5. `/456` - Post ID parameter

When designing APIs with path parameters, it's important to understand:

- Path parameters vs. query parameters
- RESTful routing conventions
- How to extract and use these values in code

### Path Parameters vs. Query Parameters

| Path Parameters | Query Parameters |
|-----------------|------------------|
| Part of the URL path: `/users/123` | Added after `?`: `/users?id=123` |
| Used for identifying resources | Used for filtering, sorting, pagination |
| More SEO-friendly | Better for optional parameters |
| Can't be omitted from the URL | Can be omitted |

## Setting Up Basic Path Routing

Let's create a simple application that demonstrates path parameter handling. Our example will be a blog API that handles posts and comments.

### Project Structure

```
blog-api/
├── main.go               # Go application entry point
├── php-files/
│   ├── api/
│   │   ├── posts.php     # Handle post collection
│   │   ├── post.php      # Handle single post
│   │   ├── comments.php  # Handle comment collection
│   │   └── comment.php   # Handle single comment
│   ├── helpers/
│   │   └── route.php     # Route helper functions
```

### Creating the Go Application

First, let's create our `main.go` file with basic routing:

```go
package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/davidroman0O/frango"
)

func main() {
	// Initialize Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php-files"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Frango: %v", err)
	}
	defer php.Shutdown()

	// Create a router
	mux := http.NewServeMux()

	// Route: GET /api/posts
	// Lists all posts
	mux.Handle("GET /api/posts", php.For("api/posts.php"))

	// Route: GET /api/posts/{postID}
	// Gets a specific post by ID
	mux.HandleFunc("GET /api/posts/", func(w http.ResponseWriter, r *http.Request) {
		// Skip "/api/posts/" prefix to extract the postID
		path := r.URL.Path[len("/api/posts/"):]
		
		// Check if there's more to the path (nested resources)
		parts := strings.Split(path, "/")
		postID := parts[0]
		
		// If postID is empty, redirect to the posts list
		if postID == "" {
			http.Redirect(w, r, "/api/posts", http.StatusSeeOther)
			return
		}
		
		// Pass the postID parameter to PHP
		// The parameter will be available in PHP via $_SERVER['PATH_PARAMS']
		php.For("api/post.php").ServeHTTP(w, r)
	})

	// Route: GET /api/posts/{postID}/comments
	// Lists all comments for a specific post
	mux.HandleFunc("GET /api/posts/", func(w http.ResponseWriter, r *http.Request) {
		// Skip "/api/posts/" prefix
		path := r.URL.Path[len("/api/posts/"):]
		
		// Check if the request is for comments
		if !strings.HasSuffix(path, "/comments") {
			// Let the previous handler take it
			return
		}
		
		// Extract the postID (remove "/comments" suffix)
		postID := path[:len(path)-len("/comments")]
		
		// Forward to PHP with the postID
		php.For("api/comments.php").ServeHTTP(w, r)
	})

	// Route: GET /api/posts/{postID}/comments/{commentID}
	// Gets a specific comment by ID for a specific post
	mux.HandleFunc("GET /api/posts/", func(w http.ResponseWriter, r *http.Request) {
		// Skip "/api/posts/" prefix
		path := r.URL.Path[len("/api/posts/"):]
		
		// Check if the path matches our pattern
		if !strings.Contains(path, "/comments/") {
			// Let other handlers take it
			return
		}
		
		// Split the path into segments
		parts := strings.Split(path, "/")
		if len(parts) < 3 || parts[1] != "comments" {
			// Invalid path structure
			http.Error(w, "Invalid URL path", http.StatusBadRequest)
			return
		}
		
		postID := parts[0]
		commentID := parts[2]
		
		// Forward to PHP
		php.For("api/comment.php").ServeHTTP(w, r)
	})

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
```

### Creating PHP Helpers

Let's create some helper functions for route handling in `php-files/helpers/route.php`:

```php
<?php
/**
 * Route parameter helper functions
 */

/**
 * Sends a JSON response
 *
 * @param mixed $data The data to encode as JSON
 * @param int $statusCode HTTP status code (default: 200)
 */
function sendJsonResponse($data, $statusCode = 200) {
    // Set status code
    http_response_code($statusCode);
    
    // Set content type
    header('Content-Type: application/json');
    
    // Encode and output data
    echo json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
    exit;
}

/**
 * Extracts a path parameter from the URL
 *
 * @param string $basePath The base path to remove (e.g., "/api/posts/")
 * @return string The extracted parameter
 */
function extractPathParameter($basePath) {
    $path = $_SERVER['REQUEST_URI'];
    $path = parse_url($path, PHP_URL_PATH);
    
    // Remove the base path
    if (strpos($path, $basePath) === 0) {
        $param = substr($path, strlen($basePath));
        // Remove trailing slash if present
        $param = rtrim($param, '/');
        return $param;
    }
    
    return '';
}

/**
 * Validates that a parameter exists and is non-empty
 *
 * @param string $param The parameter to check
 * @param string $paramName The name of the parameter (for error messages)
 * @return bool True if valid, false otherwise
 */
function validateParam($param, $paramName) {
    if (!isset($param) || empty($param)) {
        sendJsonResponse([
            'error' => "$paramName is required"
        ], 400);
        return false;
    }
    return true;
}

/**
 * Validates a numeric ID parameter
 *
 * @param mixed $id The ID to validate
 * @param string $paramName The name of the parameter (for error messages)
 * @return bool True if valid, false otherwise
 */
function validateId($id, $paramName) {
    if (!isset($id) || !is_numeric($id) || $id <= 0) {
        sendJsonResponse([
            'error' => "$paramName must be a positive number"
        ], 400);
        return false;
    }
    return true;
}
?>
```

## Handling Path Parameters in Go

In Go, you can extract path parameters in multiple ways. Let's explore a few approaches.

### Extracting Parameters from the URL Path

The most direct approach is to extract parameters from the URL path:

```go
// Handle requests for: /api/users/{userID}
mux.HandleFunc("GET /api/users/", func(w http.ResponseWriter, r *http.Request) {
    // Extract userID from URL
    userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
    
    // Forward to PHP
    php.For("api/user.php").ServeHTTP(w, r)
})
```

### Using Path Segments

For more complex paths with multiple parameters, you can split the path into segments:

```go
// Handle requests for: /api/categories/{category}/products/{productID}
mux.HandleFunc("GET /api/categories/", func(w http.ResponseWriter, r *http.Request) {
    // Remove the base path
    path := strings.TrimPrefix(r.URL.Path, "/api/categories/")
    
    // Split into segments
    segments := strings.Split(path, "/")
    
    // Check if the structure is valid
    if len(segments) < 3 || segments[1] != "products" {
        http.Error(w, "Invalid URL path", http.StatusBadRequest)
        return
    }
    
    // Extract parameters
    category := segments[0]
    productID := segments[2]
    
    // Forward to PHP
    php.For("api/product.php").ServeHTTP(w, r)
})
```

### Using Regular Expressions

For more complex matching, you can use regular expressions:

```go
// Create a regex to extract parameters
userProductPattern := regexp.MustCompile(`^/api/users/([^/]+)/products/([^/]+)$`)

// Handle requests for: /api/users/{userID}/products/{productID}
mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
    // Match the URL against our pattern
    matches := userProductPattern.FindStringSubmatch(r.URL.Path)
    if matches == nil || len(matches) < 3 {
        // Not a match for this handler
        http.NotFound(w, r)
        return
    }
    
    // Extract parameters
    userID := matches[1]
    productID := matches[2]
    
    // Forward to PHP
    php.For("api/user_product.php").ServeHTTP(w, r)
})
```

## Accessing Path Parameters in PHP

Once you've extracted path parameters in Go and forwarded the request to PHP, you need to access those parameters in your PHP scripts. There are several ways to do this.

### Method 1: Using URL Parsing

The most direct approach is to extract parameters from the URL in PHP:

```php
<?php
// Include helper functions
require_once '../helpers/route.php';

// Extract postID from URL
$postID = extractPathParameter('/api/posts/');

// Validate the parameter
if (!validateId($postID, 'Post ID')) {
    exit;
}

// Now use the postID
$response = [
    'status' => 'success',
    'data' => [
        'postID' => $postID,
        'title' => "Post $postID",
        'content' => "This is the content of post $postID."
    ]
];

// Send the response
sendJsonResponse($response);
?>
```

### Method 2: Using Frango URL Segments

Frango provides URL segment variables in the `$_SERVER` superglobal:

```php
<?php
// Include helper functions
require_once '../helpers/route.php';

// Get the URL segments
// For a URL like /api/posts/123, FRANGO_URL_SEGMENT_1 will be "123"
$postID = $_SERVER['FRANGO_URL_SEGMENT_1'] ?? null;

// Validate the parameter
if (!validateId($postID, 'Post ID')) {
    exit;
}

// Now use the postID
$response = [
    'status' => 'success',
    'data' => [
        'postID' => $postID,
        'title' => "Post $postID",
        'content' => "This is the content of post $postID."
    ]
];

// Send the response
sendJsonResponse($response);
?>
```

### Method 3: Accessing PATH_PARAMS

Frango can pass path parameters through the `$_SERVER['PATH_PARAMS']` variable:

```php
<?php
// Include helper functions
require_once '../helpers/route.php';

// Extract path parameters
$pathParams = null;
if (isset($_SERVER['PATH_PARAMS'])) {
    $pathParams = json_decode($_SERVER['PATH_PARAMS'], true);
}

// Get postID from path parameters
$postID = $pathParams['postID'] ?? null;

// If not found in PATH_PARAMS, try extracting from URL
if ($postID === null) {
    $postID = extractPathParameter('/api/posts/');
}

// Validate the parameter
if (!validateId($postID, 'Post ID')) {
    exit;
}

// Now use the postID
$response = [
    'status' => 'success',
    'data' => [
        'postID' => $postID,
        'title' => "Post $postID",
        'content' => "This is the content of post $postID."
    ]
];

// Send the response
sendJsonResponse($response);
?>
```

## Advanced Path Parameter Techniques

### Working with Optional Parameters

Sometimes you might want to have optional path parameters. Here's how to handle them:

```go
// Handle requests for: /api/posts/{year}/{month?}/{day?}
// Month and day are optional
mux.HandleFunc("GET /api/posts/", func(w http.ResponseWriter, r *http.Request) {
    // Remove the base path
    path := strings.TrimPrefix(r.URL.Path, "/api/posts/")
    
    // Split into segments
    segments := strings.Split(path, "/")
    
    // Extract parameters
    year := segments[0]
    month := ""
    day := ""
    
    if len(segments) > 1 {
        month = segments[1]
    }
    
    if len(segments) > 2 {
        day = segments[2]
    }
    
    // Forward to PHP
    php.For("api/posts_by_date.php").ServeHTTP(w, r)
})
```

In PHP, you can handle these optional parameters:

```php
<?php
// Include helper functions
require_once '../helpers/route.php';

// Extract parameters from URL segments
$year = $_SERVER['FRANGO_URL_SEGMENT_1'] ?? null;
$month = $_SERVER['FRANGO_URL_SEGMENT_2'] ?? null;
$day = $_SERVER['FRANGO_URL_SEGMENT_3'] ?? null;

// Validate year (required)
if (!validateParam($year, 'Year')) {
    exit;
}

// Build query based on available parameters
$query = "SELECT * FROM posts WHERE YEAR(published_at) = :year";
$params = ['year' => $year];

if ($month) {
    $query .= " AND MONTH(published_at) = :month";
    $params['month'] = $month;
}

if ($day) {
    $query .= " AND DAY(published_at) = :day";
    $params['day'] = $day;
}

// In a real app, execute the query here

// For the example, just return the parameters
$response = [
    'status' => 'success',
    'query' => [
        'year' => $year,
        'month' => $month,
        'day' => $day
    ],
    'sql' => $query
];

// Send the response
sendJsonResponse($response);
?>
```

### Handling Wildcards and Catch-All Routes

Sometimes you want to catch all routes under a certain prefix:

```go
// Handle all requests under /api/files/
mux.HandleFunc("GET /api/files/", func(w http.ResponseWriter, r *http.Request) {
    // Extract the file path
    filePath := strings.TrimPrefix(r.URL.Path, "/api/files/")
    
    // Forward to PHP
    php.For("api/file_viewer.php").ServeHTTP(w, r)
})
```

In PHP, you can access the full path:

```php
<?php
// Include helper functions
require_once '../helpers/route.php';

// Get the file path
$filePath = extractPathParameter('/api/files/');

// Validate the path
if (!validateParam($filePath, 'File path')) {
    exit;
}

// Security check - prevent path traversal
if (strpos($filePath, '..') !== false) {
    sendJsonResponse([
        'error' => 'Invalid file path'
    ], 400);
    exit;
}

// In a real app, check if the file exists and serve it
$fullPath = '/path/to/files/' . $filePath;

// For the example, just return the file path
$response = [
    'status' => 'success',
    'file' => [
        'path' => $filePath,
        'fullPath' => $fullPath
    ]
];

// Send the response
sendJsonResponse($response);
?>
```

## Path Parameter Validation

### Validation in Go

It's a good practice to validate path parameters in Go before forwarding the request to PHP:

```go
// Handle requests for: /api/users/{userID}
mux.HandleFunc("GET /api/users/", func(w http.ResponseWriter, r *http.Request) {
    // Extract userID from URL
    userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
    
    // Validate userID
    if userID == "" {
        http.Error(w, "User ID is required", http.StatusBadRequest)
        return
    }
    
    // Check if userID is numeric
    if _, err := strconv.Atoi(userID); err != nil {
        http.Error(w, "User ID must be a number", http.StatusBadRequest)
        return
    }
    
    // Forward to PHP
    php.For("api/user.php").ServeHTTP(w, r)
})
```

### Validation in PHP

Always validate parameters in PHP as well, even if you've already validated them in Go:

```php
<?php
// Include helper functions
require_once '../helpers/route.php';

// Extract userID from URL
$userID = extractPathParameter('/api/users/');

// Validate the parameter
if (!validateId($userID, 'User ID')) {
    exit;
}

// Now use the userID
$response = [
    'status' => 'success',
    'data' => [
        'userID' => $userID,
        'name' => "User $userID",
        'email' => "user$userID@example.com"
    ]
];

// Send the response
sendJsonResponse($response);
?>
```

## Common Patterns and Best Practices

### 1. RESTful Resource Routing

Follow RESTful conventions for your routes:

| HTTP Method | Path                   | Action           |
|-------------|------------------------|------------------|
| GET         | /api/posts             | List all posts   |
| GET         | /api/posts/{id}        | Get post by ID   |
| POST        | /api/posts             | Create new post  |
| PUT         | /api/posts/{id}        | Update post      |
| DELETE      | /api/posts/{id}        | Delete post      |
| GET         | /api/posts/{id}/comments | Get post comments |

### 2. Hierarchical Resources

For nested resources, use hierarchical paths:

```
/api/categories/{categoryID}
/api/categories/{categoryID}/products
/api/categories/{categoryID}/products/{productID}
```

### 3. Clean URLs

Use clean URLs without file extensions:

```
Good: /api/users/123
Bad:  /api/users.php?id=123
```

### 4. Consistent Casing

Use consistent casing for your path segments:

```
Good: /api/blog-posts/recent-comments
Good: /api/blogPosts/recentComments
Bad:  /api/Blog_Posts/Recent_Comments
```

### 5. Parameter Validation

Always validate parameters on both the Go and PHP sides.

## Debugging Path Parameter Issues

### Common Issues and Solutions

1. **Parameter Not Found**: If a parameter is missing, check if the URL matches your expected pattern.
2. **Parameter Parsing Error**: Ensure you're correctly extracting the parameter from the URL.
3. **Route Conflict**: Make sure you don't have overlapping routes or catch-all routes that conflict.

### Debugging Tips

#### Log URL and Parameters

Add logging to help debug path parameter issues:

```go
mux.HandleFunc("GET /api/users/", func(w http.ResponseWriter, r *http.Request) {
    // Log the full URL
    log.Printf("Request URL: %s", r.URL.Path)
    
    // Extract userID from URL
    userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
    log.Printf("Extracted userID: %s", userID)
    
    // Forward to PHP
    php.For("api/user.php").ServeHTTP(w, r)
})
```

In PHP:

```php
<?php
// Log all server variables for debugging
error_log("Request URI: " . $_SERVER['REQUEST_URI']);
error_log("URL Segments: " . json_encode([
    "SEGMENT_0" => $_SERVER['FRANGO_URL_SEGMENT_0'] ?? null,
    "SEGMENT_1" => $_SERVER['FRANGO_URL_SEGMENT_1'] ?? null,
    "SEGMENT_2" => $_SERVER['FRANGO_URL_SEGMENT_2'] ?? null,
]));
```

#### Add Test Routes

Create test routes to verify your parameter extraction logic:

```go
// Test route that just outputs the extracted parameters
mux.HandleFunc("GET /test/", func(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/test/")
    segments := strings.Split(path, "/")
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "path": path,
        "segments": segments,
    })
})
```

By following these guidelines and techniques, you can effectively work with path parameters in your Frango applications, creating clean, RESTful APIs that leverage the power of both Go and PHP.

## Next Steps

- Explore the [JSON Processing Tutorial](./json-processing.md) to learn about working with structured data
- Check out the [Form Handling Tutorial](./form-handling.md) for processing HTML forms
- See the [RESTful API Example](../examples/restful-api.md) for a complete API implementation
- Learn about [Templating](./templating.md) to generate dynamic HTML content 