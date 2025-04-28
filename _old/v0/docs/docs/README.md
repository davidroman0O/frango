# Frango Documentation

Welcome to the official documentation for Frango, the powerful Go middleware for seamlessly integrating PHP into Go applications.

## What is Frango?

Frango is a middleware that allows you to use PHP scripts within your Go applications, leveraging FrankenPHP as the underlying PHP runtime. This enables you to:

- Gradually migrate from PHP to Go
- Use PHP for templating while leveraging Go's performance and concurrency
- Access the PHP ecosystem from Go applications
- Build hybrid applications that leverage the strengths of both languages

## Documentation Structure

Our documentation is organized into several sections to help you quickly find what you need:

### 🚀 [Getting Started](./tutorials/getting-started.md)
Begin your journey with Frango by installing, setting up, and creating your first hybrid Go-PHP application.

### 📚 Tutorials
Step-by-step guides to accomplish specific tasks:
- [Basic Setup](./tutorials/basic-setup.md): Configure Frango in a Go application
- [Form Handling](./tutorials/form-handling.md): Process form submissions with PHP
- [File Uploads](./tutorials/file-uploads.md): Handle file uploads between Go and PHP
- [JSON Processing](./tutorials/json-processing.md): Work with JSON data across languages
- [Path Parameters](./tutorials/path-parameters.md): Use dynamic URL patterns
- [Templating](./tutorials/templating.md): Use PHP for view rendering
- [Virtual Filesystem](./tutorials/virtual-filesystem.md): Understand and utilize the VFS

### 📖 Guides
Conceptual explanations of Frango's architecture and features:
- [Architecture Overview](./guides/architecture.md): Understand how Frango works
- [Production Deployment](./guides/production-deployment.md): Best practices for deploying to production
- [Performance Optimization](./guides/performance.md): Tips for maximizing performance
- [Security Considerations](./guides/security.md): Keeping your applications secure
- [Error Handling](./guides/error-handling.md): Deal with errors in PHP and Go
- [Testing Strategies](./guides/testing.md): Test your hybrid applications

### 📋 API Reference
Comprehensive reference for Frango's API:
- [Middleware](./api-reference/middleware.md): Core middleware functions
- [VFS](./api-reference/vfs.md): Virtual filesystem operations
- [Request/Response](./api-reference/request-response.md): HTTP handling functionality
- [PHP Superglobals](./api-reference/php-superglobals.md): PHP environment variables
- [Configuration Options](./api-reference/options.md): Available configuration options

### 💡 Examples
Complete examples demonstrating common usage patterns:
- [Basic Web Server](./examples/basic-web-server.md): Simple HTTP server with PHP
- [RESTful API](./examples/restful-api.md): Build a REST API with PHP and Go
- [Form Processing Application](./examples/form-processing.md): Complete form handling example
- [Content Management System](./examples/cms.md): Build a simple CMS with Frango
- [Authentication System](./examples/authentication.md): Implement user authentication

## Key Concepts

Before diving in, it's helpful to understand these core concepts in Frango:

### Middleware

Frango functions as a standard Go HTTP middleware, allowing you to selectively apply PHP processing to specific routes in your application.

### Virtual Filesystem (VFS)

Frango's VFS provides a unified way to manage PHP files, whether they're stored on disk, embedded in your Go binary, or generated dynamically.

### Request Flow

Understanding how requests flow through the middleware is critical:

1. An HTTP request reaches your Go application
2. Frango middleware processes the request
3. Request data is transformed into PHP environment variables
4. The PHP script is executed by FrankenPHP
5. Output from PHP is returned as the HTTP response

### PHP Superglobals

Frango automatically populates PHP superglobals like `$_GET`, `$_POST`, and custom ones like `$_PATH` and `$_JSON`.

## Getting Help

If you encounter any issues or have questions:
- Check the [Troubleshooting Guide](./guides/troubleshooting.md)
- Use the debugging tools described in [Debugging](./guides/debugging.md)
- Review common patterns in the [Examples](./examples/) section

Now you're ready to begin! Head to the [Getting Started](./tutorials/getting-started.md) guide to set up your first Frango application.

## Documentation Status

The Frango documentation is still under active development. Several key files have been updated to match the current implementation, but you may encounter some inconsistencies in other parts of the documentation.

### Recent Updates

The following documentation sections have been aligned with the current code implementation:

- **API Reference**:
  - VFS documentation updated to match actual implementation (string-based FileOrigin, proper VFS methods)
  - Request/Response handling documentation updated with accurate path parameter handling
  - PHP Superglobals documentation improved with correct information about form processing

### Known Inconsistencies

Some parts of the documentation may still reference features that are not fully implemented:

- FrankenPHP worker mode (mentioned in FrankenPHP.md but not implemented in the codebase)
- Certain APIs may have differences in parameter types or method signatures

If you find any inconsistencies between the documentation and the code, please submit an issue or pull request to help improve the documentation. 