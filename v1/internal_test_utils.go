package frango

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnv provides a complete test environment for PHP execution
type TestEnv struct {
	TempDir     string
	PHP         *Middleware
	VFS         *VFS
	ScriptFiles map[string]string
}

// SetupTest creates a complete test environment with the provided PHP files
func SetupTest(t *testing.T, phpFiles map[string]string, opts ...Option) *TestEnv {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "frango-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create PHP files
	scriptPaths := make(map[string]string)
	for name, content := range phpFiles {
		filePath := filepath.Join(tempDir, name)
		dirPath := filepath.Dir(filePath)

		// Create directory if needed
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create directory for %s: %v", name, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to write file %s: %v", name, err)
		}

		scriptPaths[name] = filePath
	}

	// Use a null logger by default for tests
	devNull, _ := os.Open(os.DevNull)
	testLogger := log.New(devNull, "", 0)
	finalOpts := append([]Option{WithLogger(testLogger)}, opts...)

	// Add source directory
	finalOpts = append(finalOpts, WithSourceDir(tempDir))
	finalOpts = append(finalOpts, WithDevelopmentMode(true))

	// Setup middleware
	php, err := New(finalOpts...)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Create VFS
	vfs := php.NewVFS()

	// Add files to VFS
	for name, path := range scriptPaths {
		if err := vfs.AddSourceFile(path, "/"+name); err != nil {
			vfs.Cleanup()
			php.Shutdown()
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to add file %s to VFS: %v", name, err)
		}
	}

	return &TestEnv{
		TempDir:     tempDir,
		PHP:         php,
		VFS:         vfs,
		ScriptFiles: scriptPaths,
	}
}

// CleanupTest performs cleanup of test environment
func CleanupTest(env *TestEnv) {
	if env.VFS != nil {
		env.VFS.Cleanup()
	}
	if env.PHP != nil {
		env.PHP.Shutdown()
	}
	if env.TempDir != "" {
		os.RemoveAll(env.TempDir)
	}
}

// ExecutePHP executes a PHP script and returns status, headers and body
func ExecutePHP(t *testing.T, env *TestEnv, scriptPath string, req *http.Request, renderFn interface{}) (int, http.Header, string) {
	w := httptest.NewRecorder()

	// The pattern should be provided via renderFn to use for path parameters extraction
	var pattern string
	var renderData RenderData

	// If renderFn is provided, extract the pattern from it
	// Otherwise we'll use scriptPath as the pattern
	if renderFn != nil {
		switch fn := renderFn.(type) {
		case func(http.ResponseWriter, *http.Request) map[string]interface{}:
			renderData = fn

			// Call the function to get the data
			testData := fn(nil, req)
			if p, ok := testData["ROUTE_PATTERN"].(string); ok {
				pattern = p
			} else {
				pattern = scriptPath // Default to scriptPath
			}
		case func(map[string]interface{}):
			// Simple function that populates a map
			data := make(map[string]interface{})
			fn(data)

			if p, ok := data["ROUTE_PATTERN"].(string); ok {
				pattern = p
			} else {
				pattern = scriptPath // Default to scriptPath
			}

			renderData = func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
				return data
			}
		default:
			// Unsupported type
			t.Fatalf("Unsupported render function type: %T", renderFn)
			return 0, nil, ""
		}
	} else {
		// If no render function, create one that provides the pattern
		pattern = scriptPath
		renderData = func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"ROUTE_PATTERN": pattern,
			}
		}
	}

	// Patch the request to include our pattern by setting the Pattern field directly
	modifiedReq := req.Clone(req.Context())
	modifiedReq.Pattern = pattern

	// Execute the PHP script using the modified request
	env.PHP.ExecutePHP(scriptPath, env.VFS, renderData, w, modifiedReq)

	// Get response
	resp := w.Result()
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp.StatusCode, resp.Header, string(body)
}

// ParseJSON parses a JSON response body into a map
func ParseJSON(t *testing.T, body string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nResponse body: %s", err, body)
	}
	return result
}

// CheckResponseContains checks if response contains all the expected strings
func CheckResponseContains(t *testing.T, body string, expectedContents []string) {
	for _, expected := range expectedContents {
		if !strings.Contains(body, expected) {
			t.Errorf("Response does not contain expected content: %s\nActual body: %s", expected, body)
		}
	}
}

// CreateFormRequest creates a request with form data
func CreateFormRequest(method, url, contentType string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}
