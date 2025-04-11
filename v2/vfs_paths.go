package frango

// TrackLogicalPath tracks the logical path corresponding to a physical path.
// This is used by FrankenPHP to report correct file paths for magic constants like __FILE__.
func (v *VFS) TrackLogicalPath(physicalPath, logicalPath string) {
	v.cacheMutex.Lock()
	defer v.cacheMutex.Unlock()

	v.logicalPaths[physicalPath] = logicalPath
}

// GetLogicalPath returns the logical path for a given physical path.
// If no logical path is found, it returns the physical path.
// This method checks the parent VFS recursively if needed.
func (v *VFS) GetLogicalPath(physicalPath string) string {
	v.cacheMutex.RLock()
	defer v.cacheMutex.RUnlock()

	if logicalPath, ok := v.logicalPaths[physicalPath]; ok {
		return logicalPath
	}

	// Check parent VFS if available
	if v.parent != nil {
		// Release our lock before calling parent to avoid deadlocks
		v.cacheMutex.RUnlock()
		logicalPath := v.parent.GetLogicalPath(physicalPath)
		v.cacheMutex.RLock()
		return logicalPath
	}

	// Default to physical path if not found
	return physicalPath
}

// GetLogicalPathFromVirtual returns the logical path for a given virtual path.
// It first resolves the virtual path to its physical path, then looks up the logical path.
func (v *VFS) GetLogicalPathFromVirtual(virtualPath string) (string, error) {
	// First resolve the virtual path to its physical path
	physicalPath, err := v.ResolvePath(virtualPath)
	if err != nil {
		return "", err
	}

	// Then get the logical path for this physical path
	return v.GetLogicalPath(physicalPath), nil
}
