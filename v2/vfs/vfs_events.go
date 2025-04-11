package vfs

import (
	"time"
)

// AddChangeHandler registers a handler function to be called when files change
func (v *VFS) AddChangeHandler(handler FileChangeHandler) {
	v.handlerMutex.Lock()
	defer v.handlerMutex.Unlock()

	v.changeHandlers = append(v.changeHandlers, handler)
	v.logger.Printf("Added file change handler to VFS %s", v.name)
}

// RemoveAllChangeHandlers removes all registered change handlers
func (v *VFS) RemoveAllChangeHandlers() {
	v.handlerMutex.Lock()
	defer v.handlerMutex.Unlock()

	v.changeHandlers = nil
	v.logger.Printf("Removed all file change handlers from VFS %s", v.name)
}

// notifyFileChanged creates a file change event and notifies all registered handlers
func (v *VFS) notifyFileChanged(virtualPath, physicalPath, changeType string) {
	// Create the event
	event := FileChangeEvent{
		VirtualPath:  virtualPath,
		PhysicalPath: physicalPath,
		EventTime:    time.Now(),
		ChangeType:   changeType,
	}

	// Get a copy of the handlers to avoid holding the lock while calling them
	v.handlerMutex.RLock()
	handlers := make([]FileChangeHandler, len(v.changeHandlers))
	copy(handlers, v.changeHandlers)
	v.handlerMutex.RUnlock()

	// Call all handlers
	for _, handler := range handlers {
		handler(event)
	}

	v.logger.Printf("Notified %d handlers of %s for file: %s", len(handlers), changeType, virtualPath)
}
