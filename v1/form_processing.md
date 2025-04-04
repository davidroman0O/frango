# Form Data Processing in Frango

This document describes how to work with form data in Frango.

## Background

In typical PHP applications, form data would be available in the `$_GET` and `$_POST` superglobal arrays. However, in the current implementation of Frango (using FrankenPHP), there are some differences in how form data is handled.

## How Form Data is Exposed

When submitting form data to PHP scripts executed via Frango:

1. Form data is not available in the `$_POST` or `$_GET` superglobal arrays as might be expected.
2. The `php://input` stream is empty, so you cannot read the raw POST body using `file_get_contents('php://input')`.
3. Instead, form fields are exposed as environment variables in the `$_SERVER` superglobal with different prefixes:
   - POST data fields have the prefix `PHP_FORM_`
   - GET query parameters have the prefix `PHP_QUERY_`

## Accessing Form Data

### POST Data

To access POST form data in your PHP scripts:

```php
// Access a specific POST field
$name = $_SERVER['PHP_FORM_name']; // For a POST field named "name"
$email = $_SERVER['PHP_FORM_email']; // For a POST field named "email"

// Loop through all POST fields
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $formKey = substr($key, 9); // Remove PHP_FORM_ prefix
        echo "POST field '$formKey' = '$value'\n";
    }
}
```

### GET Parameters

To access GET query parameters:

```php
// Access a specific GET parameter
$id = $_SERVER['PHP_QUERY_id']; // For a query parameter ?id=123
$page = $_SERVER['PHP_QUERY_page']; // For a query parameter ?page=5

// Loop through all GET parameters
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $queryKey = substr($key, 10); // Remove PHP_QUERY_ prefix
        echo "GET param '$queryKey' = '$value'\n";
    }
}
```

## Complete Example PHP Script

Here's a complete example of a PHP script that processes both POST and GET data in Frango:

```php
<?php
header("Content-Type: text/plain");
echo "Form Data Processing\n";
echo "===================\n\n";

// Process POST form data
echo "POST Form Data:\n";
$postCount = 0;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $formKey = substr($key, 9); // Remove PHP_FORM_ prefix
        echo "- $formKey: $value\n";
        $postCount++;
    }
}
if ($postCount === 0) {
    echo "No POST data submitted.\n";
}

// Process GET query parameters
echo "\nGET Query Parameters:\n";
$getCount = 0;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $queryKey = substr($key, 10); // Remove PHP_QUERY_ prefix
        echo "- $queryKey: $value\n";
        $getCount++;
    }
}
if ($getCount === 0) {
    echo "No GET parameters submitted.\n";
}

// Access specific fields with error checking
echo "\nAccessing specific fields:\n";

// POST data example
$name = isset($_SERVER['PHP_FORM_name']) ? $_SERVER['PHP_FORM_name'] : 'Not provided';
echo "POST name: $name\n";

// GET parameter example
$page = isset($_SERVER['PHP_QUERY_page']) ? $_SERVER['PHP_QUERY_page'] : '1';
echo "GET page: $page\n";
?>
```

## Usage from Go Code

When submitting a form from Go to a PHP script:

1. The form fields will be automatically converted to environment variables with appropriate prefixes.
2. You don't need any special handling beyond normal HTTP request creation.

## Testing Forms

When testing form processing, ensure your tests check for data in the correctly prefixed `$_SERVER` array variables and not in `$_POST` or `$_GET`. 