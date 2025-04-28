# Form Handling in Frango

This tutorial covers how to handle form submissions in a Frango application, including GET and POST requests, data validation, and processing.

## Overview

Frango provides a seamless way to handle HTML forms by:

1. Automatically converting form data to PHP superglobals
2. Supporting various form types (GET, POST, multipart)
3. Allowing processing in either PHP or Go

## Basic Form Handling

Let's build a simple application with form handling capabilities.

### Project Setup

Create a directory structure for our form handling example:

```
myapp/
├── main.go           # Go application entry point
├── templates/
│   ├── form.php      # Form template
│   └── result.php    # Form result template
```

### Creating the Form Template

Create `templates/form.php` with the following content:

```php
<!DOCTYPE html>
<html>
<head>
    <title>Frango Form Handling</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input, textarea, select {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 15px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        .tabs {
            display: flex;
            margin-bottom: 20px;
        }
        .tab {
            padding: 10px 20px;
            background-color: #f0f0f0;
            border: 1px solid #ddd;
            cursor: pointer;
        }
        .tab.active {
            background-color: #fff;
            border-bottom: none;
        }
        .tab-content {
            display: none;
            border: 1px solid #ddd;
            padding: 20px;
        }
        .tab-content.active {
            display: block;
        }
    </style>
</head>
<body>
    <h1>Frango Form Handling</h1>
    
    <div class="tabs">
        <div class="tab active" onclick="showTab('get-form')">GET Form</div>
        <div class="tab" onclick="showTab('post-form')">POST Form</div>
    </div>
    
    <div id="get-form" class="tab-content active">
        <h2>GET Form Example</h2>
        <p>GET forms append data to the URL as query parameters, making them visible and bookmarkable.</p>
        
        <form action="/process-get" method="GET">
            <div class="form-group">
                <label for="get-name">Name:</label>
                <input type="text" id="get-name" name="name" required>
            </div>
            
            <div class="form-group">
                <label for="get-email">Email:</label>
                <input type="email" id="get-email" name="email" required>
            </div>
            
            <div class="form-group">
                <label for="get-category">Category:</label>
                <select id="get-category" name="category">
                    <option value="general">General</option>
                    <option value="support">Support</option>
                    <option value="feedback">Feedback</option>
                </select>
            </div>
            
            <button type="submit">Submit GET Form</button>
        </form>
    </div>
    
    <div id="post-form" class="tab-content">
        <h2>POST Form Example</h2>
        <p>POST forms send data in the request body, making them suitable for sensitive or large amounts of data.</p>
        
        <form action="/process-post" method="POST">
            <div class="form-group">
                <label for="post-name">Name:</label>
                <input type="text" id="post-name" name="name" required>
            </div>
            
            <div class="form-group">
                <label for="post-email">Email:</label>
                <input type="email" id="post-email" name="email" required>
            </div>
            
            <div class="form-group">
                <label for="post-message">Message:</label>
                <textarea id="post-message" name="message" rows="5" required></textarea>
            </div>
            
            <div class="form-group">
                <label for="post-priority">Priority:</label>
                <select id="post-priority" name="priority">
                    <option value="low">Low</option>
                    <option value="medium">Medium</option>
                    <option value="high">High</option>
                </select>
            </div>
            
            <button type="submit">Submit POST Form</button>
        </form>
    </div>
    
    <script>
    function showTab(tabId) {
        // Hide all tab contents
        var tabContents = document.getElementsByClassName('tab-content');
        for (var i = 0; i < tabContents.length; i++) {
            tabContents[i].classList.remove('active');
        }
        
        // Remove active class from all tabs
        var tabs = document.getElementsByClassName('tab');
        for (var i = 0; i < tabs.length; i++) {
            tabs[i].classList.remove('active');
        }
        
        // Show the selected tab content and activate the tab
        document.getElementById(tabId).classList.add('active');
        
        // Find the tab that was clicked and add active class
        var allTabs = document.getElementsByClassName('tab');
        for (var i = 0; i < allTabs.length; i++) {
            if (allTabs[i].textContent.toLowerCase().includes(tabId.split('-')[0])) {
                allTabs[i].classList.add('active');
            }
        }
    }
    </script>
</body>
</html>
```

### Creating the Result Template

Create `templates/result.php` to display form submission results:

```php
<!DOCTYPE html>
<html>
<head>
    <title>Form Submission Result</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .result {
            background-color: #f9f9f9;
            border: 1px solid #ddd;
            padding: 20px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        table, th, td {
            border: 1px solid #ddd;
        }
        th, td {
            padding: 10px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        .method {
            display: inline-block;
            padding: 5px 10px;
            border-radius: 3px;
            font-weight: bold;
            color: white;
        }
        .method-get {
            background-color: #3498db;
        }
        .method-post {
            background-color: #2ecc71;
        }
        .back-link {
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <h1>Form Submission Result</h1>
    
    <div class="result">
        <h2>
            <?php if ($_SERVER['REQUEST_METHOD'] === 'GET'): ?>
                <span class="method method-get">GET</span>
            <?php else: ?>
                <span class="method method-post">POST</span>
            <?php endif; ?>
            Form Data Received
        </h2>
        
        <?php if ($_SERVER['REQUEST_METHOD'] === 'GET'): ?>
            <!-- Display GET data -->
            <h3>GET Parameters:</h3>
            <?php if (count($_GET) > 0): ?>
                <table>
                    <tr>
                        <th>Field Name</th>
                        <th>Value</th>
                    </tr>
                    <?php foreach ($_GET as $key => $value): ?>
                        <tr>
                            <td><?= htmlspecialchars($key) ?></td>
                            <td><?= htmlspecialchars($value) ?></td>
                        </tr>
                    <?php endforeach; ?>
                </table>
            <?php else: ?>
                <p>No GET parameters received.</p>
            <?php endif; ?>
        <?php endif; ?>
        
        <?php if ($_SERVER['REQUEST_METHOD'] === 'POST'): ?>
            <!-- Display POST data -->
            <h3>POST Data:</h3>
            <?php if (count($_POST) > 0): ?>
                <table>
                    <tr>
                        <th>Field Name</th>
                        <th>Value</th>
                    </tr>
                    <?php foreach ($_POST as $key => $value): ?>
                        <tr>
                            <td><?= htmlspecialchars($key) ?></td>
                            <td><?= htmlspecialchars($value) ?></td>
                        </tr>
                    <?php endforeach; ?>
                </table>
            <?php else: ?>
                <p>No POST data received.</p>
            <?php endif; ?>
        <?php endif; ?>
        
        <!-- Show all available data -->
        <h3>Additional Information:</h3>
        <table>
            <tr>
                <th>Item</th>
                <th>Value</th>
            </tr>
            <tr>
                <td>Request Method</td>
                <td><?= $_SERVER['REQUEST_METHOD'] ?></td>
            </tr>
            <tr>
                <td>Request URI</td>
                <td><?= $_SERVER['REQUEST_URI'] ?></td>
            </tr>
            <tr>
                <td>Form Data Count</td>
                <td>
                    GET: <?= count($_GET) ?>,
                    POST: <?= count($_POST) ?>,
                    REQUEST: <?= count($_REQUEST) ?>
                </td>
            </tr>
            <?php if (isset($_FORM)): ?>
                <tr>
                    <td>Frango Form Data Count</td>
                    <td><?= count($_FORM) ?></td>
                </tr>
            <?php endif; ?>
        </table>
    </div>
    
    <div class="back-link">
        <a href="/">Back to Form</a>
    </div>
</body>
</html>
```

### Creating the Go Application

Create `main.go` to set up the web server and Frango middleware:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Initialize Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./templates"),
		frango.WithDevelopmentMode(true),
	)

	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Root route shows the form
	http.Handle("/", php.For("/form.php"))

	// Handle GET form submissions
	http.Handle("/process-get", php.For("/result.php"))

	// Handle POST form submissions
	http.Handle("/process-post", php.For("/result.php"))

	// Alternative: Process form in Go instead of PHP
	http.HandleFunc("/process-go", func(w http.ResponseWriter, r *http.Request) {
		// Parse form data
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
				return
			}
		}

		// Process the form data in Go
		fmt.Fprintf(w, "Received form data (processed by Go):\n")
		if r.Method == "GET" {
			for key, values := range r.URL.Query() {
				fmt.Fprintf(w, "- %s: %s\n", key, values[0])
			}
		} else {
			for key, values := range r.PostForm {
				fmt.Fprintf(w, "- %s: %s\n", key, values[0])
			}
		}
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on http://localhost%s", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, nil))
}
```

### Running the Application

Build and run your application:

```bash
go mod tidy
go run main.go
```

Visit `http://localhost:8080` in your browser to access the form. Try submitting both the GET and POST forms to see how they work.

## How Form Handling Works in Frango

Let's dive deeper into how form handling works in Frango.

### GET Form Processing

When a GET form is submitted:

1. The browser creates a URL with query parameters (e.g., `/process-get?name=John&email=john@example.com&category=feedback`)
2. Frango extracts these parameters and creates PHP environment variables with the prefix `PHP_QUERY_`
3. These variables are mapped to the `$_GET` superglobal in PHP

In the result page, you can access the submitted data using:

```php
$name = $_GET['name'];
$email = $_GET['email'];
$category = $_GET['category'];
```

### POST Form Processing

When a POST form is submitted:

1. The browser sends the form data in the request body
2. Frango extracts this data and creates PHP environment variables with the prefix `PHP_FORM_`
3. These variables are mapped to the `$_POST` superglobal in PHP

In the result page, you can access the submitted data using:

```php
$name = $_POST['name'];
$email = $_POST['email'];
$message = $_POST['message'];
$priority = $_POST['priority'];
```

### Frango-Specific Superglobals

Frango provides additional superglobals that can be useful:

- `$_FORM`: Contains form data regardless of the HTTP method (works with both GET and POST)
- `$_REQUEST`: Standard PHP superglobal that combines `$_GET`, `$_POST`, and `$_COOKIE`

These can be useful for handling form submissions in a method-agnostic way.

## Form Validation

For form validation, you can use PHP's built-in capabilities. Here's an example of how to validate a simple form:

```php
<?php
// Initialize errors array
$errors = [];

// Check if the form was submitted
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    // Validate name
    if (empty($_POST['name'])) {
        $errors['name'] = 'Name is required';
    }
    
    // Validate email
    if (empty($_POST['email'])) {
        $errors['email'] = 'Email is required';
    } elseif (!filter_var($_POST['email'], FILTER_VALIDATE_EMAIL)) {
        $errors['email'] = 'Please enter a valid email address';
    }
    
    // If no errors, process the form
    if (empty($errors)) {
        // Process the form data...
        $success = true;
    }
}
?>
```

## Processing Form Data in Go

You can also choose to process form data in Go instead of PHP. This is useful for complex business logic or when integrating with Go-based services.

```go
http.HandleFunc("/process-form", func(w http.ResponseWriter, r *http.Request) {
    // Parse form data
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }
    
    // Validate form data
    name := r.FormValue("name")
    email := r.FormValue("email")
    
    if name == "" || email == "" {
        http.Error(w, "Name and email are required", http.StatusBadRequest)
        return
    }
    
    // Process the data (e.g., save to database)
    // ...
    
    // Render a PHP template with the results
    php.Render("/success.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
        return map[string]interface{}{
            "name": name,
            "email": email,
            "processed": true,
        }
    }).ServeHTTP(w, r)
})
```

## Advanced: Handling Form Arrays

HTML forms can submit array values using square brackets in field names. Frango handles these correctly. For example:

```html
<input type="checkbox" name="interests[]" value="technology"> Technology
<input type="checkbox" name="interests[]" value="sports"> Sports
<input type="checkbox" name="interests[]" value="arts"> Arts
```

In PHP, you can access these as arrays:

```php
$interests = $_POST['interests']; // Array of selected interests
```

## Troubleshooting Form Handling

If you encounter issues with form handling, check these common problems:

1. **Missing Form Data**:
   - Ensure the `Content-Type` header is set correctly (automatically done by browsers)
   - For POST forms, verify you're using `method="POST"` in the HTML

2. **Character Encoding Issues**:
   - Add `accept-charset="UTF-8"` to your form tag
   - Check that PHP is configured to use UTF-8

3. **File Upload Issues**:
   - Make sure your form includes `enctype="multipart/form-data"` for file uploads
   - See the [File Uploads tutorial](./file-uploads.md) for more details

4. **Frango-Specific Issues**:
   - If superglobals are not populated, include the `globals_fix.php` helper:
     ```php
     <?php
     // At the top of your PHP file
     require_once 'globals_fix.php';
     ?>
     ```

## Conclusion

Frango makes form handling straightforward by automatically connecting HTTP form submissions to PHP's standard superglobals. This allows you to use familiar PHP techniques for form processing while leveraging Go's performance and concurrency features for your application as a whole.

For more advanced form handling, including file uploads, see the [File Uploads tutorial](./file-uploads.md).

For handling JSON data instead of form data, see the [JSON Processing tutorial](./json-processing.md). 