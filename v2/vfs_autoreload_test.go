package frango

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"
)

func TestAutoReloadScriptGeneration(t *testing.T) {
	// Create a temporary directory for the VFS
	tempDir, err := os.MkdirTemp("", "vfs-autoreload-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger
	logger := log.New(os.Stderr, "VFS-TEST: ", log.LstdFlags)

	// Create VFS with auto-reload enabled
	config := VFSConfig{
		TempDir:           tempDir,
		Logger:            logger,
		DevelopMode:       true,
		EnableAutoReload:  true,
		AutoReloadTrigger: "sse", // Default
	}

	vfs, err := NewVFSWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Test SSE script generation
	sseScript := vfs.GetAutoReloadScript(config)
	if !bytes.Contains([]byte(sseScript), []byte("EventSource")) {
		t.Errorf("SSE script should contain EventSource, but got: %s", sseScript)
	}

	// Test WebSocket script generation
	wsConfig := config
	wsConfig.AutoReloadTrigger = "websocket"
	wsScript := vfs.GetAutoReloadScript(wsConfig)
	if !bytes.Contains([]byte(wsScript), []byte("WebSocket")) {
		t.Errorf("WebSocket script should contain WebSocket, but got: %s", wsScript)
	}

	// Test polling script generation
	pollConfig := config
	pollConfig.AutoReloadTrigger = "polling"
	pollScript := vfs.GetAutoReloadScript(pollConfig)
	if !bytes.Contains([]byte(pollScript), []byte("polling")) || !bytes.Contains([]byte(pollScript), []byte("fetch")) {
		t.Errorf("Polling script should contain fetch, but got: %s", pollScript)
	}

	// Test custom script
	customConfig := config
	customConfig.AutoReloadScript = "<script>console.log('Custom reloader')</script>"
	customScript := vfs.GetAutoReloadScript(customConfig)
	if customScript != customConfig.AutoReloadScript {
		t.Errorf("Custom script not returned correctly, got: %s", customScript)
	}

	// Test script disabled in production mode
	prodConfig := config
	prodConfig.DevelopMode = false
	prodScript := vfs.GetAutoReloadScript(prodConfig)
	if prodScript != "" {
		t.Errorf("Script should be empty in production mode, got: %s", prodScript)
	}
}

func TestAutoReloadEventNotification(t *testing.T) {
	// Create a temporary directory for the VFS
	tempDir, err := os.MkdirTemp("", "vfs-autoreload-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger
	logger := log.New(os.Stderr, "VFS-TEST: ", log.LstdFlags)

	// Create VFS with auto-reload enabled
	config := VFSConfig{
		TempDir:          tempDir,
		Logger:           logger,
		DevelopMode:      true,
		EnableAutoReload: true,
	}

	vfs, err := NewVFSWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	// Test direct event notification (without relying on file changes)
	eventReceived := make(chan FileChangeEvent, 1)

	// Add event handler
	vfs.AddChangeHandler(func(event FileChangeEvent) {
		eventReceived <- event
	})

	// Directly trigger an event notification
	go func() {
		// Short delay to ensure handler is registered
		time.Sleep(100 * time.Millisecond)

		// Directly call notification method
		vfs.notifyFileChanged("/test.php", "/path/to/test.php", "modified")
	}()

	// Wait for the event with timeout
	select {
	case event := <-eventReceived:
		// Verify the event
		if event.VirtualPath != "/test.php" {
			t.Errorf("Expected virtual path '/test.php' but got '%s'", event.VirtualPath)
		}
		if event.ChangeType != "modified" {
			t.Errorf("Expected change type 'modified' but got '%s'", event.ChangeType)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for file change event")
	}

	// Test script injection
	htmlContent := []byte("<html><body><h1>Test</h1></body></html>")
	injected := vfs.InjectAutoReloadScript(htmlContent, "/test.html", config)

	if !bytes.Contains(injected, []byte("<script>")) {
		t.Error("Auto-reload script was not injected into HTML")
	}

	// Test that script is not injected for non-HTML files
	jsContent := []byte("console.log('test');")
	injectedJS := vfs.InjectAutoReloadScript(jsContent, "/test.js", config)

	if !bytes.Equal(injectedJS, jsContent) {
		t.Error("Auto-reload script should not be injected into JS files")
	}
}
