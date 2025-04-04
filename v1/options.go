package frango

import "log"

// WithSourceDir sets the source directory for PHP files.
func WithSourceDir(dir string) Option {
	return func(m *Middleware) {
		m.sourceDir = dir
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
