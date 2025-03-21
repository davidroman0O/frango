package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	gophp "github.com/davidroman0O/go-php"
)

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

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Find the web directory
	webDir, err := gophp.ResolveDirectory("examples/router/web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}
	log.Printf("Using web directory: %s", webDir)

	// Create server instance with functional options
	server, err := gophp.NewServer(
		gophp.WithSourceDir(webDir),
		gophp.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer server.Shutdown()

	// Initialize the server to set up FrankenPHP
	if err := server.Initialize(context.Background()); err != nil {
		log.Fatalf("Error initializing server: %v", err)
	}

	// Create our memory store
	memStore := NewMemoryStore()

	// Add sample data to the memory store
	initializeMemoryStore(memStore)

	// Create a router
	mux := server.CreateMethodRouter()

	// Register API endpoints for users
	registerUserEndpoints(mux, memStore)

	// Register API endpoints for items
	registerItemEndpoints(mux, memStore)

	// Register PHP endpoints that will communicate with our API
	server.Handle("/users", "api/users.php")
	server.Handle("/users/{id}", "api/user.php")
	server.Handle("/items", "api/items.php")
	server.Handle("/items/{id}", "api/item.php")

	// Add Go handler for memory store status
	mux.HandleFunc("GET /api/memory", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get data from the memory store
		pageViews := memStore.IncrementCounter("page_views")
		allKeys := memStore.GetAllValues()

		// Build keys list
		keys := make([]string, 0, len(allKeys))
		for k := range allKeys {
			keys = append(keys, k)
		}

		// Return memory store info
		response := map[string]interface{}{
			"status": "ok",
			"memory_store": map[string]interface{}{
				"keys":       keys,
				"page_views": pageViews,
				"uptime":     time.Now().Format(time.RFC3339),
			},
		}

		json.NewEncoder(w).Encode(response)
	})

	// Add native Go handler for status
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().Format(time.RFC3339))
	})

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Shutdown()
		os.Exit(0)
	}()

	// Start the server with our router
	log.Printf("Router Example running on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
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

	// GET /api/users/{id} - Get user by ID
	mux.HandleFunc("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		for _, user := range users {
			if userID, ok := user["id"].(int); ok && userID == id {
				foundUser = user
				break
			}
		}

		if foundUser == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Return user
		response := map[string]interface{}{
			"user":      foundUser,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	})

	// PUT /api/users/{id} - Update user
	mux.HandleFunc("PUT /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from URL
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

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

		// Update user fields
		if name, ok := input["name"].(string); ok {
			foundUser["name"] = name
		}
		if email, ok := input["email"].(string); ok {
			foundUser["email"] = email
		}
		if role, ok := input["role"].(string); ok {
			foundUser["role"] = role
		}
		foundUser["updated_at"] = time.Now().Format(time.RFC3339)

		// Update store
		users[userIndex] = foundUser
		memStore.SetValue("users", users)

		// Return updated user
		response := map[string]interface{}{
			"success": true,
			"message": "User updated successfully",
			"user":    foundUser,
		}

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

	// GET /api/items/{id} - Get item by ID
	mux.HandleFunc("GET /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get item ID from URL
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid item ID", http.StatusBadRequest)
			return
		}

		// Get items from store
		itemsVal := memStore.GetValue("items")
		items := itemsVal.([]map[string]interface{})

		// Find item by ID
		var foundItem map[string]interface{}
		for _, item := range items {
			if itemID, ok := item["id"].(int); ok && itemID == id {
				foundItem = item
				break
			}
		}

		if foundItem == nil {
			http.Error(w, "Item not found", http.StatusNotFound)
			return
		}

		// Return item
		response := map[string]interface{}{
			"item":      foundItem,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	})
}
