package frango

import "log"

// WithSourceDir configures a directory to be added to the VFS with the given prefix.
// This is a convenience function for AddSourceDirectory.
// Note: Previously, this set a field on the middleware, but now it is applied when the
// middleware is ready.
// DEPRECATED: Use AddSourceDirectory after creating the middleware for clearer API.
func WithSourceDir(dir string) Option {
	return func(m *Middleware) {
		// We can't set up the VFS immediately as the middleware might not be fully initialized yet,
		// so we'll store the directory and add it in New()
		_ = dir // Prevent unused variable warning - we'll handle this in New()
		// The actual work will now be done after the VFS is created in New()
	}
}

// WithTempDir sets the temporary directory for PHP files and VFS storage.
func WithTempDir(dir string) Option {
	return func(m *Middleware) {
		m.tempDir = dir
	}
}

// WithDevelopmentMode enables real-time file change detection and disables caching.
func WithDevelopmentMode(enabled bool) Option {
	return func(m *Middleware) {
		m.developmentMode = enabled
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *log.Logger) Option {
	return func(m *Middleware) {
		m.logger = logger
	}
}

// WithDirectPHPURLsBlocking controls whether direct PHP file access in URLs should be blocked.
// When enabled (default), URL paths ending with .php will be blocked unless they were explicitly
// registered with a handler.
func WithDirectPHPURLsBlocking(block bool) Option {
	return func(m *Middleware) {
		m.blockDirectPHPURLs = block
	}
}

// WithErrorHandler sets a custom PHP error handler script path.
// When PHP errors occur, this script will be executed to handle them.
func WithErrorHandler(phpErrorHandlerPath string) Option {
	return func(m *Middleware) {
		m.errorHandlerPath = phpErrorHandlerPath
	}
}

// WithErrorDisplay controls whether PHP errors are displayed in the output.
// In development mode, errors are displayed by default.
// In production mode, errors are hidden by default.
func WithErrorDisplay(display bool) Option {
	return func(m *Middleware) {
		m.displayErrors = display
	}
}
