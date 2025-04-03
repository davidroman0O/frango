package discovery

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

	"github.com/davidroman0O/frango"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGETFormParameters tests handling of GET form parameters
func TestGETFormParameters(t *testing.T) {
	// Create form PHP script first
	formScript := `<?php
// GET Form test
header('Content-Type: text/html; charset=UTF-8');
$name = $_GET['name'] ?? 'Unknown';
$email = $_GET['email'] ?? 'no-email';
$subscribe = isset($_GET['subscribe']) ? 'Yes' : 'No';
?>
<!DOCTYPE html>
<html>
<head>
    <title>GET Form Test</title>
</head>
<body>
    <h1>GET Form Test</h1>
    <div id="results">
        <p>Name: <?= htmlspecialchars($name) ?></p>
        <p>Email: <?= htmlspecialchars($email) ?></p>
        <p>Subscribe: <?= htmlspecialchars($subscribe) ?></p>
    </div>
</body>
</html>`

	// Write the script to a temporary file
	scriptPath := filepath.Join("form", "01_get_form.php")
	err := os.MkdirAll(filepath.Dir(scriptPath), 0755)
	require.NoError(t, err, "Failed to create form directory")

	err = os.WriteFile(scriptPath, []byte(formScript), 0644)
	require.NoError(t, err, "Failed to write form script")

	// Only remove the specific file, not the whole directory
	defer os.Remove(scriptPath)

	// Setup frango
	sourcePath, err := filepath.Abs(".")
	require.NoError(t, err)

	php, err := frango.New(
		frango.WithSourceDir(sourcePath),
		frango.WithDevelopmentMode(true),
	)
	require.NoError(t, err, "Failed to create Frango middleware")
	defer php.Shutdown()

	// Create handler using the PHP script
	handler := php.For(scriptPath)

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make GET request with form parameters in query string
	resp, err := http.Get(server.URL + "?name=John+Doe&email=john@example.com&subscribe=1")
	require.NoError(t, err, "Failed to make request")
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	// Check response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Test for expected content
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "<p>Name: John Doe</p>", "Missing or incorrect name parameter")
	assert.Contains(t, bodyStr, "<p>Email: john@example.com</p>", "Missing or incorrect email parameter")
	assert.Contains(t, bodyStr, "<p>Subscribe: Yes</p>", "Missing or incorrect subscribe parameter")
}

// TestPOSTFormParameters tests handling of POST form parameters
func TestPOSTFormParameters(t *testing.T) {
	// Ensure the form directory exists
	os.MkdirAll("form", 0755)

	// Setup frango
	sourcePath, err := filepath.Abs(".")
	require.NoError(t, err)

	php, err := frango.New(
		frango.WithSourceDir(sourcePath),
		frango.WithDevelopmentMode(true),
		frango.WithDirectPHPURLsBlocking(false), // Explicitly disable direct PHP blocking for test
	)
	require.NoError(t, err, "Failed to create Frango middleware")
	defer php.Shutdown()

	// Create form data
	formValues := url.Values{}
	formValues.Set("name", "Jane Smith")
	formValues.Set("email", "jane@example.com")
	formValues.Set("comment", "This is a test comment with special chars: <>&")
	formValues.Set("rating", "5")

	// Create request with encoded form data
	req, err := http.NewRequest("POST", "/form/02_post_form.php",
		strings.NewReader(formValues.Encode()))
	require.NoError(t, err, "Failed to create request")

	// Set content type
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute the handler directly with path specified
	handler := php.For("form/02_post_form.php")
	handler.ServeHTTP(rr, req)

	// Check response
	require.Equal(t, http.StatusOK, rr.Code, "Handler returned wrong status code")

	// Get the response body
	bodyStr := rr.Body.String()
	t.Logf("Response Body: %s", bodyStr)

	// Test for expected content
	assert.Contains(t, bodyStr, "<p>Method: POST</p>", "Missing or incorrect method")
	assert.Contains(t, bodyStr, "<p>Name: Jane Smith</p>", "Missing or incorrect name parameter")
	assert.Contains(t, bodyStr, "<p>Email: jane@example.com</p>", "Missing or incorrect email parameter")
	assert.Contains(t, bodyStr, "This is a test comment with special chars:", "Missing or incorrect comment parameter")
	assert.Contains(t, bodyStr, "<p>Rating: 5</p>", "Missing or incorrect rating parameter")
}

// TestMultipartFormWithFileUpload tests handling of multipart form data with file upload
func TestMultipartFormWithFileUpload(t *testing.T) {
	// Ensure form directory exists
	os.MkdirAll("form", 0755)

	// Setup frango
	sourcePath, err := filepath.Abs(".")
	require.NoError(t, err)

	php, err := frango.New(
		frango.WithSourceDir(sourcePath),
		frango.WithDevelopmentMode(true),
		frango.WithDirectPHPURLsBlocking(false), // Explicitly disable direct PHP blocking for test
	)
	require.NoError(t, err, "Failed to create Frango middleware")
	defer php.Shutdown()

	// Simpler approach with http.Client + multipart
	// Create a pipe for direct multipart writing
	pr, pw := io.Pipe()

	// Create a multipart writer with a known boundary
	mpWriter := multipart.NewWriter(pw)

	// Start a goroutine to write the form data
	go func() {
		defer pw.Close()
		defer mpWriter.Close()

		// Add text field
		err := mpWriter.WriteField("name", "File Uploader")
		if err != nil {
			t.Logf("Error writing field: %v", err)
			return
		}

		// Add file field
		part, err := mpWriter.CreateFormFile("uploadedFile", "test.txt")
		if err != nil {
			t.Logf("Error creating form file: %v", err)
			return
		}

		_, err = part.Write([]byte("This is a test file for PHP upload processing"))
		if err != nil {
			t.Logf("Error writing file content: %v", err)
			return
		}
	}()

	// Create request with the pipe as body
	req, err := http.NewRequest("POST", "/form/03_file_upload.php", pr)
	require.NoError(t, err, "Failed to create request")

	// Set content type with boundary
	req.Header.Set("Content-Type", mpWriter.FormDataContentType())

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute the PHP script directly (instead of using test server)
	handler := php.For("form/03_file_upload.php")
	handler.ServeHTTP(rr, req)

	// Check response
	require.Equal(t, http.StatusOK, rr.Code, "Handler returned wrong status code")

	// Check response body
	bodyStr := rr.Body.String()
	t.Logf("Response Body: %s", bodyStr)

	// Test for expected content
	assert.Contains(t, bodyStr, "<p>Name: File Uploader</p>", "Missing or incorrect name parameter")
	assert.Contains(t, bodyStr, "<p>File Uploaded: Yes</p>", "File upload not detected")
	assert.Contains(t, bodyStr, "<p>File Name: test.txt</p>", "Incorrect file name")
	assert.Contains(t, bodyStr, "File Error: No error", "Unexpected file upload error")
	assert.Contains(t, bodyStr, "This is a test file", "Missing or incorrect file content")
}

// TestJSONRequestBodyHandling tests handling of JSON request body
func TestJSONRequestBodyHandling(t *testing.T) {
	// Create JSON handling PHP script
	jsonScript := `<?php
// JSON request body test
header('Content-Type: application/json');

// Get the raw POST data
$rawData = file_get_contents('php://input');

// Decode the JSON data
$jsonData = json_decode($rawData, true);

// Check if we got valid JSON
$isValidJSON = $jsonData !== null;

// Check FRANGO_JSON variables
$frangoTitle = $_SERVER['FRANGO_JSON_title'] ?? 'Not set';
$frangoAuthor = $_SERVER['FRANGO_JSON_author'] ?? 'Not set';

// Prepare response
$response = [
    'success' => $isValidJSON,
    'message' => $isValidJSON ? 'JSON parsed successfully' : 'Failed to parse JSON',
    'received' => $jsonData,
    'frango' => [
        'title' => json_decode($frangoTitle ?? '""'),
        'author' => json_decode($frangoAuthor ?? '""')
    ]
];

// Output JSON response
echo json_encode($response, JSON_PRETTY_PRINT);`

	// Write the script to a temporary file
	scriptPath := filepath.Join("form", "04_json_body.php")
	err := os.MkdirAll(filepath.Dir(scriptPath), 0755)
	require.NoError(t, err, "Failed to create form directory")

	err = os.WriteFile(scriptPath, []byte(jsonScript), 0644)
	require.NoError(t, err, "Failed to write JSON script")

	// Only remove the specific file, not the whole directory
	defer os.Remove(scriptPath)

	// Setup frango
	sourcePath, err := filepath.Abs(".")
	require.NoError(t, err)

	php, err := frango.New(
		frango.WithSourceDir(sourcePath),
		frango.WithDevelopmentMode(true),
	)
	require.NoError(t, err, "Failed to create Frango middleware")
	defer php.Shutdown()

	// Create handler using the PHP script
	handler := php.For(scriptPath)

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create JSON data
	jsonData := map[string]interface{}{
		"title":     "Test JSON Request",
		"author":    "Frango Tester",
		"content":   "This is a test of JSON request handling",
		"tags":      []string{"test", "json", "php"},
		"published": true,
		"views":     42,
	}
	jsonBytes, err := json.Marshal(jsonData)
	require.NoError(t, err, "Failed to marshal JSON")

	// Create request
	req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(jsonBytes))
	require.NoError(t, err, "Failed to create request")

	// Set content type
	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
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

	// Parse JSON response
	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	require.NoError(t, err, "Failed to parse JSON response")

	// Check basic response
	assert.Equal(t, true, responseData["success"], "JSON parsing should succeed")
	assert.Equal(t, "JSON parsed successfully", responseData["message"],
		"Unexpected message")

	// Check received data
	received, ok := responseData["received"].(map[string]interface{})
	require.True(t, ok, "Missing received data")
	assert.Equal(t, "Test JSON Request", received["title"], "Missing title in received data")
	assert.Equal(t, "Frango Tester", received["author"], "Missing author in received data")

	// Check frango data
	frango, ok := responseData["frango"].(map[string]interface{})
	require.True(t, ok, "Missing frango data")
	assert.Equal(t, "Test JSON Request", frango["title"], "Missing title in frango data")
	assert.Equal(t, "Frango Tester", frango["author"], "Missing author in frango data")
}
