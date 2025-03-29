# Advanced Frango Usage Guide

This document covers advanced usage patterns and techniques for getting the most out of Frango.

## Advanced Routing

### Path Parameters

Frango supports path parameters in routing patterns using Go 1.22+ syntax:

```go
// Create a standard HTTP mux
mux := http.NewServeMux()

// Register a route with a path parameter
php.HandlePHP("/users/{id}", "user_detail.php")
```

In PHP, access path parameters using the `$_ENV` superglobal:

```php
<?php
// Extract the path parameter
$userId = $_ENV['PATH_PARAM_ID'] ?? null;

// Path parameters are also available as JSON in $_ENV['PATH_PARAMS']
$pathParams = json_decode($_ENV['PATH_PARAMS'] ?? '{}', true);
$userId = $pathParams['id'] ?? null;

// Use the parameter
echo "User ID: " . htmlspecialchars($userId);
?>
```

### Nested Routing Patterns

For complex APIs, you can create nested routing patterns:

```go
// Create a standard HTTP mux for different API versions
mux := http.NewServeMux()

// Register PHP endpoints for different API versions
php.HandlePHP("/api/v1/users", "api/v1/users_list.php")
php.HandlePHP("/api/v1/users/{id}", "api/v1/user_detail.php")
php.HandlePHP("/api/v2/users", "api/v2/users_list.php")
php.HandlePHP("/api/v2/users/{id}", "api/v2/user_detail.php")

// Mount the PHP middleware to handle API requests
mux.Handle("/api/", php)
```

### Custom Route Handlers with Path Parameters

You can create custom Go handlers that access path parameters:

```go
mux.HandleFunc("GET /api/users/{id}/stats", func(w http.ResponseWriter, r *http.Request) {
    // Extract the ID path parameter
    userId := r.PathValue("id")
    
    // Use the parameter
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"userId": "%s", "stats": {"views": 1203, "actions": 85}}`, userId)
})
```

## Embedding and Dynamic Content

### Dynamic PHP Template Rendering

You can generate PHP templates dynamically:

```go
// Template with placeholders
templateStr := `<?php
// Dynamic PHP template
$title = "<?= $title ?>";
$items = <?= $items_json ?>;
?>

<!DOCTYPE html>
<html>
<head>
    <title><?= htmlspecialchars($title) ?></title>
</head>
<body>
    <h1><?= htmlspecialchars($title) ?></h1>
    <ul>
        <?php foreach ($items as $item): ?>
            <li><?= htmlspecialchars($item['name']) ?> - $<?= number_format($item['price'], 2) ?></li>
        <?php endforeach; ?>
    </ul>
</body>
</html>`

// Fill in dynamic values
items := []map[string]string{
    {"name": "Product 1", "price": "19.99"},
    {"name": "Product 2", "price": "29.99"},
}
itemsJSON, _ := json.Marshal(items)

// Replace placeholders
templateStr = strings.ReplaceAll(templateStr, "<?= $title ?>", "Dynamic Products")
templateStr = strings.ReplaceAll(templateStr, "<?= $items_json ?>", string(itemsJSON))

// Write the template to a file in your web directory
templatePath := filepath.Join(webDir, "dynamic.php")
if err := os.WriteFile(templatePath, []byte(templateStr), 0644); err != nil {
    log.Fatalf("Error writing template: %v", err)
}

// Register the dynamic template
php.HandlePHP("/dynamic", "dynamic.php")
```

### Custom PHP File Handling

For more control over PHP file handling, you can use the `RenderData` function to inject variables and manipulate requests:

```go
php.HandleRender("/admin/dashboard", "admin/dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    // Check if user is authenticated
    session, err := store.Get(r, "session")
    if err != nil || session.Values["authenticated"] != true {
        // Redirect to login page if not authenticated
        http.Redirect(w, r, "/login", http.StatusFound)
        return nil
    }
    
    // User is authenticated, proceed with rendering
    userId := session.Values["user_id"].(int)
    
    // Fetch user data from database
    user, err := db.GetUser(userId)
    if err != nil {
        http.Error(w, "Error loading user data", http.StatusInternalServerError)
        return nil
    }
    
    // Return variables for the PHP template
    return map[string]interface{}{
        "user": user,
        "permissions": getUserPermissions(userId),
        "stats": fetchUserStats(userId),
        "recent_activities": getRecentActivities(userId, 10),
    }
})
```

## Advanced Middleware Usage

### Combining Multiple Middleware Components

You can chain middleware for complex processing:

```go
// Create authentication middleware
authMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check authentication token
        token := r.Header.Get("Authorization")
        if !isValidToken(token) {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"error":"Unauthorized","message":"Invalid or missing token"}`))
            return
        }
        
        // Token is valid, proceed
        next.ServeHTTP(w, r)
    })
}

// Create logging middleware
loggingMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create a response recorder to capture the response
        recorder := httptest.NewRecorder()
        
        // Call the next handler
        next.ServeHTTP(recorder, r)
        
        // Log the request
        duration := time.Since(start)
        log.Printf(
            "%s %s %s - %d %s - %s",
            r.Method,
            r.URL.Path,
            r.RemoteAddr,
            recorder.Code,
            http.StatusText(recorder.Code),
            duration,
        )
        
        // Copy the response from the recorder to the original writer
        for k, v := range recorder.Header() {
            w.Header()[k] = v
        }
        w.WriteHeader(recorder.Code)
        w.Write(recorder.Body.Bytes())
    })
}

// Apply middleware chain to the PHP middleware
phpWithMiddleware := loggingMiddleware(authMiddleware(php))

// Use in a router
mux.Handle("/api/", phpWithMiddleware)
```

### Custom Middleware for PHP Integration

You can create middleware specifically for PHP integration:

```go
// Middleware that adds PHP-specific headers and processes input
phpEnhancerMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Add headers that PHP might need
        w.Header().Set("X-Powered-By", "Frango/Go")
        
        // Check if we should process the request body
        if r.Header.Get("Content-Type") == "application/json" && r.Method == "POST" {
            // Read and parse JSON body
            var bodyData map[string]interface{}
            if err := json.NewDecoder(r.Body).Decode(&bodyData); err == nil {
                // Store the parsed data in a context value
                ctx := context.WithValue(r.Context(), "json_data", bodyData)
                r = r.WithContext(ctx)
            }
            // Reset the body for PHP to read it again
            bodyBytes, _ := json.Marshal(bodyData)
            r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
        }
        
        // Proceed to the PHP handler
        next.ServeHTTP(w, r)
    })
}

// Apply the middleware
enhancedPHP := phpEnhancerMiddleware(php)
mux.Handle("/api/", enhancedPHP)
```

## Performance Optimization

### Production Mode Settings

For optimal performance in production:

```go
// Create middleware with production settings
php, err := frango.New(
    frango.WithSourceDir(webDir),
    frango.WithDevelopmentMode(false), // Disable development mode for production
)
```

### Optimizing Embedded File Usage

When embedding files, consider these optimizations:

```go
// Embed only the PHP files you need
//go:embed php/*.php php/api/*.php php/templates/*.php
var phpFiles embed.FS

func main() {
    // Create middleware with production mode
    php, err := frango.New(
        frango.WithDevelopmentMode(false),
    )
    if err != nil {
        log.Fatalf("Error creating PHP middleware: %v", err)
    }
    defer php.Shutdown()
    
    // Add the embedded files individually for better control
    php.AddFromEmbed("/", phpFiles, "php/index.php")
    php.AddFromEmbed("/about", phpFiles, "php/about.php")
    php.AddFromEmbed("/contact", phpFiles, "php/contact.php")
    
    // Register API endpoints with method constraints for better performance
    php.AddFromEmbed("/api/users", phpFiles, "php/api/users.php")
    php.AddFromEmbed("/api/products", phpFiles, "php/api/products.php")
    
    // Start the server
    log.Fatal(http.ListenAndServe(":8082", php))
}
```

## Integration with Go Applications

### Sharing Data Between Go and PHP

To share data between your Go application and PHP:

```go
package main

import (
    "database/sql"
    "log"
    "net/http"
    "sync"
    
    _ "github.com/go-sql-driver/mysql"
    "github.com/davidroman0O/frango"
)

// Create a shared data store
type AppState struct {
    DB           *sql.DB
    CacheManager *CacheManager
    ConfigValues map[string]string
    mu           sync.RWMutex
}

// Cache manager
type CacheManager struct {
    cache map[string]interface{}
    mu    sync.RWMutex
}

func NewCacheManager() *CacheManager {
    return &CacheManager{
        cache: make(map[string]interface{}),
    }
}

func (cm *CacheManager) Set(key string, value interface{}) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    cm.cache[key] = value
}

func (cm *CacheManager) Get(key string) (interface{}, bool) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    value, ok := cm.cache[key]
    return value, ok
}

func main() {
    // Initialize shared state
    appState := &AppState{
        ConfigValues: map[string]string{
            "app_name":    "Frango App",
            "api_version": "1.0",
            "debug_mode":  "true",
        },
        CacheManager: NewCacheManager(),
    }
    
    // Connect to database
    db, err := sql.Open("mysql", "user:password@tcp(127.0.0.1:3306)/dbname")
    if err != nil {
        log.Fatalf("Error connecting to database: %v", err)
    }
    appState.DB = db
    defer db.Close()
    
    // Set up PHP middleware
    php, err := frango.New(
        frango.WithSourceDir("web"),
    )
    if err != nil {
        log.Fatalf("Error creating PHP middleware: %v", err)
    }
    defer php.Shutdown()
    
    // Register a render handler that provides access to Go state
    php.HandleRender("/dashboard", "dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
        // Get config values
        appState.mu.RLock()
        config := make(map[string]string)
        for k, v := range appState.ConfigValues {
            config[k] = v
        }
        appState.mu.RUnlock()
        
        // Query some data from the database
        rows, err := appState.DB.Query("SELECT id, name, email FROM users LIMIT 10")
        if err != nil {
            log.Printf("Database error: %v", err)
            return map[string]interface{}{
                "config": config,
                "error":  "Database error: " + err.Error(),
            }
        }
        defer rows.Close()
        
        // Process query results
        var users []map[string]interface{}
        for rows.Next() {
            var id int
            var name, email string
            if err := rows.Scan(&id, &name, &email); err != nil {
                continue
            }
            users = append(users, map[string]interface{}{
                "id":    id,
                "name":  name,
                "email": email,
            })
        }
        
        // Check cache for stats
        var stats interface{}
        if cachedStats, found := appState.CacheManager.Get("dashboard_stats"); found {
            stats = cachedStats
        } else {
            // Calculate stats (expensive operation)
            stats = calculateStats(appState.DB)
            // Cache for future use
            appState.CacheManager.Set("dashboard_stats", stats)
        }
        
        // Return data to PHP
        return map[string]interface{}{
            "config": config,
            "users":  users,
            "stats":  stats,
        }
    })
    
    // Start server
    log.Fatal(http.ListenAndServe(":8082", php))
}

func calculateStats(db *sql.DB) map[string]interface{} {
    // Simulate an expensive calculation
    time.Sleep(500 * time.Millisecond)
    
    // Return dummy stats
    return map[string]interface{}{
        "total_users":     1250,
        "active_users":    867,
        "total_products":  342,
        "recent_orders":   78,
        "revenue_30_days": 12568.99,
    }
}
```

In PHP, you can access this data:

```php
<?php
// dashboard.php
include_once 'render_helper.php';

// Get data from Go
$config = go_var('config', []);
$users = go_var('users', []);
$stats = go_var('stats', []);

// Use the data
$appName = htmlspecialchars($config['app_name'] ?? 'App');
?>

<!DOCTYPE html>
<html>
<head>
    <title><?= $appName ?> Dashboard</title>
</head>
<body>
    <h1><?= $appName ?> Dashboard</h1>
    
    <div class="stats">
        <h2>Statistics</h2>
        <p>Total Users: <?= number_format($stats['total_users'] ?? 0) ?></p>
        <p>Active Users: <?= number_format($stats['active_users'] ?? 0) ?></p>
        <p>Recent Orders: <?= number_format($stats['recent_orders'] ?? 0) ?></p>
        <p>30-Day Revenue: $<?= number_format($stats['revenue_30_days'] ?? 0, 2) ?></p>
    </div>
    
    <div class="users">
        <h2>Recent Users</h2>
        <table>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
            </tr>
            <?php foreach ($users as $user): ?>
            <tr>
                <td><?= $user['id'] ?></td>
                <td><?= htmlspecialchars($user['name']) ?></td>
                <td><?= htmlspecialchars($user['email']) ?></td>
            </tr>
            <?php endforeach; ?>
        </table>
    </div>
</body>
</html>
```

## Security Considerations

### Securing PHP Endpoints

When exposing PHP endpoints, consider these security practices:

```go
// Create a security middleware
securityMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set security headers
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Check for sensitive paths
        if strings.Contains(r.URL.Path, "/admin") {
            // Additional security for admin endpoints
            if !isSecureRequest(r) {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
        }
        
        next.ServeHTTP(w, r)
    })
}

// Apply the security middleware
securePHP := securityMiddleware(php)
mux.Handle("/", securePHP)
```

### Environment-Specific Configuration

Configure your middleware differently based on the environment:

```go
package main

import (
    "log"
    "net/http"
    "os"
    "github.com/davidroman0O/frango"
)

func main() {
    // Determine environment
    env := os.Getenv("APP_ENV")
    if env == "" {
        env = "development" // Default to development
    }
    
    // Base options
    options := []frango.Option{
        frango.WithSourceDir("web"),
    }
    
    // Environment-specific options
    switch env {
    case "production":
        options = append(options,
            frango.WithDevelopmentMode(false),
            frango.WithLogger(log.New(os.Stdout, "[prod] ", log.LstdFlags)),
        )
    case "staging":
        options = append(options,
            frango.WithDevelopmentMode(false),
            frango.WithLogger(log.New(os.Stdout, "[staging] ", log.LstdFlags)),
        )
    default: // development
        options = append(options,
            frango.WithDevelopmentMode(true),
            frango.WithLogger(log.New(os.Stdout, "[dev] ", log.LstdFlags|log.Lshortfile)),
        )
    }
    
    // Create middleware with environment-specific options
    php, err := frango.New(options...)
    if err != nil {
        log.Fatalf("Error creating PHP middleware: %v", err)
    }
    defer php.Shutdown()
    
    // Set up endpoints
    setupEndpoints(php, env)
    
    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8082" // Default port
    }
    
    log.Printf("Starting server in %s mode on port %s", env, port)
    if err := http.ListenAndServe(":"+port, php); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}

func setupEndpoints(php *frango.Middleware, env string) {
    // Common endpoints
    php.HandlePHP("/", "index.php")
    php.HandlePHP("/about", "about.php")
    php.HandlePHP("/contact", "contact.php")
    
    // Register API endpoints
    php.HandleDir("/api", "api")
    
    // Environment-specific endpoints
    if env == "development" {
        // Development-only endpoints
        php.HandlePHP("/dev/debug", "dev/debug.php")
        php.HandlePHP("/dev/phpinfo", "dev/phpinfo.php")
    }
    
    if env != "production" {
        // Non-production endpoints
        php.HandlePHP("/test", "test.php")
    }
} 