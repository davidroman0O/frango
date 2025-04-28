package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/davidroman0O/frango"
)

//go:embed index.php
var indexPHP embed.FS

func main() {
	php, err := frango.New(
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create frango instance: %v", err)
	}
	defer php.Shutdown()

	tempIndexPath, err := php.AddEmbeddedLibrary(indexPHP, "index.php", "/index.php")

	mux := http.NewServeMux()

	mux.Handle("/", php.For(tempIndexPath))

	http.ListenAndServe(":8080", mux)
}
