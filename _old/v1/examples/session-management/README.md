# Session Management Example

This example demonstrates how to use PHP sessions with Frango to maintain state across requests, implement user authentication, and create protected routes.

## What This Example Shows

- Starting and accessing PHP sessions
- Storing data in sessions
- User authentication using sessions
- Role-based access control
- Protected routes that require authentication
- Session persistence across multiple requests
- Session destruction (logout)

## Files

- `main.go`: Go HTTP server with session-enabled routes
- `php/index.php`: Home page with session information and navigation
- `php/login.php`: Login form
- `php/process-login.php`: Processes login form and creates session
- `php/dashboard.php`: User dashboard (requires login)
- `php/counter.php`: Simple example of persistent state with sessions
- `php/admin.php`: Admin panel with role-based access control
- `php/logout.php`: Session destruction and logout functionality

## Features

### Authentication Flow

1. User submits login credentials
2. Server validates credentials
3. On successful login, user info is stored in `$_SESSION['user']`
4. Protected routes check for presence of session data
5. Logout destroys the session

### Session Counter

A simple demonstration of maintaining state across requests using a counter that increments on each page visit.

### Role-Based Access Control

- Different user roles (admin, user) with different access levels
- Admin-only pages that verify role before granting access
- Automatic redirection for unauthorized access attempts

## Demo Users

For demonstration purposes, this example includes two pre-configured users:

| Username | Password | Role  |
|----------|----------|-------|
| admin    | password | admin |
| user     | password | user  |

## Running the Example

1. Navigate to this directory
2. Run:
   ```bash
   go run main.go
   ```
3. Open a browser and visit http://localhost:8080
4. Try logging in with different users and accessing various protected pages

## Important Code Snippets

```php
// Start a PHP session
session_start();

// Store data in session
$_SESSION['user'] = [
    'username' => $username,
    'role' => $role
];

// Retrieve data from session
$user = $_SESSION['user'];

// Protect a route
if (!isset($_SESSION['user'])) {
    header('Location: /login');
    exit;
}

// Role-based access control
if ($user['role'] !== 'admin') {
    // Show access denied
    exit;
}

// Destroy session (logout)
$_SESSION = array();
session_destroy();
``` 