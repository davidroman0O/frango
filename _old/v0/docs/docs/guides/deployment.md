# Deployment Guide for Frango Applications

This guide covers best practices for deploying Frango applications in production environments, including setup, optimization, security, and scaling considerations.

## Table of Contents

- [Overview](#overview)
- [Development vs Production Mode](#development-vs-production-mode)
- [Optimizing for Production](#optimizing-for-production)
- [Deployment Options](#deployment-options)
  - [Standalone Server](#standalone-server)
  - [Behind a Reverse Proxy](#behind-a-reverse-proxy)
  - [Containerized Deployment](#containerized-deployment)
- [Security Considerations](#security-considerations)
- [Performance Tuning](#performance-tuning)
- [Scaling Strategies](#scaling-strategies)
- [Monitoring and Logging](#monitoring-and-logging)
- [High Availability Setup](#high-availability-setup)
- [Deployment Checklist](#deployment-checklist)

## Overview

Deploying a Frango application to production requires careful consideration of performance, security, and reliability. Unlike traditional PHP applications that rely on a web server like Apache or Nginx with PHP-FPM, Frango applications are Go executables that embed PHP functionality.

This architectural difference means that deployment strategies for Frango applications are more aligned with Go applications than traditional PHP applications.

## Development vs Production Mode

Frango provides development and production modes with different default behaviors:

### Development Mode

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDevelopmentMode(true), // Enable development mode
)
```

Development mode features:
- Real-time file change detection
- Detailed error reporting
- Minimal caching
- More verbose logging

### Production Mode

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDevelopmentMode(false), // Disable development mode (default)
)
```

Production mode features:
- No real-time file change detection (better performance)
- Minimal error reporting (no PHP errors displayed to end users)
- Optimized caching
- Reduced logging verbosity

## Optimizing for Production

### PHP Configuration

Tune PHP settings for production using the `WithPHPIniEntries` option:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithPHPIniEntries(map[string]string{
        // Error handling
        "display_errors": "Off",
        "log_errors": "On",
        "error_log": "/var/log/php_errors.log",
        
        // Performance
        "opcache.enable": "1",
        "opcache.memory_consumption": "128",
        "opcache.interned_strings_buffer": "8",
        "opcache.max_accelerated_files": "4000",
        "opcache.revalidate_freq": "60",
        "opcache.fast_shutdown": "1",
        
        // Memory limits
        "memory_limit": "256M",
        
        // File uploads (if needed)
        "upload_max_filesize": "10M",
        "post_max_size": "12M",
        
        // Session handling
        "session.save_handler": "files",
        "session.save_path": "/tmp",
        "session.gc_maxlifetime": "1440",
    }),
)
```

### File System Optimization

1. **Embedded Files**: For production, consider embedding static PHP files into your Go binary:

```go
//go:embed php-files/*
var phpFiles embed.FS

func main() {
    php, err := frango.New(
        frango.WithEmbeddedFiles(phpFiles, "php-files", "/"),
    )
    // ...
}
```

2. **ReadOnly Mode**: If your application doesn't need to write to the virtual file system, consider using read-only mode:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithReadOnlyMode(true),
)
```

### Binary Optimization

When building your Go application for production, use build flags to optimize the binary:

```bash
go build -ldflags="-s -w" -o myapp
```

The `-s -w` flags strip debugging information, resulting in a smaller binary.

## Deployment Options

### Standalone Server

The simplest deployment method is running your Frango application as a standalone server:

```go
func main() {
    // Initialize Frango
    php, err := frango.New(
        frango.WithSourceDir("./php-files"),
    )
    if err != nil {
        log.Fatalf("Failed to initialize Frango: %v", err)
    }
    defer php.Shutdown()
    
    // Set up routes
    http.Handle("/", php.For("/index.php"))
    // ...additional routes...
    
    // Start the server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

To run the server in the background as a service, you can use systemd on Linux:

```
[Unit]
Description=Frango Application
After=network.target

[Service]
User=appuser
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/myapp
Restart=always
RestartSec=5
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=frangoapp
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

### Behind a Reverse Proxy

For production deployments, it's often better to run your Frango application behind a reverse proxy like Nginx:

**Nginx Configuration:**

```nginx
server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support (if needed)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    # Serve static assets directly from Nginx for better performance
    location /static/ {
        alias /path/to/static/files/;
        expires 30d;
    }
}
```

**In your Frango application:**

```go
func main() {
    // ...
    
    // Trust proxy headers
    http.Handle("/", php.For("/index.php"))
    
    // Listen only on localhost
    if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### Containerized Deployment

Containerizing your Frango application with Docker provides consistency across environments:

**Dockerfile:**

```dockerfile
# Build stage
FROM golang:1.18-alpine AS builder

WORKDIR /app

# Install PHP and required extensions
RUN apk add --no-cache php81 php81-fpm php81-opcache php81-json php81-phar php81-openssl php81-gd

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -ldflags="-s -w" -o frangoapp

# Final stage
FROM alpine:3.16

# Install PHP runtime
RUN apk add --no-cache php81 php81-fpm php81-opcache php81-json php81-phar php81-openssl php81-gd

WORKDIR /app

# Copy only the necessary files from the builder stage
COPY --from=builder /app/frangoapp .
COPY --from=builder /app/php-files ./php-files

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["/app/frangoapp"]
```

**docker-compose.yml for local testing:**

```yaml
version: '3'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - GO_ENV=production
    volumes:
      - ./logs:/app/logs
```

## Security Considerations

### PHP Security

1. **Restrict Direct Access to PHP Files**: Block direct access to PHP files:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDirectPHPURLsBlocking(true), // Default is true
)
```

2. **Custom Error Handler**: Implement a custom error handler to avoid exposing sensitive information:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithErrorHandler("/error_handler.php"),
)
```

Create a custom error handler in `/php-files/error_handler.php`:

```php
<?php
// Production error handler
function myErrorHandler($errno, $errstr, $errfile, $errline) {
    // Log the error
    error_log("PHP Error [$errno]: $errstr in $errfile on line $errline");
    
    // In production, show a generic error page
    if (in_array(php_sapi_name(), array('cgi', 'cgi-fcgi', 'fpm-fcgi'))) {
        header('HTTP/1.1 500 Internal Server Error');
    }
    
    echo "<h1>Something went wrong</h1>";
    echo "<p>We're sorry, but there was an error processing your request.</p>";
    
    // Don't execute PHP's internal error handler
    return true;
}

// Set the error handler
set_error_handler("myErrorHandler");
?>
```

3. **Input Validation**: Always validate and sanitize user inputs in your PHP code:

```php
// Validate and sanitize user input
$id = filter_input(INPUT_GET, 'id', FILTER_VALIDATE_INT);
if ($id === false || $id === null) {
    // Invalid input, handle the error
    http_response_code(400);
    echo "Invalid ID parameter";
    exit;
}

// Escape output
echo htmlspecialchars($userInput, ENT_QUOTES, 'UTF-8');
```

### Server Security

1. **Running as Non-Root User**: If deploying with Docker, run your container as a non-root user:

```dockerfile
# Add a non-root user
RUN adduser -D appuser

# Change ownership of the application directory
RUN chown -R appuser:appuser /app

# Switch to the non-root user
USER appuser

# Run the application
CMD ["/app/frangoapp"]
```

2. **HTTPS**: Always use HTTPS in production. With a reverse proxy:

```nginx
server {
    listen 443 ssl;
    server_name example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    location / {
        proxy_pass http://localhost:8080;
        # ... other proxy settings ...
    }
}
```

3. **Security Headers**: Add security headers in your application:

```go
func securityMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Add security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        next.ServeHTTP(w, r)
    })
}

func main() {
    // ...
    
    // Apply security middleware to all routes
    http.Handle("/", securityMiddleware(php.For("/index.php")))
    
    // ...
}
```

## Performance Tuning

### Go Performance

1. **Connection Pooling**: For database connections, use connection pooling:

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

func initDB() *sql.DB {
    db, err := sql.Open("mysql", "user:password@tcp(127.0.0.1:3306)/dbname")
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    
    // Set connection pool parameters
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    return db
}
```

2. **Memory Management**: Monitor memory usage and adjust settings if needed:

```go
import "runtime/debug"

// Set garbage collection target percentage
debug.SetGCPercent(100) // Default is 100

// Force garbage collection if needed (rarely necessary)
// debug.FreeOSMemory()
```

### PHP Performance

1. **Opcache Settings**: Optimize opcache settings for production:

```go
phpIniEntries := map[string]string{
    "opcache.enable": "1",
    "opcache.memory_consumption": "128",
    "opcache.interned_strings_buffer": "8",
    "opcache.max_accelerated_files": "4000",
    "opcache.revalidate_freq": "60",
    "opcache.fast_shutdown": "1",
    "opcache.enable_cli": "1",
}

php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithPHPIniEntries(phpIniEntries),
)
```

2. **Minimize PHP Execution**: For static content or simple responses, use Go instead of PHP when possible.

3. **PHP File Organization**: Organize PHP files to minimize includes and requires:

```php
// Use a bootstrap file to load common dependencies
require_once 'bootstrap.php';

// Use autoloading for classes
spl_autoload_register(function($class) {
    $file = str_replace('\\', '/', $class) . '.php';
    if (file_exists($file)) {
        require $file;
        return true;
    }
    return false;
});
```

## Scaling Strategies

### Horizontal Scaling

To scale your application horizontally, you'll need to make it stateless or implement shared state:

1. **Session Management**: Use external session storage instead of local files:

```php
// In PHP, configure sessions to use Redis
ini_set('session.save_handler', 'redis');
ini_set('session.save_path', 'tcp://redis-server:6379');
```

2. **Load Balancing**: Deploy multiple instances behind a load balancer:

```
[Client] -> [Load Balancer] -> [Frango Instance 1]
                            -> [Frango Instance 2]
                            -> [Frango Instance 3]
```

Nginx load balancer configuration:

```nginx
upstream frangoapp {
    server app1.example.com:8080;
    server app2.example.com:8080;
    server app3.example.com:8080;
}

server {
    listen 80;
    server_name example.com;
    
    location / {
        proxy_pass http://frangoapp;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        # ... other headers ...
    }
}
```

3. **Shared Storage**: If your application writes files that need to be accessible across instances, use shared storage:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    // Configure any temporary files to be written to shared storage
)
```

### Vertical Scaling

Vertical scaling involves increasing resources on a single server:

1. **CPU Optimization**: For CPU-bound applications, use Go's concurrency features:

```go
import "runtime"

func init() {
    // Use all available CPU cores
    runtime.GOMAXPROCS(runtime.NumCPU())
}
```

2. **Memory Optimization**: Monitor and optimize memory usage:

```go
// Track memory allocations in development
// import "runtime/pprof"
// 
// f, _ := os.Create("mem.pprof")
// pprof.WriteHeapProfile(f)
// f.Close()
```

## Monitoring and Logging

### Application Logging

Implement structured logging in your application:

```go
import (
    "log"
    "os"
)

func setupLogging() *log.Logger {
    // Create a logger that writes to a file and stdout
    logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("Failed to open log file: %v", err)
    }
    
    multiWriter := io.MultiWriter(os.Stdout, logFile)
    return log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
}

func main() {
    logger := setupLogging()
    
    php, err := frango.New(
        frango.WithSourceDir("./php-files"),
        frango.WithLogger(logger),
    )
    // ...
}
```

### Health Checks

Implement health check endpoints for monitoring:

```go
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // Check critical dependencies (database, cache, etc.)
    dbOk := checkDatabaseConnection()
    cacheOk := checkCacheConnection()
    
    if !dbOk || !cacheOk {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "error",
            "database": dbOk,
            "cache": cacheOk,
        })
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ok",
        "version": "1.0.0",
    })
}

func main() {
    // ...
    
    // Add health check route
    http.HandleFunc("/health", healthCheckHandler)
    
    // ...
}
```

### Metrics

For more advanced monitoring, integrate with Prometheus or similar tools:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func setupMetrics() {
    // Create metrics
    requestCounter := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Count of HTTP requests processed, partitioned by status code and method",
        },
        []string{"code", "method"},
    )
    
    requestDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"handler"},
    )
    
    // Register metrics
    prometheus.MustRegister(requestCounter, requestDuration)
}

func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create response wrapper to capture status code
        ww := &responseWriter{w, http.StatusOK}
        
        // Call the next handler
        next.ServeHTTP(ww, r)
        
        // Record metrics
        duration := time.Since(start).Seconds()
        requestDuration.WithLabelValues(r.URL.Path).Observe(duration)
        requestCounter.WithLabelValues(strconv.Itoa(ww.status), r.Method).Inc()
    })
}

// Response writer wrapper to capture status code
type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}

func main() {
    // Setup metrics
    setupMetrics()
    
    // ...
    
    // Add prometheus metrics endpoint
    http.Handle("/metrics", promhttp.Handler())
    
    // Apply metrics middleware to routes
    http.Handle("/", metricsMiddleware(php.For("/index.php")))
    
    // ...
}
```

## High Availability Setup

For mission-critical applications, consider a high availability setup:

1. **Multiple Regions/Zones**: Deploy in multiple cloud regions or availability zones.

2. **Database Replication**: Set up database replication for failover.

3. **Automatic Failover**: Configure your load balancer for health-based routing.

4. **Blue-Green Deployment**: Use blue-green deployment for zero-downtime updates:

```
             ┌─────────────────┐
             │   Load Balancer │
             └────────┬────────┘
                      │
           ┌──────────┴──────────┐
           │                     │
┌──────────▼─────────┐ ┌─────────▼──────────┐
│  Blue Environment  │ │  Green Environment │
│  (Current Active)  │ │  (Staged Version)  │
└────────────────────┘ └────────────────────┘
```

During deployment, traffic is gradually shifted from Blue to Green.

## Deployment Checklist

Before deploying to production, go through this checklist:

- [ ] **Turn off development mode**: `frango.WithDevelopmentMode(false)`
- [ ] **Configure PHP for production**: Set appropriate php.ini values
- [ ] **Implement logging**: Ensure logs are captured and rotated
- [ ] **Set up monitoring**: Add health checks and metrics
- [ ] **Secure the application**: Follow security best practices
- [ ] **Test performance**: Load test the application
- [ ] **Check for memory leaks**: Run with extended test periods
- [ ] **Automate deployment**: Set up CI/CD pipelines
- [ ] **Implement backup strategy**: For data and configurations
- [ ] **Document deployment process**: Include rollback procedures
- [ ] **Set up alerting**: For critical errors and performance issues

## Conclusion

Deploying Frango applications to production combines aspects of both Go and PHP deployment strategies. By following the best practices outlined in this guide, you can ensure your application is secure, performant, and reliable in a production environment.

Remember that every application is unique, so adapt these recommendations to your specific requirements and constraints. 