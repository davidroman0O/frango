package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidroman0O/frango"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupFrango creates a new Frango middleware instance with the test directory as source
func setupEnvironmentFrango(t *testing.T) *frango.Middleware {
	// Get absolute path to test directory
	sourcePath, err := filepath.Abs(".")
	require.NoError(t, err, "Failed to resolve test directory path")

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

// TestPHPIncludeRequire tests the ability to include and require other PHP files
func TestPHPIncludeRequire(t *testing.T) {
	php := setupEnvironmentFrango(t)

	// Create handler using the PHP script
	handler := php.For("environment/01_include.php")

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

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	bodyStr := string(body)

	// Verify HTML structure from included files
	assert.Contains(t, bodyStr, "<!DOCTYPE html>", "Missing DOCTYPE from header.php")
	assert.Contains(t, bodyStr, "<title>PHP Include Test</title>", "Missing title from header.php")
	assert.Contains(t, bodyStr, "<h1>PHP Include/Require Test</h1>", "Missing H1 from header.php")
	assert.Contains(t, bodyStr, "&copy;", "Missing copyright symbol from footer.php")
	assert.Contains(t, bodyStr, "Frango PHP Test Suite", "Missing footer text from footer.php")

	// Verify variables and functions from included files are accessible
	assert.Contains(t, bodyStr, "<li>Main variable: Main script variable</li>", "Missing main variable")
	assert.Contains(t, bodyStr, "<li>Header variable: Variable from header.php</li>", "Missing header variable")
	assert.Contains(t, bodyStr, "<li>Function result: 8</li>", "Missing or incorrect function result")
	assert.Contains(t, bodyStr, "<li>Constant from required file: Constant from functions.php</li>", "Missing constant")

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
}

// TestPHPEnvironmentVariables tests the ability of PHP to access environment variables
func TestPHPEnvironmentVariables(t *testing.T) {
	// Set custom environment variables for the test
	os.Setenv("TEST_ENV_VAR1", "CustomValue1")
	os.Setenv("TEST_ENV_VAR2", "CustomValue2")
	defer func() {
		// Clean up environment variables
		os.Unsetenv("TEST_ENV_VAR1")
		os.Unsetenv("TEST_ENV_VAR2")
	}()

	php := setupEnvironmentFrango(t)

	// Create handler using the PHP script
	handler := php.For("environment/02_env_variables.php")

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

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	bodyStr := string(body)

	// Verify PHP environment variables
	assert.Contains(t, bodyStr, "PHP Version:", "Missing PHP version info")
	assert.Contains(t, bodyStr, "FrankenPHP", "Missing or incorrect server software")
	assert.Contains(t, bodyStr, "HTTP/1.1", "Missing or incorrect server protocol")
	assert.Contains(t, bodyStr, "Request Method: GET", "Missing or incorrect request method")

	// Verify custom environment variables
	assert.Contains(t, bodyStr, "TEST_ENV_VAR1: CustomValue1", "Missing or incorrect custom environment variable 1")
	assert.Contains(t, bodyStr, "TEST_ENV_VAR2: CustomValue2", "Missing or incorrect custom environment variable 2")

	// Verify Frango Superglobals
	assert.Contains(t, bodyStr, "$_PATH Available: Yes", "Frango $_PATH superglobal not available")
	assert.Contains(t, bodyStr, "$_PATH_SEGMENTS Available: Yes", "Frango $_PATH_SEGMENTS superglobal not available")
	assert.Contains(t, bodyStr, "$_JSON Available: Yes", "Frango $_JSON superglobal not available")

	// Check for PHP errors
	AssertNoPHPErrors(t, bodyStr)
}
