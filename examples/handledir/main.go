package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	gophp "github.com/davidroman0O/gophp"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8082", "Port to listen on")
	prodMode := flag.Bool("prod", false, "Enable production mode")
	flag.Parse()

	// Find the web directory using the library's built-in function
	webDir, err := gophp.ResolveDirectory("web")
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

	// Register all PHP files in the "pages" directory under the "/pages" URL prefix
	if err := server.HandleDir("/pages", "pages"); err != nil {
		log.Fatalf("Error registering pages directory: %v", err)
	}

	// Register all PHP files in the "api" directory under the "/api" URL prefix
	if err := server.HandleDir("/api", "api"); err != nil {
		log.Fatalf("Error registering API directory: %v", err)
	}

	// You can also specify absolute paths
	// If you have another PHP directory outside the webDir:
	// if err := server.HandleDir("/other", "/path/to/other/php/files"); err != nil {
	//     log.Fatalf("Error registering other directory: %v", err)
	// }

	// Add a custom handler for the root
	server.HandlePHP("/", "index.php")

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Shutdown()
		os.Exit(0)
	}()

	// Start server
	log.Printf("HandleDir Example running on port %s", *port)
	log.Printf("Open http://localhost:%s/ in your browser", *port)
	if err := server.ListenAndServe(":" + *port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
