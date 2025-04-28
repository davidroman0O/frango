package main

import (
	"log"
	"net/http"
	"time"

	"github.com/davidroman0O/frango/v1"
)

// User represents a user in our application
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// DashboardData holds the data for the dashboard
type DashboardData struct {
	User              User                     `json:"user"`
	PageTitle         string                   `json:"page_title"`
	RecentActivities  []map[string]interface{} `json:"recent_activities"`
	Stats             map[string]interface{}   `json:"stats"`
	NavigationItems   []map[string]string      `json:"navigation_items"`
	NotificationCount int                      `json:"notification_count"`
}

// generateDashboardData creates sample data for the dashboard
func generateDashboardData() DashboardData {
	// Create a sample user
	user := User{
		ID:        123,
		Name:      "John Doe",
		Email:     "john@example.com",
		Role:      "Administrator",
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
	}

	// Create sample activities
	activities := []map[string]interface{}{
		{
			"id":        1,
			"type":      "login",
			"message":   "User logged in",
			"timestamp": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":        2,
			"type":      "update",
			"message":   "Profile updated",
			"timestamp": time.Now().Add(-24 * time.Hour).Format(time.RFC3339), // 1 day ago
		},
		{
			"id":        3,
			"type":      "api",
			"message":   "API key generated",
			"timestamp": time.Now().Add(-48 * time.Hour).Format(time.RFC3339), // 2 days ago
		},
		{
			"id":        4,
			"type":      "security",
			"message":   "Password changed",
			"timestamp": time.Now().Add(-120 * time.Hour).Format(time.RFC3339), // 5 days ago
		},
	}

	// Create sample statistics
	stats := map[string]interface{}{
		"visits":      1250,
		"page_views":  3840,
		"bounce_rate": 0.35,
		"avg_time":    "2m 15s",
	}

	// Create navigation items
	navItems := []map[string]string{
		{"name": "Dashboard", "url": "/", "icon": "dashboard"},
		{"name": "Profile", "url": "/profile", "icon": "user"},
		{"name": "Settings", "url": "/settings", "icon": "settings"},
		{"name": "Logout", "url": "/logout", "icon": "logout"},
	}

	// Return the dashboard data
	return DashboardData{
		User:              user,
		PageTitle:         "Admin Dashboard",
		RecentActivities:  activities,
		Stats:             stats,
		NavigationItems:   navItems,
		NotificationCount: 5,
	}
}

func main() {
	// Create a new frango middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP server
	mux := http.NewServeMux()

	// Index page (explanation)
	mux.Handle("/", php.For("index.php"))

	// Basic template data example
	mux.Handle("/basic", php.Render("basic.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"title":        "Basic Template Example",
			"message":      "This data was passed from Go to PHP!",
			"current_time": time.Now().Format(time.RFC3339),
			"items": []string{
				"Item 1",
				"Item 2",
				"Item 3",
			},
		}
	}))

	// Complex dashboard example
	mux.Handle("/dashboard", php.Render("dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		// Generate realistic dashboard data
		data := generateDashboardData()

		// Return as a map for template rendering
		return map[string]interface{}{
			"data": data, // Nested under "data" key
		}
	}))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
