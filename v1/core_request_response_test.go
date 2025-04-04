package frango

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Create test PHP files in the testdata directory
func createTestPHPFiles(t *testing.T) string {
	phpFiles := map[string]string{
		"plain_text.php": `<?php
			// Simple plain text response
			header("Content-Type: text/plain");
			echo "This is a plain text response";
		?>`,
		"html_response.php": `<?php
			// HTML response
			header("Content-Type: text/html");
			echo "<!DOCTYPE html><html><head><title>Test HTML</title></head><body><h1>Test HTML Response</h1><p>This is an HTML response from PHP</p></body></html>";
		?>`,
		"query_params.php": `<?php
			// Process query parameters
			header("Content-Type: text/plain");
			echo "Query parameters:\n";
			foreach ($_GET as $key => $value) {
				echo "$key: $value\n";
			}
		?>`,
		"request_headers.php": `<?php
			// Access request headers
			header("Content-Type: text/plain");
			echo "Request headers:\n";
			
			// Access headers from $_SERVER
			// PHP-FPM/Frango adds HTTP_ prefix to most headers and stores them in $_SERVER
			foreach ($_SERVER as $key => $value) {
				// Headers are generally prefixed with HTTP_ or PHP_HEADER_
				if (strpos($key, 'HTTP_') === 0 || strpos($key, 'PHP_HEADER_') === 0) {
					if (strpos($key, 'HTTP_') === 0) {
						$header = substr($key, 5); // Remove HTTP_ prefix
					} else {
						$header = substr($key, 11); // Remove PHP_HEADER_ prefix
					}
					
					// Convert to standard header format (dashes, proper case)
					$header = str_replace('_', '-', $header);
					echo "$header: $value\n";
				}
			}
		?>`,
		"set_response_headers.php": `<?php
			// Set custom response headers
			header("Content-Type: text/plain");
			header("X-Custom-Header: CustomValue");
			header("X-Another-Header: AnotherValue");
			echo "Response with custom headers";
		?>`,
		"json_response.php": `<?php
			// JSON response
			header("Content-Type: application/json");
			
			$data = [
				"status" => "success",
				"message" => "This is a JSON response",
				"data" => [
					"item1" => "value1",
					"item2" => "value2",
					"numbers" => [1, 2, 3, 4, 5]
				]
			];
			
			echo json_encode($data);
		?>`,
		"xml_response.php": `<?php
			// XML response
			header("Content-Type: application/xml");
			
			echo '<?xml version="1.0" encoding="UTF-8"?>';
			echo '<response>';
			echo '<status>success</status>';
			echo '<message>This is an XML response</message>';
			echo '<data>';
			echo '<item key="item1">value1</item>';
			echo '<item key="item2">value2</item>';
			echo '<numbers>';
			echo '<number>1</number>';
			echo '<number>2</number>';
			echo '<number>3</number>';
			echo '</numbers>';
			echo '</data>';
			echo '</response>';
		?>`,
		"binary_image.php": `<?php
			// Simple PNG image generator
			header("Content-Type: image/png");
			
			// Check if GD is available
			if (!extension_loaded('gd') || !function_exists('imagecreate')) {
				// Create a simple 1x1 PNG file (transparent pixel)
				echo base64_decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=');
				exit;
			}
			
			// Create a simple image (100x100 black rectangle)
			$image = imagecreate(100, 100);
			$black = imagecolorallocate($image, 0, 0, 0);
			$white = imagecolorallocate($image, 255, 255, 255);
			
			// Draw something
			imagefilledrectangle($image, 0, 0, 100, 100, $black);
			imagefilledrectangle($image, 25, 25, 75, 75, $white);
			
			// Output the image
			imagepng($image);
			imagedestroy($image);
		?>`,
	}

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-core-tests")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create each PHP file
	for fileName, content := range phpFiles {
		filePath := filepath.Join(tempDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create PHP file %s: %v", fileName, err)
		}
	}

	return tempDir
}

// TestBasicPHPExecution tests a simple plain text response from PHP
func TestBasicPHPExecution(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-basic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP file
	simplePHP := filepath.Join(tempDir, "simple.php")
	phpContent := `<?php
		// Simple plain text response
		header("Content-Type: text/plain");
		echo "This is a plain text response";
	?>`

	if err := os.WriteFile(simplePHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(simplePHP, "/simple.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/simple.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/simple.php", vfs, nil, w, req)

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
	if !strings.Contains(bodyStr, "This is a plain text response") {
		t.Errorf("Expected response to contain 'This is a plain text response', got: %s", bodyStr)
	}
}

// TestHTMLResponse tests a PHP script generating HTML response
func TestHTMLResponse(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-html-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple PHP file with HTML output
	htmlPHP := filepath.Join(tempDir, "html.php")
	phpContent := `<?php
		// HTML response
		header("Content-Type: text/html");
		echo "<!DOCTYPE html><html><head><title>Test HTML</title></head><body><h1>Test HTML Response</h1><p>This is an HTML response from PHP</p></body></html>";
	?>`

	if err := os.WriteFile(htmlPHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(htmlPHP, "/html.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/html.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/html.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "text/html", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check expected HTML content
	expectedContent := []string{
		"<!DOCTYPE html>",
		"<title>Test HTML</title>",
		"<h1>Test HTML Response</h1>",
		"<p>This is an HTML response from PHP</p>",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}

// TestQueryParameters tests a PHP script handling query parameters
func TestQueryParameters(t *testing.T) {
	// Create test PHP files
	tempDir := createTestPHPFiles(t)
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
	filePath := filepath.Join(tempDir, "query_params.php")
	err = vfs.AddSourceFile(filePath, "/query_params.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create and execute request with query parameters
	queryParams := "?name=John&age=30&city=New+York"
	req := httptest.NewRequest("GET", "/query_params.php"+queryParams, nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/query_params.php", vfs, nil, w, req)

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

	// Check that all query parameters are present in response
	expectedParams := map[string]string{
		"name": "John",
		"age":  "30",
		"city": "New York",
	}

	for key, value := range expectedParams {
		expected := key + ": " + value
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("Expected response to contain '%s', got: %s", expected, bodyStr)
		}
	}
}

// TestRequestHeaders tests a PHP script accessing request headers
func TestRequestHeaders(t *testing.T) {
	// Create test PHP files
	tempDir := createTestPHPFiles(t)
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
	filePath := filepath.Join(tempDir, "request_headers.php")
	err = vfs.AddSourceFile(filePath, "/request_headers.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request with custom headers
	req := httptest.NewRequest("GET", "/request_headers.php", nil)
	req.Header.Set("X-Custom-Header", "CustomValue")
	req.Header.Set("X-Test-Header", "TestValue")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/request_headers.php", vfs, nil, w, req)

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

	// Headers are converted variably by PHP, so we need to check case-insensitively
	bodyStrLower := strings.ToLower(bodyStr)

	// Check custom headers are present (case-insensitive check)
	expectedHeaders := map[string]string{
		"x-custom-header": "CustomValue",
		"x-test-header":   "TestValue",
		"accept-language": "en-US,en;q=0.9",
	}

	for key, value := range expectedHeaders {
		keyLower := strings.ToLower(key)
		if !strings.Contains(bodyStrLower, keyLower) {
			t.Errorf("Expected response to contain header '%s', got: %s", key, bodyStr)
		}
		// Simplified value check - PHP might modify header format slightly
		if !strings.Contains(bodyStr, value) {
			t.Errorf("Expected response to contain header value '%s', got: %s", value, bodyStr)
		}
	}
}

// TestResponseHeaders tests a PHP script setting response headers
func TestResponseHeaders(t *testing.T) {
	// Create test PHP files
	tempDir := createTestPHPFiles(t)
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
	filePath := filepath.Join(tempDir, "set_response_headers.php")
	err = vfs.AddSourceFile(filePath, "/set_response_headers.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/set_response_headers.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/set_response_headers.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check custom headers are set in the response
	expectedHeaders := map[string]string{
		"X-Custom-Header":  "CustomValue",
		"X-Another-Header": "AnotherValue",
	}

	for key, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(key)
		if actualValue != expectedValue {
			t.Errorf("Expected header '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	// Check Content-Type - allow for charset suffix
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to start with 'text/plain', got '%s'", contentType)
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
	expectedContent := "Response with custom headers"
	if !strings.Contains(bodyStr, expectedContent) {
		t.Errorf("Expected response to contain '%s', got: %s", expectedContent, bodyStr)
	}
}

// TestJSONResponse tests a PHP script generating and returning JSON
func TestJSONResponse(t *testing.T) {
	// Create test PHP files
	tempDir := createTestPHPFiles(t)
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
	filePath := filepath.Join(tempDir, "json_response.php")
	err = vfs.AddSourceFile(filePath, "/json_response.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/json_response.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/json_response.php", vfs, nil, w, req)

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

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check expected fields
	expectedFields := []string{"status", "message", "data"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected JSON to contain field '%s', got: %v", field, result)
		}
	}

	// Check specific values
	if result["status"] != "success" {
		t.Errorf("Expected 'status' to be 'success', got: %v", result["status"])
	}

	if result["message"] != "This is a JSON response" {
		t.Errorf("Expected 'message' to be 'This is a JSON response', got: %v", result["message"])
	}

	// Check data field is a map
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' to be a map, got: %v", result["data"])
	}

	// Check nested fields
	expectedNestedFields := []string{"item1", "item2", "numbers"}
	for _, field := range expectedNestedFields {
		if _, ok := data[field]; !ok {
			t.Errorf("Expected data to contain field '%s', got: %v", field, data)
		}
	}

	// Check numbers array
	numbers, ok := data["numbers"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'numbers' to be an array, got: %v", data["numbers"])
	}

	if len(numbers) != 5 {
		t.Errorf("Expected 'numbers' to have 5 elements, got: %d", len(numbers))
	}
}

// TestXMLResponse tests a PHP script generating and returning XML
func TestXMLResponse(t *testing.T) {
	// Create test PHP files
	tempDir := createTestPHPFiles(t)
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
	filePath := filepath.Join(tempDir, "xml_response.php")
	err = vfs.AddSourceFile(filePath, "/xml_response.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/xml_response.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/xml_response.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Expected Content-Type '%s', got '%s'", "application/xml", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Check expected XML structure (without fully parsing it)
	expectedXMLFragments := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<response>",
		"<status>success</status>",
		"<message>This is an XML response</message>",
		"<data>",
		"<item key=\"item1\">value1</item>",
		"<item key=\"item2\">value2</item>",
		"<numbers>",
		"<number>1</number>",
		"<number>2</number>",
		"<number>3</number>",
		"</numbers>",
		"</data>",
		"</response>",
	}

	for _, fragment := range expectedXMLFragments {
		if !strings.Contains(bodyStr, fragment) {
			t.Errorf("Expected XML to contain '%s', got: %s", fragment, bodyStr)
		}
	}
}

// TestBinaryResponse tests a PHP script generating and returning binary data (image)
func TestBinaryResponse(t *testing.T) {
	// Skip this test if GD extension isn't available in PHP
	// TODO: Check if GD is available and skip if not

	// Create test PHP files
	tempDir := createTestPHPFiles(t)
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
	filePath := filepath.Join(tempDir, "binary_image.php")
	err = vfs.AddSourceFile(filePath, "/binary_image.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/binary_image.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/binary_image.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Expected Content-Type '%s', got '%s'", "image/png", contentType)
	}

	// Read the binary data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Check that we received binary data that looks like a PNG
	// PNG files start with the signature: 89 50 4E 47 0D 0A 1A 0A
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	if len(imageData) < len(pngSignature) {
		t.Fatalf("Response too short for a PNG file")
	}

	// Check the PNG signature
	for i, b := range pngSignature {
		if imageData[i] != b {
			t.Errorf("Expected PNG signature at byte %d to be %X, got %X", i, b, imageData[i])
		}
	}

	// Ensure minimum size for a valid PNG (header + IHDR chunk + IEND chunk)
	if len(imageData) < 50 {
		t.Errorf("PNG file too small, expected at least 50 bytes, got %d", len(imageData))
	}
}
