package vfs

// This file previously contained logic for enabling auto-reload, selecting,
// and injecting JavaScript into HTML responses directly from the VFS package.

// This functionality has been moved to the main `frango` middleware package
// and the `executor` package for better separation of concerns:
// - `frango.Middleware` now manages the configuration, the ReloadEventHub,
//   and registers VFS change handlers.
// - `executor.Executor` handles the injection of the script into the final
//   HTTP response based on configuration passed from the middleware.

// The VFS package's role is now solely focused on managing the virtual
// filesystem, watching for changes, and dispatching `FileChangeEvent`s via
// the `AddChangeHandler` mechanism when `DevelopMode` is enabled.
