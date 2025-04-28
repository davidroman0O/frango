# Building a RESTful API with Frango

This example demonstrates how to create a RESTful API using Frango, combining Go's performance with PHP's flexibility for business logic.

## Table of Contents

- [Building a RESTful API with Frango](#building-a-restful-api-with-frango)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Project Structure](#project-structure)
  - [Go Server Implementation](#go-server-implementation)
  - [PHP API Implementation](#php-api-implementation)
  - [Running the API](#running-the-api)
  - [Testing the API](#testing-the-api)
    - [List all books](#list-all-books)
    - [Get a specific book](#get-a-specific-book)
    - [Create a new book](#create-a-new-book)
    - [Update a book](#update-a-book)
    - [Delete a book](#delete-a-book)
  - [Next Steps](#next-steps)

## Overview

In this example, we'll build a simple RESTful API for managing a collection of books. The API will:

1. Handle standard CRUD operations (Create, Read, Update, Delete)
2. Accept and return JSON data
3. Follow RESTful principles for endpoint design
4. Implement proper error handling
5. Support basic filtering and pagination

This approach combines Go's strong HTTP handling with PHP's ease of use for implementing business logic.

## Project Structure

```
books-api/
├── main.go                 # Go application entry point
├── data/                   # Simple data storage
│   └── books.json          # JSON file to store book data
├── php-files/              # PHP implementation files
│   ├── api/
│   │   ├── books.php       # Endpoint for collection operations
│   │   └── book.php        # Endpoint for individual book operations
│   ├── includes/
│   │   ├── responses.php   # Helper functions for API responses
│   │   ├── request.php     # Helper functions for request handling
│   │   └── database.php    # Simple database abstraction for books
│   └── utils/
│       └── validation.php  # Data validation utilities
```

## Go Server Implementation

First, let's create the Go server that will handle HTTP requests and route them to the appropriate PHP files:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/davidroman0O/frango"
)

func main() {
	// Create necessary directories and files
	setupProjectStructure()

	// Create Frango instance
	php, err := frango.New(
		frango.WithSourceDir("./php-files"),
		frango.WithMaxExecutionTime(30),
		frango.WithPHPIniSettings(map[string]string{
			"display_errors": "1",
			"error_reporting": "E_ALL",
		}),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Create a mux for routing
	mux := http.NewServeMux()

	// API endpoints
	// Collection endpoint (GET all, POST new)
	mux.Handle("GET /api/books", php.For("api/books.php"))
	mux.Handle("POST /api/books", php.For("api/books.php"))

	// Individual resource endpoints (GET, PUT, DELETE)
	mux.Handle("GET /api/books/", customHandler(php, "api/book.php"))
	mux.Handle("PUT /api/books/", customHandler(php, "api/book.php"))
	mux.Handle("DELETE /api/books/", customHandler(php, "api/book.php"))

	// Set up the server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start the server
	log.Println("Server starting on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// customHandler creates a handler that extracts the ID from the URL and passes it to PHP
func customHandler(php *frango.PHP, scriptPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract book ID from URL path
		id := strings.TrimPrefix(r.URL.Path, "/api/books/")
		
		// Add the ID to the request context so PHP can access it
		pathParams := map[string]string{
			"id": id,
		}
		
		// Create new context with path parameters
		ctx := context.WithValue(r.Context(), "path_params", pathParams)
		r = r.WithContext(ctx)
		
		// Forward to PHP script with the enhanced request
		php.For(scriptPath).ServeHTTP(w, r)
	})
}

// setupProjectStructure creates the necessary directories and files
func setupProjectStructure() {
	// Create directories
	os.MkdirAll("./data", 0755)
	os.MkdirAll("./php-files/api", 0755)
	os.MkdirAll("./php-files/includes", 0755)
	os.MkdirAll("./php-files/utils", 0755)

	// Create empty books.json if it doesn't exist
	if _, err := os.Stat("./data/books.json"); os.IsNotExist(err) {
		initialBooks := `[
			{
				"id": 1,
				"title": "The Go Programming Language",
				"author": "Alan A. A. Donovan, Brian W. Kernighan",
				"year": 2015,
				"isbn": "978-0134190440"
			},
			{
				"id": 2,
				"title": "PHP 8 in Action",
				"author": "John Smith",
				"year": 2022,
				"isbn": "978-1234567890"
			}
		]`
		os.WriteFile("./data/books.json", []byte(initialBooks), 0644)
	}
}
```

## PHP API Implementation

Now, let's create the PHP files that implement the API logic.

First, let's create the response helpers in `php-files/includes/responses.php`:

```php
<?php
/**
 * Helper functions for API responses
 */

/**
 * Send a JSON response
 */
function sendJsonResponse($data, $statusCode = 200) {
    http_response_code($statusCode);
    header('Content-Type: application/json');
    echo json_encode($data);
    exit;
}

/**
 * Send an error response
 */
function sendErrorResponse($message, $statusCode = 400) {
    sendJsonResponse([
        'error' => true,
        'message' => $message
    ], $statusCode);
}
```

Next, let's create a simple database abstraction in `php-files/includes/database.php`:

```php
<?php
/**
 * Simple database abstraction for books
 */

function getBooks($filters = []) {
    // Read books from JSON file
    $booksJson = file_get_contents(__DIR__ . '/../../data/books.json');
    $books = json_decode($booksJson, true);
    
    // Apply filters if any
    if (!empty($filters)) {
        $filteredBooks = [];
        foreach ($books as $book) {
            $match = true;
            foreach ($filters as $key => $value) {
                if (!isset($book[$key]) || $book[$key] != $value) {
                    $match = false;
                    break;
                }
            }
            if ($match) {
                $filteredBooks[] = $book;
            }
        }
        return $filteredBooks;
    }
    
    return $books;
}

function getBookById($id) {
    $books = getBooks();
    
    foreach ($books as $book) {
        if ($book['id'] == $id) {
            return $book;
        }
    }
    
    return null;
}

function saveBooks($books) {
    file_put_contents(__DIR__ . '/../../data/books.json', json_encode($books, JSON_PRETTY_PRINT));
}

function createBook($bookData) {
    $books = getBooks();
    
    // Generate a new ID
    $maxId = 0;
    foreach ($books as $book) {
        if ($book['id'] > $maxId) {
            $maxId = $book['id'];
        }
    }
    
    $bookData['id'] = $maxId + 1;
    $books[] = $bookData;
    
    saveBooks($books);
    return $bookData;
}

function updateBook($id, $bookData) {
    $books = getBooks();
    $updated = false;
    
    foreach ($books as &$book) {
        if ($book['id'] == $id) {
            // Preserve the ID
            $bookData['id'] = $id;
            $book = $bookData;
            $updated = true;
            break;
        }
    }
    
    if ($updated) {
        saveBooks($books);
        return $bookData;
    }
    
    return null;
}

function deleteBook($id) {
    $books = getBooks();
    $found = false;
    
    $newBooks = [];
    foreach ($books as $book) {
        if ($book['id'] == $id) {
            $found = true;
        } else {
            $newBooks[] = $book;
        }
    }
    
    if ($found) {
        saveBooks($newBooks);
        return true;
    }
    
    return false;
}
```

Now, let's create a simple request helper in `php-files/includes/request.php`:

```php
<?php
/**
 * Helper functions for request handling
 */

/**
 * Get JSON data from request body
 */
function getJsonInput() {
    $inputJSON = file_get_contents('php://input');
    return json_decode($inputJSON, true);
}

/**
 * Get path parameters from the request
 */
function getPathParams() {
    if (isset($_SERVER['PATH_PARAMS'])) {
        return json_decode($_SERVER['PATH_PARAMS'], true);
    }
    return [];
}

/**
 * Get a specific path parameter by name
 */
function getPathParam($name, $default = null) {
    $params = getPathParams();
    return $params[$name] ?? $default;
}
```

Now, let's implement the main API endpoints. First, the collection endpoint in `php-files/api/books.php`:

```php
<?php
require_once __DIR__ . '/../includes/responses.php';
require_once __DIR__ . '/../includes/database.php';
require_once __DIR__ . '/../includes/request.php';

// Set JSON content type
header('Content-Type: application/json');

// Handle request based on method
$method = $_SERVER['REQUEST_METHOD'];

switch ($method) {
    case 'GET':
        // Get all books, with optional filtering
        $filters = [];
        
        // Apply any filters from query params
        if (isset($_GET['year'])) {
            $filters['year'] = (int)$_GET['year'];
        }
        
        if (isset($_GET['author'])) {
            $filters['author'] = $_GET['author'];
        }
        
        $books = getBooks($filters);
        
        // Handle pagination
        $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
        $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 10;
        
        $total = count($books);
        $offset = ($page - 1) * $limit;
        $books = array_slice($books, $offset, $limit);
        
        sendJsonResponse([
            'books' => $books,
            'page' => $page,
            'limit' => $limit,
            'total' => $total
        ]);
        break;
        
    case 'POST':
        // Create a new book
        $data = getJsonInput();
        
        // Validate required fields
        if (!isset($data['title']) || !isset($data['author'])) {
            sendErrorResponse('Title and author are required', 400);
        }
        
        $newBook = createBook($data);
        sendJsonResponse(['book' => $newBook], 201);
        break;
        
    default:
        sendErrorResponse('Method not allowed', 405);
}
```

Finally, let's implement the individual resource endpoint in `php-files/api/book.php`:

```php
<?php
require_once __DIR__ . '/../includes/responses.php';
require_once __DIR__ . '/../includes/database.php';
require_once __DIR__ . '/../includes/request.php';

// Set JSON content type
header('Content-Type: application/json');

// Get the book ID from path parameters
$id = getPathParam('id');

if (!$id || !is_numeric($id)) {
    sendErrorResponse('Invalid book ID', 400);
}

// Handle request based on method
$method = $_SERVER['REQUEST_METHOD'];

switch ($method) {
    case 'GET':
        // Get a single book by ID
        $book = getBookById($id);
        
        if (!$book) {
            sendErrorResponse('Book not found', 404);
        }
        
        sendJsonResponse(['book' => $book]);
        break;
        
    case 'PUT':
        // Update a book
        $data = getJsonInput();
        
        // Validate required fields
        if (!isset($data['title']) || !isset($data['author'])) {
            sendErrorResponse('Title and author are required', 400);
        }
        
        $updatedBook = updateBook($id, $data);
        
        if (!$updatedBook) {
            sendErrorResponse('Book not found', 404);
        }
        
        sendJsonResponse(['book' => $updatedBook]);
        break;
        
    case 'DELETE':
        // Delete a book
        $success = deleteBook($id);
        
        if (!$success) {
            sendErrorResponse('Book not found', 404);
        }
        
        sendJsonResponse(['message' => 'Book deleted successfully'], 200);
        break;
        
    default:
        sendErrorResponse('Method not allowed', 405);
}
```

## Running the API

To run this example:

1. Create a new directory for your project
2. Create the files as described above
3. Initialize a new Go module:
   ```
   go mod init books-api
   ```
4. Install the Frango package:
   ```
   go get github.com/davidroman0O/frango
   ```
5. Run the server:
   ```
   go run main.go
   ```

The API will be available at `http://localhost:8080/api/books`.

## Testing the API

Here are some cURL commands to test the API:

### List all books
```sh
curl -X GET http://localhost:8080/api/books
```

### Get a specific book
```sh
curl -X GET http://localhost:8080/api/books/1
```

### Create a new book
```sh
curl -X POST http://localhost:8080/api/books \
  -H "Content-Type: application/json" \
  -d '{"title": "Learning PHP, MySQL & JavaScript", "author": "Robin Nixon", "year": 2018, "isbn": "978-1491978917"}'
```

### Update a book
```sh
curl -X PUT http://localhost:8080/api/books/1 \
  -H "Content-Type: application/json" \
  -d '{"title": "The Go Programming Language", "author": "Alan A. A. Donovan, Brian W. Kernighan", "year": 2016, "isbn": "978-0134190440"}'
```

### Delete a book
```sh
curl -X DELETE http://localhost:8080/api/books/2
```

## Next Steps

To enhance this API, you might consider:

1. **Authentication**: Add JWT-based authentication for secure endpoints
2. **Validation**: Implement more robust input validation
3. **Database Integration**: Replace the JSON file with a real database 
4. **Rate Limiting**: Add rate limiting to prevent abuse
5. **Versioning**: Implement API versioning (e.g., /api/v1/books)
6. **Documentation**: Generate API documentation using tools like Swagger
7. **Caching**: Implement response caching for improved performance
8. **Logging**: Add detailed logging for API requests and errors 