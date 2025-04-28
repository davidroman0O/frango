package main

import (
	"embed"
	"log"

	"net/http"

	"github.com/davidroman0O/frango"
)

//go:embed index.php
var indexPHP embed.FS

//go:embed side.php
var sidePHP embed.FS

func main() {
	php, err := frango.New(
		frango.WithDevelopmentMode(true),
		frango.WithDevServer(frango.DevServerConfig{
			Port: 8081,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}

	mux := http.NewServeMux()

	php.AddSourceFile("./index.php", "index.php")
	php.AddSourceFile("./side.php", "side.php")

	// php.AddEmbeddedFile(indexPHP, "index.php", "index.php")
	// php.AddEmbeddedFile(sidePHP, "side.php", "side.php")

	mux.Handle("/", php.For("index.php"))

	http.ListenAndServe(":8080", mux)
}
