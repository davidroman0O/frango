package vfs

import (
	"net/http"
)

// Middleware provides a way to execute PHP scripts using the FrankenPHP runtime
type Middleware struct {
	vfs       *VFS             // Virtual filesystem for PHP scripts
	config    MiddlewareConfig // Configuration for the middleware
	mux       *http.ServeMux   // HTTP router for special paths
	reloadHub *ReloadEventHub  // Event hub for auto-reload functionality
}

// MiddlewareConfig holds configuration options for the middleware
type MiddlewareConfig struct {
	// Auto-reload options
	EnableAutoReload  bool   // Whether to enable auto-reload functionality
	AutoReloadScript  string // Custom JavaScript to inject for auto-reload
	AutoReloadTrigger string // Custom reload trigger mechanism
}

// NewMiddleware creates a new PHP middleware with the given VFS
func NewMiddleware(vfs *VFS) *Middleware {
	return NewMiddlewareWithConfig(vfs, MiddlewareConfig{})
}

// NewMiddlewareWithConfig creates a new PHP middleware with custom configuration
func NewMiddlewareWithConfig(vfs *VFS, config MiddlewareConfig) *Middleware {
	m := &Middleware{
		vfs:    vfs,
		config: config,
		mux:    http.NewServeMux(),
	}

	// Register routes for auto-reload if needed
	if vfs.developMode && config.EnableAutoReload {
		m.RegisterReloadRoutes()
	}

	return m
}

// ServeHTTP implements the http.Handler interface
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if this is a special path for auto-reload
	if m.vfs.developMode && m.config.EnableAutoReload {
		if r.URL.Path == "/_frango_reload_events" || r.URL.Path == "/_frango_reload_poll" {
			m.mux.ServeHTTP(w, r)
			return
		}
	}

	// Regular PHP script handling would go here
	// This is a simplified version that just returns a "Not Implemented" response
	http.Error(w, "PHP execution not implemented in this example", http.StatusNotImplemented)
}
