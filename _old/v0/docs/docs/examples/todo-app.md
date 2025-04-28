# Building a TODO Application with Frango

This tutorial walks through creating a simple TODO list application using Frango. You'll learn how to set up routing, handle forms, implement data persistence, and structure a complete application.

## Table of Contents

- [Building a TODO Application with Frango](#building-a-todo-application-with-frango)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Project Setup](#project-setup)
  - [Application Structure](#application-structure)
  - [Backend Implementation](#backend-implementation)
    - [Go Server](#go-server)
    - [PHP Logic](#php-logic)
  - [Testing the Application](#testing-the-application)
  - [Enhancements](#enhancements)
    - [1. Implementing Task Due Dates](#1-implementing-task-due-dates)
    - [2. Adding Priority Levels](#2-adding-priority-levels)
    - [3. Adding User Authentication](#3-adding-user-authentication)
  - [Conclusion](#conclusion)

## Introduction

In this tutorial, we'll build a simple TODO list application with the following features:

- View all tasks
- Add new tasks
- Mark tasks as complete
- Delete tasks
- Filter tasks by status

This practical example will showcase how Frango enables you to use PHP for templating and business logic while leveraging Go for server-side operations and infrastructure.

## Project Setup

Let's start by setting up our project directory:

```bash
mkdir -p todo-app/{php-files,data}
cd todo-app
```

Create a basic Go module:

```bash
go mod init example.com/todo-app
go get github.com/davidroman0O/go-php/v1@latest
```

## Application Structure

We'll use the following structure for our application:

```
todo-app/
├── data/
│   └── todos.json     # JSON file for storing tasks
├── php-files/
│   ├── index.php      # Main page
│   ├── add.php        # Handler for adding tasks
│   ├── toggle.php     # Handler for toggling task status
│   ├── delete.php     # Handler for deleting tasks
│   ├── lib/
│   │   ├── config.php # Configuration 
│   │   ├── db.php     # Data access functions
│   │   └── utils.php  # Utility functions
│   └── partials/
│       ├── header.php # Common header
│       └── footer.php # Common footer
└── main.go            # Go server
```

## Backend Implementation

### Go Server

Create `main.go` with the following code:

```go
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	frango "github.com/davidroman0O/go-php/v1"
)

func main() {
	// Create data directory if it doesn't exist
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}

	// Create todos.json if it doesn't exist
	todosFile := filepath.Join(dataDir, "todos.json")
	if _, err := os.Stat(todosFile); os.IsNotExist(err) {
		// Initialize with empty todos array
		if err := os.WriteFile(todosFile, []byte("[]"), 0644); err != nil {
			log.Fatalf("Failed to create todos file: %v", err)
		}
	}

	// Initialize Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php-files"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Frango: %v", err)
	}
	defer php.Shutdown()

	// Set up routes
	http.Handle("/", php.For("/index.php"))
	http.Handle("/add", php.For("/add.php"))
	http.Handle("/toggle", php.For("/toggle.php"))
	http.Handle("/delete", php.For("/delete.php"))

	// Start the server
	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

### PHP Logic

Now, let's implement our PHP files:

1. First, create the utility files:

**php-files/lib/config.php**:
```php
<?php
// Configuration settings
define('DATA_DIR', dirname(dirname(__DIR__)) . '/data');
define('TODOS_FILE', DATA_DIR . '/todos.json');
?>
```

**php-files/lib/db.php**:
```php
<?php
require_once __DIR__ . '/config.php';

/**
 * Get all todos from storage
 * 
 * @param string $filter Optional filter: 'all', 'active', 'completed'
 * @return array Array of todo items
 */
function getTodos($filter = 'all') {
    if (!file_exists(TODOS_FILE)) {
        return [];
    }
    
    $content = file_get_contents(TODOS_FILE);
    $todos = json_decode($content, true);
    
    if ($filter === 'active') {
        return array_filter($todos, function($todo) {
            return !$todo['completed'];
        });
    } elseif ($filter === 'completed') {
        return array_filter($todos, function($todo) {
            return $todo['completed'];
        });
    }
    
    return $todos;
}

/**
 * Save todos to storage
 * 
 * @param array $todos Array of todo items
 * @return bool Success status
 */
function saveTodos($todos) {
    $content = json_encode($todos, JSON_PRETTY_PRINT);
    return file_put_contents(TODOS_FILE, $content) !== false;
}

/**
 * Add a new todo
 * 
 * @param string $title Todo title
 * @return bool Success status
 */
function addTodo($title) {
    $todos = getTodos();
    
    $newTodo = [
        'id' => uniqid(),
        'title' => $title,
        'completed' => false,
        'created_at' => date('Y-m-d H:i:s')
    ];
    
    $todos[] = $newTodo;
    return saveTodos($todos);
}

/**
 * Toggle todo completion status
 * 
 * @param string $id Todo ID
 * @return bool Success status
 */
function toggleTodo($id) {
    $todos = getTodos();
    
    foreach ($todos as &$todo) {
        if ($todo['id'] === $id) {
            $todo['completed'] = !$todo['completed'];
            break;
        }
    }
    
    return saveTodos($todos);
}

/**
 * Delete a todo
 * 
 * @param string $id Todo ID
 * @return bool Success status
 */
function deleteTodo($id) {
    $todos = getTodos();
    
    $todos = array_filter($todos, function($todo) use ($id) {
        return $todo['id'] !== $id;
    });
    
    return saveTodos(array_values($todos));
}
?>
```

**php-files/lib/utils.php**:
```php
<?php
/**
 * Safely redirect to a URL
 * 
 * @param string $url URL to redirect to
 * @return void
 */
function redirect($url) {
    header('Location: ' . $url);
    exit;
}

/**
 * Sanitize input
 * 
 * @param string $input Input to sanitize
 * @return string Sanitized input
 */
function sanitizeInput($input) {
    return htmlspecialchars(trim($input), ENT_QUOTES, 'UTF-8');
}
?>
```

2. Next, let's create our layout partials:

**php-files/partials/header.php**:
```php
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Todo App - Frango Example</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #2c3e50;
            text-align: center;
            margin-bottom: 30px;
        }
        .container {
            background-color: #fff;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            padding: 20px;
        }
        .todo-form {
            display: flex;
            margin-bottom: 20px;
        }
        .todo-form input[type="text"] {
            flex: 1;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px 0 0 4px;
            font-size: 16px;
        }
        .todo-form button {
            padding: 10px 15px;
            background-color: #3498db;
            color: white;
            border: none;
            border-radius: 0 4px 4px 0;
            cursor: pointer;
            font-size: 16px;
        }
        .todo-form button:hover {
            background-color: #2980b9;
        }
        .filters {
            display: flex;
            justify-content: center;
            margin-bottom: 20px;
        }
        .filters a {
            margin: 0 10px;
            color: #3498db;
            text-decoration: none;
        }
        .filters a.active {
            font-weight: bold;
        }
        .todo-list {
            list-style: none;
            padding: 0;
        }
        .todo-item {
            display: flex;
            align-items: center;
            padding: 10px;
            border-bottom: 1px solid #eee;
        }
        .todo-item:last-child {
            border-bottom: none;
        }
        .todo-checkbox {
            margin-right: 10px;
        }
        .todo-title {
            flex: 1;
        }
        .todo-title.completed {
            text-decoration: line-through;
            color: #7f8c8d;
        }
        .todo-actions a {
            color: #e74c3c;
            margin-left: 10px;
            text-decoration: none;
        }
        .todo-actions a:hover {
            text-decoration: underline;
        }
        .info-text {
            text-align: center;
            color: #7f8c8d;
            font-style: italic;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Todo App</h1>
```

**php-files/partials/footer.php**:
```php
        <div class="info-text">
            <p>Built with Frango: PHP + Go Integration</p>
        </div>
    </div>
</body>
</html>
```

3. Now, let's implement our main application files:

**php-files/index.php**:
```php
<?php
require_once __DIR__ . '/lib/db.php';
require_once __DIR__ . '/lib/utils.php';

// Determine current filter
$filter = isset($_GET['filter']) ? $_GET['filter'] : 'all';
if (!in_array($filter, ['all', 'active', 'completed'])) {
    $filter = 'all';
}

// Get filtered todos
$todos = getTodos($filter);
?>

<?php include __DIR__ . '/partials/header.php'; ?>

<!-- Add Todo Form -->
<form class="todo-form" action="/add" method="post">
    <input type="text" name="title" placeholder="What needs to be done?" required>
    <button type="submit">Add</button>
</form>

<!-- Filters -->
<div class="filters">
    <a href="/?filter=all" <?php echo $filter === 'all' ? 'class="active"' : ''; ?>>All</a>
    <a href="/?filter=active" <?php echo $filter === 'active' ? 'class="active"' : ''; ?>>Active</a>
    <a href="/?filter=completed" <?php echo $filter === 'completed' ? 'class="active"' : ''; ?>>Completed</a>
</div>

<!-- Todo List -->
<ul class="todo-list">
    <?php if (empty($todos)): ?>
        <li class="todo-item info-text">No todos found</li>
    <?php else: ?>
        <?php foreach ($todos as $todo): ?>
            <li class="todo-item">
                <form action="/toggle" method="post" style="display: inline;">
                    <input type="hidden" name="id" value="<?php echo $todo['id']; ?>">
                    <input type="checkbox" class="todo-checkbox" onchange="this.form.submit()" <?php echo $todo['completed'] ? 'checked' : ''; ?>>
                </form>
                <span class="todo-title <?php echo $todo['completed'] ? 'completed' : ''; ?>">
                    <?php echo sanitizeInput($todo['title']); ?>
                </span>
                <span class="todo-actions">
                    <form action="/delete" method="post" style="display: inline;">
                        <input type="hidden" name="id" value="<?php echo $todo['id']; ?>">
                        <a href="#" onclick="if(confirm('Are you sure?')) this.parentNode.submit(); return false;">Delete</a>
                    </form>
                </span>
            </li>
        <?php endforeach; ?>
    <?php endif; ?>
</ul>

<?php include __DIR__ . '/partials/footer.php'; ?>
```

**php-files/add.php**:
```php
<?php
require_once __DIR__ . '/lib/db.php';
require_once __DIR__ . '/lib/utils.php';

// Only allow POST requests
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $title = isset($_POST['title']) ? trim($_POST['title']) : '';
    
    if (!empty($title)) {
        addTodo($title);
    }
}

// Redirect back to the main page
redirect('/');
?>
```

**php-files/toggle.php**:
```php
<?php
require_once __DIR__ . '/lib/db.php';
require_once __DIR__ . '/lib/utils.php';

// Only allow POST requests
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $id = isset($_POST['id']) ? $_POST['id'] : '';
    
    if (!empty($id)) {
        toggleTodo($id);
    }
}

// Redirect back to the main page
redirect('/');
?>
```

**php-files/delete.php**:
```php
<?php
require_once __DIR__ . '/lib/db.php';
require_once __DIR__ . '/lib/utils.php';

// Only allow POST requests
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $id = isset($_POST['id']) ? $_POST['id'] : '';
    
    if (!empty($id)) {
        deleteTodo($id);
    }
}

// Redirect back to the main page
redirect('/');
?>
```

## Testing the Application

Now that we have all our files in place, let's run our application:

```bash
go run main.go
```

Open your browser and navigate to `http://localhost:8080`. You should see your TODO application with the following functionality:

1. Add new tasks by typing in the input field and clicking "Add"
2. Toggle tasks as complete/incomplete by clicking on the checkbox
3. Delete tasks by clicking the "Delete" link
4. Filter tasks by clicking on the "All", "Active", or "Completed" links

## Enhancements

Here are some potential enhancements you could add to this application:

### 1. Implementing Task Due Dates

Update your database functions and UI to support due dates:

```php
// In db.php, modify addTodo function:
function addTodo($title, $dueDate = null) {
    $todos = getTodos();
    
    $newTodo = [
        'id' => uniqid(),
        'title' => $title,
        'completed' => false,
        'created_at' => date('Y-m-d H:i:s'),
        'due_date' => $dueDate
    ];
    
    $todos[] = $newTodo;
    return saveTodos($todos);
}
```

### 2. Adding Priority Levels

Add priority levels to tasks:

```php
// Add a new field to the todo object
$newTodo = [
    'id' => uniqid(),
    'title' => $title,
    'completed' => false,
    'created_at' => date('Y-m-d H:i:s'),
    'priority' => $priority // 'low', 'medium', 'high'
];
```

### 3. Adding User Authentication

Implement basic authentication to secure the TODO app:

```php
// Create a simple auth.php file
function isAuthenticated() {
    return isset($_SESSION['user_id']);
}

function requireAuth() {
    if (!isAuthenticated()) {
        redirect('/login.php');
    }
}
```

Then include this at the top of your pages to protect them:

```php
<?php
session_start();
require_once __DIR__ . '/lib/auth.php';
requireAuth();
// Rest of your code...
```

## Conclusion

In this tutorial, we've built a simple TODO application using Frango, demonstrating the integration of PHP and Go. You've learned how to:

1. Set up a Frango project
2. Create a RESTful application structure
3. Implement data persistence with JSON files
4. Handle forms and user input
5. Implement filters and navigation
6. Apply a simple but functional UI

This example showcases the power of Frango for building web applications, leveraging the strength of Go for server operations while using PHP for templating and rapid development of business logic.

The application's architecture is clean and maintainable, with separation of concerns between the Go server, PHP business logic, and the presentation layer. This approach makes it easy to extend the application with more features as needed.

For more complex applications, you might consider enhancing this example with:

- Database integration (MySQL, PostgreSQL, etc.)
- User authentication and authorization
- AJAX for asynchronous operations
- More sophisticated UI with JavaScript frameworks
- API endpoints for mobile clients 