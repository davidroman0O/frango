package frango

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setUpRouterTest creates a test environment with PHP files and returns middleware, router, and temp dir
func setUpRouterTest(t *testing.T) (*Middleware, *ConventionalRouter, string, func()) {
	// Create temp dir for test files
	tempDir, err := os.MkdirTemp("", "frango-router-test-")
	require.NoError(t, err, "Failed to create temp directory")

	// Create test file structure
	webDir := filepath.Join(tempDir, "web")
	require.NoError(t, os.MkdirAll(webDir, 0755), "Failed to create web directory")

	// Create test PHP files with different patterns
	testFiles := map[string]string{
		"web/index.php":                 "<?php echo 'home page'; ?>",
		"web/about.php":                 "<?php echo 'about page'; ?>",
		"web/users/index.php":           "<?php echo 'users index'; ?>",
		"web/users/{id}.php":            "<?php echo 'user by id: ' . $_PATH['id']; ?>",
		"web/api/products.get.php":      "<?php echo 'products GET'; ?>",
		"web/api/products.post.php":     "<?php echo 'products POST'; ?>",
		"web/api/products/{id}.get.php": "<?php echo 'product GET by id: ' . $_PATH['id']; ?>",
		"web/assets/styles.css":         "body { color: blue; }",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err, "Failed to create directory for test file")

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file")
	}

	// Create middleware and router
	m, err := New(
		WithSourceDir(webDir),
		WithDevelopmentMode(true),
	)
	require.NoError(t, err, "Failed to create middleware")

	// Create router with default options
	router := m.NewConventionalRouter(nil)

	// Return cleanup function
	cleanup := func() {
		m.Shutdown()
		os.RemoveAll(tempDir)
	}

	return m, router, tempDir, cleanup
}

func TestConventionalRouter_RegisterSourceDirectory(t *testing.T) {
	_, router, tempDir, cleanup := setUpRouterTest(t)
	defer cleanup()

	// Register the source directory
	webDir := filepath.Join(tempDir, "web")
	err := router.RegisterSourceDirectory(webDir, "/")
	assert.NoError(t, err, "Failed to register source directory")

	// Check that routes were created
	routes := router.ListRoutes()
	assert.GreaterOrEqual(t, len(routes), 7, "Expected at least 7 routes to be created")

	// Check for specific routes
	routePatterns := make([]string, 0, len(routes))
	routeMethods := make(map[string]string, len(routes))
	methodPatterns := make(map[string][]string, len(routes))

	for _, route := range routes {
		routePatterns = append(routePatterns, route.Pattern)

		// Store the method for each route
		key := route.Pattern
		if route.Method != "" {
			key = route.Method + " " + route.Pattern
		}
		routeMethods[key] = route.Method

		// Group patterns by method
		if _, exists := methodPatterns[route.Pattern]; !exists {
			methodPatterns[route.Pattern] = []string{}
		}
		methodPatterns[route.Pattern] = append(methodPatterns[route.Pattern], route.Method)
	}

	// Check for expected routes
	assert.Contains(t, routePatterns, "/", "Root route should be registered")
	assert.Contains(t, routePatterns, "/about", "About route should be registered")
	assert.Contains(t, routePatterns, "/users", "Users route should be registered")
	assert.Contains(t, routePatterns, "/users/{id}", "User ID route should be registered")
	assert.Contains(t, routePatterns, "/api/products", "Products API route should be registered")
	assert.Contains(t, routePatterns, "/api/products/{id}", "Product ID API route should be registered")

	// Check method detection for API routes
	apiProductMethods := methodPatterns["/api/products"]
	assert.Contains(t, apiProductMethods, "GET", "Products route should support GET")
	assert.Contains(t, apiProductMethods, "POST", "Products route should support POST")
}

func TestConventionalRouter_AddGoHandler(t *testing.T) {
	_, router, _, cleanup := setUpRouterTest(t)
	defer cleanup()

	// Add Go handlers
	router.AddGoHandler("/status", "GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))

	router.AddGoHandler("/admin", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Admin area"))
	}))

	// Check that routes were created
	routes := router.ListRoutes()

	// Extract routes for easy checking
	goRoutes := make(map[string]string)
	for _, route := range routes {
		if route.RouteType == "go" {
			goRoutes[route.Pattern] = route.Method
		}
	}

	// Verify Go routes
	assert.Equal(t, 2, len(goRoutes), "Expected 2 Go routes")
	assert.Contains(t, goRoutes, "/status", "Status route should be registered")
	assert.Contains(t, goRoutes, "/admin", "Admin route should be registered")
	assert.Equal(t, "GET", goRoutes["/status"], "Status route should be GET")
	assert.Equal(t, "", goRoutes["/admin"], "Admin route should allow any method")
}

func TestConventionalRouter_RegisterVirtualFSEndpoints(t *testing.T) {
	middleware, router, _, cleanup := setUpRouterTest(t)
	defer cleanup()

	// Create a virtual filesystem
	vfs := middleware.NewFS()

	// Add some virtual files
	err := vfs.CreateVirtualFile("/home.php", []byte("<?php echo 'virtual home'; ?>"))
	assert.NoError(t, err)

	err = vfs.CreateVirtualFile("/contact.php", []byte("<?php echo 'contact page'; ?>"))
	assert.NoError(t, err)

	err = vfs.CreateVirtualFile("/api/data.get.php", []byte("<?php echo 'data GET'; ?>"))
	assert.NoError(t, err)

	// Register VFS endpoints
	err = router.RegisterVirtualFSEndpoints(vfs, "/v1")
	assert.NoError(t, err)

	// Check routes
	routes := router.ListRoutes()

	// Extract route patterns for checking
	routePatterns := make([]string, 0, len(routes))
	for _, route := range routes {
		routePatterns = append(routePatterns, route.Pattern)
	}

	// Check for expected routes with v1 prefix
	assert.Contains(t, routePatterns, "/v1/home", "Virtual home route should be registered")
	assert.Contains(t, routePatterns, "/v1/contact", "Virtual contact route should be registered")
	assert.Contains(t, routePatterns, "/v1/api/data", "Virtual API route should be registered")
}

func TestConventionalRouter_StaticFileServing(t *testing.T) {
	_, router, tempDir, cleanup := setUpRouterTest(t)
	defer cleanup()

	// Register the source directory
	webDir := filepath.Join(tempDir, "web")
	err := router.RegisterSourceDirectory(webDir, "/")
	assert.NoError(t, err, "Failed to register source directory")

	// Check CSS file is registered as static content
	routes := router.ListRoutes()

	// Find the static route
	var staticRoute RouteInfo
	for _, route := range routes {
		if strings.Contains(route.Pattern, "styles.css") {
			staticRoute = route
			break
		}
	}

	// Verify static route was created
	assert.NotEmpty(t, staticRoute, "Static CSS route should be registered")
	assert.Equal(t, "static", staticRoute.RouteType, "Route should be of type 'static'")
}

func TestConventionalRouter_Handler(t *testing.T) {
	_, router, _, cleanup := setUpRouterTest(t)
	defer cleanup()

	// Add Go handlers for testing
	router.AddGoHandler("/test", "GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Test handler"))
	}))

	router.AddGoHandler("/method-test", "POST", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("POST handler"))
	}))

	// Get the handler
	handler := router.Handler()

	// Test GET request to /test
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")
	assert.Equal(t, "Test handler", string(body), "Response body mismatch")

	// Test wrong method to /method-test
	req = httptest.NewRequest("GET", "http://example.com/method-test", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	resp = recorder.Result()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "Status code should be 405")
	assert.Equal(t, "POST, OPTIONS", resp.Header.Get("Allow"), "Allow header should include POST and OPTIONS")
}

func TestConventionalRouter_PatternCalculation(t *testing.T) {
	// Create router for testing pattern calculation
	m, err := New()
	require.NoError(t, err)
	defer m.Shutdown()

	router := m.NewConventionalRouter(nil)

	testCases := []struct {
		virtualPath   string
		urlPrefix     string
		expectPattern string
		expectMethod  string
	}{
		{"/index.php", "", "/", ""},
		{"/about.php", "", "/about", ""},
		{"/users/index.php", "", "/users", ""},
		{"/users/{id}.php", "", "/users/{id}", ""},
		{"/api/products.get.php", "", "/api/products", "GET"},
		{"/api/products.post.php", "", "/api/products", "POST"},
		{"/api/products/{id}.get.php", "", "/api/products/{id}", "GET"},
		{"/admin/index.php", "/v1", "/v1/admin", ""},
		{"/api/v2/data.get.php", "/api", "/api/v2/data", "GET"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case%d_%s", i, tc.virtualPath), func(t *testing.T) {
			pattern, method := router.calculateRoutePattern(tc.virtualPath, tc.urlPrefix)
			assert.Equal(t, tc.expectPattern, pattern, "Pattern mismatch")
			assert.Equal(t, tc.expectMethod, method, "Method mismatch")
		})
	}
}

func TestConventionalRouter_RouterOptions(t *testing.T) {
	m, err := New()
	require.NoError(t, err)
	defer m.Shutdown()

	// Test with custom options
	options := &ConventionalRouterOptions{
		CleanURLs:      false, // Don't remove .php extension
		MethodSuffixes: true,
		IndexFiles:     []string{"home.php", "index.php"},
		StaticExtensions: []string{
			".css", ".js",
		},
	}

	router := m.NewConventionalRouter(options)
	assert.NotNil(t, router)
	assert.Equal(t, false, router.options.CleanURLs, "CleanURLs option should be respected")
	assert.Equal(t, true, router.options.MethodSuffixes, "MethodSuffixes option should be respected")
	assert.Equal(t, []string{".css", ".js"}, router.options.StaticExtensions, "StaticExtensions option should be respected")
	assert.Equal(t, []string{"home.php", "index.php"}, router.options.IndexFiles, "IndexFiles option should be respected")
}

func TestConventionalRouter_DefaultOptions(t *testing.T) {
	// Test default options
	options := DefaultConventionalRouterOptions()
	assert.NotNil(t, options)
	assert.Equal(t, true, options.CleanURLs, "Default CleanURLs should be true")
	assert.Equal(t, false, options.CaseSensitive, "Default CaseSensitive should be false")
	assert.Equal(t, true, options.MethodSuffixes, "Default MethodSuffixes should be true")
	assert.NotEmpty(t, options.StaticExtensions, "Default StaticExtensions should not be empty")
}
