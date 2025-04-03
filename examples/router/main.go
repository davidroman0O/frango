package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	frango "github.com/davidroman0O/frango"
)

// Embed the PHP dashboard template
//
//go:embed embedded-php/dashboard.php
var dashboardTemplate embed.FS

// Embed the PHP utility library
//
//go:embed embedded-php/utils.php
var utilsLibrary embed.FS

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Item represents an item in the system
type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// MemoryStore provides thread-safe access to an in-memory data store
type MemoryStore struct {
	mu    sync.RWMutex
	store map[string]interface{}
}

// Message types for flash messaging
const (
	MessageTypeError   = "error"
	MessageTypeSuccess = "success"
	MessageTypeInfo    = "info"
)

// Message represents a flash message to display to the user
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// NewMemoryStore creates a new memory store instance
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store: make(map[string]interface{}),
	}
}

// SetValue sets a value in the memory store
func (ms *MemoryStore) SetValue(key string, value interface{}) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.store[key] = value
}

// GetValue gets a value from the memory store
func (ms *MemoryStore) GetValue(key string) interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.store[key]
}

// HasValue checks if a key exists in the memory store
func (ms *MemoryStore) HasValue(key string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	_, exists := ms.store[key]
	return exists
}

// GetAllValues returns all values in the memory store
func (ms *MemoryStore) GetAllValues() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Create a copy to avoid data races
	result := make(map[string]interface{}, len(ms.store))
	for k, v := range ms.store {
		result[k] = v
	}

	return result
}

// IncrementCounter increments a counter value in the store
func (ms *MemoryStore) IncrementCounter(key string) int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var counter int
	if val, ok := ms.store[key]; ok {
		if c, ok := val.(int); ok {
			counter = c
		}
	}

	counter++
	ms.store[key] = counter
	return counter
}

// AddMessage adds a flash message to be displayed on the next page load
func (ms *MemoryStore) AddMessage(msgType, content string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	messages, _ := ms.store["flash_messages"].([]Message)
	// Create a message with lowercase field names
	message := Message{
		Type:    msgType, // Will be marshaled to "type" in JSON
		Content: content, // Will be marshaled to "content" in JSON
	}
	messages = append(messages, message)
	ms.store["flash_messages"] = messages
}

// GetAndClearMessages returns all messages and clears them from the store
func (ms *MemoryStore) GetAndClearMessages() []Message {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Initialize flash_messages if it doesn't exist or is nil
	if _, exists := ms.store["flash_messages"]; !exists || ms.store["flash_messages"] == nil {
		ms.store["flash_messages"] = []Message{}
	}

	// Get messages
	messages, ok := ms.store["flash_messages"].([]Message)
	if !ok {
		// If type assertion fails, return empty array
		ms.store["flash_messages"] = []Message{}
		return []Message{}
	}

	// Clear messages
	ms.store["flash_messages"] = []Message{}
	return messages
}

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Define web directory
	webDir := "web"

	// Create Frango instance
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating Frango instance: %v", err)
	}
	defer php.Shutdown()

	// Add the embedded PHP utility library
	_, err = php.AddEmbeddedLibrary(utilsLibrary, "embedded-php/utils.php", "/lib/utils.php")
	assertNoError(err, "Add utils.php lib")

	// Create memory store and initialize data
	memStore := NewMemoryStore()
	initializeMemoryStore(memStore)

	// Create the main mux
	mux := http.NewServeMux()

	// --- Register Go API Endpoints ---
	registerUserEndpoints(mux, memStore) // Assume this uses mux.HandleFunc internally
	registerItemEndpoints(mux, memStore) // Assume this uses mux.HandleFunc internally
	mux.HandleFunc("GET /api/memory", func(w http.ResponseWriter, r *http.Request) { /* ... */ })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) { /* ... */ })

	// --- Register PHP Handlers ---
	// Register specific handlers for each page/view
	indexRenderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		// Get flash messages if any and clear them
		messages := memStore.GetAndClearMessages()
		if messages == nil {
			messages = []Message{} // Ensure it's initialized to an empty array
		}

		// Get query parameters for backward compatibility
		if errorMsg := r.URL.Query().Get("error"); errorMsg != "" {
			messages = append(messages, Message{Type: MessageTypeError, Content: errorMsg})
		}
		if successMsg := r.URL.Query().Get("success"); successMsg != "" {
			messages = append(messages, Message{Type: MessageTypeSuccess, Content: successMsg})
		}
		if infoMsg := r.URL.Query().Get("message"); infoMsg != "" {
			messages = append(messages, Message{Type: MessageTypeInfo, Content: infoMsg})
		}

		return map[string]interface{}{
			"flash_messages": messages,
		}
	}
	// Use Render method for index page to pass messages
	mux.Handle("GET /", php.Render("index.php", indexRenderFn))

	// Use parameterized paths for detail/edit views with the new For method
	mux.Handle("GET /users/{id}", php.For("user_detail.php"))
	mux.Handle("GET /items/{id}", php.For("item_detail.php"))
	mux.Handle("GET /users/{id}/edit", php.For("user_edit.php"))
	mux.Handle("POST /users/{id}/edit", php.For("user_edit.php")) // Standard form submission

	// Additional debug wrapper for parameterized routes to ensure pattern is available
	// Wrap the item handler to ensure the pattern is set in context
	itemDetailPattern := "GET /items/{id}"
	mux.Handle("GET /items-debug/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new context with pattern explicitly set
		type patternKey string
		ctx := context.WithValue(r.Context(), patternKey("pattern"), itemDetailPattern)
		// Call the handler with modified request
		php.For("item_detail.php").ServeHTTP(w, r.WithContext(ctx))
	}))

	// Wrap the user detail handler to ensure the pattern is set in context
	userDetailPattern := "GET /users/{id}"
	mux.Handle("GET /users-debug/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new context with pattern explicitly set
		type patternKey string
		ctx := context.WithValue(r.Context(), patternKey("pattern"), userDetailPattern)
		// Call the handler with modified request
		php.For("user_detail.php").ServeHTTP(w, r.WithContext(ctx))
	}))

	// --- Register Embedded Rendered Dashboard ---
	dashboardRenderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		log.Println("Dashboard render function called - generating data")
		pageViews := memStore.GetValue("page_views")
		if pageViews == nil {
			pageViews = 0
		}
		var users []map[string]interface{}
		if usersVal := memStore.GetValue("users"); usersVal != nil {
			if usersSlice, ok := usersVal.([]map[string]interface{}); ok {
				users = usersSlice
			}
		}
		var items []map[string]interface{}
		if itemsVal := memStore.GetValue("items"); itemsVal != nil {
			if itemsSlice, ok := itemsVal.([]map[string]interface{}); ok {
				items = itemsSlice
			}
		}
		totalUsers := len(users)
		activeUsers := int(float64(totalUsers) * 0.7)
		if activeUsers < 1 && totalUsers > 0 {
			activeUsers = 1
		}
		stats := map[string]interface{}{
			"total_users": totalUsers, "active_users": activeUsers, "total_products": len(items),
			"revenue": 12568.99, "conversion_rate": "3.2%",
		}
		return map[string]interface{}{
			"title": "Router Example - Embedded Dashboard",
			"user":  map[string]interface{}{"name": "Admin User", "email": "admin@example.com", "role": "Administrator"},
			"items": items, "stats": stats,
			"debug_info": map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339), "page_views": pageViews, "memory_keys": len(memStore.GetAllValues()),
			},
		}
	}
	// Add the embedded template file first
	tempDashboardPath, err := php.AddEmbeddedLibrary(dashboardTemplate, "embedded-php/dashboard.php", "/dashboard.php")
	assertNoError(err, "Add dashboard.php template")
	// Register the handler using the temp path with the new Render method
	mux.Handle("GET /dashboard", php.Render(tempDashboardPath, dashboardRenderFn))

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Log all requests for debugging
	wrappedMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("REQUEST: %s %s", r.Method, r.URL.Path)
		log.Printf("  Headers: %v", r.Header)
		mux.ServeHTTP(w, r)
	})

	// Start the server with the single combined mux
	log.Printf("Router Example running on port %s", *port)
	log.Printf("Using web directory: %s", php.SourceDir())
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, wrappedMux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Initialize memory store with sample data
func initializeMemoryStore(memStore *MemoryStore) {
	// Set up users
	if !memStore.HasValue("users") {
		users := []map[string]interface{}{
			{
				"id":         1,
				"name":       "John Doe",
				"email":      "john@example.com",
				"role":       "admin",
				"created_at": time.Now().Format(time.RFC3339),
			},
			{
				"id":         2,
				"name":       "Jane Smith",
				"email":      "jane@example.com",
				"role":       "user",
				"created_at": time.Now().Format(time.RFC3339),
			},
			{
				"id":         3,
				"name":       "Bob Johnson",
				"email":      "bob@example.com",
				"role":       "user",
				"created_at": time.Now().Format(time.RFC3339),
			},
		}
		memStore.SetValue("users", users)
	}

	// Set up items
	if !memStore.HasValue("items") {
		items := []map[string]interface{}{
			{
				"id":          1,
				"name":        "Item 1",
				"description": "First item",
				"created_at":  time.Now().Format(time.RFC3339),
			},
			{
				"id":          2,
				"name":        "Item 2",
				"description": "Second item",
				"created_at":  time.Now().Format(time.RFC3339),
			},
		}
		memStore.SetValue("items", items)
	}

	// Initialize page view counter
	memStore.SetValue("page_views", 0)

	// Store server start time
	memStore.SetValue("server_start_time", time.Now().Format(time.RFC3339))
}

// Register all user-related API endpoints
func registerUserEndpoints(mux *http.ServeMux, memStore *MemoryStore) {
	// GET /api/users - List all users
	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		users := memStore.GetValue("users")
		memStore.IncrementCounter("page_views")

		response := map[string]interface{}{
			"users":     users,
			"count":     len(users.([]map[string]interface{})),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	})

	// POST /api/users - Create a new user
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		// Parse JSON
		var input map[string]interface{}
		if err := json.Unmarshal(body, &input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if _, ok := input["name"]; !ok {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		// Get users from store
		usersVal := memStore.GetValue("users")
		users := usersVal.([]map[string]interface{})

		// Find highest ID
		maxID := 0
		for _, user := range users {
			if id, ok := user["id"].(int); ok && id > maxID {
				maxID = id
			}
		}

		// Create new user
		newUser := map[string]interface{}{
			"id":         maxID + 1,
			"name":       input["name"],
			"email":      input["email"].(string),
			"role":       input["role"].(string),
			"created_at": time.Now().Format(time.RFC3339),
		}

		// Add to store
		users = append(users, newUser)
		memStore.SetValue("users", users)

		// Return response
		response := map[string]interface{}{
			"success": true,
			"message": "User created successfully",
			"user":    newUser,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// GET /api/users/{id} - Get a specific user
	mux.HandleFunc("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract user ID from path
		userIdStr := r.PathValue("id")
		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Invalid user ID format",
			})
			return
		}

		// Get users from store
		usersVal := memStore.GetValue("users")
		if usersVal == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Users data not found",
			})
			return
		}

		users := usersVal.([]map[string]interface{})

		// Find the user by ID
		var user map[string]interface{}
		for _, u := range users {
			if uid, ok := u["id"].(int); ok && uid == userId {
				user = u
				break
			}
		}

		if user == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "User not found",
			})
			return
		}

		// Return the user
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user":      user,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// PUT /api/users/{id} - Update a user
	mux.HandleFunc("PUT /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Extract user ID from path
		userIdStr := r.PathValue("id")
		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Invalid user ID format",
			})
			return
		}

		// Read the entire request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Failed to read request body: " + err.Error(),
			})
			return
		}
		// Log the raw request
		log.Printf("Raw PUT request body for user %d: %s", userId, string(bodyBytes))

		// Parse the JSON body
		var requestBody map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Invalid JSON body: " + err.Error(),
			})
			return
		}

		// Get users from store
		usersVal := memStore.GetValue("users")
		if usersVal == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Users data not found",
			})
			return
		}

		users := usersVal.([]map[string]interface{})

		// Find and update the user by ID
		var updatedUser map[string]interface{}
		updated := false

		for i, user := range users {
			var uid int
			// Handle both int and float64 IDs (JSON unmarshalling might produce float64)
			switch v := user["id"].(type) {
			case int:
				uid = v
			case float64:
				uid = int(v)
			default:
				continue // Skip if ID is not a number
			}

			if uid == userId {
				// Update fields from request
				if name, ok := requestBody["name"].(string); ok && name != "" {
					users[i]["name"] = name
				}
				if email, ok := requestBody["email"].(string); ok && email != "" {
					users[i]["email"] = email
				}
				if role, ok := requestBody["role"].(string); ok && role != "" {
					users[i]["role"] = role
				}

				users[i]["updated_at"] = time.Now().Format(time.RFC3339)
				updatedUser = users[i]
				updated = true
				break
			}
		}

		if !updated {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "User not found",
			})
			return
		}

		// Save updated users back to store
		memStore.SetValue("users", users)

		// Add success message to the store
		memStore.AddMessage(MessageTypeSuccess, "User updated successfully")

		// Return the updated user
		response := map[string]interface{}{
			"user":      updatedUser,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// Log the response
		responseBytes, _ := json.Marshal(response)
		log.Printf("Response to PUT for user %d: %s", userId, string(responseBytes))

		// Send the response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// DELETE /api/users/{id} - Delete user
	mux.HandleFunc("DELETE /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from URL
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Get users from store
		usersVal := memStore.GetValue("users")
		users := usersVal.([]map[string]interface{})

		// Find user by ID
		var foundUser map[string]interface{}
		var userIndex int
		for i, user := range users {
			if userID, ok := user["id"].(int); ok && userID == id {
				foundUser = user
				userIndex = i
				break
			}
		}

		if foundUser == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Remove user from slice
		users = append(users[:userIndex], users[userIndex+1:]...)
		memStore.SetValue("users", users)

		// Return success response
		response := map[string]interface{}{
			"success":      true,
			"message":      "User deleted successfully",
			"deleted_user": foundUser,
		}

		json.NewEncoder(w).Encode(response)
	})

	// Add a handler for showing a message without redirecting to a specific path
	mux.HandleFunc("GET /message", func(w http.ResponseWriter, r *http.Request) {
		msgType := r.URL.Query().Get("type")
		content := r.URL.Query().Get("content")

		if msgType != "" && content != "" {
			memStore.AddMessage(msgType, content)
		}

		// Redirect to home page
		http.Redirect(w, r, "/", http.StatusFound)
	})
}

// Register all item-related API endpoints
func registerItemEndpoints(mux *http.ServeMux, memStore *MemoryStore) {
	// GET /api/items - List all items
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		items := memStore.GetValue("items")
		memStore.IncrementCounter("page_views")

		response := map[string]interface{}{
			"items":     items,
			"count":     len(items.([]map[string]interface{})),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	})

	// POST /api/items - Create a new item
	mux.HandleFunc("POST /api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		// Parse JSON
		var input map[string]interface{}
		if err := json.Unmarshal(body, &input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if _, ok := input["name"]; !ok {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		// Get items from store
		itemsVal := memStore.GetValue("items")
		items := itemsVal.([]map[string]interface{})

		// Find highest ID
		maxID := 0
		for _, item := range items {
			if id, ok := item["id"].(int); ok && id > maxID {
				maxID = id
			}
		}

		// Create new item
		description := "No description"
		if desc, ok := input["description"].(string); ok {
			description = desc
		}

		newItem := map[string]interface{}{
			"id":          maxID + 1,
			"name":        input["name"].(string),
			"description": description,
			"created_at":  time.Now().Format(time.RFC3339),
		}

		// Add to store
		items = append(items, newItem)
		memStore.SetValue("items", items)

		// Return response
		response := map[string]interface{}{
			"success": true,
			"message": "Item created successfully",
			"item":    newItem,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// GET /api/items/{id} - Get a specific item
	mux.HandleFunc("GET /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract item ID from path
		itemIdStr := r.PathValue("id")
		itemId, err := strconv.Atoi(itemIdStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Invalid item ID format",
			})
			return
		}

		// Get items from store
		itemsVal := memStore.GetValue("items")
		if itemsVal == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Items data not found",
			})
			return
		}

		items := itemsVal.([]map[string]interface{})

		// Find the item by ID
		var item map[string]interface{}
		for _, i := range items {
			if iid, ok := i["id"].(int); ok && iid == itemId {
				item = i
				break
			}
		}

		if item == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Item not found",
			})
			return
		}

		// Return the item
		json.NewEncoder(w).Encode(map[string]interface{}{
			"item":      item,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})
}

// Simple error helper
func assertNoError(err error, context string) {
	if err != nil {
		log.Fatalf("Error during setup (%s): %v", context, err)
	}
}
