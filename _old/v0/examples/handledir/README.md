# HandleDir Example

This example demonstrates how to use Frango's `HandleDir` function to register all PHP files in a directory for serving.

## Features

- Automatically mapping all PHP files in a directory
- Serving an entire directory structure with one command
- Preserving the directory hierarchy in URL paths
- Automatically creating clean URLs for all files

## Directory Structure

```
web/
  ├── index.php     # Root index file
  ├── about.php     # About page
  ├── contact.php   # Contact page
  ├── blog/         # Blog directory
  │   ├── index.php # Blog index
  │   ├── post1.php # Individual blog post
  │   └── post2.php # Another blog post
  └── docs/         # Documentation directory
      ├── index.php # Docs index
      ├── guide.php # Guide page
      └── api.php   # API documentation
```

## How it Works

The example registers all PHP files in the `web` directory and makes them available at corresponding URLs:

- `web/index.php` → `/index.php` or `/`
- `web/about.php` → `/about.php` or `/about`
- `web/blog/index.php` → `/blog/index.php` or `/blog/`
- `web/blog/post1.php` → `/blog/post1.php` or `/blog/post1`

The `HandleDir` function automatically:

1. Recursively finds all PHP files in the specified directory
2. Registers them with appropriate URL patterns
3. Creates clean URLs without the .php extension
4. Sets up directory indexes using index.php files

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/handledir
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

```go
// Create PHP middleware
php, err := frango.New(
    frango.WithSourceDir(webDir),
    frango.WithDevelopmentMode(true),
)

// Register the entire web directory at the root
err = php.HandleDir("/", "")
if err != nil {
    log.Fatalf("Error registering directory: %v", err)
}

// Start the server with the PHP middleware
log.Println("Server starting on :8082")
if err := http.ListenAndServe(":8082", php); err != nil {
    log.Fatalf("Server error: %v", err)
}
```

This pattern is particularly useful for porting existing PHP applications to Go with minimal changes. 