//go:build nowatcher
// +build nowatcher

package frango

// This file is only included in test builds with the nowatcher tag

func init() {
	// Set the flag to use mock handlers instead of the real FrankenPHP
	isMockBuild = true
}
