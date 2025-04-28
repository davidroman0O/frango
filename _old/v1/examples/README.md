# Frango Examples

This directory contains example implementations demonstrating different aspects of the Frango library.

## List of Examples

1. [Basic Usage](basic-usage/): Simple "Hello World" example showing how to serve PHP files from Go
2. [API Example](api-example/): JSON API with Go-to-PHP and PHP-to-Go communication
3. [Form Processing](form-processing/): Basic form submission and handling
4. [Template Data](template-data/): Passing data from Go to PHP templates
5. [Path Parameters](path-parameters/): URL pattern matching with parameters
6. [Embedded PHP Files](embedded-php/): Embedding PHP files in Go binaries
7. [VFS Branching](vfs-branching/): Using VFS branching for request isolation
8. [Error Handling](error-handling/): Catching and handling PHP errors
9. [Header Manipulation](header-manipulation/): Working with HTTP headers
10. [File Generation](file-generation/): Creating PHP files dynamically at runtime
11. [Integration Example](integration-example/): Adding PHP support to existing Go apps
12. [Environment Variables](environment-variables/): Passing environment data between Go and PHP

## Running the Examples

To run any example, navigate to its directory and run:

```bash
go run main.go
```

Then visit http://localhost:8080 in your browser to see the example in action.

## Requirements

- Go 1.22 or later
- PHP 8.0 or later with FrankenPHP support 