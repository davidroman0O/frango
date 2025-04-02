package frango

import (
	"embed"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata/embed_script.php
var embedScriptFS embed.FS

//go:embed testdata/embed_lib.php
var embedLibFS embed.FS

//go:embed testdata/render_lib.php
var renderLibFS embed.FS

//go:embed testdata/render_template.php
var renderTemplateFS embed.FS

//go:embed testdata/lib/required_lib.php
var requiredLibFS embed.FS

// Helper function to create a temporary directory with PHP files for testing
func setupTestEnv(t *testing.T, files map[string]string) (string, func()) {
	tempDir, err := os.MkdirTemp("", "frango_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	for name, content := range files {
		filePath := filepath.Join(tempDir, name)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create sub dir %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", name, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// Helper to create a middleware instance for testing
func setupTestMiddleware(t *testing.T, sourceDir string, opts ...Option) (*Middleware, func()) {
	// Suppress logger output during tests unless explicitly overridden
	devNull, _ := os.Open(os.DevNull)
	testLogger := log.New(devNull, "", 0)
	finalOpts := append([]Option{WithLogger(testLogger)}, opts...)

	php, err := New(finalOpts...)
	if err != nil {
		// Cleanup source dir if it was created by setupTestEnv
		if strings.Contains(sourceDir, "frango_test_") {
			os.RemoveAll(sourceDir)
		}
		t.Fatalf("Failed to create Middleware: %v", err)
	}

	cleanup := func() {
		php.Shutdown()
		// Cleanup source dir if it was created by setupTestEnv
		if strings.Contains(sourceDir, "frango_test_") {
			os.RemoveAll(sourceDir)
		}
	}

	return php, cleanup
}

// Integration test for basic PHP file serving via HandlerFor
func TestIntegration_BasicServe(t *testing.T) {
	files := map[string]string{
		"index.php": `<?php echo "Hello from PHP!"; ?>`,
		"about.php": `<?php echo "About Page"; ?>`,
	}
	sourceDir, _ := setupTestEnv(t, files)
	php, mwCleanup := setupTestMiddleware(t, sourceDir, WithSourceDir(sourceDir))
	defer mwCleanup()

	// Register routes using HandlerFor
	mux := http.NewServeMux()
	mux.Handle("/", php.HandlerFor("/", "index.php")) // Note: Pattern passed to HandlerFor for potential future use
	mux.Handle("GET /about", php.HandlerFor("GET /about", "about.php"))

	// Test requests
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"Root index via GET", "GET", "/", http.StatusOK, "Hello from PHP!"},
		{"Root index via POST", "POST", "/", http.StatusOK, "Hello from PHP!"},
		{"About page GET", "GET", "/about", http.StatusOK, "About Page"},
		{"About page POST (Fallback to Root)", "POST", "/about", http.StatusOK, "Hello from PHP!"},
		{"Not found (Fallback to Root)", "GET", "/contact", http.StatusOK, "Hello from PHP!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code, "Status code mismatch")
			if tt.wantStatus == http.StatusOK {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), tt.wantBody, "Body mismatch")
			}
		})
	}
}

// Test parameterized routes using Go 1.22+ stdlib mux features
func TestIntegration_ParameterizedRoutes(t *testing.T) {
	files := map[string]string{
		"user_profile.php": `<?php echo "User: " . ($_SERVER['FRANGO_PARAM_userId'] ?? 'N/A'); ?>`,
		"item_details.php": `<?php echo "Item: " . ($_SERVER['FRANGO_PARAM_itemId'] ?? 'N/A') . " Color: " . ($_SERVER['FRANGO_PARAM_color'] ?? 'N/A'); ?>`,
	}
	sourceDir, _ := setupTestEnv(t, files)
	php, mwCleanup := setupTestMiddleware(t, sourceDir, WithSourceDir(sourceDir))
	defer mwCleanup()

	// Use HandlerFor and Register with Mux
	mux := http.NewServeMux()
	// Pass the pattern used for registration to HandlerFor
	mux.Handle("GET /users/{userId}", php.HandlerFor("GET /users/{userId}", "user_profile.php"))
	mux.Handle("GET /items/{itemId}/color/{color}", php.HandlerFor("GET /items/{itemId}/color/{color}", "item_details.php"))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"User ID 123", "GET", "/users/123", http.StatusOK, "User: 123"},
		{"User ID abc", "GET", "/users/abc", http.StatusOK, "User: abc"},
		{"Item details", "GET", "/items/xyz/color/blue", http.StatusOK, "Item: xyz Color: blue"},
		{"Different Item", "GET", "/items/999/color/red", http.StatusOK, "Item: 999 Color: red"},
		{"Route mismatch", "GET", "/users/abc/extra", http.StatusNotFound, ""},
		{"Wrong method", "POST", "/users/123", http.StatusMethodNotAllowed, ""}, // Test method matching
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req) // Use the mux directly

			assert.Equal(t, tt.wantStatus, rr.Code, "Status code mismatch")
			if tt.wantStatus == http.StatusOK {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), tt.wantBody, "Body mismatch")
			}
		})
	}
}

// Test HandleRender functionality using RenderHandlerFor
func TestIntegration_HandleRender(t *testing.T) {
	files := map[string]string{
		"template.php": `<?php 
			$title = $_SERVER['FRANGO_VAR_title'] ?? 'Default Title'; 
			$message = $_SERVER['FRANGO_VAR_message'] ?? 'Default Message';
			echo "<h1>" . json_decode($title) . "</h1><p>" . json_decode($message) . "</p>"; 
		?>`,
	}
	sourceDir, _ := setupTestEnv(t, files)
	php, mwCleanup := setupTestMiddleware(t, sourceDir, WithSourceDir(sourceDir))
	defer mwCleanup()

	// Define render function
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		page := r.URL.Query().Get("page")
		message := "Default message."
		if page == "about" {
			message = "This is the about page message."
		}
		return map[string]interface{}{
			"title":   "Render Test", // Ensure keys match PHP expectations
			"message": message,
		}
	}

	// Register render route using RenderHandlerFor
	mux := http.NewServeMux()
	pattern := "GET /render"
	mux.Handle(pattern, php.RenderHandlerFor(pattern, "template.php", renderFn))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"Render default", "GET", "/render", http.StatusOK, "<h1>Render Test</h1><p>Default message.</p>"},
		{"Render with query", "GET", "/render?page=about", http.StatusOK, "<h1>Render Test</h1><p>This is the about page message.</p>"},
		{"Render wrong method", "POST", "/render", http.StatusMethodNotAllowed, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code, "Status code mismatch")
			if tt.wantStatus == http.StatusOK {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), tt.wantBody, "Body mismatch")
			}
		})
	}
}

// Test using an embedded script file
func TestIntegration_EmbedScript(t *testing.T) {
	php, mwCleanup := setupTestMiddleware(t, "") // No explicit source dir needed
	defer mwCleanup()

	// Add the embedded script to the temporary embed area
	targetPath := "/embedded/script.php" // Target path within the virtual embed space
	tempScriptPath, err := php.AddEmbeddedLibrary(embedScriptFS, "testdata/embed_script.php", targetPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, tempScriptPath)

	// Register a handler pointing to the *temporary disk path* of the embedded script
	mux := http.NewServeMux()
	pattern := "GET /run_embed"
	mux.Handle(pattern, php.HandlerFor(pattern, tempScriptPath))

	// Test request
	req := httptest.NewRequest("GET", "/run_embed", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Status code mismatch")
	body, _ := io.ReadAll(rr.Body)
	assert.Contains(t, string(body), "Hello from embedded script!", "Body mismatch")
}

// Test using an embedded library file included by another script
func TestIntegration_EmbedLibrary(t *testing.T) {
	// This script will include the library
	files := map[string]string{
		"main.php": `<?php 
			require_once __DIR__ . '/lib/util.php'; 
			echo get_greeting('Tester'); 
		?>`,
	}
	sourceDir, _ := setupTestEnv(t, files)
	php, mwCleanup := setupTestMiddleware(t, sourceDir, WithSourceDir(sourceDir))
	defer mwCleanup()

	// Add the embedded library, specifying its path within the environment
	libTargetPath := "/lib/util.php"
	_, err := php.AddEmbeddedLibrary(embedLibFS, "testdata/embed_lib.php", libTargetPath)
	assert.NoError(t, err)

	// Register a handler for the main script
	mux := http.NewServeMux()
	pattern := "GET /app"
	// Pass the pattern and the path relative to sourceDir
	mux.Handle(pattern, php.HandlerFor(pattern, "main.php"))

	// Test request
	req := httptest.NewRequest("GET", "/app", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Status code mismatch")
	body, _ := io.ReadAll(rr.Body)
	assert.Contains(t, string(body), "Hello, Tester from embedded library!", "Body mismatch")
}

// Test rendering an embedded template that includes an embedded library
func TestIntegration_EmbedRenderWithEmbedLib(t *testing.T) {
	php, mwCleanup := setupTestMiddleware(t, "") // No explicit source dir needed
	defer mwCleanup()

	// Add the embedded library
	libTargetPath := "/lib/render_util.php"
	_, err := php.AddEmbeddedLibrary(renderLibFS, "testdata/render_lib.php", libTargetPath)
	assert.NoError(t, err)

	// Add the embedded template script (which includes the lib)
	// Note: AddEmbeddedLibrary returns the *disk path* in the temp embed area
	templateTargetPath := "/templates/render.php"
	tempTemplatePath, err := php.AddEmbeddedLibrary(renderTemplateFS, "testdata/render_template.php", templateTargetPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, tempTemplatePath)

	// Define render function
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"title": "Embedded Render",
		}
	}

	// Register route using RenderHandlerFor, pointing to the temp path of the template
	mux := http.NewServeMux()
	pattern := "GET /embed_render"
	mux.Handle(pattern, php.RenderHandlerFor(pattern, tempTemplatePath, renderFn))

	// Test request
	req := httptest.NewRequest("GET", "/embed_render", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Status code mismatch")
	body, _ := io.ReadAll(rr.Body)
	// Check for content from both template and included library
	assert.Contains(t, string(body), "<h1>Embedded Render</h1>", "Body mismatch - Title")
	assert.Contains(t, string(body), "<p>Message from included rendered lib!</p>", "Body mismatch - Lib Message")
}

// Test including a file from the source directory
func TestIntegration_RequireFile(t *testing.T) {
	// We don't use setupTestEnv here as we want to use the existing testdata dir
	// testdata/ contains main_with_require.php and lib/required_lib.php
	sourceDir := "testdata"
	cwd, _ := os.Getwd()
	absSourceDir := filepath.Join(cwd, sourceDir)

	php, mwCleanup := setupTestMiddleware(t, absSourceDir, WithSourceDir(absSourceDir))
	defer mwCleanup()

	// This ensures it gets copied into the environment for main_with_require.php
	libTargetPath := "/lib/required_lib.php" // Path INSIDE the PHP environment
	_, err := php.AddEmbeddedLibrary(requiredLibFS, "testdata/lib/required_lib.php", libTargetPath)
	assert.NoError(t, err)

	// Register a handler for the main script
	mux := http.NewServeMux()
	pattern := "GET /main_req"
	// Pass the path relative to the sourceDir
	mux.Handle(pattern, php.HandlerFor(pattern, "main_with_require.php"))

	// Test request
	req := httptest.NewRequest("GET", "/main_req", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Status code mismatch")
	body, _ := io.ReadAll(rr.Body)
	assert.Contains(t, string(body), "Main script says: Output from required lib!", "Body mismatch")
}

// Test Development Mode - hash checking and environment rebuild
func TestIntegration_DevMode(t *testing.T) {
	// Initial script content
	initialContent := `<?php echo "Initial Version"; ?>`
	updatedContent := `<?php echo "UPDATED Version"; ?>`
	scriptName := "dev_mode_script.php"

	files := map[string]string{
		scriptName: initialContent,
	}
	sourceDir, _ := setupTestEnv(t, files)
	// Pass WithDevelopmentMode(true)
	php, mwCleanup := setupTestMiddleware(t, sourceDir, WithSourceDir(sourceDir), WithDevelopmentMode(true))
	defer mwCleanup()

	// Register handler
	mux := http.NewServeMux()
	pattern := "GET /dev"
	scriptPath := scriptName // Path relative to sourceDir
	mux.Handle(pattern, php.HandlerFor(pattern, scriptPath))

	// --- First Request ---
	t.Run("Initial request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dev", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Initial Status code mismatch")
		body, _ := io.ReadAll(rr.Body)
		assert.Contains(t, string(body), "Initial Version", "Initial Body mismatch")
	})

	// --- Modify the file ---
	t.Logf("Modifying script file: %s", filepath.Join(sourceDir, scriptName))
	scriptFullPath := filepath.Join(sourceDir, scriptName)
	err := os.WriteFile(scriptFullPath, []byte(updatedContent), 0644)
	assert.NoError(t, err, "Failed to write updated file content")

	// --- Second Request (with retry) ---
	t.Run("Request after modify", func(t *testing.T) {
		maxRetries := 5
		delay := 50 * time.Millisecond // Increase delay slightly from previous minimal sleep
		var success bool
		var lastBody string
		var lastCode int

		for i := 0; i < maxRetries; i++ {
			req := httptest.NewRequest("GET", "/dev", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			bodyBytes, _ := io.ReadAll(rr.Body)
			body := string(bodyBytes)
			lastBody = body
			lastCode = rr.Code

			if rr.Code == http.StatusOK && strings.Contains(body, "UPDATED Version") {
				success = true
				t.Logf("Updated content found after %d attempt(s)", i+1)
				break // Success!
			}

			t.Logf("Attempt %d failed, retrying after %v...", i+1, delay)
			time.Sleep(delay)
		}

		assert.True(t, success, "Failed to get updated content after retries")
		// Final assertions on the last attempt's results if needed, but success check is primary
		assert.Equal(t, http.StatusOK, lastCode, "Updated Status code mismatch (last attempt)")
		assert.Contains(t, lastBody, "UPDATED Version", "Updated Body mismatch (last attempt)")
	})
}

// Test PHP Parse Error handling (now checking body for error string)
func TestIntegration_ParseError(t *testing.T) {
	// No setupTestEnv needed, using file created by command
	sourceDir := "testdata"
	cwd, _ := os.Getwd()
	absSourceDir := filepath.Join(cwd, sourceDir)

	php, mwCleanup := setupTestMiddleware(t, absSourceDir, WithSourceDir(absSourceDir))
	defer mwCleanup()

	// Register handler for the script with a parse error
	mux := http.NewServeMux()
	pattern := "GET /parse_error"
	mux.Handle(pattern, php.HandlerFor(pattern, "parse_error.php"))

	// Test request
	req := httptest.NewRequest("GET", "/parse_error", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Check for 200 OK (FrankenPHP might send this even with fatal errors)
	// and assert that the body contains the PHP fatal error message.
	assert.Equal(t, http.StatusOK, rr.Code, "Status code might be 200 even on error")
	body, _ := io.ReadAll(rr.Body)
	t.Logf("Body for parse error test: %s", string(body))
	assert.Contains(t, string(body), "Fatal error", "Body should contain PHP Fatal error string")
	// Assert that the successful output is NOT present
	assert.NotContains(t, string(body), "This should not appear", "Body should not contain output after fatal error")
}

// Test MapFileSystemRoutes utility function and route execution
func TestIntegration_MapFileSystemRoutes(t *testing.T) {
	// Create a more complex structure for testing
	files := map[string]string{
		"index.php":          `<?php echo "FS Root Index"; ?>`,
		"hello.php":          `<?php echo "Hello Page"; ?>`,
		"users.get.php":      `<?php echo "Users GET"; ?>`,
		"users.post.php":     `<?php echo "Users POST"; ?>`,
		"admin/index.php":    `<?php echo "Admin Index"; ?>`,
		"admin/settings.php": `<?php echo "Admin Settings"; ?>`,
	}
	sourceDir, _ := setupTestEnv(t, files)
	php, mwCleanup := setupTestMiddleware(t, sourceDir, WithSourceDir(sourceDir))
	defer mwCleanup()

	// --- Generate Routes with different options ---

	// Case 1: Defaults (Clean=true, Index=true, Method=false)
	routes1, err1 := MapFileSystemRoutes(php, os.DirFS(sourceDir), ".", "/", nil)
	assert.NoError(t, err1)
	mux1 := http.NewServeMux()
	for _, route := range routes1 {
		muxPattern := route.Pattern
		if route.Method != "" {
			muxPattern = route.Method + " " + route.Pattern
		}
		mux1.Handle(muxPattern, route.Handler)
	}

	// Case 2: Prefix, No Implicit (Clean=false, Index=false, Method=false)
	opts2 := &FileSystemRouteOptions{
		// Use Enum constants
		GenerateCleanURLs:      OptionDisabled,
		GenerateIndexRoutes:    OptionDisabled,
		DetectMethodByFilename: OptionDisabled, // Explicitly disable (default anyway)
	}
	routes2, err2 := MapFileSystemRoutes(php, os.DirFS(sourceDir), ".", "/app", opts2)
	assert.NoError(t, err2)
	mux2 := http.NewServeMux()
	for _, route := range routes2 {
		muxPattern := route.Pattern
		if route.Method != "" {
			muxPattern = route.Method + " " + route.Pattern
		}
		mux2.Handle(muxPattern, route.Handler)
	}

	// Case 3: Prefix, Method Detection (Clean=true, Index=true BY DEFAULT but disabled for this test case)
	opts3 := &FileSystemRouteOptions{
		// Use Enum constant
		DetectMethodByFilename: OptionEnabled,
		GenerateIndexRoutes:    OptionDisabled, // Keep index disabled for this case
	}
	routes3, err3 := MapFileSystemRoutes(php, os.DirFS(sourceDir), ".", "/api", opts3)
	assert.NoError(t, err3)
	mux3 := http.NewServeMux()
	for _, route := range routes3 {
		muxPattern := route.Pattern
		if route.Method != "" {
			muxPattern = route.Method + " " + route.Pattern
		}
		mux3.Handle(muxPattern, route.Handler)
	}

	// --- Assertions ---
	type routeTest struct {
		name       string
		mux        *http.ServeMux
		method     string
		path       string
		wantStatus int
		wantBody   string
	}

	tests := []routeTest{
		// Case 1 Assertions
		{name: "C1 Root Index", mux: mux1, method: "GET", path: "/", wantStatus: http.StatusOK, wantBody: "FS Root Index"},
		{name: "C1 index.php", mux: mux1, method: "GET", path: "/index.php", wantStatus: http.StatusOK, wantBody: "FS Root Index"},
		{name: "C1 Hello Clean", mux: mux1, method: "GET", path: "/hello", wantStatus: http.StatusOK, wantBody: "Hello Page"},
		{name: "C1 hello.php", mux: mux1, method: "GET", path: "/hello.php", wantStatus: http.StatusOK, wantBody: "Hello Page"},
		{name: "C1 Users GET (No Detect)", mux: mux1, method: "GET", path: "/users.get.php", wantStatus: http.StatusOK, wantBody: "Users GET"},
		{name: "C1 Users POST (No Detect)", mux: mux1, method: "POST", path: "/users.post.php", wantStatus: http.StatusOK, wantBody: "Users POST"},
		{name: "C1 Admin Index Dir", mux: mux1, method: "GET", path: "/admin/", wantStatus: http.StatusOK, wantBody: "Admin Index"},
		{name: "C1 admin/index.php", mux: mux1, method: "GET", path: "/admin/index.php", wantStatus: http.StatusOK, wantBody: "Admin Index"},
		{name: "C1 Admin Settings Clean", mux: mux1, method: "GET", path: "/admin/settings", wantStatus: http.StatusOK, wantBody: "Admin Settings"},
		{name: "C1 admin/settings.php", mux: mux1, method: "GET", path: "/admin/settings.php", wantStatus: http.StatusOK, wantBody: "Admin Settings"},

		// Case 2 Assertions
		{name: "C2 Root Not Found", mux: mux2, method: "GET", path: "/app/", wantStatus: http.StatusNotFound, wantBody: ""}, // Index disabled
		{name: "C2 index.php", mux: mux2, method: "GET", path: "/app/index.php", wantStatus: http.StatusOK, wantBody: "FS Root Index"},
		{name: "C2 Hello Not Found", mux: mux2, method: "GET", path: "/app/hello", wantStatus: http.StatusNotFound, wantBody: ""}, // Clean disabled
		{name: "C2 hello.php", mux: mux2, method: "GET", path: "/app/hello.php", wantStatus: http.StatusOK, wantBody: "Hello Page"},
		{name: "C2 users.get.php", mux: mux2, method: "GET", path: "/app/users.get.php", wantStatus: http.StatusOK, wantBody: "Users GET"},
		{name: "C2 Admin Dir Not Found", mux: mux2, method: "GET", path: "/app/admin/", wantStatus: http.StatusNotFound, wantBody: ""}, // Index disabled
		{name: "C2 admin/index.php", mux: mux2, method: "GET", path: "/app/admin/index.php", wantStatus: http.StatusOK, wantBody: "Admin Index"},
		{name: "C2 Admin Settings Not Found", mux: mux2, method: "GET", path: "/app/admin/settings", wantStatus: http.StatusNotFound, wantBody: ""}, // Clean disabled
		{name: "C2 admin/settings.php", mux: mux2, method: "GET", path: "/app/admin/settings.php", wantStatus: http.StatusOK, wantBody: "Admin Settings"},

		// Case 3 Assertions
		{name: "C3 Users GET", mux: mux3, method: "GET", path: "/api/users", wantStatus: http.StatusOK, wantBody: "Users GET"},
		{name: "C3 Users POST", mux: mux3, method: "POST", path: "/api/users", wantStatus: http.StatusOK, wantBody: "Users POST"},
		{name: "C3 Users PUT (Not Allowed)", mux: mux3, method: "PUT", path: "/api/users", wantStatus: http.StatusMethodNotAllowed, wantBody: ""},
		{name: "C3 Hello Clean (ANY)", mux: mux3, method: "GET", path: "/api/hello", wantStatus: http.StatusOK, wantBody: "Hello Page"},
		{name: "C3 Hello.php (ANY)", mux: mux3, method: "POST", path: "/api/hello.php", wantStatus: http.StatusOK, wantBody: "Hello Page"},
		{name: "C3 Root Index (Not Found)", mux: mux3, method: "DELETE", path: "/api/", wantStatus: http.StatusNotFound, wantBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			tt.mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code, "Status code mismatch")
			if tt.wantStatus == http.StatusOK {
				body, _ := io.ReadAll(rr.Body)
				assert.Contains(t, string(body), tt.wantBody, "Body mismatch")
			}
		})
	}
}

// Test PHP URL blocking functionality
func TestPHPURLBlocking(t *testing.T) {
	// Use the existing testdata directory instead of creating temp files
	sourceDir := "testdata"
	cwd, _ := os.Getwd()
	absSourceDir := filepath.Join(cwd, sourceDir)

	// Create middleware instance with URL blocking enabled (default)
	php, mwCleanup := setupTestMiddleware(t, absSourceDir,
		WithSourceDir(absSourceDir),
		WithDirectPHPURLsBlocking(true))
	defer mwCleanup()

	// Test 1: Direct PHP access should be blocked when blocking is enabled
	t.Run("Block direct PHP access", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.Handle("/", php.HandlerFor("/", "embed_script.php"))

		req := httptest.NewRequest("GET", "/embed_script.php", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code, "Should return 404 for direct PHP access")
		body, _ := io.ReadAll(rr.Body)
		assert.Contains(t, string(body), "Not Found", "Should contain Not Found message")
	})

	// Test 2: Explicitly registered PHP paths should work even with blocking enabled
	t.Run("Allow explicitly registered PHP path", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.Handle("/embed_script.php", php.HandlerFor("/embed_script.php", "embed_script.php"))

		req := httptest.NewRequest("GET", "/embed_script.php", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Should allow explicitly registered PHP path")
		body, _ := io.ReadAll(rr.Body)
		assert.Contains(t, string(body), "Hello from embedded script!", "Should contain expected content")
	})

	// Test 3: Direct PHP access should be allowed when blocking is disabled
	// For this test, we'll directly modify the blockDirectPHPURLs field instead of creating a new instance
	t.Run("Allow PHP access when disabled", func(t *testing.T) {
		// Save original value to restore later
		originalValue := php.blockDirectPHPURLs
		// Disable blocking for this test
		php.blockDirectPHPURLs = false
		// Restore original value when test completes
		defer func() { php.blockDirectPHPURLs = originalValue }()

		mux := http.NewServeMux()
		mux.Handle("/", php.HandlerFor("/", "embed_script.php"))

		req := httptest.NewRequest("GET", "/embed_script.php", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Should allow PHP access when blocking is disabled")
		body, _ := io.ReadAll(rr.Body)
		assert.Contains(t, string(body), "Hello from embedded script!", "Should contain expected content")
	})
}
