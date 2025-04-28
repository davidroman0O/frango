


```go

// Core middleware with VFS
php := frango.New(
    frango.WithSourceDir("./php"), // we now have a root VFS with all php files at `/` in the virtual filesystem
)

// Add embedded view file to the middleware, as super global
php.AddEmbeddedFile(embedViewFileFS, "views/special-render.php", "/special-render.php")

// VFS operations
vfs := php.NewVFS() // we inherit the root VFS from the middleware, therefore we have files at `/` in the virtual filesystem
vfs.AddSourceDirectory("./templates", "/views") // add a new directory to the virtual filesystem at `/views`
vfs.AddEmbeddedDirectory(embedFS, "assets", "/assets") // add an embedded directory to the virtual filesystem at `/assets`
//vfs.AddEmbeddedDirectory(embedFS, "assets", "/assets", [ "assets/css/style.css", "assets/js/script.js" ]) // specify files to embed
vfs.AddEmbeddedFile(embedFS, "assets/css/style.css", "/assets/css/style.css") // add an embedded file to the virtual filesystem at `/assets/css/style.css`
vfs.CreateVirtualFile("/config.php", []byte("<?php return ['debug' => true];")) // create a virtual file at `/config.php`

vfs2 := vfs.Branch() // "copy" structure to another vfs
vfs2.AddSourceDirectory("./templates", "/views")
vfs2.AddEmbeddedDirectory(embedFS, "assets", "/assets")

// vfs1 and vfs2 are independent, but vfs2 share some files from vfs1
// both vfs1 and vfs2 will be watched for changes from their source files, managed by `frango`

mux := http.NewServeMux()

vfs1 := php.NewVFS(php.EmptyFS())

vfs1.AddSourceDirectory("./www", "/")

routes, err := php.NewRouter(vfs1)
if err != nil {
    log.Fatal(err)
}

for _, route := range routes {
    mux.Handle(route.Path, route.Handler)
}

if _, err := os.Stat(staticDir); os.IsNotExist(err) {
    os.MkdirAll(staticDir, 0755)
    createSampleStaticFiles(staticDir)
} else {
    log.Println("Static directory already exists.")
}
fileServer := http.FileServer(http.Dir(staticDir))
// Register static handler specifically for GET requests
mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

mux.Handle("/special-render", php.Render(vfs, "special-render.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Special Render",
        "message": "This is a special render",
    }
}))

mux.Handle("/api/v1/destruct-button", php.Render(vfs2, "views/destruct-button.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
    return map[string]interface{}{
        "title": "Destruct Button",
        "message": "This is a destruct button",
    }
}))

mux.Handle("/api/", php.For(vfs2, "view/api-list.php"))

mux.Handle("/status", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})


// Or serve directly
http.ListenAndServe(":8080", mux)
   
```

