# Practical Examples

This guide contains complete, working examples for common use cases with Frango.

## Basic Endpoint Example

A simple endpoint that serves a PHP script.

**Go code (main.go):**

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create HTTP server
	mux := http.NewServeMux()

	// Map routes to PHP scripts
	mux.Handle("/", php.For("index.php"))
	mux.Handle("/about", php.For("about.php"))
	mux.Handle("/users/", php.For("users/{id}.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

**PHP script (php/index.php):**

```php
<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango Example</title>
</head>
<body>
    <h1>Welcome to Frango Example</h1>
    <p>This is a simple PHP script served by Go.</p>
    <p>The current time is: <?= date('Y-m-d H:i:s') ?></p>
    
    <ul>
        <li><a href="/">Home</a></li>
        <li><a href="/about">About</a></li>
        <li><a href="/users/123">User 123</a></li>
    </ul>
</body>
</html>
```

**PHP script (php/users/{id}.php):**

```php
<?php
header('Content-Type: text/html');

// Get user ID from the URL pattern
$userId = $_PATH['id'] ?? 'unknown';
?>
<!DOCTYPE html>
<html>
<head>
    <title>User Profile</title>
</head>
<body>
    <h1>User Profile</h1>
    <p>You are viewing user ID: <?= htmlspecialchars($userId) ?></p>
    
    <a href="/">Back to Home</a>
</body>
</html>
```

## Form Handling Example

A complete form processing example.

**Go code (main.go):**

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create HTTP server
	mux := http.NewServeMux()

	// Map routes
	mux.Handle("/", php.For("form.php"))
	mux.Handle("/process", php.For("process.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

**PHP form (php/form.php):**

```php
<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Contact Form</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; }
        input, textarea, select { width: 100%; padding: 8px; box-sizing: border-box; }
        button { padding: 10px 15px; background: #4CAF50; color: white; border: none; cursor: pointer; }
        .error { color: red; font-size: 0.9em; }
    </style>
</head>
<body>
    <h1>Contact Form</h1>
    
    <?php if (isset($_GET['success'])): ?>
        <div style="background: #dff0d8; color: #3c763d; padding: 10px; margin-bottom: 20px; border-radius: 5px;">
            Your message has been sent successfully!
        </div>
    <?php endif; ?>
    
    <form action="/process" method="POST">
        <div class="form-group">
            <label for="name">Name:</label>
            <input type="text" id="name" name="name" required>
        </div>
        
        <div class="form-group">
            <label for="email">Email:</label>
            <input type="email" id="email" name="email" required>
        </div>
        
        <div class="form-group">
            <label for="subject">Subject:</label>
            <select id="subject" name="subject">
                <option value="general">General Inquiry</option>
                <option value="support">Technical Support</option>
                <option value="feedback">Feedback</option>
                <option value="other">Other</option>
            </select>
        </div>
        
        <div class="form-group">
            <label for="message">Message:</label>
            <textarea id="message" name="message" rows="5" required></textarea>
        </div>
        
        <button type="submit">Send Message</button>
    </form>
</body>
</html>
```

**PHP processor (php/process.php):**

```php
<?php
// Validate form submission
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    // Not a POST request, redirect to the form
    header('Location: /');
    exit;
}

// Initialize errors array
$errors = [];

// Validate name
$name = trim($_POST['name'] ?? '');
if (empty($name)) {
    $errors['name'] = 'Name is required';
}

// Validate email
$email = trim($_POST['email'] ?? '');
if (empty($email)) {
    $errors['email'] = 'Email is required';
} elseif (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
    $errors['email'] = 'Invalid email format';
}

// Get other fields
$subject = $_POST['subject'] ?? 'general';
$message = trim($_POST['message'] ?? '');
if (empty($message)) {
    $errors['message'] = 'Message is required';
}

// If we have errors, show the form again with errors
if (!empty($errors)) {
    // In a real application, you might want to pass the errors back to the form
    // For simplicity, we'll just display them here
    header('Content-Type: text/html');
    echo '<!DOCTYPE html><html><head><title>Form Errors</title></head><body>';
    echo '<h1>Please fix the following errors:</h1><ul>';
    
    foreach ($errors as $field => $error) {
        echo '<li>' . htmlspecialchars($error) . '</li>';
    }
    
    echo '</ul><p><a href="/">Back to the form</a></p></body></html>';
    exit;
}

// Process the form data (in a real app, you might save to a database, send an email, etc.)
// For this example, we'll just redirect back to the form with a success message
header('Location: /?success=1');
exit;
```

## JSON API Example

A simple JSON API example.

**Go code (main.go):**

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create HTTP server
	mux := http.NewServeMux()

	// API endpoints
	mux.Handle("/api/users", php.For("api/users.php"))
	mux.Handle("/api/users/", php.For("api/users/{id}.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

**PHP API endpoint (php/api/users.php):**

```php
<?php
// Set content type to JSON
header('Content-Type: application/json');

// Different behavior based on HTTP method
switch ($_SERVER['REQUEST_METHOD']) {
    case 'GET':
        // List users
        echo json_encode([
            'users' => [
                ['id' => 1, 'name' => 'John Doe', 'email' => 'john@example.com'],
                ['id' => 2, 'name' => 'Jane Smith', 'email' => 'jane@example.com'],
                ['id' => 3, 'name' => 'Bob Johnson', 'email' => 'bob@example.com'],
            ],
            'total' => 3
        ]);
        break;
        
    case 'POST':
        // Create a new user
        // In a real app, you would validate input and save to a database
        
        // Parse the JSON body
        $json = json_decode(file_get_contents('php://input'), true);
        
        if (!$json) {
            http_response_code(400);
            echo json_encode(['error' => 'Invalid JSON body']);
            exit;
        }
        
        // Here we would normally save to a database
        // For this example, we'll just echo back the data with a new ID
        $json['id'] = rand(100, 999);
        
        http_response_code(201); // Created
        echo json_encode([
            'message' => 'User created successfully',
            'user' => $json
        ]);
        break;
        
    default:
        http_response_code(405); // Method Not Allowed
        echo json_encode(['error' => 'Method not allowed']);
        break;
}
```

**PHP API endpoint (php/api/users/{id}.php):**

```php
<?php
// Set content type to JSON
header('Content-Type: application/json');

// Get user ID from path parameter
$userId = $_PATH['id'] ?? null;

if (!$userId) {
    http_response_code(400);
    echo json_encode(['error' => 'User ID is required']);
    exit;
}

// Simple function to simulate fetching a user from a database
function getUser($id) {
    // In a real app, this would query a database
    $users = [
        '1' => ['id' => 1, 'name' => 'John Doe', 'email' => 'john@example.com'],
        '2' => ['id' => 2, 'name' => 'Jane Smith', 'email' => 'jane@example.com'],
        '3' => ['id' => 3, 'name' => 'Bob Johnson', 'email' => 'bob@example.com'],
    ];
    
    return $users[$id] ?? null;
}

// Different behavior based on HTTP method
switch ($_SERVER['REQUEST_METHOD']) {
    case 'GET':
        // Get user details
        $user = getUser($userId);
        
        if (!$user) {
            http_response_code(404);
            echo json_encode(['error' => 'User not found']);
            exit;
        }
        
        echo json_encode(['user' => $user]);
        break;
        
    case 'PUT':
    case 'PATCH':
        // Update user
        $user = getUser($userId);
        
        if (!$user) {
            http_response_code(404);
            echo json_encode(['error' => 'User not found']);
            exit;
        }
        
        // Parse the JSON body
        $json = json_decode(file_get_contents('php://input'), true);
        
        if (!$json) {
            http_response_code(400);
            echo json_encode(['error' => 'Invalid JSON body']);
            exit;
        }
        
        // In a real app, update the user in the database
        // For this example, we'll just merge the data
        $updatedUser = array_merge($user, $json);
        $updatedUser['id'] = (int)$userId; // Ensure ID doesn't change
        
        echo json_encode([
            'message' => 'User updated successfully',
            'user' => $updatedUser
        ]);
        break;
        
    case 'DELETE':
        // Delete user
        $user = getUser($userId);
        
        if (!$user) {
            http_response_code(404);
            echo json_encode(['error' => 'User not found']);
            exit;
        }
        
        // In a real app, delete the user from the database
        echo json_encode([
            'message' => 'User deleted successfully',
            'id' => (int)$userId
        ]);
        break;
        
    default:
        http_response_code(405); // Method Not Allowed
        echo json_encode(['error' => 'Method not allowed']);
        break;
}
```

## Template Data Example

An example showing how to pass data from Go to PHP templates.

**Go code (main.go):**

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create HTTP server
	mux := http.NewServeMux()

	// Dashboard with dynamic data
	mux.Handle("/dashboard", php.Render("dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		// Get this data from wherever - database, API, etc.
		return map[string]interface{}{
			"username": "admin_user",
			"last_login": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"stats": map[string]interface{}{
				"visitors": 12547,
				"page_views": 37842,
				"bounce_rate": 0.42,
			},
			"recent_activities": []map[string]interface{}{
				{
					"action": "New signup",
					"user": "john_doe",
					"timestamp": time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
				},
				{
					"action": "Purchase",
					"user": "jane_smith",
					"timestamp": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
					"amount": 129.99,
				},
				{
					"action": "Password reset",
					"user": "bob_johnson",
					"timestamp": time.Now().Add(-4 * time.Hour).Format(time.RFC3339),
				},
			},
		}
	}))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

**PHP template (php/dashboard.php):**

```php
<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .card { border: 1px solid #ddd; border-radius: 5px; padding: 15px; margin-bottom: 20px; }
        .stats { display: flex; justify-content: space-between; }
        .stat-box { flex: 1; text-align: center; background: #f8f9fa; padding: 15px; margin: 0 5px; border-radius: 5px; }
        .activities { list-style: none; padding: 0; }
        .activity { padding: 10px; border-bottom: 1px solid #eee; }
        .timestamp { color: #6c757d; font-size: 0.8em; }
    </style>
</head>
<body>
    <h1>Welcome, <?= htmlspecialchars($username) ?></h1>
    <p>Last login: <?= htmlspecialchars($last_login) ?></p>
    
    <div class="card">
        <h2>Site Statistics</h2>
        <div class="stats">
            <div class="stat-box">
                <h3>Visitors</h3>
                <div><?= number_format($stats['visitors']) ?></div>
            </div>
            <div class="stat-box">
                <h3>Page Views</h3>
                <div><?= number_format($stats['page_views']) ?></div>
            </div>
            <div class="stat-box">
                <h3>Bounce Rate</h3>
                <div><?= number_format($stats['bounce_rate'] * 100, 2) ?>%</div>
            </div>
        </div>
    </div>
    
    <div class="card">
        <h2>Recent Activities</h2>
        <ul class="activities">
            <?php foreach ($recent_activities as $activity): ?>
                <li class="activity">
                    <strong><?= htmlspecialchars($activity['action']) ?></strong> by 
                    <em><?= htmlspecialchars($activity['user']) ?></em>
                    <?php if (isset($activity['amount'])): ?>
                        - $<?= htmlspecialchars(number_format($activity['amount'], 2)) ?>
                    <?php endif; ?>
                    <div class="timestamp"><?= htmlspecialchars($activity['timestamp']) ?></div>
                </li>
            <?php endforeach; ?>
        </ul>
    </div>
    
    <p>All data is directly passed from Go to PHP.</p>
</body>
</html>
``` 