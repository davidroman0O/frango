package main

import (
	"log"

	"net/http"

	"github.com/davidroman0O/frango"
)

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
	php.AddSourceFile("./post.php", "post.php")

	mux.Handle("/", php.For("index.php"))
	mux.Handle("POST /post", php.For("post.php"))

	http.ListenAndServe(":8080", mux)
}
