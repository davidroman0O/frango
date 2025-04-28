# Frango Documentation

Welcome to the official documentation for Frango, the PHP middleware for Go that enables seamless integration of PHP with Go applications.

## What is Frango?

Frango is a Go middleware that allows you to execute PHP code within your Go applications. It provides a virtual file system, request/response handling, and everything you need to leverage PHP's strengths within the Go ecosystem.

## Contents

### Getting Started

- [Getting Started Guide](tutorials/getting-started.md) - Set up your first Frango project

### Tutorials

- [Form Handling](tutorials/form-handling.md) - Learn how to handle GET and POST forms
- [File Uploads](tutorials/file-uploads.md) - Implement file uploads in Frango
- [Virtual Filesystem](tutorials/virtual-filesystem.md) - Learn about Frango's virtual file system

### Guides

- [Architecture Overview](guides/architecture.md) - Understand Frango's architecture
- [Deployment Guide](guides/deployment.md) - Best practices for deploying Frango applications
- [Troubleshooting Guide](guides/troubleshooting.md) - Solve common issues and problems
- [Contributing Guide](guides/contributing.md) - How to contribute to Frango

### API Reference

- [Middleware API](api-reference/middleware.md) - Complete reference for the middleware API

### Examples

- [TODO Application](examples/todo-app.md) - A complete TODO list application example

## Core Concepts

Frango builds on several core concepts:

### 1. Middleware Architecture

Frango functions as middleware in your Go HTTP server, allowing you to execute PHP scripts in response to HTTP requests. This enables you to:

- Use PHP for templating and view logic
- Handle complex form processing using PHP's capabilities
- Leverage existing PHP libraries and code in Go applications

### 2. Virtual File System (VFS)

Frango implements a virtual file system that:

- Abstracts physical file operations
- Provides runtime file creation and manipulation
- Supports embedding PHP files into your Go binary
- Enables branching into isolated environments for different requests

### 3. Request/Response Handling

Frango seamlessly translates between Go's HTTP handling and PHP's expected environment:

- Populates PHP superglobals (`$_GET`, `$_POST`, `$_SERVER`, etc.)
- Captures PHP output and headers
- Properly handles PHP session management

### 4. Data Sharing

Frango provides simple mechanisms to pass data between Go and PHP:

- Pass data from Go to PHP using the `Render` method
- Access form data, cookies, and headers in both environments

## Getting Help

If you need help with Frango, you can:

- Check the [Troubleshooting Guide](guides/troubleshooting.md)
- Open an issue on [GitHub](https://github.com/davidroman0O/go-php)
- Contribute to the documentation by submitting pull requests

## License

Frango is open source software licensed under the [MIT License](https://github.com/davidroman0O/go-php/blob/main/LICENSE). 