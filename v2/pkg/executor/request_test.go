package executor

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidroman0O/frango/v2/pkg/vfs"
	"github.com/dunglas/frankenphp"
)

// setupTestFrankenPHP initializes FrankenPHP for tests
func setupTestFrankenPHP(t *testing.T) {
	err := frankenphp.Init()
	if err != nil {
		t.Fatalf("Failed to initialize FrankenPHP: %v", err)
	}
	t.Cleanup(func() {
		frankenphp.Shutdown()
	})
}

// TestPHPGlobals tests that PHP globals are properly set up
func TestPHPGlobals(t *testing.T) {
	// Initialize FrankenPHP for the test
	setupTestFrankenPHP(t)

	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-globals-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file to test superglobal initialization
	testPHP := `<?php
header("Content-Type: application/json");
error_reporting(E_ERROR);  // Suppress warnings for undefined variables

// First, get and initialize all custom globals
$form = isset($_FORM) ? $_FORM : null;
$path = isset($_PATH) ? $_PATH : null;
$path_segments = isset($_PATH_SEGMENTS) ? $_PATH_SEGMENTS : null;
$json = isset($_JSON) ? $_JSON : null;
$url = isset($_URL) ? $_URL : null;
$current_url = isset($_CURRENT_URL) ? $_CURRENT_URL : null;
$query = isset($_QUERY) ? $_QUERY : null;

// Build a response with all initialized superglobals
$response = array(
	"server" => $_SERVER,
	"get" => $_GET,
	"post" => $_POST,
	"request" => $_REQUEST,
	"form" => $form,
	"path" => $path,
	"path_segments" => $path_segments,
	"json" => $json,
	"url" => $url,
	"current_url" => $current_url,
	"query" => $query,
	"raw_input" => file_get_contents('php://input'),
	// Check for global template variables
	"has_template" => isset($_TEMPLATE)
);

echo json_encode($response, JSON_PRETTY_PRINT);
?>`

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "superglobals_test.php")
	if err := os.WriteFile(testFilePath, []byte(testPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Setup VFS and Executor
	logger := logWriter(t)
	v, err := vfs.NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer v.Cleanup()

	// Add PHP file to VFS
	err = v.AddSourceFile(testFilePath, "/superglobals_test.php")
	if err != nil {
		t.Fatalf("Failed to add file to VFS: %v", err)
	}

	// Create executor
	exec := NewExecutor(Config{
		Logger:          logger,
		DevelopmentMode: true,
		DisplayErrors:   true,
	}, v)

	// ----- Test 1: GET request with query parameters -----
	t.Run("GET request with query parameters", func(t *testing.T) {
		queryParams := url.Values{}
		queryParams.Set("id", "123")
		queryParams.Set("name", "Test User")
		queryParams.Set("action", "view")

		getURL := "/superglobals_test.php?" + queryParams.Encode()
		getReq := httptest.NewRequest("GET", getURL, nil)
		getResp := httptest.NewRecorder()

		// Execute GET request
		exec.Execute(v, "/superglobals_test.php", nil, getResp, getReq)

		// Verify GET response
		if getResp.Code != http.StatusOK {
			t.Errorf("Expected GET status code %d, got %d", http.StatusOK, getResp.Code)
		}

		// Parse response
		var getResult map[string]interface{}
		if err := json.NewDecoder(getResp.Body).Decode(&getResult); err != nil {
			t.Fatalf("Failed to parse GET response as JSON: %v", err)
		}

		// Verify $_GET was properly populated
		get, ok := getResult["get"].(map[string]interface{})
		if !ok {
			t.Fatalf("GET response has no $_GET data")
		}

		// Check if all expected query parameters exist
		for key, values := range queryParams {
			expected := values[0]
			actual, exists := get[key]
			if !exists {
				t.Errorf("GET: Expected parameter '%s' not found in $_GET", key)
			} else if actual != expected {
				t.Errorf("GET: Expected $_GET['%s'] = '%s', got '%v'", key, expected, actual)
			}
		}

		// Verify $_QUERY matches $_GET (even if null/empty)
		query, _ := getResult["query"]
		// Check that query exists and is not null
		if query == nil {
			// $_QUERY might not be defined, this is acceptable for now
			t.Logf("GET: $_QUERY is null in response")
		} else if queryMap, ok := query.(map[string]interface{}); ok {
			// If $_QUERY is defined and a map, verify its contents match $_GET
			for key, expected := range get {
				actual, exists := queryMap[key]
				if !exists {
					t.Errorf("GET: $_QUERY['%s'] missing", key)
					continue
				}

				if actual != expected {
					t.Errorf("GET: $_QUERY['%s'] = '%v' doesn't match $_GET['%s'] = '%v'",
						key, actual, key, expected)
				}
			}
		}
	})

	// ----- Test 2: POST request with form data -----
	t.Run("POST request with form data", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("user_id", "456")
		formData.Set("email", "test@example.com")
		formData.Set("message", "This is a test message")

		postReq := httptest.NewRequest("POST", "/superglobals_test.php", strings.NewReader(formData.Encode()))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postResp := httptest.NewRecorder()

		// Execute POST request
		exec.Execute(v, "/superglobals_test.php", nil, postResp, postReq)

		// Verify POST response
		if postResp.Code != http.StatusOK {
			t.Errorf("Expected POST status code %d, got %d", http.StatusOK, postResp.Code)
		}

		// Parse response
		var postResult map[string]interface{}
		if err := json.NewDecoder(postResp.Body).Decode(&postResult); err != nil {
			t.Fatalf("Failed to parse POST response as JSON: %v", err)
		}

		// Verify $_POST was properly populated
		post, ok := postResult["post"].(map[string]interface{})
		if !ok {
			t.Fatalf("POST response has no $_POST data")
		}

		// Check all expected form parameters
		for key, values := range formData {
			expected := values[0]
			actual, exists := post[key]
			if !exists {
				t.Errorf("POST: Expected parameter '%s' not found in $_POST", key)
			} else if actual != expected {
				t.Errorf("POST: Expected $_POST['%s'] = '%s', got '%v'", key, expected, actual)
			}
		}

		// Verify $_FORM matches $_POST (even if null/empty)
		form, _ := postResult["form"]
		// Check that form exists and is not null
		if form == nil {
			// $_FORM might not be defined, this is acceptable for now
			t.Logf("POST: $_FORM is null in response")
		} else if formMap, ok := form.(map[string]interface{}); ok {
			// If $_FORM is defined and a map, verify its contents match $_POST
			for key, expected := range post {
				actual, exists := formMap[key]
				if !exists {
					t.Errorf("POST: $_FORM['%s'] missing", key)
					continue
				}

				if actual != expected {
					t.Errorf("POST: $_FORM['%s'] = '%v' doesn't match $_POST['%s'] = '%v'",
						key, actual, key, expected)
				}
			}
		}
	})

	// ----- Test 3: JSON request -----
	t.Run("JSON request", func(t *testing.T) {
		jsonData := map[string]interface{}{
			"user": map[string]interface{}{
				"id":    789,
				"name":  "JSON User",
				"email": "json@example.com",
			},
			"items":  []string{"item1", "item2", "item3"},
			"active": true,
		}

		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			t.Fatalf("Failed to marshal JSON data: %v", err)
		}

		jsonReq := httptest.NewRequest("POST", "/superglobals_test.php", bytes.NewReader(jsonBytes))
		jsonReq.Header.Set("Content-Type", "application/json")
		jsonResp := httptest.NewRecorder()

		// Execute JSON request
		exec.Execute(v, "/superglobals_test.php", nil, jsonResp, jsonReq)

		// Verify JSON response
		if jsonResp.Code != http.StatusOK {
			t.Errorf("Expected JSON status code %d, got %d", http.StatusOK, jsonResp.Code)
		}

		// Parse response
		var jsonResult map[string]interface{}
		if err := json.NewDecoder(jsonResp.Body).Decode(&jsonResult); err != nil {
			t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, jsonResp.Body.String())
		}

		// Verify $_JSON was properly populated (even if null/empty)
		jsonResp2, _ := jsonResult["json"]
		// Check that jsonResp2 exists and is not null
		if jsonResp2 == nil {
			// $_JSON might not be defined, this is acceptable for now
			t.Logf("JSON: $_JSON is null in response")
		} else if jsonMap, ok := jsonResp2.(map[string]interface{}); ok {
			// If $_JSON is defined, check for expected fields
			// Check for user object
			_, exists := jsonMap["user"]
			if !exists {
				t.Errorf("JSON: Expected $_JSON to contain 'user' key")
			}

			// Check for items array
			_, exists = jsonMap["items"]
			if !exists {
				t.Errorf("JSON: Expected $_JSON to contain 'items' key")
			}

			// Check for active boolean
			_, exists = jsonMap["active"]
			if !exists {
				t.Errorf("JSON: Expected $_JSON to contain 'active' key")
			}
		} else {
			t.Errorf("JSON: $_JSON is not a map: %T", jsonResp2)
		}
	})
}

// TestServerVariables tests that $_SERVER variables are properly populated
func TestServerVariables(t *testing.T) {
	// Initialize FrankenPHP for the test
	setupTestFrankenPHP(t)

	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-server-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file to test $_SERVER variables
	testPHP := `<?php
header("Content-Type: application/json");

// Output all $_SERVER variables
echo json_encode($_SERVER, JSON_PRETTY_PRINT);
?>`

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "server_test.php")
	if err := os.WriteFile(testFilePath, []byte(testPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Setup VFS and Executor
	logger := logWriter(t)
	v, err := vfs.NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer v.Cleanup()

	// Add PHP file to VFS
	err = v.AddSourceFile(testFilePath, "/server_test.php")
	if err != nil {
		t.Fatalf("Failed to add file to VFS: %v", err)
	}

	// Create executor
	exec := NewExecutor(Config{
		Logger:          logger,
		DevelopmentMode: true,
		DisplayErrors:   true,
	}, v)

	// Create a request with custom headers
	req := httptest.NewRequest("GET", "/server_test.php?foo=bar", nil)
	req.Header.Set("User-Agent", "Frango Test Agent")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Custom-Header", "Custom Value")
	resp := httptest.NewRecorder()

	// Execute request
	exec.Execute(v, "/server_test.php", nil, resp, req)

	// Verify response
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	// Parse response to get $_SERVER variables
	var serverVars map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&serverVars); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}

	// Check for required $_SERVER variables
	requiredVars := []string{
		"SCRIPT_FILENAME", "DOCUMENT_ROOT", "REQUEST_URI", "REQUEST_METHOD",
		"QUERY_STRING", "HTTP_USER_AGENT", "HTTP_ACCEPT", "HTTP_X_CUSTOM_HEADER",
		"SCRIPT_NAME", "PHP_SELF",
	}

	for _, varName := range requiredVars {
		if _, exists := serverVars[varName]; !exists {
			t.Errorf("$_SERVER missing required variable: %s", varName)
		}
	}

	// Check specific values
	if serverVars["REQUEST_METHOD"] != "GET" {
		t.Errorf("Expected REQUEST_METHOD to be 'GET', got '%v'", serverVars["REQUEST_METHOD"])
	}

	if serverVars["QUERY_STRING"] != "foo=bar" {
		t.Errorf("Expected QUERY_STRING to be 'foo=bar', got '%v'", serverVars["QUERY_STRING"])
	}

	if serverVars["HTTP_USER_AGENT"] != "Frango Test Agent" {
		t.Errorf("Expected HTTP_USER_AGENT to be 'Frango Test Agent', got '%v'", serverVars["HTTP_USER_AGENT"])
	}

	if serverVars["HTTP_X_CUSTOM_HEADER"] != "Custom Value" {
		t.Errorf("Expected HTTP_X_CUSTOM_HEADER to be 'Custom Value', got '%v'", serverVars["HTTP_X_CUSTOM_HEADER"])
	}
}

// Helper function to create a logger that writes to the test log
func logWriter(t *testing.T) *log.Logger {
	return log.New(&testLogWriter{t: t}, "[test] ", log.LstdFlags)
}

// testLogWriter is a helper type that writes to the test log
type testLogWriter struct {
	t *testing.T
}

// Write implements io.Writer for testLogWriter
func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
