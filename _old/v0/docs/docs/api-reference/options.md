# Configuration Options API Reference

This document provides a comprehensive reference for the configuration options available in Frango. These options control the behavior of the Frango middleware and PHP runtime environment.

## Table of Contents

- [Configuration Options API Reference](#configuration-options-api-reference)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Core Options](#core-options)
    - [WithSourceDir](#withsourcedir)
    - [WithTempDir](#withtempdir)
    - [WithDevelopmentMode](#withdevelopmentmode)
  - [Security Options](#security-options)
    - [WithDirectPHPURLsBlocking](#withdirectphpurlsblocking)
  - [Debugging Options](#debugging-options)
    - [WithLogger](#withlogger)
    - [WithErrorHandler](#witherrorhandler)
    - [WithErrorDisplay](#witherrordisplay)
  - [Proposed Future Options](#proposed-future-options)

## Overview

Frango uses a functional options pattern for configuration. When creating a new Frango instance, you can pass one or more option functions to customize its behavior:

```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDevelopmentMode(true),
    frango.WithLogger(customLogger),
    // ... more options
)
```

Each option function modifies the Frango configuration in a specific way. This pattern allows for:

- Clear, readable configuration
- Optional settings with sensible defaults
- Composable configurations
- Type-safe configuration parameters

## Core Options

These options control the fundamental behavior of Frango.

### WithSourceDir

Sets the base directory for PHP files.

```go
frango.WithSourceDir(dir string) Option
```

**Parameters:**
- `dir`: The path to the directory containing PHP files

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
)
```

**Default:** None (no source directory)

**Notes:**
- All PHP file paths used in Frango will be relative to this directory
- The path can be absolute or relative to the current working directory
- The directory must exist and be readable

### WithTempDir

Sets the temporary directory for PHP files and VFS storage.

```go
frango.WithTempDir(dir string) Option
```

**Parameters:**
- `dir`: The path to the temporary directory

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithTempDir("/tmp/frango"),
)
```

**Default:** System temporary directory

**Notes:**
- Used for storing temporary files and VFS data
- The directory should be writable
- The directory will be created if it doesn't exist
- A subdirectory with a unique identifier will be created inside this directory

### WithDevelopmentMode

Enables development mode, which includes more verbose error reporting and disables caching.

```go
frango.WithDevelopmentMode(enabled bool) Option
```

**Parameters:**
- `enabled`: Whether to enable development mode

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDevelopmentMode(true),
)
```

**Default:** `true`

**Effect when enabled:**
- More detailed error messages
- Real-time file change detection 
- Automatic file reloading when source files change
- Error display is enabled by default

## Security Options

These options control security-related behavior of Frango.

### WithDirectPHPURLsBlocking

Controls whether direct PHP file access in URLs should be blocked.

```go
frango.WithDirectPHPURLsBlocking(block bool) Option
```

**Parameters:**
- `block`: Whether to block direct PHP file access

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithDirectPHPURLsBlocking(true),
)
```

**Default:** `true`

**Notes:**
- When enabled, URL paths ending with `.php` will be blocked unless they were explicitly registered with a handler
- This helps prevent users from directly accessing PHP files that were not intended to be exposed
- This is a security feature to prevent unintended file access

## Debugging Options

These options control debugging-related behavior of Frango.

### WithLogger

Sets a custom logger for Frango operations.

```go
frango.WithLogger(logger *log.Logger) Option
```

**Parameters:**
- `logger`: A standard Go logger

**Example:**
```go
customLogger := log.New(os.Stdout, "[FRANGO] ", log.LstdFlags)
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithLogger(customLogger),
)
```

**Default:** A logger that writes to `os.Stderr` with prefix `[frango] `

**Notes:**
- The logger is used to log Frango operations, errors, and debugging information
- In production, you might want to use a logger that writes to a file or a logging service

### WithErrorHandler

Sets a custom PHP error handler script path.

```go
frango.WithErrorHandler(phpErrorHandlerPath string) Option
```

**Parameters:**
- `phpErrorHandlerPath`: Path to the PHP script that will handle errors

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithErrorHandler("/error_handler.php"),
)
```

**Default:** None (internal error handling)

**Notes:**
- The error handler script will be executed when PHP errors occur
- The script should be a PHP file that handles error information
- Path is relative to the VFS root

### WithErrorDisplay

Controls whether PHP errors are displayed in the output.

```go
frango.WithErrorDisplay(display bool) Option
```

**Parameters:**
- `display`: Whether to display PHP errors in the output

**Example:**
```go
php, err := frango.New(
    frango.WithSourceDir("./php-files"),
    frango.WithErrorDisplay(true),  // Show errors (default in development mode)
    frango.WithErrorDisplay(false), // Hide errors (default in production mode)
)
```

**Default:** `true` in development mode, `false` otherwise

**Notes:**
- When enabled, PHP errors will be included in the output sent to the client
- This is useful for debugging but should be disabled in production
- Even when disabled, errors are still logged

## Proposed Future Options

The following options are proposed for future versions of Frango but are not currently implemented.

> **Note**: These options are not available in the current version of Frango.
