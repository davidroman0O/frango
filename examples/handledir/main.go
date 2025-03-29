package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	frango "github.com/davidroman0O/frango"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Find the web directory using the library's built-in function
	webDir, err := frango.ResolveDirectory("web")
	if err != nil {
		log.Fatalf("Error finding web directory: %v", err)
	}
	log.Printf("Using web directory: %s", webDir)

	// Create PHP middleware with functional options
	php, err := frango.New(
		frango.WithSourceDir(webDir),
		frango.WithDevelopmentMode(!*prodMode),
	)
	if err != nil {
		log.Fatalf("Error creating PHP middleware: %v", err)
	}
	defer php.Shutdown()

	// Register all PHP files in the "pages" directory under the "/pages" URL prefix
	if err := php.HandleDir("/pages", "pages"); err != nil {
		log.Fatalf("Error registering pages directory: %v", err)
	}

	// Register all PHP files in the "api" directory under the "/api" URL prefix
	if err := php.HandleDir("/api", "api"); err != nil {
		log.Fatalf("Error registering API directory: %v", err)
	}

	// You can also specify absolute paths
	// If you have another PHP directory outside the webDir:
	// if err := php.HandleDir("/other", "/path/to/other/php/files"); err != nil {
	//     log.Fatalf("Error registering other directory: %v", err)
	// }

	// Add a custom handler for the root
	php.HandlePHP("/", "index.php")

	// Note: In this example, we're using the PHP middleware directly
	// as the main handler because we don't have any Go-specific routes.
	// If you wanted to add Go handlers, you would:
	//
	// mux := http.NewServeMux()
	// mux.HandleFunc("/some/go/route", yourGoHandler)
	// handler := php.Wrap(mux)
	// http.ListenAndServe(":8082", handler)

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		php.Shutdown()
		os.Exit(0)
	}()

	// Start standard Go HTTP server with PHP middleware as the handler
	log.Printf("HandleDir Example running on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := http.ListenAndServe(":"+*port, php); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
