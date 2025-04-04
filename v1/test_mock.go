//go:build nowatcher
// +build nowatcher

package frango

import (
	"fmt"
)

// This file is included when building with the nowatcher tag
// It helps with test setup for environments where FrankenPHP might be slow

func init() {
	fmt.Println("Running tests with nowatcher tag - using real FrankenPHP execution")

	// Configure any special test-time settings here if needed
	// For example, reducing timeouts or enabling special test modes
}
