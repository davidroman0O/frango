package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davidroman0O/frango"
)

func main() {
	// Create a temporary directory for test files
	webDir, err := createWebDir()
	if err != nil {
		log.Fatalf("Error creating web directory: %v", err)
	}
	defer os.RemoveAll(webDir)

	// Initialize the PHP middleware
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Error initializing Frango: %v", err)
	}
	defer php.Shutdown()

	// Create a fallback handler for non-PHP routes
	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/status" {
			// Handle API status endpoint in Go
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
			return
		}

		// For other non-PHP routes, show a 404 page
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>404 - Not Found</title>
			<style>
				body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
				h1 { color: #e74c3c; }
			</style>
		</head>
		<body>
			<h1>404 Not Found</h1>
			<p>The requested page was not found.</p>
			<p><a href="/">Go to Homepage</a></p>
		</body>
		</html>
		`))
	})

	// Create the middleware router with the fallback handler
	router := frango.NewMiddlewareRouter(php, fallbackHandler)

	// Add the source directory with URL prefix "/"
	err = router.AddSourceDirectory(webDir, "/")
	if err != nil {
		log.Fatalf("Error adding source directory: %v", err)
	}

	// Add parameterized route for user profiles
	err = router.AddRoute("/users/{id}", "/users/profile.php")
	if err != nil {
		log.Fatalf("Error adding parameterized route: %v", err)
	}

	// Start the server
	port := "8080"
	addr := ":" + port

	fmt.Printf("Server running at http://localhost:%s\n", port)
	fmt.Println("Available routes:")
	fmt.Println("  [GET] / => index.php")
	fmt.Println("  [GET] /about => about.php")
	fmt.Println("  [GET] /users => users/index.php")
	fmt.Println("  [GET] /users/{id} => users/profile.php")
	fmt.Println("  [GET] /api/status => Go handler (JSON status)")

	log.Fatal(http.ListenAndServe(addr, router))
}

// createWebDir creates a test web directory with some PHP files
func createWebDir() (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "frango-middleware-example")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}

	// Create test files
	files := map[string]string{
		"index.php": `<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Frango Middleware Router</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #3498db; }
        .nav { background: #f8f9fa; padding: 10px; margin-bottom: 20px; }
        .nav a { margin-right: 10px; }
    </style>
</head>
<body>
    <div class="nav">
        <a href="/">Home</a>
        <a href="/about">About</a>
        <a href="/users">Users</a>
        <a href="/api/status">API Status</a>
    </div>
    
    <h1>Welcome to Frango Middleware Router</h1>
    <p>This is a demonstration of using Frango as a middleware.</p>
    <p>PHP Version: <?= PHP_VERSION ?></p>
    <p>Current time: <?= date('Y-m-d H:i:s') ?></p>
</body>
</html>`,

		"about.php": `<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>About - Frango Middleware Router</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #3498db; }
        .nav { background: #f8f9fa; padding: 10px; margin-bottom: 20px; }
        .nav a { margin-right: 10px; }
    </style>
</head>
<body>
    <div class="nav">
        <a href="/">Home</a>
        <a href="/about">About</a>
        <a href="/users">Users</a>
        <a href="/api/status">API Status</a>
    </div>
    
    <h1>About This Example</h1>
    <p>This example demonstrates how to use Frango as a middleware in Go's HTTP server.</p>
    <p>Key features:</p>
    <ul>
        <li>Standard Go middleware pattern (takes next handler)</li>
        <li>Proper route handling for PHP files</li>
        <li>Fallback to Go handlers for non-PHP routes</li>
        <li>URL prefix support for mounting PHP at different paths</li>
    </ul>
</body>
</html>`,

		"users/index.php": `<?php
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Users - Frango Middleware Router</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #3498db; }
        .nav { background: #f8f9fa; padding: 10px; margin-bottom: 20px; }
        .nav a { margin-right: 10px; }
        .user-card { border: 1px solid #ddd; padding: 10px; margin-bottom: 10px; }
    </style>
</head>
<body>
    <div class="nav">
        <a href="/">Home</a>
        <a href="/about">About</a>
        <a href="/users">Users</a>
        <a href="/api/status">API Status</a>
    </div>
    
    <h1>User Directory</h1>
    
    <div class="user-card">
        <h3>John Doe</h3>
        <p>Email: john@example.com</p>
        <a href="/users/1">View Profile</a>
    </div>
    
    <div class="user-card">
        <h3>Jane Smith</h3>
        <p>Email: jane@example.com</p>
        <a href="/users/2">View Profile</a>
    </div>
    
    <div class="user-card">
        <h3>Bob Johnson</h3>
        <p>Email: bob@example.com</p>
        <a href="/users/3">View Profile</a>
    </div>
</body>
</html>`,

		"users/profile.php": `<?php
header('Content-Type: text/html');

// Get user ID from URL parameter - use both approaches for compatibility
$userId = isset($_PATH['id']) ? $_PATH['id'] : (isset($_SERVER['FRANGO_PARAM_id']) ? $_SERVER['FRANGO_PARAM_id'] : 'unknown');

// Fake user data
$users = [
    '1' => ['name' => 'John Doe', 'email' => 'john@example.com', 'role' => 'Admin'],
    '2' => ['name' => 'Jane Smith', 'email' => 'jane@example.com', 'role' => 'Editor'],
    '3' => ['name' => 'Bob Johnson', 'email' => 'bob@example.com', 'role' => 'User'],
];

$user = isset($users[$userId]) ? $users[$userId] : ['name' => 'Unknown User', 'email' => 'unknown@example.com', 'role' => 'Guest'];
?>
<!DOCTYPE html>
<html>
<head>
    <title>User Profile - Frango Middleware Router</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #3498db; }
        .nav { background: #f8f9fa; padding: 10px; margin-bottom: 20px; }
        .nav a { margin-right: 10px; }
        .profile { background: #f8f9fa; padding: 20px; border-radius: 5px; }
        .debug { background: #f5f5f5; padding: 10px; margin-top: 20px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="nav">
        <a href="/">Home</a>
        <a href="/about">About</a>
        <a href="/users">Users</a>
        <a href="/api/status">API Status</a>
    </div>
    
    <h1>User Profile</h1>
    
    <div class="profile">
        <h2><?= htmlspecialchars($user['name']) ?></h2>
        <p><strong>Email:</strong> <?= htmlspecialchars($user['email']) ?></p>
        <p><strong>Role:</strong> <?= htmlspecialchars($user['role']) ?></p>
        <p><strong>User ID:</strong> <?= htmlspecialchars($userId) ?></p>
    </div>
    
    <div class="debug">
        <h3>$_PATH Parameter (Superglobal)</h3>
        <pre><?php print_r($_PATH ?? []); ?></pre>
        
        <h3>Environment Variables</h3>
        <pre><?php 
        $params = [];
        foreach ($_SERVER as $key => $value) {
            if (strpos($key, 'FRANGO_PARAM_') === 0) {
                $params[$key] = $value;
            }
        }
        print_r($params);
        ?></pre>
        
        <h3>URL Segments</h3>
        <pre><?php print_r($_PATH_SEGMENTS ?? []); ?></pre>
    </div>
    
    <p><a href="/users">&larr; Back to Users</a></p>
</body>
</html>`,
	}

	// Write files to the temp directory
	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)

		// Create parent directories if needed
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "", fmt.Errorf("error creating directory %s: %v", parentDir, err)
		}

		// Write the file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("error writing file %s: %v", fullPath, err)
		}
	}

	return tempDir, nil
}
