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
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP server
	mux := http.NewServeMux()

	// Form routes
	mux.Handle("/", php.For("index.php"))
	mux.Handle("/contact", php.For("contact-form.php"))
	mux.Handle("POST /contact/submit", php.For("process-form.php"))
	mux.Handle("/success", php.For("success.php"))

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
