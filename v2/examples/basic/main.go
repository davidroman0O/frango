package main

import (
	"embed"
	"log"

	"net/http"

	"github.com/davidroman0O/frango/v2"
)

//go:embed index.php
var indexPHP embed.FS

func main() {
	php, err := frango.New(frango.WithDevelopmentMode(true))
	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}

	mux := http.NewServeMux()

	// mux.HandleFunc("/", php

	http.ListenAndServe(":8080", mux)
}
