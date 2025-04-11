package vfs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ReloadEventHub manages connected clients for auto-reload functionality
type ReloadEventHub struct {
	// For SSE clients
	sseClients      map[chan string]bool
	sseClientsMutex sync.RWMutex

	// For WebSocket clients (in a real implementation, this would use actual websocket connections)
	wsClients      map[string]chan []byte
	wsClientsMutex sync.RWMutex

	// For polling clients
	lastChangeTime int64
	changesMutex   sync.RWMutex
}

// NewReloadEventHub creates a new event hub for auto-reload functionality
func NewReloadEventHub() *ReloadEventHub {
	return &ReloadEventHub{
		sseClients:     make(map[chan string]bool),
		wsClients:      make(map[string]chan []byte),
		lastChangeTime: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// RegisterSSEClient registers a new SSE client
func (h *ReloadEventHub) RegisterSSEClient(client chan string) {
	h.sseClientsMutex.Lock()
	defer h.sseClientsMutex.Unlock()
	h.sseClients[client] = true
}

// UnregisterSSEClient removes an SSE client
func (h *ReloadEventHub) UnregisterSSEClient(client chan string) {
	h.sseClientsMutex.Lock()
	defer h.sseClientsMutex.Unlock()
	delete(h.sseClients, client)
	close(client)
}

// BroadcastFileChange sends a file change event to all connected clients
func (h *ReloadEventHub) BroadcastFileChange(event FileChangeEvent) {
	// Create event data as JSON
	eventData := map[string]interface{}{
		"path": event.VirtualPath,
		"type": event.ChangeType,
		"time": event.EventTime.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return // Silently fail, this is just for development convenience
	}

	// Update last change time for polling clients
	h.changesMutex.Lock()
	h.lastChangeTime = time.Now().UnixNano() / int64(time.Millisecond)
	h.changesMutex.Unlock()

	// Format SSE event message
	sseMessage := fmt.Sprintf("event: fileChange\ndata: %s\n\n", jsonData)

	// Send to SSE clients
	h.sseClientsMutex.RLock()
	for client := range h.sseClients {
		// Non-blocking send to avoid getting stuck on slow clients
		select {
		case client <- sseMessage:
			// Successfully sent
		default:
			// Client is too slow, will get it on the next event
		}
	}
	h.sseClientsMutex.RUnlock()

	// Send to WebSocket clients
	h.wsClientsMutex.RLock()
	for _, client := range h.wsClients {
		// Non-blocking send
		select {
		case client <- jsonData:
			// Successfully sent
		default:
			// Client is too slow, will get it on the next event
		}
	}
	h.wsClientsMutex.RUnlock()
}

// Middleware functions for different reload mechanisms

// SSEReloadHandler handles Server-Sent Events connections for auto-reload
func (m *Middleware) SSEReloadHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure we have a hub
	if m.reloadHub == nil {
		m.reloadHub = NewReloadEventHub()
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create channel for this client
	clientChan := make(chan string, 10) // Buffer up to 10 messages

	// Register client
	m.reloadHub.RegisterSSEClient(clientChan)
	defer m.reloadHub.UnregisterSSEClient(clientChan)

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"time\": \"%s\"}\n\n",
		time.Now().Format(time.RFC3339))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Keep the connection open
	disconnected := r.Context().Done()
	for {
		select {
		case <-disconnected:
			return
		case msg := <-clientChan:
			fmt.Fprint(w, msg)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// PollingReloadHandler handles polling requests for file changes
func (m *Middleware) PollingReloadHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure we have a hub
	if m.reloadHub == nil {
		m.reloadHub = NewReloadEventHub()
	}

	// Set JSON content type
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get the last time the client checked
	lastParam := r.URL.Query().Get("last")
	var lastClientCheck int64
	if lastParam != "" {
		var err error
		lastClientCheck, err = strconv.ParseInt(lastParam, 10, 64)
		if err != nil {
			lastClientCheck = 0
		}
	}

	// Get the current last change time
	m.reloadHub.changesMutex.RLock()
	lastChangeTime := m.reloadHub.lastChangeTime
	m.reloadHub.changesMutex.RUnlock()

	// Check if there are changes since the client last checked
	hasChanges := lastClientCheck > 0 && lastChangeTime > lastClientCheck

	// Return current status
	response := map[string]interface{}{
		"timestamp":  time.Now().UnixNano() / int64(time.Millisecond),
		"hasChanges": hasChanges,
	}

	json.NewEncoder(w).Encode(response)
}

// FileChangeHandlerMiddleware returns a middleware function that registers the reload hub
// with the VFS file change notification system
func (m *Middleware) FileChangeHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only set this up once
		if m.reloadHub == nil && m.vfs != nil {
			m.reloadHub = NewReloadEventHub()

			// Register handler for file changes
			m.vfs.AddChangeHandler(func(event FileChangeEvent) {
				if m.reloadHub != nil {
					m.reloadHub.BroadcastFileChange(event)
				}
			})
		}

		next.ServeHTTP(w, r)
	})
}

// RegisterReloadRoutes registers the auto-reload routes with the middleware
func (m *Middleware) RegisterReloadRoutes() {
	// Only register if VFS is in development mode
	if m.vfs == nil || !m.vfs.developMode {
		return
	}

	// Ensure the reload hub exists
	if m.reloadHub == nil {
		m.reloadHub = NewReloadEventHub()

		// Register handler for file changes
		m.vfs.AddChangeHandler(func(event FileChangeEvent) {
			if m.reloadHub != nil {
				m.reloadHub.BroadcastFileChange(event)
			}
		})
	}

	// Register the SSE endpoint
	if m.mux != nil {
		m.mux.HandleFunc("/_frango_reload_events", m.SSEReloadHandler)
		m.mux.HandleFunc("/_frango_reload_poll", m.PollingReloadHandler)
	}
}
