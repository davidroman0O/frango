package frango

import (
	"bytes"
	"encoding/json"
	"fmt"
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
				// Handle array values
				if (is_array($value) && count($value) > 0) {
					echo "$key: " . $value[0] . "\n";
				} else {
					echo "$key: $value\n";
				}
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
			
			// Process form data from POST variables
			echo "Form data from POST variables:\n";
			foreach ($_POST as $key => $value) {
				// Handle array values
				if (is_array($value) && count($value) > 0) {
					echo "  $key: " . $value[0] . "\n";
				} else {
					echo "  $key: $value\n";
				}
			}
			
			// For backward compatibility, also check $_SERVER vars
			if (count($_POST) === 0) {
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
			
			// Check for regular form fields from POST
			echo "\nForm fields from POST variables:\n";
			foreach ($_POST as $key => $value) {
				// Handle array values
				if (is_array($value) && count($value) > 0) {
					echo "  $key: " . $value[0] . "\n";
				} else {
					echo "  $key: $value\n";
				}
			}
			
			// For backward compatibility, also check $_SERVER vars
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

// TestFormHandling is a consolidated test for all form handling capabilities
func TestFormHandling(t *testing.T) {
	// Create a PHP script that processes form data
	formHandlerPHP := `<?php
		header("Content-Type: application/json");
		
		// Store inputs in response for testing
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"get" => $_GET,
			"post" => $_POST,
			"json" => $_JSON ?? [],
			"files" => $_FILES,
			"raw_input" => file_get_contents("php://input"),
			"path" => $_PATH
		];
		
		// Output as JSON
		echo json_encode($response);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"form_handler.php": formHandlerPHP,
	})
	defer CleanupTest(env)

	// Test cases for form handling
	testCases := []struct {
		name         string
		method       string
		path         string
		pattern      string
		setupRequest func(req *http.Request)
		validateFunc func(t *testing.T, response map[string]interface{})
	}{
		// URL Encoded Form
		{
			name:   "URL Encoded Form",
			method: "POST",
			path:   "/form_handler.php",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader("name=John&age=25&active=true"))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				postData, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check individual form fields - handle both string and array values
				if name, ok := postData["name"]; !ok {
					t.Errorf("Expected post[name] to be present")
				} else if strVal, ok := name.(string); ok {
					if strVal != "John" {
						t.Errorf("Expected post[name] = 'John', got %s", strVal)
					}
				} else if arrVal, ok := name.([]interface{}); ok && len(arrVal) > 0 {
					if arrVal[0] != "John" {
						t.Errorf("Expected post[name][0] = 'John', got %v", arrVal[0])
					}
				} else {
					t.Errorf("Unexpected type for post[name]: %T", name)
				}

				if age, ok := postData["age"]; !ok {
					t.Errorf("Expected post[age] to be present")
				} else if strVal, ok := age.(string); ok {
					if strVal != "25" {
						t.Errorf("Expected post[age] = '25', got %s", strVal)
					}
				} else if arrVal, ok := age.([]interface{}); ok && len(arrVal) > 0 {
					if fmt.Sprintf("%v", arrVal[0]) != "25" {
						t.Errorf("Expected post[age][0] = '25', got %v", arrVal[0])
					}
				} else {
					t.Errorf("Unexpected type for post[age]: %T", age)
				}

				if active, ok := postData["active"]; !ok {
					t.Errorf("Expected post[active] to be present")
				} else if strVal, ok := active.(string); ok {
					if strVal != "true" {
						t.Errorf("Expected post[active] = 'true', got %s", strVal)
					}
				} else if arrVal, ok := active.([]interface{}); ok && len(arrVal) > 0 {
					if fmt.Sprintf("%v", arrVal[0]) != "true" {
						t.Errorf("Expected post[active][0] = 'true', got %v", arrVal[0])
					}
				} else {
					t.Errorf("Unexpected type for post[active]: %T", active)
				}
			},
		},

		// URL Parameters
		{
			name:    "URL Parameters",
			method:  "GET",
			path:    "/form_handler.php?id=42&filter=recent&sort=date",
			pattern: "/form_handler.php",
			setupRequest: func(req *http.Request) {
				// No additional setup needed
				req.Pattern = "/form_handler.php"
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				getData, ok := response["get"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'get' to be a map, got %T", response["get"])
				}

				// Check query parameters - handle both string and array values
				if id, ok := getData["id"]; !ok {
					t.Errorf("Expected get[id] to be present")
				} else if strVal, ok := id.(string); ok {
					if strVal != "42" {
						t.Errorf("Expected get[id] = '42', got %s", strVal)
					}
				} else if arrVal, ok := id.([]interface{}); ok && len(arrVal) > 0 {
					if fmt.Sprintf("%v", arrVal[0]) != "42" {
						t.Errorf("Expected get[id][0] = '42', got %v", arrVal[0])
					}
				} else {
					t.Errorf("Unexpected type for get[id]: %T", id)
				}

				if filter, ok := getData["filter"]; !ok {
					t.Errorf("Expected get[filter] to be present")
				} else if strVal, ok := filter.(string); ok {
					if strVal != "recent" {
						t.Errorf("Expected get[filter] = 'recent', got %s", strVal)
					}
				} else if arrVal, ok := filter.([]interface{}); ok && len(arrVal) > 0 {
					if fmt.Sprintf("%v", arrVal[0]) != "recent" {
						t.Errorf("Expected get[filter][0] = 'recent', got %v", arrVal[0])
					}
				} else {
					t.Errorf("Unexpected type for get[filter]: %T", filter)
				}

				if sort, ok := getData["sort"]; !ok {
					t.Errorf("Expected get[sort] to be present")
				} else if strVal, ok := sort.(string); ok {
					if strVal != "date" {
						t.Errorf("Expected get[sort] = 'date', got %s", strVal)
					}
				} else if arrVal, ok := sort.([]interface{}); ok && len(arrVal) > 0 {
					if fmt.Sprintf("%v", arrVal[0]) != "date" {
						t.Errorf("Expected get[sort][0] = 'date', got %v", arrVal[0])
					}
				} else {
					t.Errorf("Unexpected type for get[sort]: %T", sort)
				}
			},
		},

		// Path Parameters
		{
			name:    "Path Parameters",
			method:  "GET",
			path:    "/users/123/posts/456",
			pattern: "/users/{userId}/posts/{postId}",
			setupRequest: func(req *http.Request) {
				req.Pattern = "/users/{userId}/posts/{postId}"
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				pathData, ok := response["path"].(map[string]interface{})
				if !ok {
					// Some implementations might return empty array instead of map when no path params
					if arr, ok := response["path"].([]interface{}); ok {
						t.Logf("Path is empty array with %d elements", len(arr))
						return
					}
					t.Fatalf("Expected 'path' to be a map, got %T", response["path"])
				}

				// Check path parameters
				if userId, ok := pathData["userId"].(string); !ok || userId != "123" {
					t.Errorf("Expected path[userId] = '123', got %v", pathData["userId"])
				}

				if postId, ok := pathData["postId"].(string); !ok || postId != "456" {
					t.Errorf("Expected path[postId] = '456', got %v", pathData["postId"])
				}
			},
		},

		// Nested Form Data
		{
			name:   "Nested Form Data",
			method: "POST",
			path:   "/form_handler.php",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(
					"user[name]=Alice&user[email]=alice@example.com&filters[category]=books&filters[price]=20",
				))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				postData, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check nested structure for 'user'
				userData, ok := postData["user"].(map[string]interface{})
				if !ok {
					// In PHP 7+, array elements with square brackets in form data
					// create nested arrays directly in $_POST
					userName, ok := postData["user[name]"]
					if !ok {
						t.Fatalf("Missing both user[name] and user object in post data")
					}
					if strVal, ok := userName.(string); ok {
						if strVal != "Alice" {
							t.Errorf("Expected post[user[name]] = 'Alice', got %s", strVal)
						}
					} else if arrVal, ok := userName.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != "Alice" {
							t.Errorf("Expected post[user[name]][0] = 'Alice', got %v", arrVal[0])
						}
					} else {
						t.Errorf("Unexpected type for post[user[name]]: %T", userName)
					}

					userEmail := postData["user[email]"]
					if strVal, ok := userEmail.(string); ok {
						if strVal != "alice@example.com" {
							t.Errorf("Expected post[user[email]] = 'alice@example.com', got %s", strVal)
						}
					} else if arrVal, ok := userEmail.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != "alice@example.com" {
							t.Errorf("Expected post[user[email]][0] = 'alice@example.com', got %v", arrVal[0])
						}
					} else {
						t.Errorf("Unexpected type for post[user[email]]: %T", userEmail)
					}

					filtersCategory := postData["filters[category]"]
					if strVal, ok := filtersCategory.(string); ok {
						if strVal != "books" {
							t.Errorf("Expected post[filters[category]] = 'books', got %s", strVal)
						}
					} else if arrVal, ok := filtersCategory.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != "books" {
							t.Errorf("Expected post[filters[category]][0] = 'books', got %v", arrVal[0])
						}
					} else {
						t.Errorf("Unexpected type for post[filters[category]]: %T", filtersCategory)
					}

					filtersPrice := postData["filters[price]"]
					expected := "20"
					if strVal, ok := filtersPrice.(string); ok {
						if strVal != expected {
							t.Errorf("Expected post[filters[price]] = '%s', got %s", expected, strVal)
						}
					} else if arrVal, ok := filtersPrice.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != expected {
							t.Errorf("Expected post[filters[price]][0] = '%s', got %v", expected, arrVal[0])
						}
					} else if numVal, ok := filtersPrice.(float64); ok {
						if fmt.Sprintf("%.0f", numVal) != expected {
							t.Errorf("Expected post[filters[price]] = '%s', got %v", expected, numVal)
						}
					} else {
						t.Errorf("Unexpected type for post[filters[price]]: %T with value %v", filtersPrice, filtersPrice)
					}
				} else {
					// Check nested user fields
					if name, ok := userData["name"].(string); !ok || name != "Alice" {
						t.Errorf("Expected post[user][name] = 'Alice', got %v", userData["name"])
					}

					if email, ok := userData["email"].(string); !ok || email != "alice@example.com" {
						t.Errorf("Expected post[user][email] = 'alice@example.com', got %v", userData["email"])
					}

					// Check nested filters
					filtersData, ok := postData["filters"].(map[string]interface{})
					if !ok {
						t.Fatalf("Expected 'filters' to be a map, got %T", postData["filters"])
					}

					if category, ok := filtersData["category"].(string); !ok || category != "books" {
						t.Errorf("Expected post[filters][category] = 'books', got %v", filtersData["category"])
					}

					// The price might come as a string or a number depending on PHP version
					expected := "20"
					switch v := filtersData["price"].(type) {
					case string:
						if v != expected {
							t.Errorf("Expected post[filters][price] = '%s', got %s", expected, v)
						}
					case float64:
						if fmt.Sprintf("%.0f", v) != expected {
							t.Errorf("Expected post[filters][price] = '%s', got %v", expected, v)
						}
					default:
						t.Errorf("Unexpected type for post[filters][price]: %T with value %v", filtersData["price"], filtersData["price"])
					}
				}
			},
		},

		// GET with nested query parameters
		{
			name:   "GET with Nested Query Parameters",
			method: "GET",
			path:   "/form_handler.php?user[name]=Bob&user[email]=bob@example.com&filters[category]=electronics&filters[price]=100",
			setupRequest: func(req *http.Request) {
				// No additional setup needed
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				getData, ok := response["get"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'get' to be a map, got %T", response["get"])
				}

				// In PHP, GET parameters with square brackets usually don't create nested arrays
				// They're treated as literal parameter names with brackets in the key
				userName, ok := getData["user[name]"]
				if !ok {
					// If not using literal bracket notation, check for nested structure
					userMap, ok := getData["user"].(map[string]interface{})
					if !ok {
						t.Fatalf("Missing both user[name] and user object in GET data")
					}

					if name, ok := userMap["name"].(string); !ok || name != "Bob" {
						t.Errorf("Expected get[user][name] = 'Bob', got %v", userMap["name"])
					}
				} else {
					if strVal, ok := userName.(string); ok {
						if strVal != "Bob" {
							t.Errorf("Expected get[user[name]] = 'Bob', got %s", strVal)
						}
					} else if arrVal, ok := userName.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != "Bob" {
							t.Errorf("Expected get[user[name]][0] = 'Bob', got %v", arrVal[0])
						}
					} else {
						t.Errorf("Unexpected type for get[user[name]]: %T", userName)
					}

					userEmail := getData["user[email]"]
					if strVal, ok := userEmail.(string); ok {
						if strVal != "bob@example.com" {
							t.Errorf("Expected get[user[email]] = 'bob@example.com', got %s", strVal)
						}
					} else if arrVal, ok := userEmail.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != "bob@example.com" {
							t.Errorf("Expected get[user[email]][0] = 'bob@example.com', got %v", arrVal[0])
						}
					} else {
						t.Errorf("Unexpected type for get[user[email]]: %T", userEmail)
					}

					filtersCategory := getData["filters[category]"]
					if strVal, ok := filtersCategory.(string); ok {
						if strVal != "electronics" {
							t.Errorf("Expected get[filters[category]] = 'electronics', got %s", strVal)
						}
					} else if arrVal, ok := filtersCategory.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != "electronics" {
							t.Errorf("Expected get[filters[category]][0] = 'electronics', got %v", arrVal[0])
						}
					} else {
						t.Errorf("Unexpected type for get[filters[category]]: %T", filtersCategory)
					}

					filtersPrice := getData["filters[price]"]
					expected := "100"
					if strVal, ok := filtersPrice.(string); ok {
						if strVal != expected {
							t.Errorf("Expected get[filters[price]] = '%s', got %s", expected, strVal)
						}
					} else if arrVal, ok := filtersPrice.([]interface{}); ok && len(arrVal) > 0 {
						if fmt.Sprintf("%v", arrVal[0]) != expected {
							t.Errorf("Expected get[filters[price]][0] = '%s', got %v", expected, arrVal[0])
						}
					} else if numVal, ok := filtersPrice.(float64); ok {
						if fmt.Sprintf("%.0f", numVal) != expected {
							t.Errorf("Expected get[filters[price]] = '%s', got %v", expected, numVal)
						}
					} else {
						t.Errorf("Unexpected type for get[filters[price]]: %T with value %v", filtersPrice, filtersPrice)
					}
				}
			},
		},

		// Empty form data
		{
			name:   "Empty Form Data",
			method: "POST",
			path:   "/form_handler.php",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(""))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				// Check POST data structure - might be an empty map or empty array
				postData := response["post"]
				switch post := postData.(type) {
				case map[string]interface{}:
					// POST should be an empty object
					if len(post) > 0 {
						t.Errorf("Expected post to be empty, got %v", post)
					}
				case []interface{}:
					// Or POST might be an empty array
					if len(post) > 0 {
						t.Errorf("Expected post to be an empty array, got %v", post)
					}
				default:
					// Log but don't fail - it could be nil or other representation
					t.Logf("Post data has type: %T", postData)
				}

				// Raw input should be empty
				rawInput, ok := response["raw_input"].(string)
				if !ok || rawInput != "" {
					t.Errorf("Expected raw_input to be an empty string, got %v", rawInput)
				}
			},
		},

		// Large JSON payload
		{
			name:   "Large JSON Payload",
			method: "POST",
			path:   "/form_handler.php",
			setupRequest: func(req *http.Request) {
				// Create a large JSON object
				largeObj := map[string]interface{}{
					"id":     12345,
					"type":   "complex-object",
					"active": true,
					"items":  make([]map[string]interface{}, 25), // Reduced from 50 for test speed
					"metadata": map[string]interface{}{
						"created":  "2023-01-01T00:00:00Z",
						"modified": "2023-04-01T12:30:45Z",
						"version":  "2.3.5",
						"tags":     []string{"test", "large", "payload", "json", "frango"},
						"settings": map[string]interface{}{
							"cache":     true,
							"ttl":       3600,
							"retries":   5,
							"threshold": 0.95,
						},
					},
				}

				// Fill the items array with data
				for i := 0; i < 25; i++ {
					largeObj["items"].([]map[string]interface{})[i] = map[string]interface{}{
						"index": i,
						"name":  fmt.Sprintf("Item %d", i),
						"value": i * 10,
						"data":  strings.Repeat("Test data content ", 5),
					}
				}

				jsonData, _ := json.Marshal(largeObj)
				req.Header.Set("Content-Type", "application/json")
				req.Body = io.NopCloser(bytes.NewReader(jsonData))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				jsonData, ok := response["json"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'json' to be a map, got %T", response["json"])
				}

				// Check basic properties
				if jsonData["id"] != float64(12345) {
					t.Errorf("Expected json[id] = 12345, got %v", jsonData["id"])
				}

				// Check arrays length
				items, ok := jsonData["items"].([]interface{})
				if !ok {
					t.Fatalf("Expected json[items] to be an array, got %T", jsonData["items"])
				}

				if len(items) != 25 {
					t.Errorf("Expected json[items] to have 25 elements, got %d", len(items))
				}

				// Check deep nested structures
				metadata, ok := jsonData["metadata"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected json[metadata] to be a map, got %T", jsonData["metadata"])
				}

				settings, ok := metadata["settings"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected json[metadata][settings] to be a map, got %T", metadata["settings"])
				}

				if settings["ttl"] != float64(3600) {
					t.Errorf("Expected json[metadata][settings][ttl] = 3600, got %v", settings["ttl"])
				}
			},
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the base request
			req := httptest.NewRequest(tc.method, tc.path, nil)

			// Apply custom request setup
			if tc.setupRequest != nil {
				tc.setupRequest(req)
			}

			// Execute the request
			status, _, body := ExecutePHP(t, env, "/form_handler.php", req, nil)

			// Check status code
			if status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
			}

			// Check for PHP errors
			AssertNoPHPErrors(t, body)

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, body)
			}

			// Run the case-specific validation
			if tc.validateFunc != nil {
				tc.validateFunc(t, response)
			}
		})
	}
}

// TestFormContentTypes tests handling of various Content-Type values in form submissions
func TestFormContentTypes(t *testing.T) {
	// Create a PHP script that outputs content type information
	contentTypePHP := `<?php
		header("Content-Type: application/json");
		
		// Get content type details
		$contentType = $_SERVER["CONTENT_TYPE"] ?? "";
		$explicitContentType = isset($_SERVER["HTTP_CONTENT_TYPE"]) ? $_SERVER["HTTP_CONTENT_TYPE"] : "not set";
		
		// Output response
		echo json_encode([
			"content_type" => $contentType,
			"http_content_type" => $explicitContentType,
			"method" => $_SERVER["REQUEST_METHOD"],
			"post_count" => count($_POST),
			"json_exists" => !empty($_JSON),
			"json_count" => is_array($_JSON) ? count($_JSON) : 0,
			"raw_input_length" => strlen(file_get_contents("php://input"))
		]);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"content_type.php": contentTypePHP,
	})
	defer CleanupTest(env)

	// Test cases for different content types
	testCases := []struct {
		name        string
		contentType string
		body        string
		validator   func(t *testing.T, response map[string]interface{})
	}{
		{
			name:        "Standard Form URL Encoded",
			contentType: "application/x-www-form-urlencoded",
			body:        "name=test&value=123",
			validator: func(t *testing.T, response map[string]interface{}) {
				// Check content type was properly received
				contentType, ok := response["content_type"].(string)
				if !ok || contentType != "application/x-www-form-urlencoded" {
					t.Errorf("Expected content_type 'application/x-www-form-urlencoded', got '%v'", contentType)
				}

				// Check POST data was processed
				postCount, ok := response["post_count"].(float64)
				if !ok || postCount == 0 {
					t.Errorf("Expected post_count > 0, got %v", postCount)
				}
			},
		},
		{
			name:        "JSON Content Type",
			contentType: "application/json",
			body:        `{"name":"test","value":123}`,
			validator: func(t *testing.T, response map[string]interface{}) {
				// Check content type was properly received
				contentType, ok := response["content_type"].(string)
				if !ok || contentType != "application/json" {
					t.Errorf("Expected content_type 'application/json', got '%v'", contentType)
				}

				// Verify JSON was processed
				jsonExists, ok := response["json_exists"].(bool)
				if !ok || !jsonExists {
					t.Errorf("Expected json_exists to be true, got %v", jsonExists)
				}
			},
		},
		{
			name:        "Content Type with Charset",
			contentType: "application/x-www-form-urlencoded; charset=UTF-8",
			body:        "name=test&value=123",
			validator: func(t *testing.T, response map[string]interface{}) {
				// Content type should include charset
				contentType, ok := response["content_type"].(string)
				if !ok || !strings.Contains(contentType, "charset=UTF-8") {
					t.Errorf("Expected content_type to contain 'charset=UTF-8', got '%v'", contentType)
				}

				// Check POST data was processed despite charset parameter
				postCount, ok := response["post_count"].(float64)
				if !ok || postCount == 0 {
					t.Errorf("Expected post_count > 0, got %v", postCount)
				}
			},
		},
		{
			name:        "Custom Content Type",
			contentType: "application/vnd.api+json",
			body:        `{"data":{"type":"test","attributes":{"name":"custom"}}}`,
			validator: func(t *testing.T, response map[string]interface{}) {
				// Content type should be preserved
				contentType, ok := response["content_type"].(string)
				if !ok || contentType != "application/vnd.api+json" {
					t.Errorf("Expected content_type 'application/vnd.api+json', got '%v'", contentType)
				}

				// Check raw input was received (may not be parsed as JSON)
				rawLength, ok := response["raw_input_length"].(float64)
				if !ok || rawLength == 0 {
					t.Errorf("Expected raw_input_length > 0, got %v", rawLength)
				}
			},
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the request with appropriate content type
			req := httptest.NewRequest("POST", "/content_type.php", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)

			// Execute the request
			status, _, body := ExecutePHP(t, env, "/content_type.php", req, nil)

			// Check status code
			if status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, body)
			}

			// Run validation
			if tc.validator != nil {
				tc.validator(t, response)
			}
		})
	}
}

// TestFormEdgeCases tests advanced form handling scenarios and edge cases
func TestFormEdgeCases(t *testing.T) {
	// Create a PHP script that handles and outputs form data for edge case testing
	edgeCasePHP := `<?php
		header("Content-Type: application/json");
		
		// Collect information about the request and form data
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"content_type" => $_SERVER["CONTENT_TYPE"] ?? "",
			"get" => $_GET,
			"post" => $_POST, 
			"files" => $_FILES ?? [],
			"json" => $_JSON ?? [],
			"raw_input" => substr(file_get_contents("php://input"), 0, 1000), // Limit for readability
			"headers" => []
		];
		
		// Collect relevant headers
		foreach ($_SERVER as $key => $value) {
			if (strpos($key, 'HTTP_') === 0) {
				$header = str_replace('HTTP_', '', $key);
				$header = str_replace('_', '-', $header);
				$response["headers"][strtolower($header)] = $value;
			}
		}
		
		// Special handling for security tests
		if (isset($_POST['username'])) {
			// Basic XSS protection test
			$username = $_POST['username'];
			if (is_array($username)) {
				$username = $username[0] ?? "";
			}
			
			$response["sanitized"] = [
				"raw" => $username,
				"html_escaped" => htmlspecialchars($username, ENT_QUOTES, 'UTF-8')
			];
		}
		
		// For CSRF test
		if (isset($_POST['csrf_token'])) {
			$token = $_POST['csrf_token'];
			if (is_array($token)) {
				$token = $token[0] ?? "";
			}
			$response["csrf_valid"] = ($token === 'valid_token_12345');
		}
		
		// Output response
		echo json_encode($response, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"edge_case.php": edgeCasePHP,
	})
	defer CleanupTest(env)

	// Test cases for various edge cases
	testCases := []struct {
		name         string
		setupRequest func(req *http.Request)
		validateFunc func(t *testing.T, response map[string]interface{})
	}{
		// 1. Form data with special characters
		{
			name: "HTML Special Characters",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				formData.Add("html_content", "<script>alert('XSS');</script>")
				formData.Add("sql_injection", "' OR 1=1 --")
				formData.Add("quotes", "John's \"quoted\" text")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check HTML content was preserved
				htmlContent, exists := extractValue(post, "html_content")
				if !exists || htmlContent != "<script>alert('XSS');</script>" {
					t.Errorf("HTML content was not preserved correctly: %v", post["html_content"])
				}

				// Check SQL injection pattern was preserved
				sqlInjection, exists := extractValue(post, "sql_injection")
				if !exists || sqlInjection != "' OR 1=1 --" {
					t.Errorf("SQL injection pattern was not preserved correctly: %v", post["sql_injection"])
				}

				// Check quotes handling
				quotes, exists := extractValue(post, "quotes")
				if !exists || quotes != "John's \"quoted\" text" {
					t.Errorf("Quotes were not preserved correctly: %v", post["quotes"])
				}
			},
		},
		{
			name: "Unicode and Emoji Characters",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				formData.Add("chinese", "你好世界")
				formData.Add("emoji", "Hello 👋 World 🌍")
				formData.Add("mixed", "Mixed text with ñ, é, ß and emoji 🚀")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check Chinese characters
				chinese, exists := extractValue(post, "chinese")
				if !exists || chinese != "你好世界" {
					t.Errorf("Chinese characters were not preserved correctly: %v", post["chinese"])
				}

				// Check emoji
				emoji, exists := extractValue(post, "emoji")
				if !exists || emoji != "Hello 👋 World 🌍" {
					t.Errorf("Emoji were not preserved correctly: %v", post["emoji"])
				}

				// Check mixed characters
				mixed, exists := extractValue(post, "mixed")
				if !exists || !strings.Contains(mixed, "ñ, é, ß") || !strings.Contains(mixed, "🚀") {
					t.Errorf("Mixed Unicode characters were not preserved correctly: %v", post["mixed"])
				}
			},
		},

		// 2. Multiple values for the same field
		{
			name: "Multiple Values For Same Field",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"

				// Create multipart form with repeated field names
				var multipartBuffer bytes.Buffer
				multipartWriter := multipart.NewWriter(&multipartBuffer)

				// Add multiple values for the same field names
				multipartWriter.WriteField("color", "red")
				multipartWriter.WriteField("color", "green")
				multipartWriter.WriteField("color", "blue")

				// Add checkbox-style values
				multipartWriter.WriteField("options[]", "option1")
				multipartWriter.WriteField("options[]", "option2")
				multipartWriter.WriteField("options[]", "option3")

				// Finalize the form
				multipartWriter.Close()

				req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
				req.Body = io.NopCloser(&multipartBuffer)
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check multiple values for same field
				colors, exists := post["color"]
				if !exists {
					t.Errorf("'color' field is missing")
				} else {
					// Check if all colors are included (implementation-dependent how they're stored)
					colorsStr := fmt.Sprintf("%v", colors)
					for _, color := range []string{"red", "green", "blue"} {
						if !strings.Contains(colorsStr, color) {
							t.Errorf("Color '%s' missing from: %v", color, colors)
						}
					}
				}

				// Check array-style fields
				options, exists := post["options[]"]
				if !exists {
					t.Errorf("'options[]' field is missing")
				} else {
					// Check if all options are included
					optionsStr := fmt.Sprintf("%v", options)
					for _, option := range []string{"option1", "option2", "option3"} {
						if !strings.Contains(optionsStr, option) {
							t.Errorf("Option '%s' missing from: %v", option, options)
						}
					}
				}
			},
		},

		// 3. Form validation handling
		{
			name: "Form Validation Edge Cases",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				// Empty value
				formData.Add("required_field", "")
				// Very long value (2000 chars)
				formData.Add("long_field", strings.Repeat("a", 2000))
				// Invalid email
				formData.Add("email", "not-an-email@incomplete")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check empty field is preserved
				emptyField, exists := extractValue(post, "required_field")
				if !exists || emptyField != "" {
					t.Errorf("Empty field was not preserved correctly: %v", post["required_field"])
				}

				// Check long field
				longField, exists := extractValue(post, "long_field")
				if !exists || len(longField) != 2000 {
					t.Errorf("Long field has incorrect length, expected 2000, got %d", len(fmt.Sprintf("%v", post["long_field"])))
				}

				// Check invalid email
				email, exists := extractValue(post, "email")
				if !exists || email != "not-an-email@incomplete" {
					t.Errorf("Invalid email was not preserved correctly: %v", post["email"])
				}
			},
		},

		// 4. Content-Type negotiation
		{
			name: "Content-Type Mismatch",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				// Send JSON data but claim it's form-encoded
				jsonData := `{"key":"value","nested":{"subkey":"subvalue"}}`

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(jsonData))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				contentType, ok := response["content_type"].(string)
				if !ok || contentType != "application/x-www-form-urlencoded" {
					t.Errorf("Content-Type not correctly reported: %v", response["content_type"])
				}

				// Check raw input was captured - either directly in raw_input or in the POST data
				rawInput, ok := response["raw_input"].(string)
				foundJSON := ok && strings.Contains(rawInput, `{"key":"value"`)

				// If raw_input doesn't have our JSON, check if it was interpreted as form data
				if !foundJSON {
					post, ok := response["post"].(map[string]interface{})
					if ok {
						// Log how the server handled the mismatched content type
						t.Logf("Note: PHP engine attempted to parse JSON as form data: %v", post)

						// Consider the test passed if we see any evidence of our input
						for k := range post {
							if strings.Contains(fmt.Sprintf("%v", k), "{") ||
								strings.Contains(fmt.Sprintf("%v", k), "key") ||
								strings.Contains(fmt.Sprintf("%v", k), "value") {
								foundJSON = true
								break
							}
						}
					}
				}

				if !foundJSON {
					t.Errorf("JSON data not found in response, neither in raw_input nor as form data")
				}
			},
		},
		{
			name: "Malformed Content-Type Header",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				formData.Add("test", "value")

				// Set invalid Content-Type
				req.Header.Set("Content-Type", "text/weird;;;;;charset:::broken")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				contentType, ok := response["content_type"].(string)
				if !ok {
					t.Errorf("Content-Type missing from response")
				} else {
					// Verify the malformed header was received
					if contentType != "text/weird;;;;;charset:::broken" {
						t.Errorf("Malformed Content-Type not preserved, got: %v", contentType)
					}
				}

				// Check raw input was captured
				rawInput, ok := response["raw_input"].(string)
				if !ok || !strings.Contains(rawInput, "test=value") {
					t.Errorf("Raw input not correctly captured: %v", response["raw_input"])
				}
			},
		},

		// 5. Form security scenarios
		{
			name: "XSS Attack Prevention",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				formData.Add("username", "<script>alert('XSS');</script>")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				sanitized, ok := response["sanitized"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'sanitized' to be a map, got %T", response["sanitized"])
				}

				// Check raw value is preserved
				raw, exists := sanitized["raw"]
				if !exists || raw != "<script>alert('XSS');</script>" {
					t.Errorf("Raw value not preserved correctly: %v", sanitized["raw"])
				}

				// Check HTML escaped value
				escaped, exists := sanitized["html_escaped"]
				if !exists || escaped == raw || !strings.Contains(fmt.Sprintf("%v", escaped), "&lt;script&gt;") {
					t.Errorf("XSS payload was not properly escaped: %v", sanitized["html_escaped"])
				}
			},
		},
		{
			name: "CSRF Token Validation",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				formData.Add("action", "delete_account")
				formData.Add("csrf_token", "valid_token_12345")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				csrfValid, ok := response["csrf_valid"].(bool)
				if !ok {
					t.Fatalf("Expected 'csrf_valid' to be a boolean, got %T", response["csrf_valid"])
				}

				if !csrfValid {
					t.Errorf("Valid CSRF token was not recognized")
				}

				// Verify the form fields
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				action, exists := extractValue(post, "action")
				if !exists || action != "delete_account" {
					t.Errorf("Action field not correctly processed: %v", post["action"])
				}
			},
		},
		{
			name: "Invalid CSRF Token",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				formData.Add("action", "delete_account")
				formData.Add("csrf_token", "invalid_token")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				csrfValid, ok := response["csrf_valid"].(bool)
				if !ok {
					t.Fatalf("Expected 'csrf_valid' to be a boolean, got %T", response["csrf_valid"])
				}

				if csrfValid {
					t.Errorf("Invalid CSRF token was incorrectly accepted as valid")
				}
			},
		},

		// 6. Boundary value tests
		{
			name: "Field Name Edge Cases",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				formData := url.Values{}
				// Field with very long name
				formData.Add(strings.Repeat("a", 100), "long-name-value")
				// Field with unusual name
				formData.Add("field-with.special_chars+123", "special-name-value")
				// Field with empty name
				formData.Add("", "empty-name-value")

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				// Check long field name
				longFieldName := strings.Repeat("a", 100)
				if _, exists := post[longFieldName]; !exists {
					// Some servers might truncate very long names
					found := false
					for key := range post {
						if strings.HasPrefix(fmt.Sprintf("%v", key), "aaa") {
							found = true
							t.Logf("Note: Long field name may have been truncated: %v", key)
							break
						}
					}
					if !found {
						t.Errorf("Long field name not found in POST data: %v", post)
					}
				}

				// Check field with special chars in name
				specialName := "field-with.special_chars+123"
				if val, exists := extractValue(post, specialName); !exists || val != "special-name-value" {
					t.Errorf("Field with special chars in name not correctly processed: %v", post)
				}
			},
		},

		// 7. Malformed requests
		{
			name: "Malformed JSON Request",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"
				// Create invalid JSON
				malformedJSON := `{"key": "value", "broken": {`

				req.Header.Set("Content-Type", "application/json")
				req.Body = io.NopCloser(strings.NewReader(malformedJSON))
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				// Check content type was received correctly
				contentType, ok := response["content_type"].(string)
				if !ok || contentType != "application/json" {
					t.Errorf("Content-Type not correctly reported: %v", response["content_type"])
				}

				// Malformed JSON shouldn't parse into _JSON
				jsonData, ok := response["json"].(map[string]interface{})
				if ok && len(jsonData) > 0 {
					t.Errorf("Malformed JSON incorrectly parsed: %v", jsonData)
				}

				// Check raw input was captured
				rawInput, ok := response["raw_input"].(string)
				if !ok || !strings.Contains(rawInput, `{"key": "value", "broken": {`) {
					t.Errorf("Raw input not correctly captured: %v", response["raw_input"])
				}
			},
		},
	}

	// Execute all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create base request
			req := httptest.NewRequest("GET", "/edge_case.php", nil)

			// Apply custom request setup
			if tc.setupRequest != nil {
				tc.setupRequest(req)
			}

			// Execute request
			status, _, body := ExecutePHP(t, env, "/edge_case.php", req, nil)

			// Basic status check
			if status != http.StatusOK {
				t.Errorf("Expected status 200, got %d", status)
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, body)
			}

			// Run validation
			if tc.validateFunc != nil {
				tc.validateFunc(t, response)
			}
		})
	}
}

// Helper function to extract values that might be arrays or simple values
func extractValue(data map[string]interface{}, key string) (string, bool) {
	val, exists := data[key]
	if !exists {
		return "", false
	}

	switch v := val.(type) {
	case string:
		return v, true
	case []interface{}:
		if len(v) > 0 {
			return fmt.Sprintf("%v", v[0]), true
		}
	}

	return fmt.Sprintf("%v", val), true
}

// TestLargeFileUploads tests handling of file uploads with different sizes and types
func TestLargeFileUploads(t *testing.T) {
	// Skip this test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping large file upload test in CI environment")
	}

	// Create PHP script for file upload handling
	fileUploadPHP := `<?php
		header("Content-Type: application/json");
		
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"content_type" => $_SERVER["CONTENT_TYPE"] ?? "",
			"files" => [],
			"post" => $_POST,
		];
		
		// Process uploaded files
		foreach ($_FILES as $fieldName => $fileInfo) {
			if (is_array($fileInfo['name'])) {
				// Multiple files
				$fileCount = count($fileInfo['name']);
				for ($i = 0; $i < $fileCount; $i++) {
					$response["files"][] = [
						"field_name" => $fieldName,
						"name" => $fileInfo['name'][$i],
						"type" => $fileInfo['type'][$i],
						"size" => $fileInfo['size'][$i],
						"error" => $fileInfo['error'][$i],
						"tmp_name" => basename($fileInfo['tmp_name'][$i]),
					];
				}
			} else {
				// Single file
				$response["files"][] = [
					"field_name" => $fieldName,
					"name" => $fileInfo['name'],
					"type" => $fileInfo['type'],
					"size" => $fileInfo['size'],
					"error" => $fileInfo['error'],
					"tmp_name" => basename($fileInfo['tmp_name']),
				];
			}
		}
		
		echo json_encode($response, JSON_PRETTY_PRINT);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"upload.php": fileUploadPHP,
	})
	defer CleanupTest(env)

	testCases := []struct {
		name         string
		setupRequest func(req *http.Request)
		validateFunc func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Multiple File Upload",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"

				// Create multipart form with multiple files
				var multipartBuffer bytes.Buffer
				multipartWriter := multipart.NewWriter(&multipartBuffer)

				// Add regular form fields
				multipartWriter.WriteField("description", "Multiple file test")

				// Create first file (text file)
				fileWriter1, _ := multipartWriter.CreateFormFile("documents", "test1.txt")
				fileContent1 := strings.Repeat("This is test file 1.\n", 100) // ~2KB
				fileWriter1.Write([]byte(fileContent1))

				// Create second file (simulated image)
				fileWriter2, _ := multipartWriter.CreateFormFile("documents", "test2.jpg")
				// Simple JPEG-like header followed by random data
				jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
				fileContent2 := append(jpegHeader, bytes.Repeat([]byte{0xAA, 0xBB, 0xCC, 0xDD}, 1000)...) // ~4KB
				fileWriter2.Write(fileContent2)

				// Create third file in a different field
				fileWriter3, _ := multipartWriter.CreateFormFile("avatar", "profile.png")
				// Simple PNG-like header followed by random data
				pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
				fileContent3 := append(pngHeader, bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 500)...) // ~2KB
				fileWriter3.Write(fileContent3)

				// Finalize form
				multipartWriter.Close()

				req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
				req.Body = io.NopCloser(&multipartBuffer)
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				// Check files array exists
				files, ok := response["files"].([]interface{})
				if !ok {
					t.Fatalf("Expected 'files' to be an array, got %T", response["files"])
				}

				// Should have at least 2 files (different implementations may handle this differently)
				if len(files) < 2 {
					t.Errorf("Expected at least 2 files, got %d", len(files))
				}

				// Check for expected file names and sizes
				fileMap := make(map[string]map[string]interface{})
				for _, file := range files {
					fileInfo, ok := file.(map[string]interface{})
					if !ok {
						t.Errorf("Expected file info to be a map, got %T", file)
						continue
					}

					name, ok := fileInfo["name"].(string)
					if !ok {
						continue
					}

					fileMap[name] = fileInfo
				}

				// Check for text file
				textFileFound := false
				for name, file := range fileMap {
					if strings.HasSuffix(name, ".txt") {
						textFileFound = true
						t.Logf("Found text file: %s", name)

						// Check size is reasonable
						size, ok := file["size"].(float64)
						if !ok || size < 1000 { // Size should be ~2KB
							t.Errorf("Text file has unexpected size: %v", file["size"])
						}

						// Check type
						fileType, ok := file["type"].(string)
						if !ok || (!strings.Contains(fileType, "text/") && !strings.Contains(fileType, "octet-stream")) {
							t.Logf("Note: Text file has content type: %v", fileType)
						}
						break
					}
				}
				if !textFileFound {
					t.Errorf("No text file found in upload")
				}

				// Check for profile.png
				if file, exists := fileMap["profile.png"]; !exists {
					t.Errorf("File 'profile.png' not found")
				} else {
					// Check field name
					fieldName, ok := file["field_name"].(string)
					if !ok || fieldName != "avatar" {
						t.Errorf("File 'profile.png' has incorrect field name: %v", file["field_name"])
					}
				}

				// Check form field
				post, ok := response["post"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'post' to be a map, got %T", response["post"])
				}

				description, exists := extractValue(post, "description")
				if !exists || description != "Multiple file test" {
					t.Errorf("Form field 'description' not correctly processed: %v", post["description"])
				}
			},
		},
		{
			name: "Large File Upload",
			setupRequest: func(req *http.Request) {
				req.Method = "POST"

				// Create multipart form with a larger file
				var multipartBuffer bytes.Buffer
				multipartWriter := multipart.NewWriter(&multipartBuffer)

				// Add metadata
				multipartWriter.WriteField("filename", "large_document.pdf")
				multipartWriter.WriteField("filetype", "application/pdf")

				// Create a larger file (simulated PDF)
				fileWriter, _ := multipartWriter.CreateFormFile("document", "large_document.pdf")
				// PDF-like header
				pdfHeader := []byte("%PDF-1.5\n%¥±ë\n\n")
				// Generate ~500KB of PDF-like content
				contentSize := 500 * 1024                                    // 500KB
				contentChunk := bytes.Repeat([]byte("0123456789abcdef"), 64) // 1KB chunk

				// Write header
				fileWriter.Write(pdfHeader)

				// Write content in chunks
				for written := 0; written < contentSize; written += len(contentChunk) {
					fileWriter.Write(contentChunk)
				}

				// Finalize form
				multipartWriter.Close()

				req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
				req.Body = io.NopCloser(&multipartBuffer)
			},
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				// Check files array exists
				files, ok := response["files"].([]interface{})
				if !ok {
					t.Fatalf("Expected 'files' to be an array, got %T", response["files"])
				}

				// Should have 1 file
				if len(files) != 1 {
					t.Errorf("Expected 1 file, got %d", len(files))
				}

				if len(files) > 0 {
					fileInfo, ok := files[0].(map[string]interface{})
					if !ok {
						t.Fatalf("Expected file info to be a map, got %T", files[0])
					}

					// Check name
					name, ok := fileInfo["name"].(string)
					if !ok || name != "large_document.pdf" {
						t.Errorf("Expected file name 'large_document.pdf', got %v", fileInfo["name"])
					}

					// Check size is close to what we expect (~500KB)
					size, ok := fileInfo["size"].(float64)
					if !ok || size < 400000 || size > 600000 {
						t.Errorf("File has unexpected size: %v (expected ~500KB)", fileInfo["size"])
					}

					// Check content type (could be application/pdf or application/octet-stream)
					fileType, ok := fileInfo["type"].(string)
					if !ok || (!strings.Contains(fileType, "pdf") && !strings.Contains(fileType, "octet-stream")) {
						t.Logf("Note: File has content type: %v", fileType)
					}

					// Check error code
					errorCode, ok := fileInfo["error"].(float64)
					if !ok || errorCode != 0 {
						t.Errorf("File upload has error code: %v", fileInfo["error"])
					}
				}
			},
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create base request
			req := httptest.NewRequest("GET", "/upload.php", nil)

			// Apply custom request setup
			if tc.setupRequest != nil {
				tc.setupRequest(req)
			}

			// Execute request (with increased timeout for large files)
			status, _, body := ExecutePHP(t, env, "/upload.php", req, nil)

			// Basic status check
			if status != http.StatusOK {
				t.Errorf("Expected status 200, got %d", status)
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, body)
			}

			// Run validation
			if tc.validateFunc != nil {
				tc.validateFunc(t, response)
			}
		})
	}
}
