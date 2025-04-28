# Understanding Frango's Virtual Filesystem (VFS)

The Virtual Filesystem (VFS) is one of Frango's most powerful features. It provides an abstraction layer for managing PHP files, allowing you to seamlessly use files from different sources - physical files, embedded content, and dynamically generated content.

## What is the VFS?

The VFS in Frango is a component that:

1. **Manages PHP files** from multiple sources
2. **Virtualizes file paths** into a unified namespace
3. **Handles file changes** in development mode
4. **Optimizes performance** with caching
5. **Ensures security** by preventing path traversal attacks

## Why Use the VFS?

Using the VFS offers several benefits:

- **Embedding PHP files** directly into your Go binary for easy deployment
- **Hot-reloading** PHP files during development
- **Branching** to create isolated environments
- **Dynamic content generation** without writing to disk
- **Secure file access** with built-in protections

## VFS Sources

The VFS can obtain PHP files from three main sources:

1. **Source Files** - Files from your local filesystem
2. **Embedded Files** - Files embedded in your Go binary (using Go's embed package)
3. **Virtual Files** - Files generated dynamically in memory

Let's explore how to use each of these sources.

## Basic VFS Setup

The VFS is automatically initialized when you create a Frango middleware instance. Here's a basic example:

```go
package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Initialize Frango with a source directory
	php, err := frango.New(
		frango.WithSourceDir("./php-files"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}
	defer php.Shutdown()

	// The php-files directory is now accessible through the VFS
	http.Handle("/", php.For("/index.php"))
	
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

In this example, the VFS is automatically set up to use files from the `./php-files` directory.

## Adding Files to the VFS

### 1. Source Files from the Filesystem

You can add individual files or entire directories from your local filesystem:

```go
// Add a single file
err := php.AddSourceFile("/path/to/local/file.php", "/virtual/path/file.php")
if err != nil {
    log.Fatalf("Failed to add source file: %v", err)
}

// Add all files from a directory
err = php.AddSourceDirectory("/path/to/local/dir", "/virtual/path")
if err != nil {
    log.Fatalf("Failed to add source directory: %v", err)
}
```

### 2. Embedded Files

Go 1.16+ allows embedding files directly into your binary using the `embed` package. This is great for deployment as it eliminates the need to ship PHP files separately:

```go
package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

//go:embed templates/*.php
var templates embed.FS

func main() {
	php, err := frango.New()
	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Add embedded files to the VFS
	err = php.AddEmbeddedDirectory(templates, "templates", "/")
	if err != nil {
		log.Fatalf("Failed to add embedded directory: %v", err)
	}

	// The embedded PHP files are now accessible
	http.Handle("/", php.For("/index.php"))
	
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### 3. Virtual Files (Generated Content)

You can create PHP files dynamically in memory without writing them to disk:

```go
// Create a virtual PHP file
content := []byte(`<?php echo "This file was generated dynamically at " . date('Y-m-d H:i:s'); ?>`)
err := php.CreateVirtualFile("/virtual/dynamic.php", content)
if err != nil {
    log.Fatalf("Failed to create virtual file: %v", err)
}

// The virtual file is now accessible
http.Handle("/dynamic", php.For("/virtual/dynamic.php"))
```

This is particularly useful for:
- Generating PHP files from templates
- Creating temporary PHP scripts for specific use cases
- Testing different PHP code versions without filesystem changes

## VFS Branching

One of the most powerful features of Frango's VFS is the ability to create branches. A branch inherits files from its parent VFS but can add or override files without affecting the parent.

This is useful for:
- Creating isolated environments for different requests
- Testing changes without affecting the main VFS
- Implementing multi-tenancy where each tenant gets its own VFS branch

Here's how to create and use a VFS branch:

```go
func main() {
	// Create the root VFS
	php, err := frango.New(
		frango.WithSourceDir("./shared-php-files"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer php.Shutdown()

	// Handler that creates a customized VFS branch for each tenant
	http.HandleFunc("/tenant/", func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant ID from URL
		tenantID := r.URL.Path[len("/tenant/"):]
		if tenantID == "" {
			http.Error(w, "Tenant ID required", http.StatusBadRequest)
			return
		}

		// Create a VFS branch
		tenantVFS := php.NewVFS()
		
		// Add tenant-specific virtual file
		customWelcome := []byte(`<?php echo "Welcome to tenant: <?= htmlspecialchars($tenantID) ?>"; ?>`)
		err := tenantVFS.CreateVirtualFile("/welcome.php", customWelcome)
		if err != nil {
			http.Error(w, "Error creating tenant file", http.StatusInternalServerError)
			return
		}
		
		// Render using the tenant-specific VFS branch
		php.RenderVFS(tenantVFS, "/welcome.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"tenantID": tenantID,
			}
		}).ServeHTTP(w, r)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Each tenant gets its own isolated VFS environment, but they all share the base files from the parent VFS.

## Development Mode

When development mode is enabled, the VFS continuously monitors source files for changes and automatically refreshes them:

```go
php, err := frango.New(
	frango.WithSourceDir("./php-files"),
	frango.WithDevelopmentMode(true),  // Enable development mode
)
```

This allows you to:
1. Edit PHP files while the application is running
2. See changes immediately without restarting the server
3. Rapidly iterate on your PHP code

In production, you would disable development mode to optimize performance:

```go
php, err := frango.New(
	frango.WithSourceDir("./php-files"),
	frango.WithDevelopmentMode(false),  // Disable for production
)
```

## File Operations

The VFS supports various file operations:

```go
// Check if a file exists
exists := vfs.FileExists("/virtual/path/file.php")

// Get the real disk path for a virtual file
diskPath, err := vfs.ResolvePath("/virtual/path/file.php")

// Copy a file within the VFS
err = vfs.CopyFile("/virtual/source.php", "/virtual/destination.php")

// Move a file within the VFS
err = vfs.MoveFile("/virtual/old-path.php", "/virtual/new-path.php")

// Delete a file from the VFS
err = vfs.DeleteFile("/virtual/path/file.php")
```

## Advanced: Custom VFS for Different Routes

You can create multiple VFS instances for different parts of your application:

```go
func main() {
	// Create a VFS for admin area
	adminPHP, err := frango.New(
		frango.WithSourceDir("./admin-templates"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer adminPHP.Shutdown()

	// Create a VFS for public area
	publicPHP, err := frango.New(
		frango.WithSourceDir("./public-templates"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer publicPHP.Shutdown()

	// Public routes use public VFS
	http.Handle("/", publicPHP.For("/index.php"))
	http.Handle("/products/", publicPHP.For("/products/index.php"))

	// Admin routes use admin VFS
	http.Handle("/admin/", adminPHP.For("/admin.php"))
	http.Handle("/admin/dashboard/", adminPHP.For("/dashboard.php"))

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

This approach provides isolation between different areas of your application.

## Complete VFS Example

Let's put everything together in a complete example that demonstrates various VFS features:

```go
package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davidroman0O/frango/v1"
)

//go:embed embedded-templates/*.php
var embeddedTemplates embed.FS

func main() {
	// Create base Frango instance
	php, err := frango.New(
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}
	defer php.Shutdown()

	// 1. Add source directory
	if _, err := os.Stat("./templates"); err == nil {
		err = php.AddSourceDirectory("./templates", "/")
		if err != nil {
			log.Printf("Warning: Failed to add source directory: %v", err)
		} else {
			log.Println("Added source directory: ./templates")
		}
	}

	// 2. Add embedded files
	err = php.AddEmbeddedDirectory(embeddedTemplates, "embedded-templates", "/embedded")
	if err != nil {
		log.Printf("Warning: Failed to add embedded templates: %v", err)
	} else {
		log.Println("Added embedded templates")
	}

	// 3. Create a dynamic PHP file
	dynamicContent := []byte(`<?php
	$time = date('Y-m-d H:i:s');
	echo "<h1>Dynamic PHP File</h1>";
	echo "<p>This file was generated dynamically by the application.</p>";
	echo "<p>Current server time: {$time}</p>";
	?>`)

	err = php.CreateVirtualFile("/dynamic.php", dynamicContent)
	if err != nil {
		log.Printf("Warning: Failed to create dynamic file: %v", err)
	} else {
		log.Println("Created dynamic PHP file: /dynamic.php")
	}

	// 4. Create a branch VFS with overridden files
	branchVFS := php.NewVFS()
	branchContent := []byte(`<?php
	echo "<h1>Branch-Specific Page</h1>";
	echo "<p>This file exists only in the branch VFS.</p>";
	?>`)

	err = branchVFS.CreateVirtualFile("/branch.php", branchContent)
	if err != nil {
		log.Printf("Warning: Failed to create file in branch VFS: %v", err)
	}

	// Define routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve an index page with links to demos
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Frango VFS Demo</title>
			<style>
				body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
				h1 { color: #333; }
				ul { line-height: 1.6; }
			</style>
		</head>
		<body>
			<h1>Frango VFS Demo</h1>
			<ul>
				<li><a href="/source">PHP file from source directory</a></li>
				<li><a href="/embedded">PHP file from embedded template</a></li>
				<li><a href="/dynamic">Dynamically generated PHP file</a></li>
				<li><a href="/branch">PHP file from branch VFS</a></li>
				<li><a href="/vfs-info">VFS Information</a></li>
			</ul>
		</body>
		</html>
		`))
	})

	// Route for source file
	http.Handle("/source", php.For("/index.php"))

	// Route for embedded file
	http.Handle("/embedded", php.For("/embedded/index.php"))

	// Route for dynamic file
	http.Handle("/dynamic", php.For("/dynamic.php"))

	// Route using branch VFS
	http.Handle("/branch", php.ForVFS(branchVFS, "/branch.php"))

	// Route to show VFS info
	http.HandleFunc("/vfs-info", func(w http.ResponseWriter, r *http.Request) {
		// Create a simple report about the VFS
		w.Header().Set("Content-Type", "text/html")
		
		// Get all available files
		sourceFiles := "Could not list files"
		tmpDir := php.TempDir()
		if files, err := filepath.Glob(filepath.Join(tmpDir, "*.php")); err == nil {
			sourceFiles = ""
			for _, file := range files {
				sourceFiles += "- " + filepath.Base(file) + "<br>"
			}
		}

		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>VFS Information</title>
			<style>
				body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
				h1, h2 { color: #333; }
				.info-section { margin-bottom: 30px; background: #f9f9f9; padding: 15px; border-radius: 5px; }
			</style>
		</head>
		<body>
			<h1>VFS Information</h1>

			<div class="info-section">
				<h2>VFS Configuration</h2>
				<p><strong>Source Directory:</strong> ` + php.SourceDir() + `</p>
				<p><strong>Temp Directory:</strong> ` + php.TempDir() + `</p>
			</div>

			<div class="info-section">
				<h2>Available PHP Files</h2>
				` + sourceFiles + `
			</div>

			<p><a href="/">Back to Demo Index</a></p>
		</body>
		</html>
		`))
	})

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

To run this example:

1. Create a directory named `templates` with a file named `index.php`
2. Create a directory named `embedded-templates` with a file named `index.php`
3. Build and run the application

## Best Practices

Here are some best practices for working with Frango's VFS:

1. **Organize your files logically**
   - Keep a clear structure that mirrors your URL paths
   - Use consistent naming conventions

2. **Use embedding for production**
   - Embed PHP files in the Go binary for simplified deployment
   - Use source files during development for easier editing

3. **Use branching for isolation**
   - Create branches when you need separate environments
   - Clean up branches when they're no longer needed

4. **Be mindful of file watching**
   - In development mode, file watching consumes resources
   - For large projects, consider selective watching

5. **Handle errors appropriately**
   - Check for errors when adding files to the VFS
   - Have fallback strategies for missing files

## Troubleshooting

Here are some common issues and their solutions:

1. **Files not found**
   - Verify the virtual path is correct (should start with '/')
   - Check if the file exists in the VFS using `FileExists()`
   - Ensure the source directory path is absolute

2. **Changes not reflecting**
   - Confirm development mode is enabled
   - Check file permissions
   - Verify you're editing the correct file

3. **Performance issues**
   - Disable development mode in production
   - Use embedding instead of source files
   - Minimize dynamic file generation

4. **Memory usage**
   - Clean up unused VFS branches
   - Be careful with large files in virtual memory
   - Use `Shutdown()` to free resources

## Conclusion

Frango's Virtual Filesystem is a powerful abstraction that simplifies PHP file management in your Go applications. It provides flexibility in how you source, organize, and serve PHP files while maintaining security and optimizing performance.

By understanding how to effectively use the VFS, you can build more maintainable and deployable Go-PHP hybrid applications.

For more advanced use cases, see:
- [Path Parameters Tutorial](./path-parameters.md) - for integrating URL patterns with VFS
- [Embedding Examples](../examples/embedding.md) - for embedding PHP in Go binaries
- [Production Deployment Guide](../guides/production-deployment.md) - for optimizing VFS in production 