# Troubleshooting Guide

This guide helps you diagnose and resolve common issues that may arise when using Frango.

## Table of Contents

- [Installation Issues](#installation-issues)
- [PHP Configuration Issues](#php-configuration-issues)
- [Virtual File System Issues](#virtual-file-system-issues)
- [Request Handling Issues](#request-handling-issues)
- [PHP Execution Issues](#php-execution-issues)
- [Environment Variables and Superglobals](#environment-variables-and-superglobals)
- [Performance Issues](#performance-issues)
- [Development vs. Production](#development-vs-production)
- [Common Error Messages](#common-error-messages)
- [Debugging Techniques](#debugging-techniques)

## Installation Issues

### PHP Not Found

**Symptoms**: Errors mentioning `PHP-CGI not found` or similar when initializing Frango.

**Solution**:

1. Ensure PHP is installed on your system:

   ```bash
   # Check PHP version
   php --version
   
   # Check PHP-CGI availability
   php-cgi --version
   ```

2. If PHP-CGI is not available, install it:

   **On Ubuntu/Debian**:
   ```bash
   sudo apt-get install php-cgi
   ```

   **On macOS with Homebrew**:
   ```bash
   brew install php
   ```

   **On Windows**:
   Download and install PHP from [php.net](https://www.php.net/downloads.php) and ensure the CGI version is included.

3. Make sure PHP is in your PATH environment variable.

### Go Module Issues

**Symptoms**: Errors when trying to `go get` the Frango package or build your application.

**Solution**:

1. Ensure you're using a recent version of Go (1.16+).

2. Initialize your Go module if you haven't:

   ```bash
   go mod init your-module-name
   ```

3. Try forcing a clean download:

   ```bash
   go clean -modcache
   go get -u github.com/davidroman0O/go-php/v1
   ```

## PHP Configuration Issues

### PHP Extensions Missing

**Symptoms**: PHP scripts fail with errors about missing functions or classes.

**Solution**:

1. Identify the required PHP extensions from the error messages.

2. Install the needed extensions:

   **On Ubuntu/Debian**:
   ```bash
   sudo apt-get install php-[extension-name]
   ```

   **On macOS with Homebrew**:
   ```bash
   brew install php-[extension-name]
   ```

3. Configure Frango to use these extensions:

   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(map[string]string{
           "extension": "extension_name.so",
       }),
   )
   ```

### PHP.ini Settings

**Symptoms**: PHP scripts behave unexpectedly or have limitations that seem configuration-related.

**Solution**:

1. Check your current PHP settings with a diagnostic script:

   ```php
   <?php phpinfo(); ?>
   ```

2. Configure specific PHP settings in your Go code:

   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(map[string]string{
           "memory_limit": "256M",
           "max_execution_time": "30",
           "display_errors": "On",
           "error_reporting": "E_ALL",
       }),
   )
   ```

## Virtual File System Issues

### Files Not Found

**Symptoms**: "File not found" errors even though you're sure the file exists.

**Solution**:

1. Check your VFS path mappings:

   ```go
   // Debug: Print the list of files in the VFS
   vfs := php.NewVFS()
   // Add your files as you normally would
   vfs.AddSourceDirectory("./php-files", "/")
   // Print available files (implementation depends on VFS internals)
   ```

2. Verify correct path usage:

   - Remember that VFS paths typically start with a forward slash `/`
   - Paths are case-sensitive
   - Use absolute paths within the VFS context

3. Check for trailing slashes in directory paths:

   ```go
   // This may not work as expected
   vfs.AddSourceDirectory("./php-files", "app")
   
   // Better approaches
   vfs.AddSourceDirectory("./php-files", "/app")
   // or
   vfs.AddSourceDirectory("./php-files", "/app/")
   ```

### File Permissions

**Symptoms**: Permission denied errors when trying to read or write files.

**Solution**:

1. Check the permissions of your source files on disk:

   ```bash
   ls -la ./php-files
   ```

2. Ensure the process has read/write access to the temporary directory:

   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithTempDir("/tmp/my-app"), // Ensure this directory is writable
   )
   ```

## Request Handling Issues

### POST Data Not Available

**Symptoms**: `$_POST` is empty in PHP even though you're submitting a form.

**Solution**:

1. Ensure the form has the correct method and enctype:

   ```html
   <!-- For normal forms -->
   <form method="post" action="/process.php">
   
   <!-- For file uploads -->
   <form method="post" action="/upload.php" enctype="multipart/form-data">
   ```

2. Check if you're using the correct PHP superglobal:

   ```php
   // Debug
   echo "<pre>POST: ";
   var_dump($_POST);
   echo "</pre>";
   
   echo "<pre>REQUEST: ";
   var_dump($_REQUEST);
   echo "</pre>";
   ```

3. Verify that Frango is properly configured to handle POST requests:

   ```go
   // Make sure you're using the right HTTP handler
   http.Handle("/process.php", php.For("/process.php"))
   ```

### Query Parameters Missing

**Symptoms**: `$_GET` parameters not available in PHP.

**Solution**:

1. Check the URL structure and encoding:

   ```
   Correct: /script.php?name=John&age=30
   ```

2. Debug by outputting the raw query:

   ```php
   echo "Query string: " . $_SERVER['QUERY_STRING'];
   ```

3. Check for any middleware that might be modifying the request:

   ```go
   // Ensure your middleware preserves query parameters
   func myMiddleware(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           // If you modify r.URL here, be careful not to lose query parameters
           next.ServeHTTP(w, r)
       })
   }
   ```

## PHP Execution Issues

### Script Execution Failures

**Symptoms**: PHP scripts don't execute or produce unexpected errors.

**Solution**:

1. Check for PHP syntax errors:

   ```bash
   php -l your_script.php
   ```

2. Enable detailed error reporting in PHP:

   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(map[string]string{
           "display_errors": "On",
           "error_reporting": "E_ALL",
       }),
   )
   ```

3. Create a debug script that outputs system information:

   ```php
   <?php
   echo "<h1>Debug Info</h1>";
   echo "<h2>PHP Version</h2>";
   echo phpversion();
   
   echo "<h2>Loaded Extensions</h2>";
   echo "<pre>";
   print_r(get_loaded_extensions());
   echo "</pre>";
   
   echo "<h2>Include Path</h2>";
   echo get_include_path();
   
   echo "<h2>Current Working Directory</h2>";
   echo getcwd();
   
   echo "<h2>Server Variables</h2>";
   echo "<pre>";
   print_r($_SERVER);
   echo "</pre>";
   ?>
   ```

### Timeouts

**Symptoms**: PHP scripts timeout or seem to hang.

**Solution**:

1. Increase execution time limits:

   ```go
   php, err := frango.New(
       frango.WithPHPIniEntries(map[string]string{
           "max_execution_time": "60",  // 60 seconds
           "max_input_time": "60",
       }),
   )
   ```

2. Check for infinite loops or blocking operations in your PHP code:

   ```php
   // Add timeouts to potentially long-running operations
   $context = stream_context_create([
       'http' => [
           'timeout' => 5, // 5 seconds timeout
       ]
   ]);
   $response = file_get_contents('http://example.com', false, $context);
   ```

## Environment Variables and Superglobals

### Environment Variables Not Set

**Symptoms**: PHP can't access expected environment variables.

**Solution**:

1. Explicitly set environment variables in your Go code:

   ```go
   env := map[string]string{
       "APP_ENV": "development",
       "DB_HOST": "localhost",
   }
   
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithEnvironment(env),
   )
   ```

2. Check if environment variables are correctly propagated:

   ```php
   <?php
   echo "<pre>";
   print_r($_ENV);
   echo "</pre>";
   
   echo "<pre>";
   print_r(getenv());
   echo "</pre>";
   ?>
   ```

### Superglobals Empty or Incorrect

**Symptoms**: PHP superglobals like `$_SERVER`, `$_COOKIE`, etc. are empty or have unexpected values.

**Solution**:

1. Debug superglobal contents:

   ```php
   <?php
   echo "<h2>SERVER</h2><pre>";
   print_r($_SERVER);
   echo "</pre>";
   
   echo "<h2>GET</h2><pre>";
   print_r($_GET);
   echo "</pre>";
   
   echo "<h2>POST</h2><pre>";
   print_r($_POST);
   echo "</pre>";
   
   echo "<h2>FILES</h2><pre>";
   print_r($_FILES);
   echo "</pre>";
   
   echo "<h2>COOKIE</h2><pre>";
   print_r($_COOKIE);
   echo "</pre>";
   
   echo "<h2>SESSION</h2><pre>";
   if (session_status() == PHP_SESSION_ACTIVE) {
       print_r($_SESSION);
   } else {
       echo "Session not active";
   }
   echo "</pre>";
   ?>
   ```

2. Check if you've included the globals initialization file:

   ```php
   <?php
   // At the beginning of your entry point script
   require_once 'globals_fix.php';
   
   // Rest of your code
   ?>
   ```

## Performance Issues

### Slow Response Times

**Symptoms**: The application responds slowly, especially under load.

**Solution**:

1. Enable opcache in production:

   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(map[string]string{
           "opcache.enable": "1",
           "opcache.memory_consumption": "128",
           "opcache.interned_strings_buffer": "8",
           "opcache.max_accelerated_files": "4000",
           "opcache.revalidate_freq": "60",
           "opcache.fast_shutdown": "1",
       }),
   )
   ```

2. Optimize PHP script inclusion:

   ```php
   // Instead of including many files
   require_once 'file1.php';
   require_once 'file2.php';
   require_once 'file3.php';
   
   // Use an autoloader
   spl_autoload_register(function($class) {
       $file = str_replace('\\', '/', $class) . '.php';
       if (file_exists($file)) {
           require $file;
           return true;
       }
       return false;
   });
   ```

3. Enable caching for VFS:

   ```go
   // Ensure development mode is OFF in production
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithDevelopmentMode(false), // Important for performance
   )
   ```

### Memory Leaks

**Symptoms**: Memory usage grows over time, eventually leading to crashes or performance degradation.

**Solution**:

1. Implement proper cleanup:

   ```go
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
   )
   // Important!
   defer php.Shutdown()
   ```

2. Limit PHP memory usage:

   ```go
   php, err := frango.New(
       frango.WithPHPIniEntries(map[string]string{
           "memory_limit": "128M",
       }),
   )
   ```

3. Properly close resources in PHP:

   ```php
   // Open a file
   $handle = fopen('file.txt', 'r');
   // Use the file
   $content = fread($handle, filesize('file.txt'));
   // Close the resource when done
   fclose($handle);
   ```

## Development vs. Production

### Different Behavior in Production

**Symptoms**: Application works in development but fails in production.

**Solution**:

1. Use consistent configurations with environment-specific overrides:

   ```go
   // Base configuration
   iniEntries := map[string]string{
       "date.timezone": "UTC",
   }
   
   // Environment-specific overrides
   if os.Getenv("GO_ENV") == "production" {
       iniEntries["display_errors"] = "Off"
       iniEntries["error_reporting"] = "E_ALL & ~E_DEPRECATED & ~E_STRICT"
       iniEntries["opcache.enable"] = "1"
   } else {
       iniEntries["display_errors"] = "On"
       iniEntries["error_reporting"] = "E_ALL"
       iniEntries["opcache.enable"] = "0"
   }
   
   php, err := frango.New(
       frango.WithSourceDir("./php-files"),
       frango.WithPHPIniEntries(iniEntries),
       frango.WithDevelopmentMode(os.Getenv("GO_ENV") != "production"),
   )
   ```

2. Implement environment-specific logic in PHP:

   ```php
   <?php
   $isProduction = (getenv('GO_ENV') === 'production');
   
   if ($isProduction) {
       // Production settings
       error_reporting(E_ALL & ~E_DEPRECATED & ~E_STRICT);
       ini_set('display_errors', 'Off');
   } else {
       // Development settings
       error_reporting(E_ALL);
       ini_set('display_errors', 'On');
   }
   ?>
   ```

### File Path Issues

**Symptoms**: Path-related errors that appear in production but not development.

**Solution**:

1. Always use correct path functions and constants:

   ```php
   // Correct approach for file paths
   $configPath = __DIR__ . '/config.ini';
   
   // For web URLs, use proper construction
   $baseUrl = (isset($_SERVER['HTTPS']) ? 'https://' : 'http://') . 
              $_SERVER['HTTP_HOST'] . 
              rtrim(dirname($_SERVER['SCRIPT_NAME']), '/\\');
   $imageUrl = $baseUrl . '/images/logo.png';
   ```

2. Use the VFS paths consistently:

   ```go
   // In Go code
   vfs.AddSourceDirectory("./templates", "/templates")
   
   // Then in PHP, use the VFS path
   require_once "/templates/header.php";
   ```

## Common Error Messages

### "PHP executable not found"

**Cause**: Frango can't locate the PHP-CGI binary.

**Solution**:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithPHPCGIPath("/custom/path/to/php-cgi"), // Point to your PHP-CGI binary
)
```

### "Could not create temporary directory"

**Cause**: Insufficient permissions or disk space for creating temporary files.

**Solution**:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithTempDir("/path/with/write/permissions"), // Custom temp directory
)
```

### "Headers already sent"

**Cause**: In PHP, output has been sent to the browser before header() or session functions were called.

**Solution**:

1. Check for whitespace or output before `<?php` tags.
2. Use output buffering:

```php
<?php
// Start output buffering at the beginning of your script
ob_start();

// Your code here
// ...

// Headers can still be set
header('Content-Type: application/json');

// Flush the buffer when ready to send output
ob_end_flush();
?>
```

## Debugging Techniques

### Enable PHP Debugging

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithPHPIniEntries(map[string]string{
        "display_errors": "On",
        "error_reporting": "E_ALL",
        "display_startup_errors": "On",
        "log_errors": "On",
        "error_log": "/path/to/php_error.log",
    }),
)
```

### Debug PHP Output

```php
<?php
// For debugging variables
var_dump($variable);

// For debugging complex objects/arrays with better formatting
echo "<pre>";
print_r($complexArray);
echo "</pre>";

// For logging without affecting output
error_log("Debug message: " . json_encode($data));
?>
```

### Trace Requests in Go

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Log request details
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        
        // Create a response wrapper to capture status code
        rw := &responseWriter{w, http.StatusOK}
        
        // Process the request
        next.ServeHTTP(rw, r)
        
        // Log response details
        duration := time.Since(start)
        log.Printf("Response: %d %s in %v", rw.status, http.StatusText(rw.status), duration)
    })
}

// Apply middleware
http.Handle("/", loggingMiddleware(php.For("/index.php")))
```

### Check PHP Process Execution

To debug PHP process execution issues:

```go
// Optional: Configure PHP to log its startup process
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithPHPIniEntries(map[string]string{
        "log_errors": "On",
        "error_log": "/tmp/php_errors.log",
    }),
)
```

Then check the PHP error log for any startup or execution issues.

## Conclusion

This troubleshooting guide covers the most common issues you might encounter when working with Frango. If you're experiencing a problem not covered here, consider:

1. Checking the GitHub repository for open or closed issues that might address your problem
2. Enabling detailed logging and error reporting to gather more information
3. Isolating the issue to determine if it's related to Frango, PHP, or your application code
4. Creating a minimal reproducible example to help diagnose the problem

Remember that Frango integrates two different technologies (Go and PHP), and issues can arise from either environment or their interaction. Systematic debugging by isolating components can help identify where the problem originates. 