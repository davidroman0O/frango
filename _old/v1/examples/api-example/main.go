package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/davidroman0O/frango/v1"
)

// Product represents a product in our API
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

// ProductList is a simple in-memory storage for products
var ProductList = []Product{
	{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "High-performance laptop",
		CreatedAt:   time.Now().Format(time.RFC3339),
	},
	{
		ID:          2,
		Name:        "Smartphone",
		Price:       599.99,
		Description: "Latest smartphone model",
		CreatedAt:   time.Now().Format(time.RFC3339),
	},
}

func main() {
	// Create a new frango middleware instance
	php, err := frango.New(
		frango.WithSourceDir("./php"),
		frango.WithDevelopmentMode(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create a standard HTTP server
	mux := http.NewServeMux()

	// Serve index page
	mux.Handle("/", php.For("index.php"))

	// PHP API endpoints
	mux.Handle("GET /api/php/products", php.For("api/products.php"))
	mux.Handle("GET /api/php/products/{id}", php.For("api/product-{id}.php"))

	// Go API endpoints
	mux.HandleFunc("GET /api/go/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Products retrieved successfully",
			"data": map[string]interface{}{
				"products": ProductList,
				"count":    len(ProductList),
			},
		})
	})

	mux.HandleFunc("GET /api/go/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from URL pattern
		idStr := r.PathValue("id")

		// convert idStr to int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid product ID",
			})
			return
		}

		// Find product
		var product *Product
		for i, p := range ProductList {
			if id == p.ID {
				product = &ProductList[i]
				break
			}
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		if product != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Product retrieved successfully",
				"data":    product,
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Product not found",
			})
		}
	})

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
