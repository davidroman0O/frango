# PHP Templating in Frango

This tutorial explains how to use PHP for templating in Frango applications, taking advantage of PHP's mature templating capabilities while leveraging Go's performance and concurrency.

## Table of Contents

- [PHP Templating in Frango](#php-templating-in-frango)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Setting Up Basic Templates](#setting-up-basic-templates)
    - [Project Structure](#project-structure)
    - [Go Application](#go-application)
  - [Creating Template Helpers](#creating-template-helpers)
  - [Building a Template Layout System](#building-a-template-layout-system)
  - [Passing Data to Templates](#passing-data-to-templates)
    - [Basic Data Passing](#basic-data-passing)
    - [Data Types](#data-types)
    - [Complex Data Structures](#complex-data-structures)
    - [Passing Go Structs](#passing-go-structs)
  - [Working with Includes and Partials](#working-with-includes-and-partials)
    - [Simple Includes](#simple-includes)
    - [Using the Partial Helper](#using-the-partial-helper)
    - [Dynamic Includes](#dynamic-includes)
  - [Advanced Templating Techniques](#advanced-templating-techniques)
    - [Template Inheritance](#template-inheritance)
    - [Caching Template Output](#caching-template-output)
    - [Using Existing PHP Template Libraries](#using-existing-php-template-libraries)
  - [Best Practices](#best-practices)
    - [Security](#security)
    - [Performance](#performance)
    - [Maintainability](#maintainability)
    - [Testing](#testing)
  - [Conclusion](#conclusion)

## Overview

Templating is an essential part of web development, and PHP has long been appreciated for its straightforward and powerful templating capabilities. Frango allows you to leverage PHP's templating within a Go application, giving you the best of both worlds:

- **Go's strengths**: Performance, concurrency, and strong typing
- **PHP's strengths**: Familiar syntax, easy template logic, and a wealth of templating libraries

This approach is particularly useful if:

- You have existing PHP templates you want to reuse
- Your team is more familiar with PHP syntax than Go's templating options
- You need complex template logic that's easier to express in PHP

## Setting Up Basic Templates

Let's start by creating a simple application with PHP templates. We'll build a small website with home and about pages.

### Project Structure

```
template-app/
├── main.go               # Go application entry point
├── php-files/
│   ├── helpers/
│   │   └── template.php  # Template helper functions
│   └── templates/
│       ├── home.php      # Home page template
│       ├── about.php     # About page template
│       ├── layout.php    # Layout template (header/footer)
│       └── partials/     # Partial templates
│           ├── header.php
│           ├── nav.php
│           └── footer.php
```

### Go Application

First, let's create our `main.go` file:

```go
package main

import (
	"log"
	"net/http"

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

	// Home page route
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Forward to PHP template
		// Create context with data we want to pass to the template
		r = r.WithContext(contextWithTemplateData(r.Context(), map[string]interface{}{
			"title":    "Welcome to Frango",
			"subtitle": "Simple PHP Templating in Go",
		}))
		
		// Serve the PHP template
		php.For("templates/home.php").ServeHTTP(w, r)
	})

	// About page route
	mux.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) {
		// Create team members data
		teamMembers := []map[string]interface{}{
			{
				"name":     "Jane Doe",
				"position": "CEO",
			},
			{
				"name":     "John Smith",
				"position": "CTO",
			},
			{
				"name":     "Alice Johnson",
				"position": "Lead Developer",
			},
		}
		
		// Forward to PHP template with data
		r = r.WithContext(contextWithTemplateData(r.Context(), map[string]interface{}{
			"title":       "About Us",
			"subtitle":    "Learn more about our team",
			"teamMembers": teamMembers,
		}))
		
		// Serve the PHP template
		php.For("templates/about.php").ServeHTTP(w, r)
	})

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// Helper function to add template data to the request context
func contextWithTemplateData(ctx context.Context, data map[string]interface{}) context.Context {
	return context.WithValue(ctx, "template_data", data)
}
```

## Creating Template Helpers

To make our PHP templates more powerful and secure, let's create some helper functions. Create a file at `php-files/helpers/template.php`:

```php
<?php
/**
 * Template helper functions
 */

/**
 * HTML escape a string to prevent XSS
 */
function h($text) {
    return htmlspecialchars($text, ENT_QUOTES, 'UTF-8');
}

/**
 * Get the current URL
 */
function currentUrl() {
    $protocol = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') ? 'https' : 'http';
    $host = $_SERVER['HTTP_HOST'];
    $uri = $_SERVER['REQUEST_URI'];
    return "$protocol://$host$uri";
}

/**
 * Check if a path matches the current URL path
 */
function isActivePath($path) {
    return $_SERVER['REQUEST_URI'] === $path;
}

/**
 * Format a date with a specified pattern
 */
function formatDate($date, $format = 'Y-m-d') {
    return date($format, strtotime($date));
}

/**
 * Include a partial template
 */
function partial($name, $vars = []) {
    // Extract variables to make them available in the partial
    extract($vars);
    
    // Include the partial file
    include __DIR__ . "/../templates/partials/$name.php";
}

/**
 * Render a template with the layout
 */
function layout($content, $vars = []) {
    // Extract variables to make them available in the layout
    extract($vars);
    
    // Include the layout file
    include __DIR__ . "/../templates/layout.php";
}
?>
```

## Building a Template Layout System

Now, let's create our template layout system. First, create the main layout file at `php-files/templates/layout.php`:

```php
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title><?= h($title ?? 'Frango Application') ?></title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" rel="stylesheet">
</head>
<body>
    <div class="container">
        <?php partial('header', ['title' => $title ?? 'Frango Application']); ?>
        
        <?php partial('nav'); ?>
        
        <main>
            <?= $content ?>
        </main>
        
        <?php partial('footer'); ?>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>
```

Create the header partial at `php-files/templates/partials/header.php`:

```php
<header class="pb-3 mb-4 border-bottom">
    <div class="d-flex align-items-center text-dark text-decoration-none">
        <span class="fs-4"><?= h($title) ?></span>
    </div>
</header>
```

Create the navigation partial at `php-files/templates/partials/nav.php`:

```php
<nav class="navbar navbar-expand-lg navbar-light bg-light mb-4">
    <div class="container-fluid">
        <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
            <span class="navbar-toggler-icon"></span>
        </button>
        <div class="collapse navbar-collapse" id="navbarNav">
            <ul class="navbar-nav">
                <li class="nav-item">
                    <a class="nav-link <?= $_SERVER['REQUEST_URI'] === '/' ? 'active' : '' ?>" href="/">Home</a>
                </li>
                <li class="nav-item">
                    <a class="nav-link <?= $_SERVER['REQUEST_URI'] === '/about' ? 'active' : '' ?>" href="/about">About</a>
                </li>
            </ul>
        </div>
    </div>
</nav>
```

Create the footer partial at `php-files/templates/partials/footer.php`:

```php
<footer class="pt-4 my-md-5 pt-md-5 border-top">
    <div class="row">
        <div class="col-12 col-md">
            <small class="d-block mb-3 text-muted">&copy; <?= date('Y') ?> Frango Template Example</small>
        </div>
    </div>
</footer>
```

Now, let's create the home page template at `php-files/templates/home.php`:

```php
<?php
// Include template helpers
require_once __DIR__ . '/../helpers/template.php';

// Access data passed from Go using $GLOBALS
$title = $title ?? 'Home';
$subtitle = $subtitle ?? 'Welcome to our application';

// Start output buffering to capture content for the layout
ob_start();
?>

<div class="px-4 py-5 my-5 text-center">
    <h1 class="display-5 fw-bold"><?= h($title) ?></h1>
    <div class="col-lg-6 mx-auto">
        <p class="lead mb-4"><?= h($subtitle) ?></p>
        <p>This is a simple example of using PHP templates with Frango. The current time is <?= date('H:i:s') ?>.</p>
        <div class="d-grid gap-2 d-sm-flex justify-content-sm-center">
            <a href="/about" class="btn btn-primary btn-lg px-4 gap-3">Learn more</a>
        </div>
    </div>
</div>

<div class="row">
    <div class="col-md-4">
        <h2>Go + PHP</h2>
        <p>Combine the performance of Go with the flexibility of PHP for templating.</p>
    </div>
    <div class="col-md-4">
        <h2>Easy Integration</h2>
        <p>Seamlessly pass data from your Go application to PHP templates.</p>
    </div>
    <div class="col-md-4">
        <h2>Familiar Syntax</h2>
        <p>Use the PHP templating syntax you already know and love.</p>
    </div>
</div>

<?php
// Get the captured content
$content = ob_get_clean();

// Render with layout
layout($content, ['title' => $title]);
?>
```

Finally, let's create the about page template at `php-files/templates/about.php`:

```php
<?php
// Include template helpers
require_once __DIR__ . '/../helpers/template.php';

// Access data passed from Go
$title = $title ?? 'About';
$subtitle = $subtitle ?? 'About our company';
$teamMembers = $teamMembers ?? [];

// Start output buffering to capture content for the layout
ob_start();
?>

<div class="px-4 py-5 my-5 text-center">
    <h1 class="display-5 fw-bold"><?= h($title) ?></h1>
    <div class="col-lg-6 mx-auto">
        <p class="lead mb-4"><?= h($subtitle) ?></p>
    </div>
</div>

<div class="row mb-5">
    <div class="col-md-12">
        <h2>Our Story</h2>
        <p>
            This is a sample about page that demonstrates how to use PHP templates with Frango.
            You can include any HTML content here and use PHP variables passed from Go.
        </p>
        <p>
            The current date and time is: <?= date('Y-m-d H:i:s') ?>
        </p>
    </div>
</div>

<h2 class="text-center mb-4">Our Team</h2>

<div class="row">
    <?php foreach ($teamMembers as $member): ?>
        <div class="col-md-4 mb-4">
            <div class="card">
                <div class="card-body">
                    <h5 class="card-title"><?= h($member['name']) ?></h5>
                    <p class="card-text"><?= h($member['position']) ?></p>
                </div>
            </div>
        </div>
    <?php endforeach; ?>
</div>

<?php
// Get the captured content
$content = ob_get_clean();

// Render with layout
layout($content, ['title' => $title]);
?>
```

## Passing Data to Templates

In Frango, you can pass data from Go to PHP templates using request context. The data can be accessed directly in your PHP template as variables.

### Basic Data Passing

```go
// In Go
r = r.WithContext(context.WithValue(r.Context(), "template_data", map[string]interface{}{
    "title": "Page Title",
    "items": []string{"Item 1", "Item 2", "Item 3"},
}))
php.For("templates/page.php").ServeHTTP(w, r)
```

In the PHP template, these variables are available directly:

```php
<?php
// Access the title variable
echo h($title);

// Loop through items
foreach ($items as $item) {
    echo h($item) . "<br>";
}
?>
```

### Data Types

Frango supports passing various data types from Go to PHP:

| Go Type | PHP Type |
|---------|----------|
| string | string |
| int, int32, int64 | integer |
| float32, float64 | float |
| bool | boolean |
| nil | null |
| []interface{} | array (numeric keys) |
| map[string]interface{} | array (associative) |
| struct | array (associative) |

### Complex Data Structures

You can pass complex data structures such as nested maps and slices:

```go
// In Go
data := map[string]interface{}{
    "categories": []map[string]interface{}{
        {
            "name": "Electronics",
            "products": []map[string]interface{}{
                {"name": "Laptop", "price": 999.99},
                {"name": "Smartphone", "price": 599.99},
            },
        },
        {
            "name": "Books",
            "products": []map[string]interface{}{
                {"name": "Go Programming", "price": 29.99},
                {"name": "PHP Cookbook", "price": 39.99},
            },
        },
    },
}

r = r.WithContext(context.WithValue(r.Context(), "template_data", data))
php.For("templates/products.php").ServeHTTP(w, r)
```

In PHP, you can access this data using familiar array syntax:

```php
<?php foreach ($categories as $category): ?>
    <h2><?= h($category['name']) ?></h2>
    <ul>
        <?php foreach ($category['products'] as $product): ?>
            <li><?= h($product['name']) ?> - $<?= number_format($product['price'], 2) ?></li>
        <?php endforeach; ?>
    </ul>
<?php endforeach; ?>
```

### Passing Go Structs

You can also pass Go structs, which will be converted to associative arrays in PHP:

```go
// Define a Go struct
type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Description string  `json:"description"`
    InStock     bool    `json:"inStock"`
}

// Pass it to PHP
product := Product{
    ID:          123,
    Name:        "Ergonomic Keyboard",
    Price:       129.99,
    Description: "A comfortable keyboard for long typing sessions.",
    InStock:     true,
}

r = r.WithContext(context.WithValue(r.Context(), "template_data", map[string]interface{}{
    "product": product,
    "title":   "Product: " + product.Name,
}))
php.For("templates/product-detail.php").ServeHTTP(w, r)
```

In PHP, you can access the struct fields as array keys:

```php
<h1><?= h($product['name']) ?></h1>
<p><?= h($product['description']) ?></p>
<p>Price: $<?= number_format($product['price'], 2) ?></p>
<?php if ($product['inStock']): ?>
    <p class="text-success">In Stock</p>
<?php else: ?>
    <p class="text-danger">Out of Stock</p>
<?php endif; ?>
```

## Working with Includes and Partials

PHP's include system works naturally with Frango, allowing you to create modular templates.

### Simple Includes

You can use PHP's standard include functions:

```php
<?php
// Include a header file
include 'templates/partials/header.php';

// Include with variables
$title = "My Page";
include 'templates/partials/header.php';
?>
```

### Using the Partial Helper

Our template helper includes a `partial()` function for cleaner includes:

```php
<?php
// Include the header partial with variables
partial('header', ['title' => 'My Page']);

// Include a sidebar with active menu item
partial('sidebar', ['activeMenu' => 'products']);
?>
```

### Dynamic Includes

You can dynamically determine which partial to include:

```php
<?php
// Determine the template based on a variable
$template = $view ?? 'default';

// Validate the template name to prevent directory traversal
$allowedTemplates = ['default', 'compact', 'detailed'];
if (!in_array($template, $allowedTemplates)) {
    $template = 'default';
}

// Include the appropriate template
partial("product-views/$template");
?>
```

## Advanced Templating Techniques

### Template Inheritance

Our layout system already implements a simple template inheritance pattern. You can extend this to support multiple layout types:

```php
<?php
// Add this to your template.php helpers

/**
 * Render a template with a specific layout
 */
function renderWithLayout($content, $layout = 'default', $vars = []) {
    // Extract variables to make them available in the layout
    extract($vars);
    
    // Include the specified layout file
    include __DIR__ . "/../templates/layouts/$layout.php";
}
?>
```

Then create multiple layouts like `admin.php`, `blank.php`, etc.

### Caching Template Output

For performance, you can cache template output:

```php
<?php
// Add this to your template.php helpers

/**
 * Get or create cached template output
 */
function cachedOutput($key, $ttl = 300, $callback) {
    $cacheFile = sys_get_temp_dir() . '/template_cache_' . md5($key);
    
    // Check if cache file exists and is still valid
    if (file_exists($cacheFile) && (time() - filemtime($cacheFile) < $ttl)) {
        return file_get_contents($cacheFile);
    }
    
    // Generate new content
    $content = $callback();
    
    // Save to cache
    file_put_contents($cacheFile, $content);
    
    return $content;
}
?>
```

Use it in your templates:

```php
<?php
$content = cachedOutput('home_page_' . $language, 3600, function() use ($title, $subtitle) {
    ob_start();
    // ... template logic here ...
    return ob_get_clean();
});

echo $content;
?>
```

### Using Existing PHP Template Libraries

Frango works with existing PHP template libraries. For example, with Twig:

1. Install Twig using Composer in your PHP directory
2. Create a template helper for Twig:

```php
<?php
// Include Composer autoloader
require_once __DIR__ . '/../vendor/autoload.php';

use Twig\Environment;
use Twig\Loader\FilesystemLoader;

// Initialize Twig
$loader = new FilesystemLoader(__DIR__ . '/../templates');
$twig = new Environment($loader, [
    'cache' => __DIR__ . '/../cache',
    'auto_reload' => true,
]);

/**
 * Render a Twig template
 */
function renderTwig($template, $vars = []) {
    global $twig;
    echo $twig->render($template, $vars);
}
?>
```

## Best Practices

### Security

1. **Always escape output**: Use the `h()` helper function to prevent XSS attacks
2. **Validate dynamic includes**: Prevent directory traversal by validating include paths
3. **Be careful with user input**: Don't use user input directly in template includes or requires

### Performance

1. **Use output buffering**: Capture output before sending to avoid partial page renders
2. **Consider caching**: Implement caching for templates that don't change frequently
3. **Be mindful of includes**: Each include requires a file read, so don't overuse them

### Maintainability

1. **Separate logic and presentation**: Keep complex PHP logic out of templates
2. **Use consistent naming**: Establish naming conventions for templates and partials
3. **Organize templates**: Group related templates in subdirectories
4. **Comment your templates**: Add comments to explain complex sections

### Testing

1. **Create a test harness**: Build a simple page to test templates with mock data
2. **Test with different data**: Ensure templates work with edge cases (empty arrays, nulls, etc.)
3. **Validate HTML output**: Use an HTML validator to ensure your templates produce valid HTML

## Conclusion

PHP templates provide a powerful, familiar way to generate HTML in your Frango applications. By combining Go's performance with PHP's template flexibility, you can create maintainable, scalable web applications that are easy to develop and extend.

This tutorial has shown you how to:

1. Set up a basic template system with Frango
2. Create helper functions to make templating easier and safer
3. Implement a layout system for consistent page structure
4. Pass data from Go to PHP templates
5. Work with partials and includes
6. Use advanced templating techniques for more complex needs

With these tools and techniques, you can build sophisticated web applications that leverage the strengths of both Go and PHP. 