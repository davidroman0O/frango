package executor

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidroman0O/frango/pkg/php"
	"github.com/davidroman0O/frango/pkg/vfs"
	"github.com/dunglas/frankenphp"
)

// TestExecutorEnv holds the necessary components for executor tests.
type TestExecutorEnv struct {
	VFS      *vfs.VFS
	Executor *Executor
	Config   Config
	TempDir  string
	Cleanup  func()
}

// ConfigOption defines a function type for applying configuration options.
// This allows for flexible configuration in tests using functional options pattern.
type ConfigOption func(*Config)

// TestExecutorBasic tests basic functionality of the executor
func TestExecutorBasic(t *testing.T) {
	t.Helper()

	// Create a basic PHP file for testing
	basicPhp := `<?php echo "Hello, Executor!"; ?>`

	// Setup test environment using the exported helper
	env, err := SetupExecutorTest(t, map[string]string{
		"basic.php": basicPhp,
	}, WithTestLogger(t)) // Use logger option
	if err != nil {
		t.Fatalf("SetupExecutorTest failed: %v", err)
	}
	defer env.Cleanup()

	// Execute PHP file using the exported helper
	req := httptest.NewRequest("GET", "/basic.php", nil)
	resp := ExecuteRequest(t, env, "/basic.php", req)

	// Check response
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}
	body := resp.Body.String()
	CheckNoPHPErrors(t, body) // Check for PHP errors
	if !strings.Contains(body, "Hello, Executor!") {
		t.Errorf("Expected response to contain 'Hello, Executor!', got: %s", body)
	}
}

// TestRelativeIncludePaths tests that relative include paths work correctly
func TestRelativeIncludePaths(t *testing.T) {
	t.Helper()
	// Removed t.Skip - testing relative includes now

	// Initialize FrankenPHP for the test
	setupFrankenPHP(t)

	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-relative-include-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	categoryDir := filepath.Join(tempDir, "categories")
	subCategoryDir := filepath.Join(categoryDir, "electronics")
	includeDir := filepath.Join(tempDir, "includes")

	// Create directories
	for _, dir := range []string{categoryDir, subCategoryDir, includeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create a shared include file
	includeFile := `<?php
function getHeader() {
	return "Header from include file";
}
?>`
	err = os.WriteFile(filepath.Join(includeDir, "header.php"), []byte(includeFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create include file: %v", err)
	}

	// Create a main PHP file that uses relative path to include
	mainFile := `<?php
header("Content-Type: application/json");

// Use relative path for include
include_once("../includes/header.php");

$response = [
	"success" => function_exists("getHeader"),
	"header" => getHeader(),
	"file" => __FILE__,
	"dir" => __DIR__,
];

echo json_encode($response, JSON_PRETTY_PRINT);
?>`
	err = os.WriteFile(filepath.Join(categoryDir, "index.php"), []byte(mainFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Create a subcategory PHP file that uses relative path to include
	subcategoryFile := `<?php
header("Content-Type: application/json");

// Use relative path for include
include_once("../../includes/header.php");

$response = [
	"success" => function_exists("getHeader"),
	"header" => getHeader(),
	"file" => __FILE__,
	"dir" => __DIR__,
];

echo json_encode($response, JSON_PRETTY_PRINT);
?>`
	err = os.WriteFile(filepath.Join(subCategoryDir, "smartphones.php"), []byte(subcategoryFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create subcategory file: %v", err)
	}

	// Create a VFS
	vfsLogger := log.New(os.Stdout, "[vfs-test] ", log.LstdFlags)
	v, err := createTestVFS(tempDir, vfsLogger, false)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer v.Cleanup()

	// Add all PHP files to VFS
	if err := v.AddSourceDirectory(tempDir, "/"); err != nil {
		t.Fatalf("Failed to add directory to VFS: %v", err)
	}

	// Create executor
	exec := NewExecutor(Config{
		Logger:          vfsLogger,
		DevelopmentMode: true,
		DisplayErrors:   true,
	}, v)

	// Test main file with a relative include
	t.Run("Main file with relative include", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/categories/index.php", nil)
		resp := httptest.NewRecorder()

		exec.Execute(v, "/categories/index.php", nil, resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
			t.Logf("Response: %s", resp.Body.String())
			return
		}

		bodyStr := resp.Body.String()

		// Verify no PHP errors occurred
		if strings.Contains(bodyStr, "Warning") || strings.Contains(bodyStr, "Fatal error") {
			t.Errorf("PHP error occurred: %s", bodyStr)
		}

		// Check if the include worked
		if !strings.Contains(bodyStr, "Header from include file") {
			t.Errorf("Include file content not found in response: %s", bodyStr)
		}

		// Check if success is true
		if !strings.Contains(bodyStr, `"success": true`) {
			t.Errorf("Expected 'success: true' in response, got: %s", bodyStr)
		}
	})

	// Test subcategory file with a relative include going up two levels
	t.Run("Subcategory file with relative include", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/categories/electronics/smartphones.php", nil)
		resp := httptest.NewRecorder()

		exec.Execute(v, "/categories/electronics/smartphones.php", nil, resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
			t.Logf("Response: %s", resp.Body.String())
			return
		}

		bodyStr := resp.Body.String()

		// Verify no PHP errors occurred
		if strings.Contains(bodyStr, "Warning") || strings.Contains(bodyStr, "Fatal error") {
			t.Errorf("PHP error occurred: %s", bodyStr)
		}

		// Check if the include worked
		if !strings.Contains(bodyStr, "Header from include file") {
			t.Errorf("Include file content not found in response: %s", bodyStr)
		}

		// Check if success is true
		if !strings.Contains(bodyStr, `"success": true`) {
			t.Errorf("Expected 'success: true' in response, got: %s", bodyStr)
		}
	})
}

// TestErrorHandling tests the error handling functionality
func TestErrorHandling(t *testing.T) {
	t.Helper()

	// Create PHP files for error scenarios
	syntaxErrorPhp := `<?php // Missing semicolon
echo "This will fail" $var = 42; ?>`
	runtimeErrorPhp := `<?php // Division by zero
$a = 10; $b = 0; $result = $a / $b; echo $result; ?>`
	errorHandlerPhp := `<?php
header('Content-Type: application/json'); http_response_code(500);
$errorType = $_SERVER['PHP_ERROR_TYPE'] ?? 'unknown';
$errorDetail = $_SERVER['PHP_LAST_ERROR'] ?? 'Unknown error';
echo json_encode(['status' => 'error', 'details' => $errorDetail]);
?>`

	// Common setup
	commonFiles := map[string]string{
		"syntax_error.php":  syntaxErrorPhp,
		"runtime_error.php": runtimeErrorPhp,
		"error_handler.php": errorHandlerPhp,
	}

	// Test syntax error without custom error handler
	t.Run("Syntax error without custom handler", func(t *testing.T) {
		t.Helper()
		env, err := SetupExecutorTest(t, commonFiles, WithTestLogger(t), WithTestDevelopmentMode(true), WithTestErrorDisplay(true))
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		defer env.Cleanup()

		req := httptest.NewRequest("GET", "/syntax_error.php", nil)
		resp := ExecuteRequest(t, env, "/syntax_error.php", req)
		body := resp.Body.String()

		if !strings.Contains(strings.ToLower(body), "syntax error") && !strings.Contains(strings.ToLower(body), "parse error") {
			t.Errorf("Expected syntax/parse error message, got: %s", body)
		}
	})

	// Test runtime error without custom error handler
	t.Run("Runtime error without custom handler", func(t *testing.T) {
		t.Helper()
		env, err := SetupExecutorTest(t, commonFiles, WithTestLogger(t), WithTestDevelopmentMode(true), WithTestErrorDisplay(true))
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		defer env.Cleanup()

		req := httptest.NewRequest("GET", "/runtime_error.php", nil)
		resp := ExecuteRequest(t, env, "/runtime_error.php", req)
		body := resp.Body.String()

		// Check for the script name in the error context, not the exact phrase
		if !strings.Contains(strings.ToLower(body), "runtime_error.php") {
			t.Errorf("Expected error message containing 'runtime_error.php', got: %s", body)
		}
	})

	// Test runtime error with custom error handler
	t.Run("Runtime error with custom handler", func(t *testing.T) {
		t.Helper()
		env, err := SetupExecutorTest(t, commonFiles, WithTestLogger(t), WithTestDevelopmentMode(true), WithTestErrorDisplay(true), WithTestErrorHandler("/error_handler.php"))
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		defer env.Cleanup()

		req := httptest.NewRequest("GET", "/runtime_error.php", nil)
		resp := ExecuteRequest(t, env, "/runtime_error.php", req)
		body := resp.Body.String()

		if resp.Code != http.StatusInternalServerError {
			t.Errorf("Expected status code %d with custom handler, got %d", http.StatusInternalServerError, resp.Code)
		}

		if !strings.Contains(body, `"status":"error"`) {
			t.Errorf("Expected JSON error response, got: %s", body)
		}

		if !strings.Contains(strings.ToLower(body), "division by zero") {
			t.Errorf("Expected division by zero in error details, got: %s", body)
		}
	})
}

// Sanity check test function
func TestExecutorSanityCheck(t *testing.T) {
	t.Helper()
	t.Log("Executor sanity check test running.")
	if 1+1 != 2 {
		t.Error("Math is broken!")
	}
}

// --- Placeholder for actual tests ---
// func TestExecutorSomething(t *testing.T) { ... }

// setupFrankenPHP initializes FrankenPHP for tests (unexported)
// Includes mutex for safety if tests run in parallel.
func setupFrankenPHP(t *testing.T) {
	t.Helper()
	mu.Lock()
	defer mu.Unlock() // Ensure mutex is always unlocked

	if !frankenphpInitialized {
		err := frankenphp.Init()
		if err != nil {
			// No t.Fatalf here, return error or panic?
			// For tests, Fatalf is usually appropriate.
			t.Fatalf("Failed to initialize FrankenPHP: %v", err)
		}
		frankenphpInitialized = true
		log.Println("FrankenPHP initialized for tests.")

		// Use a global mechanism or test suite feature if available to ensure
		// Shutdown happens only once after all tests in the package.
		// t.Cleanup is per-test. For package-level, TestMain is needed.
		// For simplicity here, we rely on the last test's Cleanup.
		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()
			if frankenphpInitialized {
				frankenphp.Shutdown()
				frankenphpInitialized = false
				log.Println("FrankenPHP shut down.")
			}
		})
	}
}

// Global state for FrankenPHP initialization tracking
var (
	frankenphpInitialized bool
	mu                    sync.Mutex
)

// WithTestLogger configures the executor to use a test-specific logger. (Exported)
func WithTestLogger(t *testing.T) ConfigOption {
	return func(cfg *Config) {
		cfg.Logger = log.New(testWriter{t}, "[test-executor] ", log.LstdFlags|log.Lshortfile)
	}
}

// WithTestDevelopmentMode sets the development mode. (Exported)
func WithTestDevelopmentMode(enabled bool) ConfigOption {
	return func(cfg *Config) {
		cfg.DevelopmentMode = enabled
	}
}

// WithTestErrorDisplay sets whether PHP errors should be displayed. (Exported)
func WithTestErrorDisplay(enabled bool) ConfigOption {
	return func(cfg *Config) {
		cfg.DisplayErrors = enabled
	}
}

// WithTestErrorHandler sets a custom PHP error handler path. (Exported)
func WithTestErrorHandler(path string) ConfigOption {
	return func(cfg *Config) {
		cfg.ErrorHandlerPath = path
	}
}

// testWriter adapts testing.T to io.Writer for logging.
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(bytes.TrimSpace(p)))
	return len(p), nil
}

// SetupExecutorTest prepares a test environment with VFS and Executor. (Exported)
func SetupExecutorTest(t *testing.T, files map[string]string, opts ...ConfigOption) (*TestExecutorEnv, error) {
	t.Helper()

	// Initialize FrankenPHP needed for tests
	setupFrankenPHP(t)

	// Default config
	cfg := Config{
		Logger:          log.New(io.Discard, "", 0), // Default to discard logger
		DevelopmentMode: true,                       // Default to true for tests unless overridden
		DisplayErrors:   true,                       // Default to true for tests unless overridden
	}

	// Apply options
	for _, opt := range opts {
		opt(&cfg)
	}

	tempDir, err := os.MkdirTemp("", "frango-executor-test-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	vfsConfig := vfs.VFSConfig{
		TempDir:         tempDir,
		Logger:          cfg.Logger,
		DevelopMode:     cfg.DevelopmentMode,
		GlobalsProvider: &php.StandardGlobalsProvider{},
	}
	testVFS, err := vfs.NewVFSWithConfig(vfsConfig)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create VFS: %w", err)
	}

	for name, content := range files {
		virtualPath := "/" + name
		filePath := filepath.Join(tempDir, name)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			testVFS.Cleanup()
			return nil, fmt.Errorf("failed to write temp file %s: %w", name, err)
		}
		err = testVFS.AddSourceFile(filePath, virtualPath)
		if err != nil {
			testVFS.Cleanup()
			return nil, fmt.Errorf("failed to add source file %s to VFS: %w", name, err)
		}
	}

	executor := NewExecutor(cfg, testVFS)

	cleanup := func() {
		testVFS.Cleanup()
		time.Sleep(20 * time.Millisecond) // Slightly increased delay
	}

	env := &TestExecutorEnv{
		VFS:      testVFS,
		Executor: executor,
		Config:   cfg,
		TempDir:  tempDir,
		Cleanup:  cleanup,
	}

	return env, nil
}

// ExecuteRequest runs the executor for a given request and returns a recorder. (Exported)
func ExecuteRequest(t *testing.T, env *TestExecutorEnv, scriptPath string, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	// The current Execute signature takes (vfs, scriptPath, renderFn, w, r)
	// renderFn is nil as we are not testing template rendering here yet
	env.Executor.Execute(env.VFS, scriptPath, nil, recorder, req)
	return recorder
}

// CheckNoPHPErrors checks the response body for PHP error messages using the php package. (Exported)
func CheckNoPHPErrors(t *testing.T, body string) {
	t.Helper()
	php.AssertNoErrors(t, body) // Use the helper from the php package
}

// CheckValueMatch checks if a value in a map matches the expected value, handling potential array wrapping. (Exported)
func CheckValueMatch(t *testing.T, data map[string]interface{}, key string, expectedValue string, fieldName string) {
	t.Helper()
	rawValue, exists := data[key]
	if !exists {
		t.Errorf("Expected field '%s' missing from data", fieldName)
		return
	}

	var actualValue string
	switch v := rawValue.(type) {
	case string:
		actualValue = v
	case []interface{}:
		if len(v) > 0 {
			actualValue = fmt.Sprintf("%v", v[0])
		} else {
			actualValue = ""
		}
	default:
		actualValue = fmt.Sprintf("%v", v)
	}

	if actualValue != expectedValue {
		t.Errorf("Field '%s': expected '%s', got '%s' (type: %T)", fieldName, expectedValue, actualValue, rawValue)
	}
}
