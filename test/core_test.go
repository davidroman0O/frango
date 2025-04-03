package discovery

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidroman0O/frango"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupFrango creates a new Frango middleware instance with the discovery dir as source
func setupFrango(t *testing.T) *frango.Middleware {
	// Get absolute path to discovery directory
	sourcePath, err := filepath.Abs(".")
	require.NoError(t, err, "Failed to resolve discovery directory path")

	// Create Frango middleware
	php, err := frango.New(
		frango.WithSourceDir(sourcePath),
		frango.WithDevelopmentMode(true),
	)
	require.NoError(t, err, "Failed to create Frango middleware")

	// Register cleanup
	t.Cleanup(func() {
		php.Shutdown()
	})

	return php
}

// TestPlainTextResponse tests the plain text response from PHP
func TestPlainTextResponse(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/01_plain_text.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check content type - Note that PHP may add charset
	contentType := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(contentType, "text/plain"),
		"Content-Type should start with text/plain")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	assert.Contains(t, string(body), "Hello from PHP!", "Response body doesn't contain expected content")
}

// TestHTMLResponse tests the HTML response from PHP
func TestHTMLResponse(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/02_html_response.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(contentType, "text/html"),
		"Unexpected content type: %s", contentType)

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Test for expected HTML elements
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "<title>Frango Test - HTML Response</title>", "Missing title")
	assert.Contains(t, bodyStr, "<h1>Hello from PHP!</h1>", "Missing h1 content")
	assert.Contains(t, bodyStr, "<li>Item 1</li>", "Missing list item")
}

// TestQueryParameters tests query parameter handling in PHP
func TestQueryParameters(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/03_query_params.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request with query parameters
	resp, err := http.Get(server.URL + "?name=Tester&age=42&interests[]=Golang&interests[]=PHP")
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Test for expected content
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Name: Tester", "Missing name parameter")
	assert.Contains(t, bodyStr, "Age: 42", "Missing age parameter")
	assert.Contains(t, bodyStr, "<li>Golang</li>", "Missing interests parameter")
	assert.Contains(t, bodyStr, "<li>PHP</li>", "Missing interests parameter")
	assert.Contains(t, bodyStr, "Name via FRANGO_QUERY: Tester",
		"Missing FRANGO_QUERY variable")
}

// TestRequestHeaders tests request header handling in PHP
func TestRequestHeaders(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/04_request_headers.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create custom request with headers
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err, "Failed to create request")

	// Add custom headers
	req.Header.Set("X-Test-Header", "Test Value")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "Frango-Test-Agent")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Test for expected content
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "User-Agent: Frango-Test-Agent",
		"Missing User-Agent header")
	assert.Contains(t, bodyStr, "User-Agent via FRANGO:",
		"Missing FRANGO_HEADER variable")
}

// TestResponseHeaders tests setting response headers in PHP
func TestResponseHeaders(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/05_response_headers.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check custom headers
	assert.Equal(t, "Custom Value", resp.Header.Get("X-Custom-Header"),
		"Missing custom header")
	assert.Equal(t, "Testing Custom Headers", resp.Header.Get("X-Frango-Test"),
		"Missing Frango test header")
	assert.Equal(t, "no-cache, no-store, must-revalidate",
		resp.Header.Get("Cache-Control"), "Missing cache control header")
}

// TestJSONResponse tests JSON response from PHP
func TestJSONResponse(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/06_json_response.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check content type
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"),
		"Unexpected content type")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Parse JSON
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	require.NoError(t, err, "Failed to parse JSON response")

	// Check structure
	assert.Equal(t, true, data["success"], "JSON success should be true")
	assert.Equal(t, "This is a JSON response from PHP", data["message"],
		"Unexpected message in JSON")

	// Check nested data
	jsonData, ok := data["data"].(map[string]interface{})
	require.True(t, ok, "Missing data object in JSON")
	assert.Equal(t, float64(3), jsonData["count"], "Unexpected item count in JSON")

	// Check items array
	items, ok := jsonData["items"].([]interface{})
	require.True(t, ok, "Missing items array in JSON")
	assert.Len(t, items, 3, "Expected 3 items in JSON")
}

// TestXMLResponse tests XML response from PHP
func TestXMLResponse(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/07_xml_response.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check content type
	assert.Equal(t, "application/xml", resp.Header.Get("Content-Type"),
		"Unexpected content type")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Check for XML structure (basic validation)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"Missing XML declaration")
	assert.Contains(t, bodyStr, "<response>", "Missing root element")
	assert.Contains(t, bodyStr, "<message>This is an XML response from PHP</message>",
		"Missing message element")
	assert.Contains(t, bodyStr, "<item id=\"1\">", "Missing item element")
}

// TestBinaryResponse tests binary (image) response from PHP
func TestBinaryResponse(t *testing.T) {
	php := setupFrango(t)

	// Create handler using the PHP script
	handler := php.For("core/08_binary_response.php")

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check content type
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"),
		"Unexpected content type")

	// Check content disposition
	assert.Equal(t, "inline; filename=\"frango_test.png\"",
		resp.Header.Get("Content-Disposition"), "Unexpected content disposition")

	// Read binary data (don't need to validate content, just ensure it's not empty)
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read binary response")
	assert.NotEmpty(t, data, "Binary response should not be empty")

	// Basic PNG validation - check for PNG signature (first 8 bytes)
	if len(data) >= 8 {
		pngSignature := []byte{137, 80, 78, 71, 13, 10, 26, 10}
		assert.Equal(t, pngSignature, data[:8], "Response doesn't have PNG signature")
	}
}
