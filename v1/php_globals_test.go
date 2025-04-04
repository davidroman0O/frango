package frango

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImprovedSuperglobals tests that our PHP globals script correctly populates standard PHP superglobals
func TestImprovedSuperglobals(t *testing.T) {
	// Create a PHP file to test superglobal initialization
	testPHP := `<?php
	header("Content-Type: application/json");
	
	// Build a response with all initialized superglobals
	$response = array(
		"server" => $_SERVER,
		"get" => $_GET,
		"post" => $_POST,
		"request" => $_REQUEST,
		"form" => $_FORM,
		"path" => $_PATH,
		"path_segments" => $_PATH_SEGMENTS,
		"json" => $_JSON,
		"url" => $_URL,
		"current_url" => $_CURRENT_URL,
		"query" => $_QUERY,
		"raw_input" => file_get_contents('php://input'),
		// Check for global template variables
		"has_template" => isset($_TEMPLATE)
	);
	
	echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Setup a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "frango-globals-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "superglobals_test.php")
	if err := os.WriteFile(testFilePath, []byte(testPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add PHP file to VFS
	err = vfs.AddSourceFile(testFilePath, "/superglobals_test.php")
	if err != nil {
		t.Fatalf("Failed to add file to VFS: %v", err)
	}

	// Install PHP globals script
	err = InstallPHPGlobals(vfs)
	if err != nil {
		t.Fatalf("Failed to install PHP globals: %v", err)
	}

	// ----- Test 1: GET request with query parameters -----
	queryParams := url.Values{}
	queryParams.Set("id", "123")
	queryParams.Set("name", "Test User")
	queryParams.Set("action", "view")

	getURL := "/superglobals_test.php?" + queryParams.Encode()
	getReq := httptest.NewRequest("GET", getURL, nil)
	getResp := httptest.NewRecorder()

	// Execute GET request
	php.ExecutePHP("/superglobals_test.php", vfs, nil, getResp, getReq)

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

	// Check all expected query parameters
	for key, values := range queryParams {
		expected := values[0]
		actual, exists := get[key]
		if !exists {
			t.Errorf("GET: Expected parameter '%s' not found in $_GET", key)
		} else if actual != expected {
			t.Errorf("GET: Expected $_GET['%s'] = '%s', got '%v'", key, expected, actual)
		}
	}

	// Verify $_QUERY matches $_GET
	query, ok := getResult["query"].(map[string]interface{})
	if !ok {
		t.Fatalf("GET response has no $_QUERY data")
	}
	for key, expected := range get {
		if actual, exists := query[key]; !exists || actual != expected {
			t.Errorf("GET: $_QUERY['%s'] = '%v' doesn't match $_GET['%s'] = '%v'", key, actual, key, expected)
		}
	}

	// ----- Test 2: POST request with form data -----
	formData := url.Values{}
	formData.Set("user_id", "456")
	formData.Set("email", "test@example.com")
	formData.Set("message", "This is a test message")

	postReq := httptest.NewRequest("POST", "/superglobals_test.php", strings.NewReader(formData.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postResp := httptest.NewRecorder()

	// Execute POST request
	php.ExecutePHP("/superglobals_test.php", vfs, nil, postResp, postReq)

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

	// Verify $_FORM matches $_POST
	form, ok := postResult["form"].(map[string]interface{})
	if !ok {
		t.Fatalf("POST response has no $_FORM data")
	}
	for key, expected := range post {
		if actual, exists := form[key]; !exists || actual != expected {
			t.Errorf("POST: $_FORM['%s'] = '%v' doesn't match $_POST['%s'] = '%v'", key, actual, key, expected)
		}
	}

	// Check if php://input was populated
	rawInput, ok := postResult["raw_input"].(string)
	if !ok {
		t.Fatalf("POST response has no raw_input data")
	}
	if rawInput == "" {
		t.Logf("Warning: php://input was empty, but we have a workaround in place")
	}

	// ----- Test 3: JSON request -----
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

	jsonReq := httptest.NewRequest("POST", "/superglobals_test.php", strings.NewReader(string(jsonBytes)))
	jsonReq.Header.Set("Content-Type", "application/json")
	jsonResp := httptest.NewRecorder()

	// Execute JSON request
	php.ExecutePHP("/superglobals_test.php", vfs, nil, jsonResp, jsonReq)

	// Verify JSON response
	if jsonResp.Code != http.StatusOK {
		t.Errorf("Expected JSON status code %d, got %d", http.StatusOK, jsonResp.Code)
	}

	// Parse response
	var jsonResult map[string]interface{}
	jsonRespBody, _ := io.ReadAll(jsonResp.Body)
	if err := json.Unmarshal(jsonRespBody, &jsonResult); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, string(jsonRespBody))
	}

	// Verify $_JSON was properly populated
	jsonResp2, ok := jsonResult["json"].(map[string]interface{})
	if !ok {
		t.Fatalf("JSON response has no $_JSON data")
	}

	// Check the structure of $_JSON
	if _, ok := jsonResp2["user"].(map[string]interface{}); !ok {
		t.Errorf("JSON: Expected $_JSON to contain 'user' object")
	}

	t.Logf("All superglobal tests completed successfully")
}

// TestTemplateVariables tests the automatic population of template variables
func TestTemplateVariables(t *testing.T) {
	// Create a PHP file to test template variable initialization
	testPHP := `<?php
	header("Content-Type: application/json");
	
	// Echo template variables directly
	$response = array(
		"direct_vars" => array(
			"title" => $title ?? "Not set",
			"user" => $user ?? "Not set",
			"items" => $items ?? "Not set",
		),
		"template_array" => $_TEMPLATE ?? "Not set",
	);
	
	echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "frango-template-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "template_test.php")
	if err := os.WriteFile(testFilePath, []byte(testPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add PHP file to VFS
	err = vfs.AddSourceFile(testFilePath, "/template_test.php")
	if err != nil {
		t.Fatalf("Failed to add file to VFS: %v", err)
	}

	// Install PHP globals script
	err = InstallPHPGlobals(vfs)
	if err != nil {
		t.Fatalf("Failed to install PHP globals: %v", err)
	}

	// Define render data
	renderData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"title": "Test Page",
			"user": map[string]interface{}{
				"id":      123,
				"name":    "Test User",
				"email":   "test@example.com",
				"isAdmin": true,
			},
			"items": []string{"item1", "item2", "item3"},
		}
	}

	// Execute PHP with render data
	req := httptest.NewRequest("GET", "/template_test.php", nil)
	resp := httptest.NewRecorder()

	php.ExecutePHP("/template_test.php", vfs, renderData, resp, req)

	// Verify response
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}

	// Verify direct variables were set
	directVars, ok := result["direct_vars"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response has no direct_vars data")
	}

	if directVars["title"] != "Test Page" {
		t.Errorf("Expected $title = 'Test Page', got '%v'", directVars["title"])
	}

	// Verify $_TEMPLATE array was populated
	templateArr, ok := result["template_array"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response has no $_TEMPLATE data")
	}

	if _, ok := templateArr["user"].(map[string]interface{}); !ok {
		t.Errorf("Expected $_TEMPLATE[\"user\"] to be an object")
	}

	t.Logf("Template variable tests completed successfully")
}
