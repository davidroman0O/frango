package executor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestFormHandlingURLEncodedTable tests processing of application/x-www-form-urlencoded POST data using a table.
func TestFormHandlingURLEncodedTable(t *testing.T) {
	t.Helper()

	// PHP script remains the same
	formHandlerPHP := `<?php
		header("Content-Type: application/json");
		$response = [
			"method" => $_SERVER["REQUEST_METHOD"],
			"post" => $_POST ?? [],
			"raw_input" => file_get_contents("php://input"),
		];
		echo json_encode($response);
	?>`

	// Setup test environment once (using helper from executor_test.go)
	env, err := SetupExecutorTest(t, map[string]string{
		"form_handler.php": formHandlerPHP,
	}, WithTestLogger(t)) // Add logger for visibility during test run
	if err != nil {
		t.Fatalf("SetupExecutorTest failed: %v", err)
	}
	defer env.Cleanup()

	// Define test cases in a table
	testCases := []struct {
		name             string
		formData         string
		expectedPostData map[string]interface{} // Using interface{} for flexibility with arrays/strings
		expectedRawInput string
	}{
		{
			name:     "Simple Fields",
			formData: "name=John Doe&age=30",
			expectedPostData: map[string]interface{}{
				"name": "John Doe",
				"age":  "30", // PHP often reads form data as strings
			},
			expectedRawInput: "name=John Doe&age=30",
		},
		{
			name:     "Fields with Array Syntax",
			formData: "tags[]=coding&tags[]=testing",
			expectedPostData: map[string]interface{}{
				// NOTE: PHP parses 'tags[]' into an array under the key 'tags'
				"tags": []interface{}{"coding", "testing"},
			},
			expectedRawInput: "tags[]=coding&tags[]=testing",
		},
		{
			name:     "Mixed Simple and Array Fields",
			formData: "name=Alice&age=25&hobbies[]=reading&hobbies[]=hiking",
			expectedPostData: map[string]interface{}{
				"name":    "Alice",
				"age":     "25",
				"hobbies": []interface{}{"reading", "hiking"},
			},
			expectedRawInput: "name=Alice&age=25&hobbies[]=reading&hobbies[]=hiking",
		},
		{
			name:             "Empty Form Data",
			formData:         "",
			expectedPostData: map[string]interface{}{ // PHP's $_POST will likely be an empty array/map
			},
			expectedRawInput: "",
		},
		{
			name:     "Special Characters",
			formData: "message=Hello%20World%21&symbol=%26%3D%2B", // URL encoded: Hello World! &=+
			expectedPostData: map[string]interface{}{
				"message": "Hello World!",
				"symbol":  "&=+",
			},
			expectedRawInput: "message=Hello%20World%21&symbol=%26%3D%2B",
		},
		{
			name:     "Field with dot in name", // PHP might convert '.' to '_'
			formData: "user.name=Dotty",
			expectedPostData: map[string]interface{}{
				"user.name": "Dotty", // Check if PHP preserves the dot or converts to underscore
			},
			expectedRawInput: "user.name=Dotty",
		},
	}

	// Loop through test cases
	for _, tc := range testCases {
		tc := tc // Capture range variable

		// Run each case as a sub-test
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			// Prepare the request body for this specific case
			reqBody := strings.NewReader(tc.formData)

			// Create the request
			req := httptest.NewRequest("POST", "/form_handler.php", reqBody)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Execute the request (using helper from executor_test.go)
			recorder := ExecuteRequest(t, env, "/form_handler.php", req)
			resp := recorder.Result()
			defer resp.Body.Close() // Ensure body is closed
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			body := string(bodyBytes)

			// --- Validation for this case ---

			// Check status code
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}

			// Check content type
			expectedContentType := "application/json"
			actualContentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(actualContentType, expectedContentType) { // Use HasPrefix for potential charset
				t.Errorf("Expected Content-Type starting with '%s', got '%s'", expectedContentType, actualContentType)
			}

			// Check for PHP errors (using helper from executor_test.go)
			CheckNoPHPErrors(t, body)

			// Parse JSON response
			var response map[string]interface{}
			err = json.Unmarshal(bodyBytes, &response) // Use bodyBytes here
			if err != nil {
				// Add more context to parse failures
				t.Fatalf("Failed to parse JSON response: %v\nBody:\n%s", err, body)
			}

			// Validate method
			method, ok := response["method"].(string)
			if !ok || method != "POST" {
				t.Errorf("Expected request method 'POST', got '%v'", response["method"])
			}

			// Validate POST data
			postData := make(map[string]interface{}) // Initialize to avoid nil map panic
			if response["post"] != nil {
				switch p := response["post"].(type) {
				case map[string]interface{}:
					postData = p
				case []interface{}:
					if len(p) == 0 && len(tc.expectedPostData) == 0 {
						// Empty array is acceptable for empty expected data
						t.Logf("Received empty array for POST data, which is acceptable for empty form.")
					} else {
						t.Fatalf("Expected 'post' data to be a map or empty array, got %T. Response: %v", response["post"], response)
					}
				default:
					t.Fatalf("Expected 'post' data to be a map or empty array, got %T. Response: %v", response["post"], response)
				}
			} else if len(tc.expectedPostData) != 0 {
				// POST data is nil, but we expected something
				t.Fatalf("'post' data is nil, but expected %d fields. Response: %v", len(tc.expectedPostData), response)
			}

			// Compare actual POST data with expected POST data for this case
			// Need to handle the dot-to-underscore conversion possibility carefully
			tempExpectedData := make(map[string]interface{})
			for k, v := range tc.expectedPostData {
				tempExpectedData[k] = v
			}
			handledKeys := make(map[string]bool)

			for key, actualVal := range postData {
				expectedVal, exists := tempExpectedData[key]
				underscoreKeyForDot := ""

				if !exists && strings.Contains(key, "_") {
					// Maybe this key corresponds to an expected key with a dot?
					dotKey := strings.ReplaceAll(key, "_", ".")
					if _, dotKeyExists := tempExpectedData[dotKey]; dotKeyExists {
						t.Logf("Note: PHP converted dot to underscore in field name '%s' -> '%s'", dotKey, key)
						expectedVal = tempExpectedData[dotKey]
						exists = true
						underscoreKeyForDot = key        // Track this key was handled via conversion
						delete(tempExpectedData, dotKey) // Remove original expected key
					}
				} else if !exists {
					// Key genuinely not expected
					t.Errorf("Unexpected field '%s' found in POST data: %v", key, actualVal)
					handledKeys[key] = true
					continue
				}

				// Compare based on expected type
				switch expected := expectedVal.(type) {
				case string:
					CheckValueMatch(t, postData, key, expected, fmt.Sprintf("post[%s]", key))
				case []interface{}:
					actualSlice, ok := actualVal.([]interface{})
					if !ok {
						t.Errorf("Expected field '%s' to be a slice, got %T", key, actualVal)
					} else if !reflect.DeepEqual(expected, actualSlice) {
						// Provide a more detailed error for slice comparison
						t.Errorf("Field '%s' slice mismatch:\nExpected: %v\nGot:      %v", key, expected, actualSlice)
					}
				default:
					t.Errorf("Unsupported expected type '%T' in test case for key '%s'", expectedVal, key)
				}
				handledKeys[key] = true
				// If this key was handled via underscore conversion, remove it from tempExpectedData
				if underscoreKeyForDot == key {
					// We already removed the dot version above
				} else if exists {
					delete(tempExpectedData, key) // Remove key from expected map as it's been checked
				}
			}

			// Check if any expected keys were not found in the actual data
			if len(tempExpectedData) > 0 {
				missingKeys := []string{}
				for k := range tempExpectedData {
					missingKeys = append(missingKeys, k)
				}
				t.Errorf("Expected fields missing from POST data: %v", missingKeys)
			}

			// Validate raw input matches the sent form data for this case
			rawInput, ok := response["raw_input"].(string)
			if !ok {
				// Handle case where raw_input might be nil if body was empty
				if tc.expectedRawInput == "" && response["raw_input"] == nil {
					// OK
				} else {
					t.Errorf("Expected raw_input to be string, got %T", response["raw_input"])
				}
			} else if rawInput != tc.expectedRawInput {
				t.Errorf("Expected raw input '%s', got '%s'", tc.expectedRawInput, rawInput)
			}
		})
	}
}
