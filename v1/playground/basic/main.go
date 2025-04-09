package main

import (
	"log"
	"net/http"

	"github.com/davidroman0O/frango/v1"
)

func main() {
	php, err := frango.New(frango.WithDevelopmentMode(true), frango.WithSourceDir("./php"))
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.Handle("/", php.For("index.php"))
	mux.Handle("GET /nested/{id}", php.For("nested/data.php"))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
