# Frango Middleware: Lessons Learned

This document captures key learnings and best practices when working with the Frango middleware for PHP in Go.

## Form Handling in Frango

### Key Discovery

**Important**: Frango processes form data differently than standard PHP applications. Form data is placed in `$_SERVER` with `FRANGO_FORM_` prefixes rather than in `$_POST`.

### Example

Instead of:
```php
$name = $_POST['name'] ?? '';
$email = $_POST['email'] ?? '';
```

Use:
```php
$name = $_SERVER['FRANGO_FORM_name'] ?? '';
$email = $_SERVER['FRANGO_FORM_email'] ?? '';
```

### Best Practice for Form Processing

```php
// Process form data from Frango middleware
function getFormValue($fieldName, $default = '') {
    $key = 'FRANGO_FORM_' . $fieldName;
    return isset($_SERVER[$key]) ? trim($_SERVER[$key]) : $default;
}

// Usage
$name = getFormValue('name');
$email = getFormValue('email');
$role = getFormValue('role', 'user'); // With default value
```

## PHP Environment Limitations

FrankenPHP provides a minimal PHP environment that may lack common extensions.

### Notable Limitations

1. **No cURL Extension**: Use `file_get_contents()` with stream contexts instead
2. **Limited PHP Extensions**: Stick to core PHP functionality

### API Calls Example

Instead of cURL:

```php
// DON'T use cURL (not available in FrankenPHP)
$ch = curl_init($url);
curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "PUT"); 
curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
$response = curl_exec($ch);
$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);
```

Use `file_get_contents()` with a stream context:

```php
// DO use file_get_contents with stream context
$jsonData = json_encode($data);
$options = [
    'http' => [
        'method' => 'PUT',
        'header' => [
            'Content-Type: application/json',
            'Accept: application/json',
            'Content-Length: ' . strlen($jsonData)
        ],
        'content' => $jsonData,
        'ignore_errors' => true,
        'timeout' => 15
    ]
];
$context = stream_context_create($options);

// Disable error reporting for this call
$oldErrorReporting = error_reporting(0);
$response = @file_get_contents($url, false, $context);
error_reporting($oldErrorReporting);

// Get HTTP status code from response headers
$httpCode = $http_response_header ? intval(substr($http_response_header[0], 9, 3)) : 0;
```

## Debugging Frango Applications

Comprehensive debugging is crucial when working with Frango.

### Effective Debug Information

Add this debug output section to troubleshoot form issues:

```php
<!-- Debug info -->
<div style="margin-top: 30px; padding: 15px; background: #f5f5f5; border: 1px solid #ddd; border-radius: 5px; font-family: monospace; font-size: 12px;">
    <h3>Debug Information</h3>
    
    <h4>URL Segments</h4>
    <p>Raw URL Path: <?= htmlspecialchars($_SERVER['FRANGO_URL_PATH'] ?? 'not available') ?></p>
    
    <?php for ($i = 0; $i < ($_SERVER['FRANGO_URL_SEGMENT_COUNT'] ?? 0); $i++): ?>
        <div>Segment <?= $i ?>: <?= htmlspecialchars($_SERVER["FRANGO_URL_SEGMENT_$i"] ?? 'none') ?></div>
    <?php endfor; ?>
    
    <h4>Form Data</h4>
    <?php foreach ($_SERVER as $key => $value): ?>
        <?php if (strpos($key, 'FRANGO_FORM_') === 0): ?>
            <div><?= htmlspecialchars($key) ?> = <?= htmlspecialchars($value) ?></div>
        <?php endif; ?>
    <?php endforeach; ?>
    
    <h4>Request Details</h4>
    <div>Method: <?= htmlspecialchars($_SERVER['REQUEST_METHOD']) ?></div>
    <div>Content-Type: <?= htmlspecialchars($_SERVER['CONTENT_TYPE'] ?? $_SERVER['FRANGO_HEADER_CONTENT_TYPE'] ?? 'not set') ?></div>
</div>
```

## Form Submission Best Practices

### HTML Form Setup

```html
<form method="post" action="">
    <!-- Simple action ensures posting to current URL -->
    <input type="text" name="name" value="<?= htmlspecialchars($userData['name'] ?? '') ?>">
    <input type="email" name="email" value="<?= htmlspecialchars($userData['email'] ?? '') ?>">
    <select name="role">
        <option value="user" <?= (isset($userData['role']) && $userData['role'] === 'user') ? 'selected' : '' ?>>User</option>
        <option value="admin" <?= (isset($userData['role']) && $userData['role'] === 'admin') ? 'selected' : '' ?>>Admin</option>
    </select>
    <button type="submit">Submit</button>
</form>
```

### PHP Processing Template

```php
// Initialize variables
$success = null;
$error = null;

// Process form submission
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    // Get form data from Frango middleware
    $name = $_SERVER['FRANGO_FORM_name'] ?? '';
    $email = $_SERVER['FRANGO_FORM_email'] ?? '';
    $role = $_SERVER['FRANGO_FORM_role'] ?? 'user';
    
    // Validate form data
    if (empty($name) || empty($email)) {
        $error = 'Name and email are required';
    } else {
        // Process data or call API
        // ...
        
        $success = 'Data saved successfully';
    }
}
```

## Common Pitfalls and Solutions

1. **Empty $_POST Array**: If your $_POST array is empty, remember Frango stores form data in $_SERVER with FRANGO_FORM_ prefixes.

2. **PHP Fatal Errors with cURL**: FrankenPHP may not include cURL extension. Use file_get_contents() instead.

3. **Missing Form Data**: Check if the form attributes (method, action, enctype) might be interfering with submission.

4. **Routing Issues**: Use relative form action (empty action="") to post back to the current URL.

5. **HTTP vs HTTPS**: Ensure your API calls match the protocol of your application.

## Conclusion

Working with Frango requires understanding its unique approach to handling form data and PHP execution. By following these patterns and best practices, you can avoid common pitfalls and build reliable PHP applications within the Go ecosystem. 

