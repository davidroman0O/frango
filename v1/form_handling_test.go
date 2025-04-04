package frango

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Create test PHP files for form handling tests
func createFormTestPHPFiles(t *testing.T) string {
	formPHPFiles := map[string]string{
		"get_form.php": `<?php
			// Process GET form parameters
			header("Content-Type: text/plain");
			echo "GET form parameters:\n";
			foreach ($_GET as $key => $value) {
				echo "$key: $value\n";
			}
		?>`,
		"post_form.php": `<?php
			// Process POST form parameters
			header("Content-Type: text/plain");
			echo "POST form parameters:\n";
			
			// Debug info
			echo "Content-Type: " . $_SERVER['CONTENT_TYPE'] . "\n";
			echo "Request method: " . $_SERVER['REQUEST_METHOD'] . "\n";
			
			// Show raw input
			echo "Raw input: " . file_get_contents('php://input') . "\n";
			
			// Process form data from environment variables
			echo "Form data from \$_SERVER variables:\n";
			$formCount = 0;
			foreach ($_SERVER as $key => $value) {
				if (strpos($key, 'PHP_FORM_') === 0) {
					$formKey = substr($key, 9); // Remove PHP_FORM_ prefix
					echo "  $formKey: $value\n";
					$formCount++;
				}
			}
			if ($formCount === 0) {
				echo "  <no form data found>\n";
			}
		?>`,
		"file_upload.php": `<?php
			// Process file uploads
			header("Content-Type: text/plain");
			
			// Debug info
			echo "Content-Type: " . $_SERVER['CONTENT_TYPE'] . "\n";
			echo "Request method: " . $_SERVER['REQUEST_METHOD'] . "\n";
			
			// Show raw input
			$raw = file_get_contents('php://input');
			echo "Raw input length: " . strlen($raw) . "\n";
			
			// Basic file info
			echo "File uploads:\n";
			
			// Check for file data in $_SERVER
			echo "File data from \$_SERVER variables:\n";
			$fileVars = [];
			
			// Find all PHP_FILE_ variables
			foreach ($_SERVER as $key => $value) {
				if (strpos($key, 'PHP_FILE_') === 0) {
					$parts = explode('_', $key, 3);
					if (count($parts) >= 3) {
						$fieldName = $parts[2];
						$fileVars[$fieldName] = $value;
						echo "File field found: $fieldName\n";
					}
				}
			}
			
			// Check for regular form fields
			echo "\nForm fields from \$_SERVER variables:\n";
			foreach ($_SERVER as $key => $value) {
				if (strpos($key, 'PHP_FORM_') === 0) {
					$formKey = substr($key, 9); // Remove PHP_FORM_ prefix
					echo "  $formKey: $value\n";
				}
			}
		?>`,
		"json_body.php": `<?php
			// Process JSON request body
			header("Content-Type: application/json");
			
			// Read raw POST data
			$jsonData = file_get_contents("php://input");
			
			// Try to decode JSON
			$data = json_decode($jsonData, true);
			
			if ($data === null && json_last_error() !== JSON_ERROR_NONE) {
				// JSON error
				$response = [
					"status" => "error",
					"message" => "Invalid JSON: " . json_last_error_msg(),
					"raw_data" => $jsonData
				];
			} else {
				// JSON valid, echo back the data
				$response = [
					"status" => "success",
					"message" => "JSON received and parsed successfully",
					"data" => $data,
					"size" => strlen($jsonData) . " bytes"
				];
			}
			
			echo json_encode($response);
		?>`,
	}

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-form-tests")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create each PHP file
	for fileName, content := range formPHPFiles {
		filePath := filepath.Join(tempDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create PHP file %s: %v", fileName, err)
		}
	}

	return tempDir
}

// TestGETFormHandling tests a PHP script handling GET form parameters
func TestGETFormHandling(t *testing.T) {
	// Create test PHP files
	tempDir := createFormTestPHPFiles(t)
	defer os.RemoveAll(tempDir)

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

	// Add test file to VFS
	filePath := filepath.Join(tempDir, "get_form.php")
	err = vfs.AddSourceFile(filePath, "/get_form.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create form data
	formData := url.Values{}
	formData.Set("name", "John Doe")
	formData.Set("email", "john@example.com")
	formData.Set("message", "This is a test message")
	formData.Set("subscribe", "true")

	// Create and execute request with form parameters
	queryString := formData.Encode()
	req := httptest.NewRequest("GET", "/get_form.php?"+queryString, nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/get_form.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check that all form parameters are present in response
	for key, values := range formData {
		expected := key + ": " + values[0]
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}

// TestPOSTFormHandling tests a PHP script handling POST form parameters
func TestPOSTFormHandling(t *testing.T) {
	// Create test PHP files
	tempDir := createFormTestPHPFiles(t)
	defer os.RemoveAll(tempDir)

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

	// Add test file to VFS
	filePath := filepath.Join(tempDir, "post_form.php")
	err = vfs.AddSourceFile(filePath, "/post_form.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create form data
	formData := url.Values{}
	formData.Set("name", "Jane Smith")
	formData.Set("email", "jane@example.com")
	formData.Set("message", "This is another test message")
	formData.Set("agree_terms", "yes")

	// Create and execute request with form parameters
	req := httptest.NewRequest("POST", "/post_form.php", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/post_form.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check expected content
	expectedParams := []string{
		"Content-Type: application/x-www-form-urlencoded",
		"Request method: POST",
		"Raw input:",
		"Form data from $_SERVER variables:",
		"  name: Jane Smith",
		"  email: jane@example.com",
		"  message: This is another test message",
		"  agree_terms: yes",
	}

	for _, expected := range expectedParams {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}

// TestMultipartFormData tests a PHP script handling multipart form data with file uploads
func TestMultipartFormData(t *testing.T) {
	// Create test PHP files
	tempDir := createFormTestPHPFiles(t)
	defer os.RemoveAll(tempDir)

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

	// Add test file to VFS
	filePath := filepath.Join(tempDir, "file_upload.php")
	err = vfs.AddSourceFile(filePath, "/file_upload.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create a multipart form with a file
	var multipartBuffer bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBuffer)

	// Add form fields
	multipartWriter.WriteField("field1", "value1")
	multipartWriter.WriteField("field2", "value2")

	// Create a file field with some content
	fileWriter, err := multipartWriter.CreateFormFile("uploaded_file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	fileContent := "This is a test file for upload testing.\nIt has multiple lines.\nTesting file upload."
	fileWriter.Write([]byte(fileContent))

	// Close the multipart writer to finalize the form
	multipartWriter.Close()

	// Create and execute request with multipart form
	req := httptest.NewRequest("POST", "/file_upload.php", &multipartBuffer)
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/file_upload.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check that file upload was processed
	expectedOutputs := []string{
		"Content-Type: multipart/form-data",
		"Request method: POST",
		"Form fields from $_SERVER variables:",
	}

	for _, expected := range expectedOutputs {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}

// TestJSONRequestBody tests a PHP script handling JSON request body
func TestJSONRequestBody(t *testing.T) {
	// Create test PHP files
	tempDir := createFormTestPHPFiles(t)
	defer os.RemoveAll(tempDir)

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

	// Add test file to VFS
	filePath := filepath.Join(tempDir, "json_body.php")
	err = vfs.AddSourceFile(filePath, "/json_body.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create JSON data
	jsonData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    123,
			"name":  "Alice Johnson",
			"email": "alice@example.com",
			"roles": []string{"admin", "editor"},
			"preferences": map[string]interface{}{
				"theme":         "dark",
				"language":      "en-US",
				"newsletter":    true,
				"twoFactorAuth": false,
			},
		},
		"action":    "update",
		"timestamp": 1617123456,
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Create and execute request with JSON body
	req := httptest.NewRequest("POST", "/json_body.php", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/json_body.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type '%s', got '%s'", "application/json", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check status
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", result["status"])
	}

	// Check that the echoed data matches what we sent
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' to be a map, got: %v", result["data"])
	}

	// Check user data is present
	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'user' to be a map, got: %v", data["user"])
	}

	// Check user ID
	if int(user["id"].(float64)) != 123 {
		t.Errorf("Expected user id 123, got %v", user["id"])
	}

	// Check user name
	if user["name"] != "Alice Johnson" {
		t.Errorf("Expected user name 'Alice Johnson', got '%v'", user["name"])
	}

	// Check action
	if data["action"] != "update" {
		t.Errorf("Expected action 'update', got '%v'", data["action"])
	}
}

// TestSimpleGETForm tests a basic GET form submission
func TestSimpleGETForm(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-get-form-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP file for GET form handling
	getFormPHP := filepath.Join(tempDir, "get_form.php")
	phpContent := `<?php
		// Process GET form parameters
		header("Content-Type: text/plain");
		echo "GET form parameters:\n";
		foreach ($_GET as $key => $value) {
			echo "$key: $value\n";
		}
	?>`

	if err := os.WriteFile(getFormPHP, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to create PHP file: %v", err)
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

	// Add test file to VFS
	err = vfs.AddSourceFile(getFormPHP, "/get_form.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create query parameters
	params := url.Values{}
	params.Add("name", "John Doe")
	params.Add("email", "john@example.com")
	params.Add("message", "This is a test message")
	params.Add("subscribe", "true")

	// Create request with query parameters
	req := httptest.NewRequest("GET", "/get_form.php?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/get_form.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "text/plain", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check expected content
	expectedParams := []string{
		"name: John Doe",
		"email: john@example.com",
		"message: This is a test message",
		"subscribe: true",
	}

	for _, expected := range expectedParams {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}

// TestSimplePOSTForm tests a basic POST form submission
func TestSimplePOSTForm(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-post-form-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP file for POST form handling
	postFormPHP := filepath.Join(tempDir, "post_form.php")
	phpContent := `<?php
		// Process POST form parameters
		header("Content-Type: text/plain");
		echo "POST form parameters:\n";
		
		// Debug info
		echo "Content-Type: " . $_SERVER['CONTENT_TYPE'] . "\n";
		echo "Request method: " . $_SERVER['REQUEST_METHOD'] . "\n";
		
		// Show raw input
		echo "Raw input: " . file_get_contents('php://input') . "\n";
		
		// Show form data from environment variables
		echo "Form data from \$_SERVER variables:\n";
		$formCount = 0;
		foreach ($_SERVER as $key => $value) {
			if (strpos($key, 'PHP_FORM_') === 0) {
				$formKey = substr($key, 9); // Remove PHP_FORM_ prefix
				echo "  $formKey: $value\n";
				$formCount++;
			}
		}
		if ($formCount === 0) {
			echo "  <no form data found>\n";
		}
		
		// Name directly
		if (isset($_SERVER['PHP_FORM_name'])) {
			echo "Name: " . $_SERVER['PHP_FORM_name'] . "\n";
		}
	?>`

	if err := os.WriteFile(postFormPHP, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to create PHP file: %v", err)
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

	// Add test file to VFS
	err = vfs.AddSourceFile(postFormPHP, "/post_form.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create form data
	formData := url.Values{}
	formData.Add("name", "Jane Smith")
	formData.Add("email", "jane@example.com")
	formData.Add("message", "This is another test message")
	formData.Add("agree_terms", "yes")

	// Create POST request
	req := httptest.NewRequest("POST", "/post_form.php", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/post_form.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "text/plain", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check expected content
	expectedParams := []string{
		"Content-Type: application/x-www-form-urlencoded",
		"Request method: POST",
		"Raw input:",
		"Form data from $_SERVER variables:",
		"  name: Jane Smith",
		"  email: jane@example.com",
		"  message: This is another test message",
		"  agree_terms: yes",
		"Name: Jane Smith",
	}

	for _, expected := range expectedParams {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}
