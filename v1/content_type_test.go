package frango

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContentHtml tests a PHP script generating HTML response
func TestContentHtml(t *testing.T) {
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

	for _, expect := range expectedContent {
		if !strings.Contains(bodyStr, expect) {
			t.Errorf("Response does not contain expected content: %s", expect)
		}
	}
}

// TestContentJson tests a PHP script generating and returning JSON
func TestContentJson(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-json-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file that returns JSON
	jsonPHP := filepath.Join(tempDir, "json.php")
	phpContent := `<?php
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
	?>`

	if err := os.WriteFile(jsonPHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(jsonPHP, "/json.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/json.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/json.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "application/json", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Verify it's valid JSON by checking for expected content
	expectedContent := []string{
		"\"status\":\"success\"",
		"\"message\":\"This is a JSON response\"",
		"\"data\":",
		"\"item1\":\"value1\"",
		"\"numbers\":[1,2,3,4,5]",
	}

	for _, expect := range expectedContent {
		if !strings.Contains(bodyStr, expect) {
			t.Errorf("Response does not contain expected JSON content: %s", expect)
		}
	}
}

// TestContentXml tests a PHP script generating and returning XML
func TestContentXml(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-xml-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file that returns XML
	xmlPHP := filepath.Join(tempDir, "xml.php")
	phpContent := `<?php
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
	?>`

	if err := os.WriteFile(xmlPHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(xmlPHP, "/xml.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/xml.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/xml.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/xml") {
		t.Errorf("Expected Content-Type starting with %s, got %s", "application/xml", contentType)
	}

	// Check body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)

	// Verify it's valid XML by checking for expected tags
	expectedContent := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<response>",
		"<status>success</status>",
		"<message>This is an XML response</message>",
		"<data>",
		"<item key=\"item1\">value1</item>",
		"<numbers>",
		"<number>1</number>",
	}

	for _, expect := range expectedContent {
		if !strings.Contains(bodyStr, expect) {
			t.Errorf("Response does not contain expected XML content: %s", expect)
		}
	}
}

// TestContentBinary tests a PHP script generating and returning binary data (image)
func TestContentBinary(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "frango-binary-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file that returns a PNG image
	pngPHP := filepath.Join(tempDir, "binary_image.php")
	phpContent := `<?php
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
	?>`

	if err := os.WriteFile(pngPHP, []byte(phpContent), 0644); err != nil {
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
	err = vfs.AddSourceFile(pngPHP, "/binary_image.php")
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

	// Check body to verify it's a PNG file
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Check for PNG signature (should start with these bytes)
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	if len(imageData) < len(pngSignature) {
		t.Fatalf("Response data too short to be a PNG file")
	}

	for i, b := range pngSignature {
		if imageData[i] != b {
			t.Errorf("Expected PNG signature at byte %d to be %X, got %X", i, b, imageData[i])
		}
	}

	// Verify minimum size (should be larger than the signature)
	if len(imageData) < 50 {
		t.Errorf("PNG file too small, expected at least 50 bytes, got %d", len(imageData))
	}
}
