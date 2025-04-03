package test

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

// NOTE: checkPHPErrors has been replaced by AssertNoPHPErrors in php_error_utils.go

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

	// Check for PHP errors
	AssertNoPHPErrors(t, string(body))
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

	// Check content type - Note that PHP may add charset
	contentType := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(contentType, "text/html"),
		"Content-Type should start with text/html")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Test for expected HTML structure
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "<!DOCTYPE html>", "Missing DOCTYPE")
	assert.Contains(t, bodyStr, "<title>HTML Test</title>", "Missing title")
	assert.Contains(t, bodyStr, "<h1>HTML Response Test</h1>", "Missing H1")
	assert.Contains(t, bodyStr, "<p>This is an HTML response from PHP.</p>", "Missing paragraph")

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
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
	resp, err := http.Get(server.URL + "?name=John&age=30&active=true")
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Test for expected content
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Name: John", "Missing name parameter")
	assert.Contains(t, bodyStr, "Age: 30", "Missing age parameter")
	assert.Contains(t, bodyStr, "Active: true", "Missing active parameter")

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
}

// TestRequestHeaders tests request header access in PHP
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

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
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

	// Check custom headers set by PHP
	assert.Equal(t, "Custom Value", resp.Header.Get("X-Custom-Header"),
		"Missing custom X-Custom-Header")
	assert.Equal(t, "Testing Custom Headers", resp.Header.Get("X-Frango-Test"),
		"Missing X-Frango-Test header")
	assert.Equal(t, "no-cache, no-store, must-revalidate", resp.Header.Get("Cache-Control"),
		"Missing Cache-Control header")

	// Check expires header (just check it exists)
	assert.NotEmpty(t, resp.Header.Get("Expires"),
		"Missing Expires header")

	// Check content type - matching the actual header sent by the PHP script
	contentType := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(contentType, "text/html"),
		"Content-Type should be HTML")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	bodyStr := string(body)

	// Check for content that's actually in the response
	assert.Contains(t, bodyStr, "Response Headers Test",
		"Response body doesn't contain expected title")
	assert.Contains(t, bodyStr, "This page has set the following headers:",
		"Response body doesn't contain expected content")

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
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

	// Check for PHP errors
	AssertNoPHPErrors(t, string(body))
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

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
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

	// Check response body (shouldn't be empty)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	assert.NotEmpty(t, body, "Binary response should not be empty")

	// Check PNG signature (first 8 bytes)
	if len(body) >= 8 {
		// PNG signature: 89 50 4E 47 0D 0A 1A 0A
		pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		assert.Equal(t, pngSignature, body[0:8], "Response is not a valid PNG image")
	} else {
		t.Error("Binary response too short for a PNG image")
	}

	// Since this is binary data, we can't check for PHP errors in the usual way
	// Instead, we just make sure it's a valid image (which we did above)
}
