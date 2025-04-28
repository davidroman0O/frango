


```go

php, err := frango.New(
    frango.WithSourceDir("./php"),
	frango.WithDevelopmentMode(!*prodMode),
)

// Create a container filesystem for a dashboard feature
dashboard := php.NewFS()

// Add source directory (live files) - basically load everything under into the root of the container
dashboard.AddSourceDirectory("./php/dashboard/*", "/libs")

// Add embedded templates into the container at /dashboard.php
dashboard.AddEmbeddedFiles(embedFS, "templates/dashboard.php", "/dashboard.php")

// Use the container
mux.Handle("/dashboard", dashboard.Render("dashboard.php", func(w http.ResponseWriter, r *http.Request) map[string]interface{}) {
    return map[string]interface{}{
        "title": "Dashboard",
        "message": "Welcome to the dashboard",
    }
})

// Use the global php instance to handle the stats.php file
mux.Handle("/dashboard/stats", php.For("stats.php"))

```

