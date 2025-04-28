# Form Processing Example

This example demonstrates how to handle form submissions with Frango, including validation, processing, and displaying success messages.

## What This Example Shows

- Creating HTML forms in PHP
- Form validation with error handling
- Processing POST form submissions
- Redirecting after form submission
- Accessing form data with different PHP superglobals
- HTTP method constraints in route patterns
- Displaying submission results

## Files

- `main.go`: Go HTTP server with form routes
- `php/index.php`: Home page with explanation
- `php/contact-form.php`: Contact form with validation support
- `php/process-form.php`: Form processing script
- `php/success.php`: Success page showing submitted data

## Form Data Access

This example demonstrates two ways to access form data:

1. Using standard PHP `$_POST` superglobal
2. Using Frango's unified `$_FORM` superglobal (works with both GET and POST)

## Running the Example

1. Navigate to this directory
2. Run:
   ```bash
   go run main.go
   ```
3. Open a browser and visit http://localhost:8080
4. Click "Go to Contact Form" and submit the form

## How It Works

1. User fills out the contact form at `/contact`
2. Form is submitted to `/contact/submit` via POST
3. PHP validates the form data
4. If validation fails, user is redirected back to the form with error messages
5. If validation succeeds, data is stored in session and user is redirected to success page
6. Success page displays the submitted data and explanation

## Key Concepts

```go
// Route that only accepts POST requests
mux.Handle("POST /contact/submit", php.For("process-form.php"))
```

```php
// Form validation in PHP
$name = trim($_POST['name'] ?? '');
if (empty($name)) {
    $errors[] = 'Name is required';
    $errorFields[] = 'name';
}

// Redirecting with error information
header("Location: /contact?$queryParams");

// Accessing form data (multiple methods)
$name = $_POST['name'] ?? 'Not provided';  // Standard way
$name = $_FORM['name'] ?? 'Not provided';  // Frango unified way
``` 