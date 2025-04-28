package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	// Create a new frango middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
		// Note: Session handling is enabled by default in PHP
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP server
	mux := http.NewServeMux()

	// Routes
	mux.Handle("/", php.For("index.php"))
	mux.Handle("/login", php.For("login.php"))
	mux.Handle("POST /process-login", php.For("process-login.php"))
	mux.Handle("/dashboard", php.For("dashboard.php"))
	mux.Handle("/counter", php.For("counter.php"))
	mux.Handle("/logout", php.For("logout.php"))

	// Protected routes (require session authentication)
	mux.Handle("/admin", php.For("admin.php"))
	mux.Handle("/profile", php.For("profile.php"))
	mux.Handle("/settings", php.For("settings.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
