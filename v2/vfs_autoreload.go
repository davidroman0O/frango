package frango

import (
	"bytes"
	"path/filepath"
	"strings"
)

// Default JavaScript code for auto-reload functionality using Server-Sent Events (SSE)
const defaultSSEReloadScript = `
<script>
(function() {
    // Create EventSource for SSE connection
    const eventSource = new EventSource('/_frango_reload_events');
    
    // Listen for file change events
    eventSource.addEventListener('fileChange', (event) => {
        const data = JSON.parse(event.data);
        console.log('File changed:', data.path);
        
        // Reload the page when any PHP file changes
        if (data.type === 'modified' && data.path.endsWith('.php')) {
            console.log('Reloading page due to PHP file change');
            window.location.reload();
        }
    });
    
    // Handle connection errors
    eventSource.onerror = (error) => {
        console.error('EventSource error:', error);
        eventSource.close();
        
        // Try to reconnect after a short delay
        setTimeout(() => {
            window.location.reload();
        }, 2000);
    };
})();
</script>
`

// Default JavaScript code for auto-reload functionality using WebSocket
const defaultWebSocketReloadScript = `
<script>
(function() {
    // Function to connect to WebSocket server
    function connect() {
        const ws = new WebSocket('ws://' + window.location.host + '/_frango_reload_ws');
        
        // Handle connection open
        ws.onopen = () => {
            console.log('Connected to reload websocket');
        };
        
        // Handle messages from server
        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            console.log('File changed:', data.path);
            
            // Reload the page when any PHP file changes
            if (data.type === 'modified' && data.path.endsWith('.php')) {
                console.log('Reloading page due to PHP file change');
                window.location.reload();
            }
        };
        
        // Handle connection close and try to reconnect
        ws.onclose = () => {
            console.log('Reload websocket closed, reconnecting...');
            setTimeout(connect, 2000);
        };
        
        // Handle connection errors
        ws.onerror = (error) => {
            console.error('Websocket error:', error);
            ws.close();
        };
    }
    
    // Start the connection
    connect();
})();
</script>
`

// Default JavaScript code for auto-reload functionality using polling
const defaultPollingReloadScript = `
<script>
(function() {
    // Track the last update timestamp
    let lastUpdate = Date.now();
    
    // Function to check for updates
    function checkForUpdates() {
        fetch('/_frango_reload_poll?last=' + lastUpdate)
            .then(response => response.json())
            .then(data => {
                if (data.hasChanges) {
                    console.log('Files changed since last check');
                    window.location.reload();
                } else {
                    lastUpdate = data.timestamp;
                    setTimeout(checkForUpdates, 1000);
                }
            })
            .catch(error => {
                console.error('Error checking for updates:', error);
                setTimeout(checkForUpdates, 2000);
            });
    }
    
    // Start polling for changes
    checkForUpdates();
})();
</script>
`

// EnableAutoReload sets up auto-reload functionality for the VFS
func (v *VFS) EnableAutoReload(config VFSConfig) {
	if !config.EnableAutoReload || !v.developMode {
		v.logger.Printf("Auto-reload not enabled (EnableAutoReload: %v, DevelopMode: %v)",
			config.EnableAutoReload, v.developMode)
		return
	}

	// Register file change handler
	v.AddChangeHandler(func(event FileChangeEvent) {
		v.logger.Printf("Auto-reload detected file change: %s (%s)", event.VirtualPath, event.ChangeType)
		// The middleware/handlers will use the registered handlers to send notifications to clients
	})

	v.logger.Printf("Auto-reload enabled with trigger mechanism: %s",
		getReloadTriggerType(config.AutoReloadTrigger))
}

// getReloadTriggerType returns the reload trigger type with a default if empty
func getReloadTriggerType(triggerType string) string {
	if triggerType == "" {
		return "sse" // Default to SSE
	}
	return strings.ToLower(triggerType)
}

// GetAutoReloadScript returns the appropriate auto-reload script based on configuration
func (v *VFS) GetAutoReloadScript(config VFSConfig) string {
	// Check if auto-reload is enabled and we're in development mode
	if !config.EnableAutoReload || !config.DevelopMode {
		return "" // No script if auto-reload is not enabled or not in development mode
	}

	// If custom script is provided, use that
	if config.AutoReloadScript != "" {
		return config.AutoReloadScript
	}

	// Otherwise use default script based on trigger type
	switch getReloadTriggerType(config.AutoReloadTrigger) {
	case "websocket", "ws":
		return defaultWebSocketReloadScript
	case "polling", "poll":
		return defaultPollingReloadScript
	default: // "sse" or any unknown value
		return defaultSSEReloadScript
	}
}

// InjectAutoReloadScript injects the auto-reload script into HTML content if appropriate
func (v *VFS) InjectAutoReloadScript(content []byte, virtualPath string, config VFSConfig) []byte {
	if !config.EnableAutoReload || !config.DevelopMode {
		return content // No injection if auto-reload is not enabled or not in development mode
	}

	// Only inject into HTML or PHP files
	ext := strings.ToLower(filepath.Ext(virtualPath))
	if ext != ".php" && ext != ".html" && ext != ".htm" {
		return content
	}

	// Check if content looks like HTML (contains <html> tag)
	if !bytes.Contains(bytes.ToLower(content), []byte("<html")) {
		return content
	}

	// Get the auto-reload script
	script := v.GetAutoReloadScript(config)
	if script == "" {
		return content
	}

	// Inject before closing </body> tag if it exists
	if idx := bytes.LastIndex(bytes.ToLower(content), []byte("</body>")); idx != -1 {
		return bytes.Join([][]byte{
			content[:idx],
			[]byte(script),
			content[idx:],
		}, nil)
	}

	// Otherwise, append to the end
	return append(content, []byte(script)...)
}
