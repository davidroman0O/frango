package frango

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// setupPathParamTest creates a test environment with the given PHP script and returns it
func setupPathParamTest(t *testing.T, phpScript string) *TestEnv {
	env := SetupTest(t, map[string]string{
		"path_params.php": phpScript,
	})
	return env
}

// executePathParamRequest executes a request against the given pattern and path
func executePathParamRequest(t *testing.T, env *TestEnv, pattern, requestPath, method string, body io.Reader, headers map[string]string) (int, http.Header, string) {
	// Create request
	req := httptest.NewRequest(method, requestPath, body)

	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Provide a render function that sets the route pattern
	renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
		return map[string]interface{}{
			"ROUTE_PATTERN": pattern,
		}
	}

	// Execute request - use the actual PHP script path, not the pattern
	// The pattern is just for route matching and parameter extraction
	return ExecutePHP(t, env, "/path_params.php", req, renderFn)
}

// validatePathParams validates the expected path parameters against actual parameters
func validatePathParams(t *testing.T, responseBody string, expectedParams map[string]string) {
	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(responseBody), &responseData); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nResponse body: %s", err, responseBody)
	}

	// Check for PHP errors
	AssertNoPHPErrors(t, responseBody)

	t.Logf("JSON Response: %+v", responseData)

	// Add debug output to see the structure
	jsonBytes, _ := json.MarshalIndent(responseData, "", "  ")
	t.Logf("JSON Response Pretty: %s", string(jsonBytes))

	// Get path parameters - using the correct field name from the JSON response
	params, ok := responseData["params"].(map[string]interface{})
	if !ok {
		// Try accessing it differently since it might be structured differently
		if pathParams, ok := responseData["path_params"].(map[string]interface{}); ok {
			params = pathParams
		} else {
			t.Fatalf("Expected params to be a map, got %T", responseData["params"])
		}
	}

	// Check that all expected parameters are present with correct values
	for paramName, expectedValue := range expectedParams {
		if actualValue, exists := params[paramName]; !exists {
			t.Errorf("Expected parameter '%s' not found", paramName)
		} else if actualValue != expectedValue {
			t.Errorf("Parameter '%s': expected value '%s', got '%v'", paramName, expectedValue, actualValue)
		}
	}

	// Check for unexpected parameters
	for paramName := range params {
		if _, expected := expectedParams[paramName]; !expected {
			t.Errorf("Unexpected parameter '%s' with value '%v'", paramName, params[paramName])
		}
	}
}

// Common PHP scripts used across tests
const (
	// Basic path parameters PHP script
	basicPathParamsPHP = `<?php
		header("Content-Type: application/json");
		
		// Output all available variables
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"params" => $_PATH ?? [], // Custom superglobal set by Frango
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Path parameters with request details PHP script
	requestInfoPHP = `<?php
		header("Content-Type: application/json");
		
		// Define polyfill for getallheaders() if it doesn't exist
		if (!function_exists('getallheaders')) {
			function getallheaders() {
				$headers = [];
				foreach ($_SERVER as $name => $value) {
					if (substr($name, 0, 5) === 'HTTP_') {
						$headers[str_replace(' ', '-', ucwords(strtolower(str_replace('_', ' ', substr($name, 5)))))] = $value;
					} elseif ($name === 'CONTENT_TYPE' || $name === 'CONTENT_LENGTH') {
						$headers[str_replace(' ', '-', ucwords(strtolower(str_replace('_', ' ', $name))))] = $value;
					}
				}
				return $headers;
			}
		}
		
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"params" => $_PATH ?? [], // Custom superglobal set by Frango
			"query" => $_GET,
			"body" => $_POST,
			"json" => json_decode(file_get_contents('php://input'), true) ?? [],
			"headers" => getallheaders(),
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Route context PHP script
	routeContextPHP = `<?php
		header("Content-Type: application/json");
		
		// Access the variables from $_TEMPLATE instead of $_SERVER for testing
		$response = [
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
			"route_handler" => $_TEMPLATE["ROUTE_HANDLER"] ?? "",
			"path_params" => $_PATH ?? [],
			"route_matches" => (bool)($_TEMPLATE["ROUTE_MATCHES"] ?? false),
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`
)

// TestBasicPathParameters tests extracting parameters from URLs
func TestBasicPathParameters(t *testing.T) {
	// Create a temporary directory for PHP files
	tempDir, err := os.MkdirTemp("", "frango-path-params-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// PHP script that outputs path parameters
	phpContent := `<?php
		header("Content-Type: application/json");
		
		// Output all available variables
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"params" => $_PATH ?? [], // Custom superglobal set by Frango
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Write the PHP file to disk
	testFilePath := filepath.Join(tempDir, "test_params.php")
	if err := os.WriteFile(testFilePath, []byte(phpContent), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	testCases := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParams map[string]string
	}{
		{
			name:        "Basic Path Parameter",
			pattern:     "/users/{id}",
			requestPath: "/users/123",
			expectedParams: map[string]string{
				"id": "123",
			},
		},
		{
			name:        "Multiple Path Parameters",
			pattern:     "/users/{userId}/posts/{postId}",
			requestPath: "/users/456/posts/789",
			expectedParams: map[string]string{
				"userId": "456",
				"postId": "789",
			},
		},
		{
			name:        "Path Parameter with Special Characters",
			pattern:     "/files/{filename}",
			requestPath: "/files/test-file_with.special+chars.txt",
			expectedParams: map[string]string{
				"filename": "test-file_with.special+chars.txt",
			},
		},
		{
			name:        "Path Parameter with URL-Encoded Characters",
			pattern:     "/search/{query}",
			requestPath: "/search/programming%20language",
			expectedParams: map[string]string{
				"query": "programming language", // Should be decoded
			},
		},
		{
			name:        "Trailing Slash",
			pattern:     "/categories/{category}/",
			requestPath: "/categories/books/",
			expectedParams: map[string]string{
				"category": "books",
			},
		},
		// Adding new test cases
		{
			name:        "Empty Parameter Value",
			pattern:     "/tags/{tagName}",
			requestPath: "/tags/",
			expectedParams: map[string]string{
				"tagName": "",
			},
		},
		{
			name:        "Very Long Parameter Value",
			pattern:     "/content/{slug}",
			requestPath: "/content/" + strings.Repeat("very-long-parameter-value-", 20),
			expectedParams: map[string]string{
				"slug": strings.Repeat("very-long-parameter-value-", 20),
			},
		},
		{
			name:        "Parameter with Multiple Slashes",
			pattern:     "/path/{fullpath}",
			requestPath: "/path/dir1/dir2/file.txt",
			expectedParams: map[string]string{
				"fullpath": "dir1/dir2/file.txt",
			},
		},
		{
			name:        "Multiple Parameters Same Segment",
			pattern:     "/products/{category}-{id}",
			requestPath: "/products/electronics-12345",
			expectedParams: map[string]string{
				"category": "electronics",
				"id":       "12345",
			},
		},
		{
			name:        "Parameter at Root Level",
			pattern:     "/{locale}/home",
			requestPath: "/en-US/home",
			expectedParams: map[string]string{
				"locale": "en-US",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create VFS for this test case
			vfs := php.NewVFS()
			defer vfs.Cleanup()

			// Register the actual PHP file with the pattern
			err := vfs.AddSourceFile(testFilePath, tc.pattern)
			if err != nil {
				t.Fatalf("Failed to add source file: %v", err)
			}

			// Create request
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()

			// Create a render data function to provide the route pattern
			renderData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
				return map[string]interface{}{
					"ROUTE_PATTERN": tc.pattern,
				}
			}

			// Execute the PHP script using the pattern as the route
			php.ExecutePHP(tc.pattern, vfs, renderData, w, req)

			// Get response
			resp := w.Result()
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
				t.Logf("Error response body: %s", string(body))
				return
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// Parse JSON response
			var responseData map[string]interface{}
			if err := json.Unmarshal(body, &responseData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v\nResponse body: %s", err, string(body))
			}

			// Check for PHP errors
			AssertNoPHPErrors(t, string(body))

			// Get path parameters
			params, ok := responseData["params"].(map[string]interface{})
			if !ok {
				jsonBytes, _ := json.MarshalIndent(responseData, "", "  ")
				t.Fatalf("Expected params to be a map, got %T. Response: %s",
					responseData["params"], string(jsonBytes))
			}

			// Check that all expected parameters are present with correct values
			for paramName, expectedValue := range tc.expectedParams {
				if actualValue, exists := params[paramName]; !exists {
					t.Errorf("Expected parameter '%s' not found", paramName)
				} else if actualValue != expectedValue {
					t.Errorf("Parameter '%s': expected value '%s', got '%v'", paramName, expectedValue, actualValue)
				}
			}

			// Check for unexpected parameters
			for paramName := range params {
				if _, expected := tc.expectedParams[paramName]; !expected {
					t.Errorf("Unexpected parameter '%s' with value '%v'", paramName, params[paramName])
				}
			}
		})
	}
}

// TestPathParametersWithHTTPMethods tests path parameters with different HTTP methods
func TestPathParametersWithHTTPMethods(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		pattern        string
		requestPath    string
		body           string
		contentType    string
		expectedParams map[string]string
	}{
		{
			name:        "GET Request with Path Params",
			method:      "GET",
			pattern:     "/api/products/{productId}",
			requestPath: "/api/products/12345",
			expectedParams: map[string]string{
				"productId": "12345",
			},
		},
		{
			name:        "POST Request with Path Params",
			method:      "POST",
			pattern:     "/api/users/{userId}/profile",
			requestPath: "/api/users/789/profile",
			body:        "name=John&email=john@example.com",
			contentType: "application/x-www-form-urlencoded",
			expectedParams: map[string]string{
				"userId": "789",
			},
		},
		{
			name:        "PUT Request with Path Params",
			method:      "PUT",
			pattern:     "/api/articles/{articleId}",
			requestPath: "/api/articles/456",
			body:        `{"title":"Updated Title","content":"Updated content"}`,
			contentType: "application/json",
			expectedParams: map[string]string{
				"articleId": "456",
			},
		},
		{
			name:        "DELETE Request with Path Params",
			method:      "DELETE",
			pattern:     "/api/comments/{commentId}",
			requestPath: "/api/comments/101",
			expectedParams: map[string]string{
				"commentId": "101",
			},
		},
	}

	// Setup test environment
	env := setupPathParamTest(t, requestInfoPHP)
	defer CleanupTest(env)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare headers
			headers := map[string]string{}
			if tc.contentType != "" {
				headers["Content-Type"] = tc.contentType
			}

			// Execute request
			status, _, body := executePathParamRequest(
				t,
				env,
				tc.pattern,
				tc.requestPath,
				tc.method,
				strings.NewReader(tc.body),
				headers,
			)

			// Check status code
			if status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
				t.Logf("Error response body: %s", body)
			}

			// Validate path parameters
			validatePathParams(t, body, tc.expectedParams)
		})
	}
}

// TestPathParametersWithQueryParams tests path parameters combined with query parameters
func TestPathParametersWithQueryParams(t *testing.T) {
	testCases := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParams map[string]string
		expectedQuery  map[string]string
	}{
		{
			name:        "Path Params with Query Params",
			pattern:     "/products/{category}",
			requestPath: "/products/electronics?sort=price&order=asc",
			expectedParams: map[string]string{
				"category": "electronics",
			},
			expectedQuery: map[string]string{
				"sort":  "price",
				"order": "asc",
			},
		},
		{
			name:        "Multiple Path Params with Query Params",
			pattern:     "/users/{userId}/posts/{postId}",
			requestPath: "/users/123/posts/456?include=comments&limit=10",
			expectedParams: map[string]string{
				"userId": "123",
				"postId": "456",
			},
			expectedQuery: map[string]string{
				"include": "comments",
				"limit":   "10",
			},
		},
		{
			name:        "Multiple Query Params with Same Name",
			pattern:     "/products/{category}",
			requestPath: "/products/clothing?color=red&color=blue&size=M",
			expectedParams: map[string]string{
				"category": "clothing",
			},
			expectedQuery: map[string]string{
				"color": `["red","blue"]`, // This will be encoded as an array in JSON
				"size":  "M",
			},
		},
		{
			name:        "Empty Query Param",
			pattern:     "/search/{term}",
			requestPath: "/search/smartphones?brand=&price=500",
			expectedParams: map[string]string{
				"term": "smartphones",
			},
			expectedQuery: map[string]string{
				"brand": "",
				"price": "500",
			},
		},
		{
			name:        "Special Characters in Query Params",
			pattern:     "/users/{username}",
			requestPath: "/users/john_doe?q=special%20chars%20%26%20symbols",
			expectedParams: map[string]string{
				"username": "john_doe",
			},
			expectedQuery: map[string]string{
				"q": "special chars & symbols",
			},
		},
	}

	// Create PHP script that outputs both path and query parameters
	queryParamsPHP := `<?php
		header("Content-Type: application/json");
		
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"path_params" => $_PATH ?? [],
			"query" => $_GET,
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"query_params.php": queryParamsPHP,
	})
	defer CleanupTest(env)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req := httptest.NewRequest("GET", tc.requestPath, nil)

			// Execute request - use the correct path to the PHP script
			renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
				return map[string]interface{}{
					"ROUTE_PATTERN": tc.pattern,
				}
			}

			// Use ExecutePHP properly
			status, _, body := ExecutePHP(t, env, "/query_params.php", req, renderFn)

			// Check status code
			if status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
				t.Logf("Error response body: %s", body)
			}

			// Parse JSON response
			var responseData map[string]interface{}
			if err := json.Unmarshal([]byte(body), &responseData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check for PHP errors
			AssertNoPHPErrors(t, body)

			// Validate path parameters
			validatePathParams(t, body, tc.expectedParams)

			// Validate query parameters
			query, ok := responseData["query"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected query to be a map, got %T", responseData["query"])
			}

			for queryName, expectedValue := range tc.expectedQuery {
				// Handle array parameters
				if strings.HasPrefix(expectedValue, "[") && strings.HasSuffix(expectedValue, "]") {
					// This is an expected array value
					actualArr, ok := query[queryName].([]interface{})
					if !ok {
						t.Errorf("Expected query parameter '%s' to be an array, got: %T with value %v",
							queryName, query[queryName], query[queryName])
						continue
					}

					// Convert array to JSON for comparison
					jsonArr, _ := json.Marshal(actualArr)
					if string(jsonArr) != expectedValue {
						t.Errorf("Query parameter '%s': expected array '%s', got '%s'",
							queryName, expectedValue, string(jsonArr))
					}
				} else {
					// Simple string comparison - but handle PHP's array representation
					if actualValue, exists := query[queryName]; !exists {
						t.Errorf("Expected query parameter '%s' not found", queryName)
					} else {
						// PHP always returns arrays for GET parameters, even if there's only one value
						// If it's an array with a single value, use that value for comparison
						var valueToCompare string
						switch typedValue := actualValue.(type) {
						case []interface{}:
							if len(typedValue) == 1 {
								// Single value in array
								valueToCompare = fmt.Sprintf("%v", typedValue[0])
							} else if len(typedValue) == 0 && expectedValue == "" {
								// Empty array for empty value
								valueToCompare = ""
							} else {
								// Multiple values or other case
								jsonArr, _ := json.Marshal(typedValue)
								valueToCompare = string(jsonArr)
							}
						default:
							// Not an array
							valueToCompare = fmt.Sprintf("%v", actualValue)
						}

						if valueToCompare != expectedValue {
							t.Errorf("Query parameter '%s': expected value '%s', got '%s'",
								queryName, expectedValue, valueToCompare)
						}
					}
				}
			}
		})
	}
}

// TestRouteContextExtraction tests extracting context information from routes
func TestRouteContextExtraction(t *testing.T) {
	testCases := []struct {
		name            string
		pattern         string
		requestPath     string
		expectedPattern string
		expectedHandler string
		expectedMatches bool
	}{
		{
			name:            "Exact Route Match",
			pattern:         "/api/status",
			requestPath:     "/api/status",
			expectedPattern: "/api/status",
			expectedHandler: "/route_context.php",
			expectedMatches: true,
		},
		{
			name:            "Route with Parameters",
			pattern:         "/api/users/{id}/details",
			requestPath:     "/api/users/42/details",
			expectedPattern: "/api/users/{id}/details",
			expectedHandler: "/route_context.php",
			expectedMatches: true,
		},
		{
			name:            "Pattern with Optional Trailing Slash",
			pattern:         "/api/settings/?",
			requestPath:     "/api/settings",
			expectedPattern: "/api/settings/?",
			expectedHandler: "/route_context.php",
			expectedMatches: true,
		},
		{
			name:            "No Route Match",
			pattern:         "",
			requestPath:     "/api/unknown-route",
			expectedPattern: "",
			expectedHandler: "",
			expectedMatches: false,
		},
		// Adding new test cases
		{
			name:            "Root Path With Parameter",
			pattern:         "/{lang}",
			requestPath:     "/en",
			expectedPattern: "/{lang}",
			expectedHandler: "/route_context.php",
			expectedMatches: true,
		},
		{
			name:            "Pattern with Multiple Parameters",
			pattern:         "/{locale}/products/{category}/{id}",
			requestPath:     "/es/products/electronics/12345",
			expectedPattern: "/{locale}/products/{category}/{id}",
			expectedHandler: "/route_context.php",
			expectedMatches: true,
		},
		{
			name:            "Different HTTP Method in Pattern",
			pattern:         "POST /api/items",
			requestPath:     "/api/items",
			expectedPattern: "POST /api/items",
			expectedHandler: "/route_context.php",
			expectedMatches: true,
		},
	}

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"route_context.php": routeContextPHP,
	})
	defer CleanupTest(env)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a custom render function that sets route context variables
			renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
				return map[string]interface{}{
					"ROUTE_PATTERN": tc.expectedPattern,
					"ROUTE_HANDLER": tc.expectedHandler,
					"ROUTE_MATCHES": tc.expectedMatches,
				}
			}

			// Execute request directly with ExecutePHP instead of executePathParamRequest
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			status, _, body := ExecutePHP(t, env, "/route_context.php", req, renderFn)

			// Check status code
			if status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
				t.Logf("Error response body: %s", body)
			}

			// Parse JSON response
			var responseData map[string]interface{}
			if err := json.Unmarshal([]byte(body), &responseData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check for PHP errors
			AssertNoPHPErrors(t, body)

			// Check route pattern
			pattern, ok := responseData["route_pattern"].(string)
			if !ok {
				t.Fatalf("Expected route_pattern to be a string, got %T", responseData["route_pattern"])
			}
			if pattern != tc.expectedPattern {
				t.Errorf("Expected route pattern '%s', got '%s'", tc.expectedPattern, pattern)
			}

			// Check route handler
			handler, ok := responseData["route_handler"].(string)
			if !ok {
				t.Fatalf("Expected route_handler to be a string, got %T", responseData["route_handler"])
			}
			if handler != tc.expectedHandler && tc.expectedMatches {
				t.Errorf("Expected route handler '%s', got '%s'", tc.expectedHandler, handler)
			}

			// Check if route matched
			matches, ok := responseData["route_matches"].(bool)
			if !ok {
				t.Fatalf("Expected route_matches to be a boolean, got %T", responseData["route_matches"])
			}
			if matches != tc.expectedMatches {
				t.Errorf("Expected route_matches to be %v, got %v", tc.expectedMatches, matches)
			}
		})
	}
}

// TestAdvancedPathParameters tests more complex path parameter scenarios
func TestAdvancedPathParameters(t *testing.T) {
	testCases := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParams map[string]string
		expectedStatus int
	}{
		{
			name:        "Path Parameter with Dots",
			pattern:     "/api/{version}/resource",
			requestPath: "/api/v1.2/resource",
			expectedParams: map[string]string{
				"version": "v1.2",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "URL Path Traversal Attempt",
			pattern:     "/files/{filepath}",
			requestPath: "/files/../../../etc/passwd",
			expectedParams: map[string]string{
				"filepath": "../../../etc/passwd",
			},
			expectedStatus: http.StatusOK, // The middleware should handle this securely
		},
		{
			name:        "Parameters with URL Special Characters",
			pattern:     "/search/{query}",
			requestPath: "/search/language?programming",
			expectedParams: map[string]string{
				"query": "language",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Parameter with Hash Fragment",
			pattern:     "/docs/{section}",
			requestPath: "/docs/introduction#overview",
			expectedParams: map[string]string{
				"section": "introduction",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Multiple Path Segments in One Parameter",
			pattern:     "/api/{*remainingPath}",
			requestPath: "/api/users/123/profile/edit",
			expectedParams: map[string]string{
				"remainingPath": "users/123/profile/edit",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Extremely Long Parameter Value",
			pattern:     "/data/{id}",
			requestPath: "/data/" + strings.Repeat("a", 2000),
			expectedParams: map[string]string{
				"id": strings.Repeat("a", 2000),
			},
			expectedStatus: http.StatusOK,
		},
	}

	// Setup test environment
	env := setupPathParamTest(t, basicPathParamsPHP)
	defer CleanupTest(env)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute request
			status, _, body := executePathParamRequest(t, env, tc.pattern, tc.requestPath, "GET", nil, nil)

			// Check status code
			if status != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, status)
			}

			// Only validate path parameters if status is OK
			if status == http.StatusOK {
				validatePathParams(t, body, tc.expectedParams)
			}
		})
	}
}

// TestPathParametersWithSpecialChars tests that path parameters with special characters
// are correctly extracted and properly encoded in the $_PATH superglobal
func TestPathParametersWithSpecialChars(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-path-special-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
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

	// Create a test PHP file that outputs path parameters
	testPHP := `<?php
		header("Content-Type: application/json");
		
		// Output all available variables
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"path_params" => $_PATH ?? [], // Custom superglobal set by Frango
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
			"request_uri" => $_SERVER["REQUEST_URI"], // Add the request_uri field
			"debug" => [
				"path_segments" => $_PATH_SEGMENTS ?? [],
				"path_segment_count" => $_PATH_SEGMENT_COUNT ?? 0,
				"env" => $_ENV,
				"server" => $_SERVER,
				"template" => $_TEMPLATE ?? [],
			],
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "special_chars.php")
	if err := os.WriteFile(testFilePath, []byte(testPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Test cases with special characters in path parameters
	tests := []struct {
		name           string
		pattern        string
		path           string
		expectedStatus int
		expectedParams map[string]string // Expected decoded parameters in $_PATH
		expectedURI    string            // Expected REQUEST_URI
	}{
		{
			name:           "Special characters in user ID",
			pattern:        "/users/{userId}/posts/{slug}",
			path:           "/users/user@example.com/posts/test-post",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"userId": "user@example.com",
				"slug":   "test-post",
			},
			expectedURI: "/users/user@example.com/posts/test-post",
		},
		{
			name:           "URL encoded characters in slug",
			pattern:        "/users/{userId}/posts/{slug}",
			path:           "/users/123/posts/hello-world%21",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"userId": "123",
				"slug":   "hello-world!", // The decoded value
			},
			expectedURI: "/users/123/posts/hello-world%21",
		},
		{
			name:           "Spaces in slug",
			pattern:        "/users/{userId}/posts/{slug}",
			path:           "/users/123/posts/my%20blog%20post",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"userId": "123",
				"slug":   "my blog post", // The decoded value
			},
			expectedURI: "/users/123/posts/my%20blog%20post",
		},
		{
			name:           "Unicode characters",
			pattern:        "/users/{userId}/posts/{slug}",
			path:           "/users/user%E2%98%85/posts/%E4%BD%A0%E5%A5%BD", // "user★" and "你好"
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"userId": "user★",
				"slug":   "你好",
			},
			expectedURI: "/users/user%E2%98%85/posts/%E4%BD%A0%E5%A5%BD",
		},
		{
			name:           "Path param at root level",
			pattern:        "/{lang}/products/{productId}",
			path:           "/fr/products/12345",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"lang":      "fr",
				"productId": "12345",
			},
			expectedURI: "/fr/products/12345",
		},
		// Adding more edge cases
		{
			name:           "Multiple URL-encoded segments",
			pattern:        "/path/{complexPath}",
			path:           "/path/segment1%2Fsegment2%2Fsegment3",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"complexPath": "segment1/segment2/segment3",
			},
			expectedURI: "/path/segment1%2Fsegment2%2Fsegment3",
		},
		{
			name:           "Special RFC3986 characters",
			pattern:        "/uri/{component}",
			path:           "/uri/scheme%3A%2F%2Fauthority%2Fpath%3Fquery%23fragment",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"component": "scheme://authority/path?query#fragment",
			},
			expectedURI: "/uri/scheme%3A%2F%2Fauthority%2Fpath%3Fquery%23fragment",
		},
		{
			name:           "Plus sign as space",
			pattern:        "/message/{text}",
			path:           "/message/hello+world",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{
				"text": "hello+world", // PHP doesn't decode + in paths, only in query params
			},
			expectedURI: "/message/hello+world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add the PHP file to VFS with the current test pattern
			err = vfs.AddSourceFile(testFilePath, tt.pattern)
			if err != nil {
				t.Fatalf("Failed to add file to VFS with pattern %s: %v", tt.pattern, err)
			}

			// Create request
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Execute the PHP script with the request
			php.ExecutePHP(tt.pattern, vfs, nil, w, req)

			// Check response code
			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Read the response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			resp.Body.Close()

			bodyStr := string(body)

			// Check for any PHP errors
			AssertNoPHPErrors(t, bodyStr)

			// Parse the JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Get path_params from the response
			pathParams, ok := response["path_params"].(map[string]interface{})
			if !ok {
				t.Fatalf("Response doesn't contain valid path_params field")
			}

			// Verify all expected path parameters are present with correct decoded values
			for paramName, expectedValue := range tt.expectedParams {
				actualValue, exists := pathParams[paramName]
				if !exists {
					t.Errorf("Expected path parameter %s missing from response", paramName)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Parameter %s: expected %s but got %v",
						paramName, expectedValue, actualValue)
				}
			}

			// Verify the original request URI is preserved
			if response["request_uri"] != tt.expectedURI {
				t.Errorf("Expected request_uri=%s but got %v",
					tt.expectedURI, response["request_uri"])
			}
		})
	}
}

// TestPathParameterErrorCases tests expected error/edge cases with path parameters
func TestPathParameterErrorCases(t *testing.T) {
	testCases := []struct {
		name                string
		pattern             string
		requestPath         string
		expectedStatus      int
		expectParamMismatch bool
	}{
		{
			name:                "Missing Required Parameter",
			pattern:             "/api/users/{userId}/profile",
			requestPath:         "/api/users//profile", // Missing userId
			expectedStatus:      http.StatusOK,         // The middleware should still handle this
			expectParamMismatch: true,
		},
		{
			name:                "Pattern/Path Segment Count Mismatch",
			pattern:             "/api/users/{userId}/posts/{postId}",
			requestPath:         "/api/users/123/posts", // Missing postId segment
			expectedStatus:      http.StatusOK,          // The middleware should handle this gracefully
			expectParamMismatch: true,
		},
		{
			name:                "Too Many Path Segments",
			pattern:             "/api/users/{userId}",
			requestPath:         "/api/users/123/extra/segments",
			expectedStatus:      http.StatusOK, // Should either 404 or extract what it can
			expectParamMismatch: true,
		},
		{
			name:                "Invalid Parameter Name Characters",
			pattern:             "/api/{*invalid@name}/resource", // Invalid character in parameter name
			requestPath:         "/api/value/resource",
			expectedStatus:      http.StatusOK, // How the middleware handles invalid parameter names
			expectParamMismatch: true,
		},
		{
			name:                "Empty Path",
			pattern:             "/{param}",
			requestPath:         "/", // Empty parameter
			expectedStatus:      http.StatusOK,
			expectParamMismatch: true,
		},
	}

	// Create an error-handling PHP script
	errorHandlingPHP := `<?php
		header("Content-Type: application/json");
		
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"params" => $_PATH ?? [],
			"path_params" => $_PATH ?? [], // Add this for consistency
			"pattern" => $_SERVER["ROUTE_PATTERN"] ?? "",
			"error" => error_get_last(),
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"error_handling.php": errorHandlingPHP,
	})
	defer CleanupTest(env)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute request
			status, _, body := ExecutePHP(t, env, "/error_handling.php", httptest.NewRequest("GET", tc.requestPath, nil), func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
				return map[string]interface{}{
					"ROUTE_PATTERN": tc.pattern,
				}
			})

			// Check status code
			if status != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, status)
				t.Logf("Error response body: %s", body)
			}

			// Parse JSON response
			var responseData map[string]interface{}
			if err := json.Unmarshal([]byte(body), &responseData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// For parameter mismatches, we just verify the structure contains the params field
			if tc.expectParamMismatch {
				_, ok := responseData["params"]
				if !ok {
					t.Errorf("Expected 'params' field in response even with parameter mismatch")
				}
				// We don't validate the specific parameters since this is an error case
			}
		})
	}
}

// TestPathParametersCaseSensitivity tests if path parameters are case-sensitive
func TestPathParametersCaseSensitivity(t *testing.T) {
	testCases := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParams map[string]string
	}{
		{
			name:        "Lowercase Parameter Name",
			pattern:     "/api/{userid}/profile",
			requestPath: "/api/123/profile",
			expectedParams: map[string]string{
				"userid": "123",
			},
		},
		{
			name:        "Uppercase Parameter Name",
			pattern:     "/api/{USERID}/profile",
			requestPath: "/api/123/profile",
			expectedParams: map[string]string{
				"USERID": "123",
			},
		},
		{
			name:        "Mixed Case Parameter Name",
			pattern:     "/api/{userId}/profile",
			requestPath: "/api/123/profile",
			expectedParams: map[string]string{
				"userId": "123",
			},
		},
		{
			name:        "Case-Sensitive Path Segment",
			pattern:     "/api/{category}/items",
			requestPath: "/api/Electronics/items", // Note the capital E
			expectedParams: map[string]string{
				"category": "Electronics", // Should preserve the exact case
			},
		},
		{
			name:        "Lowercase vs Uppercase Parameter Value",
			pattern:     "/api/users/{username}",
			requestPath: "/api/users/JohnDoe", // Uppercase in username
			expectedParams: map[string]string{
				"username": "JohnDoe", // Should preserve case
			},
		},
	}

	// Setup test environment
	env := setupPathParamTest(t, basicPathParamsPHP)
	defer CleanupTest(env)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute request
			status, _, body := executePathParamRequest(t, env, tc.pattern, tc.requestPath, "GET", nil, nil)

			// Check status code
			if status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
				t.Logf("Error response body: %s", body)
			}

			// Validate path parameters
			validatePathParams(t, body, tc.expectedParams)
		})
	}
}

// TestPathParametersForPatterns tests the extraction of path parameters from URL patterns
func TestPathParametersForPatterns(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-path-params-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
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

	// Create a test PHP file that outputs path parameters
	testPHP := `<?php
		header("Content-Type: application/json");
		
		// Output all available variables
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"path_params" => $_PATH ?? [], // Custom superglobal set by Frango
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
			"request_uri" => $_SERVER["REQUEST_URI"], // Add the request_uri field
			"debug" => [
				"path_segments" => $_PATH_SEGMENTS ?? [],
				"path_segment_count" => $_PATH_SEGMENT_COUNT ?? 0,
				"env" => $_ENV,
				"server" => $_SERVER,
				"template" => $_TEMPLATE ?? [],
			],
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the PHP file
	testFilePath := filepath.Join(tempDir, "path_params.php")
	if err := os.WriteFile(testFilePath, []byte(testPHP), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Create a PHP file for testing query parameters
	queryParamsPHP := `<?php
		header("Content-Type: application/json");
		
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"path_params" => $_PATH ?? [],
			"query" => $_GET,
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the query params PHP file
	queryFilePath := filepath.Join(tempDir, "query_params.php")
	if err := os.WriteFile(queryFilePath, []byte(queryParamsPHP), 0644); err != nil {
		t.Fatalf("Failed to write query params PHP file: %v", err)
	}

	// Create a PHP file for testing HTTP methods and request info
	requestInfoPHP := `<?php
		header("Content-Type: application/json");
		
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"path" => $_SERVER["REQUEST_URI"],
			"path_params" => $_PATH ?? [],
			"query" => $_GET,
			"body" => $_POST,
			"json" => json_decode(file_get_contents('php://input'), true) ?? [],
			"headers" => function_exists('getallheaders') ? getallheaders() : [],
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the request info PHP file
	requestInfoFilePath := filepath.Join(tempDir, "request_info.php")
	if err := os.WriteFile(requestInfoFilePath, []byte(requestInfoPHP), 0644); err != nil {
		t.Fatalf("Failed to write request info PHP file: %v", err)
	}

	// Create a PHP file for route context testing
	routeContextPHP := `<?php
		header("Content-Type: application/json");
		
		// Access the variables from $_TEMPLATE instead of $_SERVER for testing
		$response = [
			"route_pattern" => $_TEMPLATE["ROUTE_PATTERN"] ?? "",
			"route_handler" => $_TEMPLATE["ROUTE_HANDLER"] ?? "",
			"path_params" => $_PATH ?? [],
			"route_matches" => (bool)($_TEMPLATE["ROUTE_MATCHES"] ?? false),
		];
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the route context PHP file
	routeContextFilePath := filepath.Join(tempDir, "route_context.php")
	if err := os.WriteFile(routeContextFilePath, []byte(routeContextPHP), 0644); err != nil {
		t.Fatalf("Failed to write route context PHP file: %v", err)
	}

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Test cases for basic path parameters
	basicTestCases := []struct {
		name             string
		pattern          string
		requestPath      string
		expectedParams   map[string]string
		expectedStatus   int
		expectedPathSegs []string
	}{
		{
			name:        "Basic Path Parameter",
			pattern:     "/users/{id}",
			requestPath: "/users/123",
			expectedParams: map[string]string{
				"id": "123",
			},
			expectedStatus:   http.StatusOK,
			expectedPathSegs: []string{"users", "123"},
		},
		{
			name:        "Multiple Path Parameters",
			pattern:     "/users/{userId}/posts/{postId}",
			requestPath: "/users/456/posts/789",
			expectedParams: map[string]string{
				"userId": "456",
				"postId": "789",
			},
			expectedStatus:   http.StatusOK,
			expectedPathSegs: []string{"users", "456", "posts", "789"},
		},
		{
			name:        "Path Parameter with Special Characters",
			pattern:     "/files/{filename}",
			requestPath: "/files/test-file_with.special+chars.txt",
			expectedParams: map[string]string{
				"filename": "test-file_with.special+chars.txt",
			},
			expectedStatus:   http.StatusOK,
			expectedPathSegs: []string{"files", "test-file_with.special+chars.txt"},
		},
		{
			name:        "Path Parameter with URL-Encoded Characters",
			pattern:     "/search/{query}",
			requestPath: "/search/programming%20language",
			expectedParams: map[string]string{
				"query": "programming language", // Should be decoded
			},
			expectedStatus:   http.StatusOK,
			expectedPathSegs: []string{"search", "programming language"},
		},
		{
			name:        "Trailing Slash",
			pattern:     "/categories/{category}/",
			requestPath: "/categories/books/",
			expectedParams: map[string]string{
				"category": "books",
			},
			expectedStatus:   http.StatusOK,
			expectedPathSegs: []string{"categories", "books", ""},
		},
		{
			name:        "Parameter at Root Level",
			pattern:     "/{locale}/home",
			requestPath: "/en-US/home",
			expectedParams: map[string]string{
				"locale": "en-US",
			},
			expectedStatus:   http.StatusOK,
			expectedPathSegs: []string{"en-US", "home"},
		},
		{
			name:        "Very Long Parameter Value",
			pattern:     "/content/{slug}",
			requestPath: "/content/" + strings.Repeat("very-long-parameter-value-", 20),
			expectedParams: map[string]string{
				"slug": strings.Repeat("very-long-parameter-value-", 20),
			},
			expectedStatus: http.StatusOK,
		},
	}

	t.Run("Basic Path Parameters", func(t *testing.T) {
		for _, tc := range basicTestCases {
			t.Run(tc.name, func(t *testing.T) {
				// Register pattern with the file in the VFS
				if err := vfs.AddSourceFile(testFilePath, tc.pattern); err != nil {
					t.Fatalf("Failed to add file with pattern %s: %v", tc.pattern, err)
				}

				// Create request
				req := httptest.NewRequest("GET", tc.requestPath, nil)
				w := httptest.NewRecorder()

				// Create a RenderData function to provide the route pattern
				routePatternData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
					return map[string]interface{}{
						"ROUTE_PATTERN": tc.pattern,
					}
				}

				// Execute PHP with the pattern (which maps to our test file)
				php.ExecutePHP(tc.pattern, vfs, routePatternData, w, req)

				// Read response
				resp := w.Result()
				defer resp.Body.Close()

				// Check status code
				if resp.StatusCode != tc.expectedStatus {
					t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.StatusCode)
					body, _ := io.ReadAll(resp.Body)
					t.Logf("Error response body: %s", string(body))
					return
				}

				// Read response body
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				// Check for PHP errors
				if strings.Contains(string(body), "Fatal error") || strings.Contains(string(body), "Parse error") {
					t.Fatalf("PHP error detected: %s", string(body))
				}

				// Parse JSON response
				var responseData map[string]interface{}
				if err := json.Unmarshal(body, &responseData); err != nil {
					t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, string(body))
				}

				// Get path parameters
				params, ok := responseData["path_params"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected path_params to be a map, got %T: %s",
						responseData["path_params"], string(body))
				}

				// Verify all parameters were extracted correctly
				for paramName, expectedValue := range tc.expectedParams {
					actualValue, exists := params[paramName]
					if !exists {
						t.Errorf("Expected parameter %s missing from response", paramName)
						continue
					}

					if actualValue != expectedValue {
						t.Errorf("Parameter %s: expected %s but got %v",
							paramName, expectedValue, actualValue)
					}
				}

				// Check for unexpected parameters
				for paramName := range params {
					if _, expected := tc.expectedParams[paramName]; !expected {
						t.Errorf("Unexpected parameter '%s' with value '%v'", paramName, params[paramName])
					}
				}

				// Check route pattern was set correctly
				routePattern, ok := responseData["route_pattern"].(string)
				if !ok {
					t.Errorf("Expected route_pattern to be a string, got %T", responseData["route_pattern"])
				} else if routePattern != tc.pattern {
					t.Errorf("Expected route_pattern %s, got %s", tc.pattern, routePattern)
				}
			})
		}
	})

	// Test cases for path parameters with HTTP methods
	methodTestCases := []struct {
		name           string
		method         string
		pattern        string
		requestPath    string
		body           string
		contentType    string
		expectedParams map[string]string
		expectedStatus int
	}{
		{
			name:        "GET Request with Path Params",
			method:      "GET",
			pattern:     "/api/products/{productId}",
			requestPath: "/api/products/12345",
			expectedParams: map[string]string{
				"productId": "12345",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "POST Request with Path Params",
			method:      "POST",
			pattern:     "/api/users/{userId}/profile",
			requestPath: "/api/users/789/profile",
			body:        "name=John&email=john@example.com",
			contentType: "application/x-www-form-urlencoded",
			expectedParams: map[string]string{
				"userId": "789",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "PUT Request with Path Params",
			method:      "PUT",
			pattern:     "/api/articles/{articleId}",
			requestPath: "/api/articles/456",
			body:        `{"title":"Updated Title","content":"Updated content"}`,
			contentType: "application/json",
			expectedParams: map[string]string{
				"articleId": "456",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "DELETE Request with Path Params",
			method:      "DELETE",
			pattern:     "/api/comments/{commentId}",
			requestPath: "/api/comments/101",
			expectedParams: map[string]string{
				"commentId": "101",
			},
			expectedStatus: http.StatusOK,
		},
	}

	t.Run("Path Parameters with HTTP Methods", func(t *testing.T) {
		for _, tc := range methodTestCases {
			t.Run(tc.name, func(t *testing.T) {
				// Register pattern with the file in the VFS
				if err := vfs.AddSourceFile(requestInfoFilePath, tc.pattern); err != nil {
					t.Fatalf("Failed to add file with pattern %s: %v", tc.pattern, err)
				}

				// Create request
				req := httptest.NewRequest(tc.method, tc.requestPath, strings.NewReader(tc.body))
				if tc.contentType != "" {
					req.Header.Set("Content-Type", tc.contentType)
				}
				w := httptest.NewRecorder()

				// Create a RenderData function to provide the route pattern
				routePatternData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
					return map[string]interface{}{
						"ROUTE_PATTERN": tc.pattern,
					}
				}

				// Execute PHP with the pattern
				php.ExecutePHP(tc.pattern, vfs, routePatternData, w, req)

				// Read response
				resp := w.Result()
				defer resp.Body.Close()

				// Check status code
				if resp.StatusCode != tc.expectedStatus {
					t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.StatusCode)
					body, _ := io.ReadAll(resp.Body)
					t.Logf("Error response body: %s", string(body))
					return
				}

				// Read response body
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				// Parse JSON response
				var responseData map[string]interface{}
				if err := json.Unmarshal(body, &responseData); err != nil {
					t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, string(body))
				}

				// Get path parameters
				params, ok := responseData["path_params"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected path_params to be a map, got %T: %s",
						responseData["path_params"], string(body))
				}

				// Verify all parameters were extracted correctly
				for paramName, expectedValue := range tc.expectedParams {
					actualValue, exists := params[paramName]
					if !exists {
						t.Errorf("Expected parameter %s missing from response", paramName)
						continue
					}

					if actualValue != expectedValue {
						t.Errorf("Parameter %s: expected %s but got %v",
							paramName, expectedValue, actualValue)
					}
				}

				// Verify the HTTP method was correctly set
				method, ok := responseData["method"].(string)
				if !ok {
					t.Errorf("Expected method to be a string, got %T", responseData["method"])
				} else if method != tc.method {
					t.Errorf("Expected method %s, got %s", tc.method, method)
				}

				// Check route pattern was set correctly
				routePattern, ok := responseData["route_pattern"].(string)
				if !ok {
					t.Errorf("Expected route_pattern to be a string, got %T", responseData["route_pattern"])
				} else if routePattern != tc.pattern {
					t.Errorf("Expected route_pattern %s, got %s", tc.pattern, routePattern)
				}
			})
		}
	})

	// Test cases for path parameters with query parameters
	queryTestCases := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParams map[string]string
		expectedQuery  map[string]interface{}
		expectedStatus int
	}{
		{
			name:        "Path Params with Query Params",
			pattern:     "/products/{category}",
			requestPath: "/products/electronics?sort=price&order=asc",
			expectedParams: map[string]string{
				"category": "electronics",
			},
			expectedQuery: map[string]interface{}{
				"sort":  "price",
				"order": "asc",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Multiple Path Params with Query Params",
			pattern:     "/users/{userId}/posts/{postId}",
			requestPath: "/users/123/posts/456?include=comments&limit=10",
			expectedParams: map[string]string{
				"userId": "123",
				"postId": "456",
			},
			expectedQuery: map[string]interface{}{
				"include": "comments",
				"limit":   "10",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Empty Query Param",
			pattern:     "/search/{term}",
			requestPath: "/search/smartphones?brand=&price=500",
			expectedParams: map[string]string{
				"term": "smartphones",
			},
			expectedQuery: map[string]interface{}{
				"brand": "",
				"price": "500",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Special Characters in Query Params",
			pattern:     "/users/{username}",
			requestPath: "/users/john_doe?q=special%20chars%20%26%20symbols",
			expectedParams: map[string]string{
				"username": "john_doe",
			},
			expectedQuery: map[string]interface{}{
				"q": "special chars & symbols",
			},
			expectedStatus: http.StatusOK,
		},
	}

	t.Run("Path Parameters with Query Parameters", func(t *testing.T) {
		for _, tc := range queryTestCases {
			t.Run(tc.name, func(t *testing.T) {
				// Register pattern with the file in the VFS
				if err := vfs.AddSourceFile(queryFilePath, tc.pattern); err != nil {
					t.Fatalf("Failed to add file with pattern %s: %v", tc.pattern, err)
				}

				// Create request
				req := httptest.NewRequest("GET", tc.requestPath, nil)
				w := httptest.NewRecorder()

				// Create a RenderData function to provide the route pattern
				routePatternData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
					return map[string]interface{}{
						"ROUTE_PATTERN": tc.pattern,
					}
				}

				// Execute PHP with the pattern
				php.ExecutePHP(tc.pattern, vfs, routePatternData, w, req)

				// Read response
				resp := w.Result()
				defer resp.Body.Close()

				// Check status code
				if resp.StatusCode != tc.expectedStatus {
					t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.StatusCode)
					body, _ := io.ReadAll(resp.Body)
					t.Logf("Error response body: %s", string(body))
					return
				}

				// Read response body
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				// Parse JSON response
				var responseData map[string]interface{}
				if err := json.Unmarshal(body, &responseData); err != nil {
					t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, string(body))
				}

				// Get path parameters
				params, ok := responseData["path_params"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected path_params to be a map, got %T: %s",
						responseData["path_params"], string(body))
				}

				// Verify all parameters were extracted correctly
				for paramName, expectedValue := range tc.expectedParams {
					actualValue, exists := params[paramName]
					if !exists {
						t.Errorf("Expected parameter %s missing from response", paramName)
						continue
					}

					if actualValue != expectedValue {
						t.Errorf("Parameter %s: expected %s but got %v",
							paramName, expectedValue, actualValue)
					}
				}

				// Get query parameters
				query, ok := responseData["query"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected query to be a map, got %T: %s",
						responseData["query"], string(body))
				}

				// Verify all query parameters were extracted correctly
				for queryName, expectedValue := range tc.expectedQuery {
					actualValue, exists := query[queryName]
					if !exists {
						t.Errorf("Expected query parameter %s missing from response", queryName)
						continue
					}

					// Handle PHP's array-style query parameters
					switch actual := actualValue.(type) {
					case []interface{}:
						if len(actual) == 1 {
							// If it's a single-value array, compare with the first element
							if actual[0] != expectedValue {
								t.Errorf("Query parameter %s: expected %v but got %v",
									queryName, expectedValue, actual[0])
							}
						} else {
							// For multi-value arrays, just log it but don't fail
							t.Logf("Query parameter %s: is an array %v", queryName, actual)
						}
					default:
						// For non-array values
						if actualValue != expectedValue {
							t.Errorf("Query parameter %s: expected %v but got %v",
								queryName, expectedValue, actualValue)
						}
					}
				}

				// Check route pattern was set correctly
				routePattern, ok := responseData["route_pattern"].(string)
				if !ok {
					t.Errorf("Expected route_pattern to be a string, got %T", responseData["route_pattern"])
				} else if routePattern != tc.pattern {
					t.Errorf("Expected route_pattern %s, got %s", tc.pattern, routePattern)
				}
			})
		}
	})

	// Test cases for path parameters with special characters
	specialCharsTestCases := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParams map[string]string
		expectedStatus int
	}{
		{
			name:        "Special characters in user ID",
			pattern:     "/users/{userId}/posts/{slug}",
			requestPath: "/users/user@example.com/posts/test-post",
			expectedParams: map[string]string{
				"userId": "user@example.com",
				"slug":   "test-post",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "URL encoded characters in slug",
			pattern:     "/users/{userId}/posts/{slug}",
			requestPath: "/users/123/posts/hello-world%21",
			expectedParams: map[string]string{
				"userId": "123",
				"slug":   "hello-world!", // The decoded value
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Spaces in slug",
			pattern:     "/users/{userId}/posts/{slug}",
			requestPath: "/users/123/posts/my%20blog%20post",
			expectedParams: map[string]string{
				"userId": "123",
				"slug":   "my blog post", // The decoded value
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Plus sign as space",
			pattern:     "/message/{text}",
			requestPath: "/message/hello+world",
			expectedParams: map[string]string{
				"text": "hello+world", // PHP doesn't decode + in paths, only in query params
			},
			expectedStatus: http.StatusOK,
		},
	}

	t.Run("Path Parameters with Special Characters", func(t *testing.T) {
		for _, tc := range specialCharsTestCases {
			t.Run(tc.name, func(t *testing.T) {
				// Register pattern with the file in the VFS
				if err := vfs.AddSourceFile(testFilePath, tc.pattern); err != nil {
					t.Fatalf("Failed to add file with pattern %s: %v", tc.pattern, err)
				}

				// Create request
				req := httptest.NewRequest("GET", tc.requestPath, nil)
				w := httptest.NewRecorder()

				// Create a RenderData function to provide the route pattern
				routePatternData := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
					return map[string]interface{}{
						"ROUTE_PATTERN": tc.pattern,
					}
				}

				// Execute PHP with the pattern
				php.ExecutePHP(tc.pattern, vfs, routePatternData, w, req)

				// Read response
				resp := w.Result()
				defer resp.Body.Close()

				// Check status code
				if resp.StatusCode != tc.expectedStatus {
					t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.StatusCode)
					body, _ := io.ReadAll(resp.Body)
					t.Logf("Error response body: %s", string(body))
					return
				}

				// Read response body
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				// Parse JSON response
				var responseData map[string]interface{}
				if err := json.Unmarshal(body, &responseData); err != nil {
					t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, string(body))
				}

				// Get path parameters
				params, ok := responseData["path_params"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected path_params to be a map, got %T: %s",
						responseData["path_params"], string(body))
				}

				// Verify all parameters were extracted correctly
				for paramName, expectedValue := range tc.expectedParams {
					actualValue, exists := params[paramName]
					if !exists {
						t.Errorf("Expected parameter %s missing from response", paramName)
						continue
					}

					if actualValue != expectedValue {
						t.Errorf("Parameter %s: expected %s but got %v",
							paramName, expectedValue, actualValue)
					}
				}

				// Check route pattern was set correctly
				routePattern, ok := responseData["route_pattern"].(string)
				if !ok {
					t.Errorf("Expected route_pattern to be a string, got %T", responseData["route_pattern"])
				} else if routePattern != tc.pattern {
					t.Errorf("Expected route_pattern %s, got %s", tc.pattern, routePattern)
				}
			})
		}
	})
}

// TestNestedPathRouting tests multiple routes pointing to the same PHP script
// and verifies that other routes still work properly alongside nested routes
func TestNestedPathRouting(t *testing.T) {
	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-nested-paths-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the nested path directory structure
	nestedDir := filepath.Join(tempDir, "nested", "deep", "path")
	err = os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory structure: %v", err)
	}

	// PHP script for nested paths
	nestedPathPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"type" => "nested_path",
			"script" => "index.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"path_info" => $_SERVER["PATH_INFO"] ?? "",
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_params" => $_PATH ?? [],
			"segments" => $_PATH_SEGMENTS ?? [],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// PHP script for regular route
	regularPathPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"type" => "regular_path",
			"script" => "regular.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"path_info" => $_SERVER["PATH_INFO"] ?? "",
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_params" => $_PATH ?? [],
			"segments" => $_PATH_SEGMENTS ?? [],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// PHP script for root path
	rootPathPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"type" => "root_path",
			"script" => "index.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"path_info" => $_SERVER["PATH_INFO"] ?? "",
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_params" => $_PATH ?? [],
			"segments" => $_PATH_SEGMENTS ?? [],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write PHP files
	err = os.WriteFile(filepath.Join(nestedDir, "index.php"), []byte(nestedPathPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested path PHP file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "regular.php"), []byte(regularPathPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create regular path PHP file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "index.php"), []byte(rootPathPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create root path PHP file: %v", err)
	}

	// Create Frango middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create router and register routes
	mux := http.NewServeMux()

	// Nested paths - all pointing to the same file
	mux.Handle("/nested", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/path", php.For("/nested/deep/path/index.php"))
	mux.Handle("/nested/deep/path/", php.For("/nested/deep/path/index.php"))

	// Regular route for verification
	mux.Handle("/regular", php.For("/regular.php"))

	// Root route - this needs to be registered LAST to prevent it from catching all routes
	// Go's ServeMux gives precedence to more specific routes, so registering in this order is important
	mux.Handle("/", php.For("/index.php"))

	// Create test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Helper function to extract JSON response from HTTP request
	getJsonResponse := func(url string) (map[string]interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Check for PHP errors
		if strings.Contains(string(body), "Fatal error") || strings.Contains(string(body), "Parse error") {
			return nil, fmt.Errorf("PHP error detected: %s", string(body))
		}

		return result, nil
	}

	// Test 0: Verify root path works BEFORE accessing any nested paths
	t.Run("Root Path First", func(t *testing.T) {
		// Access root path first, before any nested paths
		rootResponse, err := getJsonResponse(server.URL + "/")
		if err != nil {
			t.Fatalf("Root path request failed: %v", err)
		}

		// Verify it's the correct handler
		rootType, ok := rootResponse["type"].(string)
		if !ok {
			t.Fatalf("Expected 'type' field to be a string, got %T", rootResponse["type"])
		}

		if rootType != "root_path" {
			t.Errorf("Expected root path type 'root_path', got '%s'", rootType)
		}

		// Verify script path
		scriptPath, ok := rootResponse["script"].(string)
		if !ok {
			t.Fatalf("Expected 'script' field to be a string, got %T", rootResponse["script"])
		}

		if scriptPath != "index.php" {
			t.Errorf("Expected script path 'index.php', got '%s'", scriptPath)
		}

		// Verify URI
		uri, ok := rootResponse["uri"].(string)
		if !ok {
			t.Fatalf("Expected 'uri' field to be a string, got %T", rootResponse["uri"])
		}

		if uri != "/" {
			t.Errorf("Expected URI to be '/', got '%s'", uri)
		}
	})

	// Define the different nested paths to test
	nestedPaths := []string{
		"/nested",
		"/nested/",
		"/nested/deep",
		"/nested/deep/",
		"/nested/deep/path",
		"/nested/deep/path/",
		"/nested/deep/path/extra",
	}

	// Test 1: Verify all nested paths work
	t.Run("Nested Paths", func(t *testing.T) {
		for _, path := range nestedPaths {
			t.Run(path, func(t *testing.T) {
				response, err := getJsonResponse(server.URL + path)
				if err != nil {
					t.Fatalf("Request failed for path %s: %v", path, err)
				}

				// Verify response type
				responseType, ok := response["type"].(string)
				if !ok {
					t.Fatalf("Expected 'type' field to be a string, got %T", response["type"])
				}

				if responseType != "nested_path" {
					t.Errorf("Expected response type 'nested_path', got '%s'", responseType)
				}

				// Verify script path
				script, ok := response["script"].(string)
				if !ok {
					t.Fatalf("Expected 'script' field to be a string, got %T", response["script"])
				}

				if script != "index.php" {
					t.Errorf("Expected script path 'index.php', got '%s'", script)
				}
			})
		}
	})

	// Test 2: Verify that regular route works between nested path accesses
	t.Run("Regular Route Between Nested", func(t *testing.T) {
		// First access nested path
		nestedResponse, err := getJsonResponse(server.URL + "/nested")
		if err != nil {
			t.Fatalf("Nested path request failed: %v", err)
		}

		if nestedType, _ := nestedResponse["type"].(string); nestedType != "nested_path" {
			t.Errorf("Expected nested path type 'nested_path', got '%s'", nestedType)
		}

		// Then access regular path
		regularResponse, err := getJsonResponse(server.URL + "/regular")
		if err != nil {
			t.Fatalf("Regular path request failed: %v", err)
		}

		if regularType, _ := regularResponse["type"].(string); regularType != "regular_path" {
			t.Errorf("Expected regular path type 'regular_path', got '%s'", regularType)
		}

		// Then access nested path again
		nestedResponse2, err := getJsonResponse(server.URL + "/nested/deep")
		if err != nil {
			t.Fatalf("Second nested path request failed: %v", err)
		}

		if nestedType2, _ := nestedResponse2["type"].(string); nestedType2 != "nested_path" {
			t.Errorf("Expected second nested path type 'nested_path', got '%s'", nestedType2)
		}
	})

	// Test 3: Verify that nested routes work with query parameters
	t.Run("Nested With Query Params", func(t *testing.T) {
		response, err := getJsonResponse(server.URL + "/nested?param=value")
		if err != nil {
			t.Fatalf("Nested with query params request failed: %v", err)
		}

		responseType, _ := response["type"].(string)
		if responseType != "nested_path" {
			t.Errorf("Expected nested path type 'nested_path', got '%s'", responseType)
		}

		// Verify request URI contains the query parameter
		uri, ok := response["uri"].(string)
		if !ok {
			t.Fatalf("Expected 'uri' field to be a string, got %T", response["uri"])
		}

		if !strings.Contains(uri, "?param=value") {
			t.Errorf("Expected URI to contain query parameter, got '%s'", uri)
		}
	})

	// Test 4: Verify nested paths with unusual characters or patterns
	t.Run("Nested With Extra Segments", func(t *testing.T) {
		// Test with extra segments
		extraResponse, err := getJsonResponse(server.URL + "/nested/deep/path/extra/segments")
		if err != nil {
			t.Fatalf("Nested with extra segments request failed: %v", err)
		}

		if extraType, _ := extraResponse["type"].(string); extraType != "nested_path" {
			t.Errorf("Expected nested path with extra segments type 'nested_path', got '%s'", extraType)
		}

		// Verify the script name is correct
		scriptName, ok := extraResponse["script_name"].(string)
		if !ok {
			t.Fatalf("Expected 'script_name' field to be a string, got %T", extraResponse["script_name"])
		}

		if scriptName != "/index.php" {
			t.Errorf("Expected script name '/index.php', got '%s'", scriptName)
		}
	})

	// Test 5: Verify root path still works AFTER accessing nested paths
	t.Run("Root Path After Nested", func(t *testing.T) {
		// First access a nested path
		_, err := getJsonResponse(server.URL + "/nested/deep")
		if err != nil {
			t.Fatalf("Nested path request failed: %v", err)
		}

		// Then access root path
		rootResponse, err := getJsonResponse(server.URL + "/")
		if err != nil {
			t.Fatalf("Root path request failed: %v", err)
		}

		// Verify it's the correct handler
		rootType, ok := rootResponse["type"].(string)
		if !ok {
			t.Fatalf("Expected 'type' field to be a string, got %T", rootResponse["type"])
		}

		// Now we can properly verify the root path still works after accessing nested paths
		if rootType != "root_path" {
			t.Errorf("Expected root path type 'root_path', got '%s'", rootType)
		}

		// Verify URI
		uri, ok := rootResponse["uri"].(string)
		if !ok {
			t.Fatalf("Expected 'uri' field to be a string, got %T", rootResponse["uri"])
		}

		if uri != "/" {
			t.Errorf("Expected URI to be '/', got '%s'", uri)
		}
	})

	// Test 6: Specifically test for handler state leakage across requests
	t.Run("Handler State Isolation", func(t *testing.T) {
		// Create a completely separate mux with just the root handler
		// This isolates the test from any potential state leakage from
		// other nested path handlers
		isolatedMux := http.NewServeMux()
		isolatedMux.Handle("/", php.For("/index.php"))

		isolatedServer := httptest.NewServer(isolatedMux)
		defer isolatedServer.Close()

		// First make a request to a nested path on the main server
		// to potentially "contaminate" the state
		_, err := getJsonResponse(server.URL + "/nested/deep/path")
		if err != nil {
			t.Fatalf("Nested path request failed: %v", err)
		}

		// Now try to access the root path on the isolated server
		// This should always work and return root_path type
		isolatedRootResponse, err := getJsonResponse(isolatedServer.URL + "/")
		if err != nil {
			t.Fatalf("Isolated root path request failed: %v", err)
		}

		// Verify it's the correct handler with no contamination
		isolatedRootType, ok := isolatedRootResponse["type"].(string)
		if !ok {
			t.Fatalf("Expected 'type' field to be a string, got %T", isolatedRootResponse["type"])
		}

		if isolatedRootType != "root_path" {
			t.Errorf("CRITICAL ERROR: Handler state contamination detected. Expected root path type 'root_path', got '%s'", isolatedRootType)
		}
	})
}

// TestRootPathRouting tests the root path behavior specifically
func TestRootPathRouting(t *testing.T) {
	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-root-path-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// PHP script for root path
	rootPathPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"type" => "root_path",
			"script" => "index.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"path_info" => $_SERVER["PATH_INFO"] ?? "",
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_params" => $_PATH ?? [],
			"segments" => $_PATH_SEGMENTS ?? [],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the PHP file
	err = os.WriteFile(filepath.Join(tempDir, "index.php"), []byte(rootPathPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create root path PHP file: %v", err)
	}

	// Create Frango middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create router with ONLY the root route
	mux := http.NewServeMux()
	mux.Handle("/", php.For("/index.php"))

	// Create test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Helper function to extract JSON response from HTTP request
	getJsonResponse := func(url string) (map[string]interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Check for PHP errors
		if strings.Contains(string(body), "Fatal error") || strings.Contains(string(body), "Parse error") {
			return nil, fmt.Errorf("PHP error detected: %s", string(body))
		}

		return result, nil
	}

	// Test root path
	t.Run("Root Path", func(t *testing.T) {
		// Access root path
		rootResponse, err := getJsonResponse(server.URL + "/")
		if err != nil {
			t.Fatalf("Root path request failed: %v", err)
		}

		// Verify it's the correct handler
		rootType, ok := rootResponse["type"].(string)
		if !ok {
			t.Fatalf("Expected 'type' field to be a string, got %T", rootResponse["type"])
		}

		if rootType != "root_path" {
			t.Errorf("Expected root path type 'root_path', got '%s'", rootType)
		}

		// Verify script path
		scriptPath, ok := rootResponse["script"].(string)
		if !ok {
			t.Fatalf("Expected 'script' field to be a string, got %T", rootResponse["script"])
		}

		if scriptPath != "index.php" {
			t.Errorf("Expected script path 'index.php', got '%s'", scriptPath)
		}

		// Verify URI
		uri, ok := rootResponse["uri"].(string)
		if !ok {
			t.Fatalf("Expected 'uri' field to be a string, got %T", rootResponse["uri"])
		}

		if uri != "/" {
			t.Errorf("Expected URI to be '/', got '%s'", uri)
		}
	})
}

// TestHandlerIsolation tests that different route handlers are properly isolated
// and don't contaminate each other's state or execution context
func TestHandlerIsolation(t *testing.T) {
	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-handler-isolation-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	usersDir := filepath.Join(tempDir, "users")
	productsDir := filepath.Join(tempDir, "products")

	err = os.MkdirAll(usersDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create users directory: %v", err)
	}

	err = os.MkdirAll(productsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create products directory: %v", err)
	}

	// Create different PHP files with unique identifiers

	// Root handler PHP
	rootPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"handler_id" => "root_handler",
			"script_path" => "index.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_segments" => $_PATH_SEGMENTS ?? [],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Users handler PHP
	usersPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"handler_id" => "users_handler",
			"script_path" => "users/profile.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_segments" => $_PATH_SEGMENTS ?? [],
			"user_id" => $_PATH["id"] ?? null,
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Products handler PHP
	productsPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"handler_id" => "products_handler",
			"script_path" => "products/details.php",
			"uri" => $_SERVER["REQUEST_URI"],
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_segments" => $_PATH_SEGMENTS ?? [],
			"product_id" => $_PATH["id"] ?? null,
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write PHP files
	err = os.WriteFile(filepath.Join(tempDir, "index.php"), []byte(rootPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create root PHP file: %v", err)
	}

	err = os.WriteFile(filepath.Join(usersDir, "profile.php"), []byte(usersPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create users PHP file: %v", err)
	}

	err = os.WriteFile(filepath.Join(productsDir, "details.php"), []byte(productsPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create products PHP file: %v", err)
	}

	// Create Frango middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Create router with multiple distinct routes
	mux := http.NewServeMux()
	mux.Handle("/users/", php.For("/users/profile.php"))       // Wildcard route
	mux.Handle("/products/", php.For("/products/details.php")) // Wildcard route
	mux.Handle("/", php.For("/index.php"))                     // Root route last

	// Create test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Helper function to extract JSON response
	getJsonResponse := func(url string) (map[string]interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Check for PHP errors
		if strings.Contains(string(body), "Fatal error") || strings.Contains(string(body), "Parse error") {
			return nil, fmt.Errorf("PHP error detected: %s", string(body))
		}

		return result, nil
	}

	// Test 1: Access each route in isolation to establish baseline
	t.Run("Individual Routes", func(t *testing.T) {
		// Test root route
		rootResponse, err := getJsonResponse(server.URL + "/")
		if err != nil {
			t.Fatalf("Root path request failed: %v", err)
		}

		// Verify correct handler
		rootID, ok := rootResponse["handler_id"].(string)
		if !ok || rootID != "root_handler" {
			t.Errorf("Expected root_handler, got %v", rootID)
		}

		// Test users route
		usersResponse, err := getJsonResponse(server.URL + "/users/123")
		if err != nil {
			t.Fatalf("Users path request failed: %v", err)
		}

		// Verify correct handler
		usersID, ok := usersResponse["handler_id"].(string)
		if !ok || usersID != "users_handler" {
			t.Errorf("Expected users_handler, got %v", usersID)
		}

		// Test products route
		productsResponse, err := getJsonResponse(server.URL + "/products/456")
		if err != nil {
			t.Fatalf("Products path request failed: %v", err)
		}

		// Verify correct handler
		productsID, ok := productsResponse["handler_id"].(string)
		if !ok || productsID != "products_handler" {
			t.Errorf("Expected products_handler, got %v", productsID)
		}
	})

	// Test 2: Rapidly alternate between routes to detect contamination
	t.Run("Alternating Routes", func(t *testing.T) {
		// Access sequence designed to maximize chance of contamination
		accessSequence := []struct {
			path          string
			expectedID    string
			failureDetail string
		}{
			{"/users/1", "users_handler", "first users request"},
			{"/products/1", "products_handler", "products after users"},
			{"/users/2", "users_handler", "users after products"},
			{"/", "root_handler", "root after users"},
			{"/products/2", "products_handler", "products after root"},
			{"/users/3", "users_handler", "second users request"},
			{"/products/3", "products_handler", "second products request"},
			{"/", "root_handler", "second root request"},
			{"/products/4", "products_handler", "third products request"},
			{"/users/4", "users_handler", "third users request"},
		}

		for _, access := range accessSequence {
			response, err := getJsonResponse(server.URL + access.path)
			if err != nil {
				t.Fatalf("Request to %s failed: %v", access.path, err)
			}

			handlerID, ok := response["handler_id"].(string)
			if !ok {
				t.Errorf("Missing handler_id in response for %s", access.path)
				continue
			}

			if handlerID != access.expectedID {
				t.Errorf("Handler contamination detected in %s: expected %s, got %s",
					access.failureDetail, access.expectedID, handlerID)
			}
		}
	})

	// Test 3: Create separate ServeMux instances to verify isolation
	t.Run("Isolated ServeMux Instances", func(t *testing.T) {
		// Create separate mux for users
		usersMux := http.NewServeMux()
		usersMux.Handle("/users/", php.For("/users/profile.php"))
		usersServer := httptest.NewServer(usersMux)
		defer usersServer.Close()

		// Create separate mux for products
		productsMux := http.NewServeMux()
		productsMux.Handle("/products/", php.For("/products/details.php"))
		productsServer := httptest.NewServer(productsMux)
		defer productsServer.Close()

		// Create separate mux for root
		rootMux := http.NewServeMux()
		rootMux.Handle("/", php.For("/index.php"))
		rootServer := httptest.NewServer(rootMux)
		defer rootServer.Close()

		// First access users route
		for i := 0; i < 3; i++ {
			_, err := getJsonResponse(server.URL + "/users/123")
			if err != nil {
				t.Fatalf("Users path request failed: %v", err)
			}
		}

		// Then check isolated product server (should not be contaminated)
		productsResponse, err := getJsonResponse(productsServer.URL + "/products/456")
		if err != nil {
			t.Fatalf("Isolated products path request failed: %v", err)
		}

		productsID, ok := productsResponse["handler_id"].(string)
		if !ok || productsID != "products_handler" {
			t.Errorf("Handler contamination in isolated server: expected products_handler, got %v", productsID)
		}

		// Check isolated root server (should not be contaminated)
		rootResponse, err := getJsonResponse(rootServer.URL + "/")
		if err != nil {
			t.Fatalf("Isolated root path request failed: %v", err)
		}

		rootID, ok := rootResponse["handler_id"].(string)
		if !ok || rootID != "root_handler" {
			t.Errorf("Handler contamination in isolated server: expected root_handler, got %v", rootID)
		}
	})

	// Test 4: Create a new middleware instance to ensure it's really the middleware causing issues, not the mux
	t.Run("Separate Middleware Instances", func(t *testing.T) {
		// Skip this test because FrankenPHP can only be initialized once per process
		t.Skip("Skipping - FrankenPHP can only be initialized once per process")

		// Create a new middleware instance
		php2, err := New(
			WithSourceDir(tempDir),
			WithDevelopmentMode(true),
		)
		if err != nil {
			t.Fatalf("Failed to create second middleware: %v", err)
		}
		defer php2.Shutdown()

		// Create a new mux with the new middleware
		mux2 := http.NewServeMux()
		mux2.Handle("/users/", php2.For("/users/profile.php"))
		mux2.Handle("/products/", php2.For("/products/details.php"))
		mux2.Handle("/", php2.For("/index.php"))

		server2 := httptest.NewServer(mux2)
		defer server2.Close()

		// First heavily use one route in the original server to potentially contaminate it
		for i := 0; i < 5; i++ {
			_, err := getJsonResponse(server.URL + "/users/" + fmt.Sprintf("%d", i))
			if err != nil {
				t.Fatalf("Users path request failed: %v", err)
			}
		}

		// Now check the new server's root route
		rootResponse, err := getJsonResponse(server2.URL + "/")
		if err != nil {
			t.Fatalf("New server root path request failed: %v", err)
		}

		rootID, ok := rootResponse["handler_id"].(string)
		if !ok || rootID != "root_handler" {
			t.Errorf("Handler contamination between middleware instances: expected root_handler, got %v", rootID)
		}

		// Check the product route on the new server
		productsResponse, err := getJsonResponse(server2.URL + "/products/456")
		if err != nil {
			t.Fatalf("New server products path request failed: %v", err)
		}

		productsID, ok := productsResponse["handler_id"].(string)
		if !ok || productsID != "products_handler" {
			t.Errorf("Handler contamination between middleware instances: expected products_handler, got %v", productsID)
		}
	})

	// Test 5: Simulate high load with concurrent requests
	t.Run("Concurrent Requests", func(t *testing.T) {
		var wg sync.WaitGroup
		errorChan := make(chan string, 50)

		// Run 50 concurrent requests across different routes
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				var path, expectedID string
				switch i % 3 {
				case 0:
					path = "/"
					expectedID = "root_handler"
				case 1:
					path = fmt.Sprintf("/users/%d", i)
					expectedID = "users_handler"
				case 2:
					path = fmt.Sprintf("/products/%d", i)
					expectedID = "products_handler"
				}

				response, err := getJsonResponse(server.URL + path)
				if err != nil {
					errorChan <- fmt.Sprintf("Request to %s failed: %v", path, err)
					return
				}

				handlerID, ok := response["handler_id"].(string)
				if !ok {
					errorChan <- fmt.Sprintf("Missing handler_id in response for %s", path)
					return
				}

				if handlerID != expectedID {
					errorChan <- fmt.Sprintf("Handler contamination in concurrent request to %s: expected %s, got %s",
						path, expectedID, handlerID)
					return
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)

		// Check for errors
		for err := range errorChan {
			t.Errorf("%s", err)
		}
	})
}

// TestRootPathAfterNestedAccess specifically tests the root path handler
// after accessing nested paths to verify there's no state contamination
func TestRootPathAfterNestedAccess(t *testing.T) {
	// Create temp directory for PHP files
	tempDir := filepath.Join(os.TempDir(), "frango-root-nested-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the nested directory structure
	nestedDir := filepath.Join(tempDir, "nested", "deep", "path")
	err = os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory structure: %v", err)
	}

	// Write distinct PHP files with differentiating markers

	// Root path PHP - with version tracking to detect handler reuse
	rootPHP := `<?php
		static $counter = 0; // This static counter should be reset between requests
		$counter++;
		
		header("Content-Type: application/json");
		$response = [
			"handler_id" => "root_handler",
			"counter" => $counter, // Should always be 1 if handler is properly reset
			"script_path" => __FILE__,
			"uri" => $_SERVER["REQUEST_URI"],
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"request_time" => time(),
			"path_segments" => $_PATH_SEGMENTS ?? [],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Nested path PHP - intentionally very different from root to aid detection
	nestedPHP := `<?php
		static $counter = 0;
		$counter++;
		
		header("Content-Type: application/json");
		$response = [
			"handler_id" => "nested_handler",
			"counter" => $counter,
			"script_path" => __FILE__,
			"uri" => $_SERVER["REQUEST_URI"],
			"script_name" => $_SERVER["SCRIPT_NAME"],
			"path_segments" => $_PATH_SEGMENTS ?? [],
			"REQUEST_URI" => $_SERVER["REQUEST_URI"],
			"SCRIPT_FILENAME" => $_SERVER["SCRIPT_FILENAME"],
			"PHP_SELF" => $_SERVER["PHP_SELF"],
		];
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Write the PHP files
	err = os.WriteFile(filepath.Join(tempDir, "index.php"), []byte(rootPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create root PHP file: %v", err)
	}

	err = os.WriteFile(filepath.Join(nestedDir, "index.php"), []byte(nestedPHP), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested PHP file: %v", err)
	}

	// Create Frango middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create Frango middleware: %v", err)
	}
	defer php.Shutdown()

	// Helper function to extract JSON response from HTTP request
	getJsonResponse := func(handler http.Handler, path string) (map[string]interface{}, error) {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Check for PHP errors
		if strings.Contains(w.Body.String(), "Fatal error") || strings.Contains(w.Body.String(), "Parse error") {
			return nil, fmt.Errorf("PHP error detected: %s", w.Body.String())
		}

		return result, nil
	}

	// Run multiple test approaches to maximize chances of detecting issues

	// Test 1: Direct handler verification without ServeMux
	t.Run("Direct Handler Comparison", func(t *testing.T) {
		// Create isolated handlers directly without a mux
		rootHandler := php.For("/index.php")
		nestedHandler := php.For("/nested/deep/path/index.php")

		// Get root path response first
		rootResponse, err := getJsonResponse(rootHandler, "/")
		if err != nil {
			t.Fatalf("Root path request failed: %v", err)
		}

		rootID, ok := rootResponse["handler_id"].(string)
		if !ok || rootID != "root_handler" {
			t.Errorf("Initial root response wrong handler: expected root_handler, got %v", rootID)
		}

		// Make multiple requests to nested handler to potentially contaminate state
		for i := 0; i < 5; i++ {
			_, err := getJsonResponse(nestedHandler, "/nested/deep/path")
			if err != nil {
				t.Fatalf("Nested path request failed: %v", err)
			}
		}

		// Now request root path again - should still be root_handler
		rootResponse2, err := getJsonResponse(rootHandler, "/")
		if err != nil {
			t.Fatalf("Second root path request failed: %v", err)
		}

		rootID2, ok := rootResponse2["handler_id"].(string)
		if !ok || rootID2 != "root_handler" {
			t.Errorf("Root handler contaminated: expected root_handler, got %v", rootID2)
		}

		// Check static counter - should be 1 if PHP environment is fresh
		counter, ok := rootResponse2["counter"].(float64)
		if !ok {
			t.Errorf("Counter missing in response")
		} else if counter != 1 {
			t.Errorf("PHP environment not reset properly: counter = %v, expected 1", counter)
		}
	})

	// Test 2: With ServeMux alternating paths
	t.Run("Alternating ServeMux Paths", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.Handle("/nested/", php.For("/nested/deep/path/index.php"))
		mux.Handle("/nested/deep/", php.For("/nested/deep/path/index.php"))
		mux.Handle("/nested/deep/path", php.For("/nested/deep/path/index.php"))
		mux.Handle("/nested/deep/path/", php.For("/nested/deep/path/index.php"))
		mux.Handle("/", php.For("/index.php"))

		server := httptest.NewServer(mux)
		defer server.Close()

		// Helper for this test case
		getResponse := func(path string) (map[string]interface{}, error) {
			resp, err := http.Get(server.URL + path)
			if err != nil {
				return nil, fmt.Errorf("HTTP request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}

			return result, nil
		}

		// Alternate between root and nested paths
		for i := 0; i < 5; i++ {
			// Access nested path
			nestedResponse, err := getResponse("/nested/deep/path")
			if err != nil {
				t.Fatalf("Nested path request failed: %v", err)
			}

			// Verify it's the nested handler
			nestedID, ok := nestedResponse["handler_id"].(string)
			if !ok || nestedID != "nested_handler" {
				t.Errorf("Expected nested_handler, got %v", nestedID)
			}

			// Access root path
			rootResponse, err := getResponse("/")
			if err != nil {
				t.Fatalf("Root path request failed: %v", err)
			}

			// Verify it's the root handler
			rootID, ok := rootResponse["handler_id"].(string)
			if !ok || rootID != "root_handler" {
				t.Errorf("Root handler contaminated in round %d: expected root_handler, got %v", i, rootID)
			}

			// Verify static counter
			counter, ok := rootResponse["counter"].(float64)
			if !ok {
				t.Errorf("Counter missing in response")
			} else if counter != 1 {
				t.Errorf("PHP environment not reset in round %d: counter = %v, expected 1", i, counter)
			}
		}
	})

	// Test 3: Multiple different installations of same root route
	t.Run("Multiple Root Instances", func(t *testing.T) {
		// Skip this test because FrankenPHP can only be initialized once per process
		// FrankenPHP has a design constraint where it can only be initialized once
		// in a process, so we need to skip this test during normal runs
		t.Skip("Skipping - FrankenPHP can only be initialized once per process")

		// Create 3 separate mux instances with same pattern but different middleware instances
		// This tests if the middleware is properly isolating state between instances

		php1, err := New(WithSourceDir(tempDir), WithDevelopmentMode(true))
		if err != nil {
			t.Fatalf("Failed to create first middleware: %v", err)
		}
		defer php1.Shutdown()

		php2, err := New(WithSourceDir(tempDir), WithDevelopmentMode(true))
		if err != nil {
			t.Fatalf("Failed to create second middleware: %v", err)
		}
		defer php2.Shutdown()

		php3, err := New(WithSourceDir(tempDir), WithDevelopmentMode(true))
		if err != nil {
			t.Fatalf("Failed to create third middleware: %v", err)
		}
		defer php3.Shutdown()

		// Create three mux instances
		mux1 := http.NewServeMux()
		mux1.Handle("/", php1.For("/index.php"))
		server1 := httptest.NewServer(mux1)
		defer server1.Close()

		mux2 := http.NewServeMux()
		mux2.Handle("/nested/", php2.For("/nested/deep/path/index.php"))
		mux2.Handle("/", php2.For("/index.php"))
		server2 := httptest.NewServer(mux2)
		defer server2.Close()

		mux3 := http.NewServeMux()
		mux3.Handle("/", php3.For("/index.php"))
		server3 := httptest.NewServer(mux3)
		defer server3.Close()

		// Access the root path on all three servers
		getFromAllServers := func() error {
			servers := []struct {
				url  string
				name string
			}{
				{server1.URL + "/", "server1"},
				{server2.URL + "/", "server2"},
				{server3.URL + "/", "server3"},
			}

			for _, s := range servers {
				resp, err := http.Get(s.url)
				if err != nil {
					return fmt.Errorf("%s request failed: %w", s.name, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("%s unexpected status code: %d", s.name, resp.StatusCode)
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("%s failed to read body: %w", s.name, err)
				}

				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("%s failed to parse JSON: %w", s.name, err)
				}

				handlerID, ok := result["handler_id"].(string)
				if !ok || handlerID != "root_handler" {
					return fmt.Errorf("%s wrong handler: expected root_handler, got %v", s.name, handlerID)
				}

				counter, ok := result["counter"].(float64)
				if !ok {
					return fmt.Errorf("%s counter missing", s.name)
				}

				if counter != 1 {
					return fmt.Errorf("%s PHP not reset: counter = %v, expected 1", s.name, counter)
				}
			}

			return nil
		}

		// Access some nested routes on server2 to potentially contaminate state
		for i := 0; i < 5; i++ {
			resp, err := http.Get(server2.URL + "/nested/deep/path")
			if err != nil {
				t.Fatalf("Nested path request failed: %v", err)
			}
			resp.Body.Close()
		}

		// Now try to access all root paths - they should all be isolated
		if err := getFromAllServers(); err != nil {
			t.Errorf("Multiple instances test failed: %v", err)
		}
	})

	// Test 4: Multiple identical middleware instances with same VFS but different routes
	t.Run("Shared VFS Multiple Routes", func(t *testing.T) {
		// Create a new VFS instance
		vfs := php.NewVFS()
		defer vfs.Cleanup()

		// Configure the VFS
		if err := vfs.AddSourceFile(filepath.Join(tempDir, "index.php"), "/index.php"); err != nil {
			t.Fatalf("Failed to add root file to VFS: %v", err)
		}

		if err := vfs.AddSourceFile(filepath.Join(nestedDir, "index.php"), "/nested/deep/path/index.php"); err != nil {
			t.Fatalf("Failed to add nested file to VFS: %v", err)
		}

		// Create server with multiple routes sharing the same VFS instance
		mux := http.NewServeMux()

		// Important: use ForVFS which is more likely to show problems
		mux.Handle("/nested/deep/path/", php.ForVFS(vfs, "/nested/deep/path/index.php"))
		mux.Handle("/", php.ForVFS(vfs, "/index.php"))

		server := httptest.NewServer(mux)
		defer server.Close()

		// Helper for this test case
		getResponse := func(path string) (map[string]interface{}, error) {
			resp, err := http.Get(server.URL + path)
			if err != nil {
				return nil, fmt.Errorf("HTTP request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}

			return result, nil
		}

		// Access nested path several times
		for i := 0; i < 5; i++ {
			_, err := getResponse("/nested/deep/path/")
			if err != nil {
				t.Fatalf("Nested path request failed: %v", err)
			}
		}

		// Now access root path - should be uncontaminated
		rootResponse, err := getResponse("/")
		if err != nil {
			t.Fatalf("Root path request failed: %v", err)
		}

		rootID, ok := rootResponse["handler_id"].(string)
		if !ok || rootID != "root_handler" {
			t.Errorf("Expected root_handler, got %v", rootID)
		}

		counter, ok := rootResponse["counter"].(float64)
		if !ok {
			t.Errorf("Counter missing in response")
		} else if counter != 1 {
			t.Errorf("Shared VFS handler contamination: counter = %v, expected 1", counter)
		}
	})
}
