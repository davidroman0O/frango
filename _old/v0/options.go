package frango

import "log"

// WithSourceDir sets the source directory for PHP files.
func WithSourceDir(dir string) Option {
	return func(m *Middleware) {
		m.sourceDir = dir
	}
}

// WithDevelopmentMode enables immediate file change detection and disables caching.
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
func WithDirectPHPURLsBlocking(block bool) Option {
	return func(m *Middleware) {
		m.blockDirectPHPURLs = block
	}
}
