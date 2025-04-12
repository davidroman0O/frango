package main

import (
	"embed"
	"log"

	"net/http"

	"github.com/davidroman0O/frango/v2"
	"github.com/davidroman0O/frango/v2/pkg/vfs"
)

//go:embed index.php
var indexPHP embed.FS

func main() {
	php, err := frango.New(frango.WithDevelopmentMode(true))
	if err != nil {
		log.Fatalf("Failed to create Frango instance: %v", err)
	}

	mux := http.NewServeMux()

	vfs, err := vfs.NewVFS()
	if err != nil {
		log.Fatalf("No VFS %s", err)
	}

	if err := vfs.AddEmbeddedFile(indexPHP, "index.php", "index.php"); err != nil {
		log.Fatalf("Failed to add embedded file: %v", err)
	}

	mux.Handle("/", php.ForVFS(vfs, "index.php"))

	http.ListenAndServe(":8080", mux)
}
