# HandleDir Example

This example demonstrates how to use the `HandleDir` function in Go-PHP to automatically register PHP files in a directory structure.

## Features

- Automatically registers all PHP files under a URL prefix
- Creates clean URLs without .php extensions
- Works with nested directories
- Supports both relative and absolute paths

## Directory Structure

```
web/
  ├── index.php           # Main page with links
  ├── pages/              # Pages directory
  │   ├── about.php       # About page
  │   └── contact.php     # Contact page
  └── api/                # API directory
      ├── users.php       # Users API endpoint
      └── items.php       # Items API endpoint
```

## How it Works

1. The `HandleDir` function scans a directory for PHP files
2. It automatically registers each file under the specified URL prefix
3. All PHP files are accessible with or without the `.php` extension
4. The original directory structure is preserved in the URL paths

## Running the Example

```bash
# From the go-php directory
go run -tags=nowatcher ./examples/handledir
```

Then open your browser to `http://localhost:8082/`

> **Important**: The `nowatcher` build tag is required to make this example work properly.

## Key Code

The main functionality is provided by the `HandleDir` function:

```go
// Register all PHP files in the "pages" directory under the "/pages" URL prefix
if err := server.HandleDir("/pages", "pages"); err != nil {
    log.Fatalf("Error registering pages directory: %v", err)
}

// Register all PHP files in the "api" directory under the "/api" URL prefix
if err := server.HandleDir("/api", "api"); err != nil {
    log.Fatalf("Error registering API directory: %v", err)
}
```

This automatically makes all PHP files in the "pages" and "api" directories available under the "/pages" and "/api" URL prefixes, respectively. 