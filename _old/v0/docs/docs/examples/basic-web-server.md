# Building a Basic Web Server with Frango

This example demonstrates how to create a complete basic web server using Frango, combining Go's performance with PHP's ease of use for templating and business logic.

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Go Server Implementation](#go-server-implementation)
- [PHP Templates and Logic](#php-templates-and-logic)
- [Static Files](#static-files)
- [Running the Server](#running-the-server)
- [Next Steps](#next-steps)

## Overview

In this example, we'll build a simple web server that:

1. Handles routing for different pages
2. Uses PHP templates for rendering HTML
3. Implements a contact form with validation
4. Serves static assets
5. Includes basic error handling

The example demonstrates how Frango allows you to leverage Go's strong HTTP server capabilities while using PHP for templating and business logic.

## Project Structure

Let's start by setting up our project structure:

```
basic-web-server/
├── main.go                   # Go application entry point
├── static/                   # Static files
│   ├── css/
│   │   └── style.css
│   ├── js/
│   │   └── main.js
│   └── images/
│       └── logo.png
└── php-files/                # PHP files
    ├── templates/            # Templates
    │   ├── layout.php        # Main layout template
    │   ├── home.php          # Home page template
    │   ├── about.php         # About page template
    │   ├── contact.php       # Contact page template
    │   └── partials/         # Partial templates
    │       ├── header.php
    │       ├── footer.php
    │       └── nav.php
    ├── includes/             # Includes
    │   └── helpers.php       # Helper functions
    └── controllers/          # Controllers
        ├── home.php          # Home page controller
        ├── about.php         # About page controller
        └── contact.php       # Contact page controller
```

## Go Server Implementation

Let's start by creating the main Go application that will handle HTTP requests and route them to the appropriate PHP handlers.

### `main.go`

```go
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/davidroman0O/go-php/v1"
)

func main() {
	// Configure the Frango middleware
	php, err := frango.New(
		frango.WithSourceDir("./php-files"),
		frango.WithDevelopmentMode(true),
		frango.WithPHPIniSettings(map[string]string{
			"display_errors": "On",
			"error_reporting": "E_ALL",
			"date.timezone": "UTC",
		}),
		frango.WithErrorHandler(func(err error) {
			log.Printf("PHP Error: %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Frango: %v", err)
	}
	defer php.Shutdown()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Define routes
	setupRoutes(php)

	// Create and configure the server
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func setupRoutes(php *frango.PHP) {
	// Home page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle the root path
		if r.URL.Path != "/" {
			notFound(w, r, php)
			return
		}
		
		// Serve the home page
		php.For("/controllers/home.php").ServeHTTP(w, r)
	})

	// About page
	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		php.For("/controllers/about.php").ServeHTTP(w, r)
	})

	// Contact page
	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		php.For("/controllers/contact.php").ServeHTTP(w, r)
	})
}

func notFound(w http.ResponseWriter, r *http.Request, php *frango.PHP) {
	// Render the 404 page using PHP
	php.Render("/templates/error.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"code": 404,
			"message": "Page not found",
			"requestPath": r.URL.Path,
		}
	}).ServeHTTP(w, r)
}
```

## PHP Templates and Logic

Now let's create the PHP files that will handle the rendering and business logic of our web server.

### Helper Functions

First, let's create some helper functions in `php-files/includes/helpers.php`:

```php
<?php
/**
 * Helper functions for the basic web server example
 */

/**
 * HTML escape a string
 *
 * @param string $string The string to escape
 * @return string The escaped string
 */
function h($string) {
    return htmlspecialchars($string, ENT_QUOTES, 'UTF-8');
}

/**
 * Get the current page URL
 *
 * @return string The current URL
 */
function current_url() {
    $scheme = isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? 'https' : 'http';
    return $scheme . '://' . $_SERVER['HTTP_HOST'] . $_SERVER['REQUEST_URI'];
}

/**
 * Check if a URL path is active
 *
 * @param string $path The path to check
 * @return bool True if the path is active
 */
function is_active($path) {
    return $_SERVER['REQUEST_URI'] === $path;
}

/**
 * Format a date
 *
 * @param string $date The date string
 * @param string $format The format string
 * @return string The formatted date
 */
function format_date($date, $format = 'Y-m-d H:i:s') {
    return date($format, strtotime($date));
}

/**
 * Include a template partial
 *
 * @param string $name The partial name
 * @param array $data Optional data to pass to the partial
 * @return void
 */
function partial($name, $data = []) {
    // Extract data variables
    if (!empty($data)) {
        extract($data);
    }
    
    // Include the partial
    include __DIR__ . "/../templates/partials/$name.php";
}

/**
 * Render a template with layout
 * 
 * @param string $template The template name
 * @param array $data Optional data to pass to the template
 * @return void
 */
function render($template, $data = []) {
    // Extract data variables
    if (!empty($data)) {
        extract($data);
    }
    
    // Start output buffering
    ob_start();
    
    // Include the template
    include __DIR__ . "/../templates/$template.php";
    
    // Get the content
    $content = ob_get_clean();
    
    // Include the layout with the content
    include __DIR__ . "/../templates/layout.php";
}
```

### Main Layout

Next, let's create the main layout in `php-files/templates/layout.php`:

```php
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title><?= isset($title) ? h($title) : 'Basic Web Server Example' ?></title>
    <link rel="stylesheet" href="/static/css/style.css">
    <?php if (isset($meta_description)): ?>
    <meta name="description" content="<?= h($meta_description) ?>">
    <?php endif; ?>
</head>
<body>
    <div class="container">
        <?php partial('header', ['title' => $title ?? 'Basic Web Server Example']); ?>
        
        <?php partial('nav'); ?>
        
        <main role="main">
            <?php if (isset($page_title)): ?>
            <h1><?= h($page_title) ?></h1>
            <?php endif; ?>
            
            <?= $content ?>
        </main>
        
        <?php partial('footer'); ?>
    </div>
    
    <script src="/static/js/main.js"></script>
</body>
</html>
```

### Partials

Let's create the partial templates:

#### `php-files/templates/partials/header.php`

```php
<header class="site-header">
    <div class="logo">
        <img src="/static/images/logo.png" alt="Logo" width="50">
        <span class="site-title"><?= h($title) ?></span>
    </div>
</header>
```

#### `php-files/templates/partials/nav.php`

```php
<nav class="main-nav">
    <ul>
        <li><a href="/" class="<?= is_active('/') ? 'active' : '' ?>">Home</a></li>
        <li><a href="/about" class="<?= is_active('/about') ? 'active' : '' ?>">About</a></li>
        <li><a href="/contact" class="<?= is_active('/contact') ? 'active' : '' ?>">Contact</a></li>
    </ul>
</nav>
```

#### `php-files/templates/partials/footer.php`

```php
<footer class="site-footer">
    <p>&copy; <?= date('Y') ?> Basic Web Server Example. Powered by Frango.</p>
</footer>
```

### Page Templates

Now let's create the templates for each page:

#### `php-files/templates/home.php`

```php
<div class="welcome-section">
    <p class="lead"><?= h($description) ?></p>
</div>

<div class="features">
    <?php foreach ($features as $feature): ?>
    <div class="feature-box">
        <h3><?= h($feature['title']) ?></h3>
        <p><?= h($feature['description']) ?></p>
    </div>
    <?php endforeach; ?>
</div>

<div class="cta-section">
    <h2>Ready to get started?</h2>
    <p>Learn more about our services or contact us to discuss your needs.</p>
    <a href="/contact" class="button">Contact Us</a>
</div>
```

#### `php-files/templates/about.php`

```php
<div class="about-section">
    <p class="lead"><?= h($description) ?></p>
    
    <h2>Our Team</h2>
    <div class="team-members">
        <?php foreach ($team as $member): ?>
        <div class="team-member">
            <h3><?= h($member['name']) ?></h3>
            <p class="position"><?= h($member['position']) ?></p>
            <p class="bio"><?= h($member['bio']) ?></p>
        </div>
        <?php endforeach; ?>
    </div>
    
    <h2>Our Story</h2>
    <p><?= h($story) ?></p>
</div>
```

#### `php-files/templates/contact.php`

```php
<div class="contact-section">
    <p class="lead"><?= h($description) ?></p>
    
    <?php if (isset($success)): ?>
    <div class="message success">
        <p><?= h($success) ?></p>
    </div>
    <?php endif; ?>
    
    <?php if (isset($errors) && !empty($errors)): ?>
    <div class="message error">
        <p>Please correct the following errors:</p>
        <ul>
            <?php foreach ($errors as $error): ?>
            <li><?= h($error) ?></li>
            <?php endforeach; ?>
        </ul>
    </div>
    <?php endif; ?>
    
    <form class="contact-form" method="post" action="/contact">
        <div class="form-group">
            <label for="name">Name</label>
            <input type="text" id="name" name="name" value="<?= h($name ?? '') ?>" required>
        </div>
        
        <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" value="<?= h($email ?? '') ?>" required>
        </div>
        
        <div class="form-group">
            <label for="subject">Subject</label>
            <input type="text" id="subject" name="subject" value="<?= h($subject ?? '') ?>" required>
        </div>
        
        <div class="form-group">
            <label for="message">Message</label>
            <textarea id="message" name="message" rows="5" required><?= h($message ?? '') ?></textarea>
        </div>
        
        <div class="form-group">
            <button type="submit" class="button">Send Message</button>
        </div>
    </form>
    
    <div class="contact-info">
        <h2>Contact Information</h2>
        <p><strong>Address:</strong> <?= h($contact_info['address']) ?></p>
        <p><strong>Phone:</strong> <?= h($contact_info['phone']) ?></p>
        <p><strong>Email:</strong> <?= h($contact_info['email']) ?></p>
    </div>
</div>
```

#### `php-files/templates/error.php`

```php
<div class="error-section">
    <h1>Error <?= h($code) ?></h1>
    <p class="lead"><?= h($message) ?></p>
    
    <?php if (isset($requestPath)): ?>
    <p>The requested path <code><?= h($requestPath) ?></code> could not be found.</p>
    <?php endif; ?>
    
    <p>Go back to <a href="/">home page</a>.</p>
</div>
```

### Controllers

Finally, let's create the controllers that will handle the business logic:

#### `php-files/controllers/home.php`

```php
<?php
// Include helper functions
require_once __DIR__ . '/../includes/helpers.php';

// Define page data
$title = 'Home - Basic Web Server Example';
$page_title = 'Welcome to Our Basic Web Server';
$description = 'This is a simple example of a web server built with Frango, combining Go and PHP.';
$meta_description = 'A basic web server example using Frango to combine Go and PHP.';

// Define features
$features = [
    [
        'title' => 'Go Performance',
        'description' => 'Benefit from Go\'s excellent performance and concurrency for handling HTTP requests.'
    ],
    [
        'title' => 'PHP Templating',
        'description' => 'Use PHP\'s powerful templating capabilities for rendering dynamic content.'
    ],
    [
        'title' => 'Best of Both Worlds',
        'description' => 'Combine Go and PHP to leverage the strengths of each language.'
    ]
];

// Render the template
render('home', [
    'title' => $title,
    'page_title' => $page_title,
    'description' => $description,
    'meta_description' => $meta_description,
    'features' => $features
]);
```

#### `php-files/controllers/about.php`

```php
<?php
// Include helper functions
require_once __DIR__ . '/../includes/helpers.php';

// Define page data
$title = 'About Us - Basic Web Server Example';
$page_title = 'About Our Company';
$description = 'Learn more about our company and our team.';
$meta_description = 'About our company and team - Basic Web Server Example.';

// Define team members
$team = [
    [
        'name' => 'John Doe',
        'position' => 'CEO',
        'bio' => 'John has over 15 years of experience in software development and has been leading our company since its founding.'
    ],
    [
        'name' => 'Jane Smith',
        'position' => 'CTO',
        'bio' => 'Jane is a technology enthusiast with expertise in Go, PHP, and various other programming languages.'
    ],
    [
        'name' => 'Bob Johnson',
        'position' => 'Lead Developer',
        'bio' => 'Bob leads our development team and ensures the quality and performance of our products.'
    ]
];

// Define company story
$story = 'Our company was founded in 2020 with the goal of creating innovative web solutions. We specialize in building high-performance web applications using modern technologies like Go and PHP.';

// Render the template
render('about', [
    'title' => $title,
    'page_title' => $page_title,
    'description' => $description,
    'meta_description' => $meta_description,
    'team' => $team,
    'story' => $story
]);
```

#### `php-files/controllers/contact.php`

```php
<?php
// Include helper functions
require_once __DIR__ . '/../includes/helpers.php';

// Define page data
$title = 'Contact Us - Basic Web Server Example';
$page_title = 'Contact Us';
$description = 'Have questions or want to learn more? Reach out to us using the form below.';
$meta_description = 'Contact our team - Basic Web Server Example.';

// Define contact information
$contact_info = [
    'address' => '123 Web Dev Street, Servertown, 12345',
    'phone' => '(555) 123-4567',
    'email' => 'info@example.com'
];

// Initialize variables
$name = '';
$email = '';
$subject = '';
$message = '';
$errors = [];
$success = null;

// Handle form submission
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    // Get form data
    $name = $_POST['name'] ?? '';
    $email = $_POST['email'] ?? '';
    $subject = $_POST['subject'] ?? '';
    $message = $_POST['message'] ?? '';
    
    // Validate form data
    if (empty($name)) {
        $errors[] = 'Name is required.';
    }
    
    if (empty($email)) {
        $errors[] = 'Email is required.';
    } elseif (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
        $errors[] = 'Email is invalid.';
    }
    
    if (empty($subject)) {
        $errors[] = 'Subject is required.';
    }
    
    if (empty($message)) {
        $errors[] = 'Message is required.';
    } elseif (strlen($message) < 10) {
        $errors[] = 'Message must be at least 10 characters long.';
    }
    
    // Process the form if there are no errors
    if (empty($errors)) {
        // In a real application, you would send an email or save to a database
        // For this example, we'll just simulate success
        
        // Log the message (in a real app, you might send an email)
        error_log("Contact form submission: $name <$email> - $subject");
        
        // Set success message
        $success = 'Your message has been sent successfully! We will get back to you soon.';
        
        // Reset form fields after successful submission
        $name = '';
        $email = '';
        $subject = '';
        $message = '';
    }
}

// Render the template
render('contact', [
    'title' => $title,
    'page_title' => $page_title,
    'description' => $description,
    'meta_description' => $meta_description,
    'contact_info' => $contact_info,
    'name' => $name,
    'email' => $email,
    'subject' => $subject,
    'message' => $message,
    'errors' => $errors,
    'success' => $success
]);
```

## Static Files

Now let's create the static files for our web server.

### `static/css/style.css`

```css
/* Basic styling for the web server example */

/* Global styles */
:root {
    --primary-color: #2c3e50;
    --secondary-color: #3498db;
    --accent-color: #e74c3c;
    --text-color: #333;
    --background-color: #f9f9f9;
    --light-gray: #ecf0f1;
    --dark-gray: #7f8c8d;
    --success-color: #2ecc71;
    --error-color: #e74c3c;
}

* {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
}

body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    line-height: 1.6;
    color: var(--text-color);
    background-color: var(--background-color);
    padding: 0;
    margin: 0;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 20px;
}

a {
    color: var(--secondary-color);
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}

h1, h2, h3, h4, h5, h6 {
    color: var(--primary-color);
    margin: 1rem 0;
}

p {
    margin: 0 0 1rem;
}

/* Header styles */
.site-header {
    background-color: var(--primary-color);
    color: white;
    padding: 1rem 0;
    margin-bottom: 1rem;
}

.logo {
    display: flex;
    align-items: center;
}

.site-title {
    margin-left: 10px;
    font-size: 1.5rem;
    font-weight: bold;
}

/* Navigation styles */
.main-nav {
    background-color: var(--light-gray);
    margin-bottom: 2rem;
}

.main-nav ul {
    display: flex;
    list-style: none;
}

.main-nav li {
    padding: 1rem;
}

.main-nav a {
    color: var(--primary-color);
    font-weight: bold;
}

.main-nav a.active {
    color: var(--secondary-color);
}

/* Main content styles */
main {
    min-height: 60vh;
    padding: 2rem 0;
}

.lead {
    font-size: 1.2rem;
    margin-bottom: 2rem;
}

/* Home page styles */
.welcome-section {
    margin-bottom: 3rem;
}

.features {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    margin-bottom: 3rem;
}

.feature-box {
    background-color: var(--light-gray);
    padding: 1.5rem;
    border-radius: 5px;
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
}

.feature-box h3 {
    color: var(--secondary-color);
    margin-top: 0;
}

.cta-section {
    background-color: var(--secondary-color);
    color: white;
    padding: 2rem;
    border-radius: 5px;
    text-align: center;
    margin-bottom: 2rem;
}

.cta-section h2 {
    color: white;
    margin-top: 0;
}

.button {
    display: inline-block;
    background-color: var(--primary-color);
    color: white;
    padding: 0.75rem 1.5rem;
    border-radius: 5px;
    text-decoration: none;
    font-weight: bold;
    cursor: pointer;
    border: none;
    transition: background-color 0.3s;
}

.button:hover {
    background-color: #1a252f;
    text-decoration: none;
}

/* About page styles */
.team-members {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    margin-bottom: 3rem;
}

.team-member {
    background-color: var(--light-gray);
    padding: 1.5rem;
    border-radius: 5px;
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
}

.team-member h3 {
    margin-top: 0;
    color: var(--secondary-color);
}

.position {
    color: var(--dark-gray);
    font-style: italic;
    margin-bottom: 1rem;
}

/* Contact page styles */
.contact-section {
    margin-bottom: 2rem;
}

.contact-form {
    background-color: var(--light-gray);
    padding: 2rem;
    border-radius: 5px;
    margin-bottom: 2rem;
}

.form-group {
    margin-bottom: 1.5rem;
}

label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: bold;
}

input, textarea {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--dark-gray);
    border-radius: 5px;
    font-family: inherit;
    font-size: 1rem;
}

.message {
    padding: 1rem;
    border-radius: 5px;
    margin-bottom: 1.5rem;
}

.success {
    background-color: var(--success-color);
    color: white;
}

.error {
    background-color: var(--error-color);
    color: white;
}

.error ul {
    margin-left: 1.5rem;
}

.contact-info {
    background-color: var(--light-gray);
    padding: 2rem;
    border-radius: 5px;
}

/* Error page styles */
.error-section {
    text-align: center;
    padding: 3rem 1rem;
}

.error-section h1 {
    font-size: 3rem;
    color: var(--accent-color);
}

code {
    background-color: var(--light-gray);
    padding: 0.2rem 0.4rem;
    border-radius: 3px;
    font-family: 'Courier New', Courier, monospace;
}

/* Footer styles */
.site-footer {
    background-color: var(--primary-color);
    color: white;
    padding: 1.5rem 0;
    text-align: center;
    margin-top: 2rem;
}

.site-footer p {
    margin-bottom: 0;
}

/* Responsive adjustments */
@media (max-width: 768px) {
    .main-nav ul {
        flex-direction: column;
    }
    
    .main-nav li {
        padding: 0.5rem 1rem;
    }
    
    .features, .team-members {
        grid-template-columns: 1fr;
    }
}
```

### `static/js/main.js`

```javascript
// Main JavaScript file for the basic web server example

document.addEventListener('DOMContentLoaded', function() {
    console.log('Basic Web Server Example - JavaScript loaded');
    
    // Form validation
    const contactForm = document.querySelector('.contact-form');
    if (contactForm) {
        contactForm.addEventListener('submit', function(event) {
            let valid = true;
            
            // Get form fields
            const nameInput = document.getElementById('name');
            const emailInput = document.getElementById('email');
            const subjectInput = document.getElementById('subject');
            const messageInput = document.getElementById('message');
            
            // Validate name
            if (!nameInput.value.trim()) {
                valid = false;
                nameInput.classList.add('error');
            } else {
                nameInput.classList.remove('error');
            }
            
            // Validate email
            const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailInput.value.trim() || !emailPattern.test(emailInput.value)) {
                valid = false;
                emailInput.classList.add('error');
            } else {
                emailInput.classList.remove('error');
            }
            
            // Validate subject
            if (!subjectInput.value.trim()) {
                valid = false;
                subjectInput.classList.add('error');
            } else {
                subjectInput.classList.remove('error');
            }
            
            // Validate message
            if (!messageInput.value.trim() || messageInput.value.length < 10) {
                valid = false;
                messageInput.classList.add('error');
            } else {
                messageInput.classList.remove('error');
            }
            
            // Client-side form validation will supplement server-side validation
            // The form will still submit and the server will perform its own validation
        });
    }
    
    // Add smooth scrolling for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            
            const targetId = this.getAttribute('href');
            const targetElement = document.querySelector(targetId);
            
            if (targetElement) {
                window.scrollTo({
                    top: targetElement.offsetTop - 100,
                    behavior: 'smooth'
                });
            }
        });
    });
});
```

## Running the Server

To run the server, you'll need to:

1. Make sure Go is installed on your system
2. Make sure PHP is installed and available in your PATH
3. Create the project structure as described above
4. Save all the code files to their respective locations
5. Install the Frango package

```bash
# Create the project directory
mkdir -p basic-web-server/static/{css,js,images} basic-web-server/php-files/{templates/partials,includes,controllers}

# Initialize the Go module
cd basic-web-server
go mod init basic-web-server

# Install Frango
go get github.com/davidroman0O/go-php/v1

# Run the server
go run main.go
```

After starting the server, you can access it at http://localhost:8080 in your web browser.

## Next Steps

This example provides a basic foundation for a web server using Frango. Here are some ways you could extend it:

1. **Add a Database** - Connect to a database to store contact form submissions or other data
2. **Implement Authentication** - Add user authentication and sessions
3. **Create an API** - Add JSON API endpoints for AJAX requests
4. **Add More Pages** - Expand the site with additional pages and functionality
5. **Improve SEO** - Add more metadata and implement SEO best practices
6. **Add Analytics** - Integrate with an analytics service
7. **Optimize Performance** - Implement caching and other performance optimizations
8. **Add Tests** - Write tests for both the Go and PHP components

By combining Go's performance with PHP's ease of use for templating and business logic, Frango allows you to build robust web applications that leverage the strengths of both languages. 