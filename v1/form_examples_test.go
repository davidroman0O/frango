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

// TestDocumentedFormHandling tests the form handling approach documented in form_processing.md
func TestDocumentedFormHandling(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "frango-form-example-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP script using the documented approach
	phpFile := filepath.Join(tempDir, "form_processor.php")
	phpContent := `<?php
header("Content-Type: text/plain");
echo "Form Data Processing\n";
echo "===================\n\n";

// Process POST form data
echo "POST Form Data:\n";
$postCount = 0;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_FORM_') === 0) {
        $formKey = substr($key, 9); // Remove PHP_FORM_ prefix
        echo "- $formKey: $value\n";
        $postCount++;
    }
}
if ($postCount === 0) {
    echo "No POST data submitted.\n";
}

// Process GET query parameters
echo "\nGET Query Parameters:\n";
$getCount = 0;
foreach ($_SERVER as $key => $value) {
    if (strpos($key, 'PHP_QUERY_') === 0) {
        $queryKey = substr($key, 10); // Remove PHP_QUERY_ prefix
        echo "- $queryKey: $value\n";
        $getCount++;
    }
}
if ($getCount === 0) {
    echo "No GET parameters submitted.\n";
}

// Access specific fields with error checking
echo "\nAccessing specific fields:\n";

// POST data example
$name = isset($_SERVER['PHP_FORM_name']) ? $_SERVER['PHP_FORM_name'] : 'Not provided';
echo "POST name: $name\n";

// GET parameter example
$page = isset($_SERVER['PHP_QUERY_page']) ? $_SERVER['PHP_QUERY_page'] : '1';
echo "GET page: $page\n";
?>`

	if err := os.WriteFile(phpFile, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to create PHP file: %v", err)
	}

	// Setup frango
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add test file to VFS
	err = vfs.AddSourceFile(phpFile, "/form_processor.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Test cases for form submissions
	tests := []struct {
		name        string
		formFields  map[string]string
		method      string
		expectedOut []string
	}{
		{
			name: "POST form submission",
			formFields: map[string]string{
				"name":    "John Doe",
				"email":   "john@example.com",
				"message": "Hello from test",
			},
			method: "POST",
			expectedOut: []string{
				"POST Form Data:",
				"- name: John Doe",
				"- email: john@example.com",
				"- message: Hello from test",
				"No GET parameters submitted.",
				"POST name: John Doe",
			},
		},
		{
			name: "GET form submission",
			formFields: map[string]string{
				"product": "Widget",
				"qty":     "5",
				"page":    "3",
			},
			method: "GET",
			expectedOut: []string{
				"No POST data submitted.",
				"GET Query Parameters:",
				"- product: Widget",
				"- qty: 5",
				"- page: 3",
				"POST name: Not provided",
				"GET page: 3",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create form body for POST (GET query params are handled differently)
			var formBody string
			if tc.method == "POST" {
				formParts := make([]string, 0, len(tc.formFields))
				for k, v := range tc.formFields {
					formParts = append(formParts, k+"="+v)
				}
				formBody = strings.Join(formParts, "&")
			}

			// Create request URL with query params for GET
			url := "/form_processor.php"
			if tc.method == "GET" {
				queryParts := make([]string, 0, len(tc.formFields))
				for k, v := range tc.formFields {
					queryParts = append(queryParts, k+"="+v)
				}
				if len(queryParts) > 0 {
					url += "?" + strings.Join(queryParts, "&")
				}
			}

			// Create request
			var req *http.Request
			if tc.method == "POST" {
				req = httptest.NewRequest("POST", url, strings.NewReader(formBody))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest("GET", url, nil)
			}

			w := httptest.NewRecorder()

			// Execute PHP script
			php.ExecutePHP("/form_processor.php", vfs, nil, w, req)

			// Get response
			resp := w.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			bodyStr := string(body)

			// Log full response
			t.Logf("Response: %s", bodyStr)

			// Check for PHP errors
			AssertNoPHPErrors(t, bodyStr)

			// Check for expected output
			for _, expected := range tc.expectedOut {
				if !strings.Contains(bodyStr, expected) {
					t.Errorf("Expected to find '%s' in response", expected)
				}
			}
		})
	}
}
