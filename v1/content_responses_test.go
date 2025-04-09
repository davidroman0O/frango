package frango

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ValidationFunc is a function that can validate different aspects of a response
type ValidationFunc interface{}

// TestContentResponses tests various content types returned by PHP scripts
func TestContentResponses(t *testing.T) {
	testCases := []struct {
		name           string
		phpCode        string
		expectedType   string
		expectedStatus int
		validationFunc ValidationFunc
	}{
		{
			name: "HTML Response",
			phpCode: `<?php 
				header("Content-Type: text/html");
				echo "<!DOCTYPE html><html><head><title>Test HTML</title></head><body><h1>Test HTML Response</h1><p>This is an HTML response from PHP</p></body></html>";
			?>`,
			expectedType:   "text/html",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string) {
				expectedContent := []string{
					"<!DOCTYPE html>",
					"<title>Test HTML</title>",
					"<h1>Test HTML Response</h1>",
					"<p>This is an HTML response from PHP</p>",
				}
				CheckResponseContains(t, body, expectedContent)
			},
		},
		{
			name: "JSON Response",
			phpCode: `<?php 
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
			expectedType:   "application/json",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string) {
				expectedContent := []string{
					"\"status\":\"success\"",
					"\"message\":\"This is a JSON response\"",
					"\"data\":",
					"\"item1\":\"value1\"",
					"\"numbers\":[1,2,3,4,5]",
				}
				CheckResponseContains(t, body, expectedContent)
			},
		},
		{
			name: "XML Response",
			phpCode: `<?php
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
			expectedType:   "application/xml",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string) {
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
				CheckResponseContains(t, body, expectedContent)
			},
		},
		{
			name: "Plain Text Response",
			phpCode: `<?php
				header("Content-Type: text/plain");
				echo "This is a plain text response.\nNo HTML or other formatting.";
			?>`,
			expectedType:   "text/plain",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string) {
				expectedContent := []string{
					"This is a plain text response.",
					"No HTML or other formatting.",
				}
				CheckResponseContains(t, body, expectedContent)
			},
		},
		{
			name: "Binary Response",
			phpCode: `<?php
				// Simple PNG image generator
				header("Content-Type: image/png");
				
				// Create a simple 1x1 PNG file (transparent pixel)
				echo base64_decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=');
			?>`,
			expectedType:   "image/png",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string) {
				// Check that the response looks like a PNG (starts with the PNG signature)
				if len(body) < 8 {
					t.Errorf("Binary response too short")
					return
				}

				// PNG signature (hex: 89 50 4E 47 0D 0A 1A 0A)
				pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
				bodyBytes := []byte(body)

				for i, b := range pngSignature {
					if i < len(bodyBytes) && bodyBytes[i] != b {
						t.Errorf("Binary response does not start with PNG signature at byte %d", i)
					}
				}
			},
		},
		{
			name: "Custom Status Code",
			phpCode: `<?php
				// Set a 404 status code
				http_response_code(404);
				header("Content-Type: text/plain");
				echo "Not Found: The requested resource does not exist.";
			?>`,
			expectedType:   "text/plain",
			expectedStatus: http.StatusNotFound,
			validationFunc: func(t *testing.T, body string) {
				if !strings.Contains(body, "Not Found") {
					t.Errorf("Expected 'Not Found' in response, got: %s", body)
				}
			},
		},
		{
			name: "Custom HTTP Headers",
			phpCode: `<?php
				header("Content-Type: text/plain");
				header("X-Custom-Header: CustomValue");
				header("X-Response-Time: 42ms");
				header("Cache-Control: no-cache, no-store");
				echo "Response with custom headers";
			?>`,
			expectedType:   "text/plain",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string, headers http.Header) {
				expectedHeaders := map[string]string{
					"X-Custom-Header": "CustomValue",
					"X-Response-Time": "42ms",
					"Cache-Control":   "no-cache, no-store",
				}

				for name, value := range expectedHeaders {
					if headers.Get(name) != value {
						t.Errorf("Expected header %s: %s, got: %s", name, value, headers.Get(name))
					}
				}
			},
		},
		{
			name: "HTTP Redirect",
			phpCode: `<?php
				// Set redirect code and location
				http_response_code(302);
				header("Location: /redirected-page");
				echo "You are being redirected...";
			?>`,
			expectedType:   "text/html",      // Default content type
			expectedStatus: http.StatusFound, // 302 Found
			validationFunc: func(t *testing.T, body string, headers http.Header) {
				// Check for redirect location header
				location := headers.Get("Location")
				if location != "/redirected-page" {
					t.Errorf("Expected Location header '/redirected-page', got: %s", location)
				}

				// Check body contains redirect message
				if !strings.Contains(body, "You are being redirected") {
					t.Errorf("Expected redirect message in body, got: %s", body)
				}
			},
		},
		{
			name: "Unicode Content",
			phpCode: `<?php
				header("Content-Type: text/html; charset=utf-8");
				
				// Output content with various Unicode characters
				echo "<!DOCTYPE html><html><body>";
				echo "<h1>Unicode Test</h1>";
				echo "<p>English: Hello World</p>";
				echo "<p>Chinese: 你好世界</p>";
				echo "<p>Japanese: こんにちは世界</p>";
				echo "<p>Arabic: مرحبا بالعالم</p>";
				echo "<p>Emoji: 🌍 🌎 🌏 👋 😊</p>";
				echo "</body></html>";
			?>`,
			expectedType:   "text/html",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string) {
				expectedContent := []string{
					"<h1>Unicode Test</h1>",
					"<p>English: Hello World</p>",
					"<p>Chinese: 你好世界</p>",
					"<p>Japanese: こんにちは世界</p>",
					"<p>Arabic: مرحبا بالعالم</p>",
					"<p>Emoji: 🌍 🌎 🌏 👋 😊</p>",
				}
				CheckResponseContains(t, body, expectedContent)
			},
		},
		{
			name: "Setting Cookies",
			phpCode: `<?php
				// Set multiple cookies
				setcookie("session_id", "abc123", time()+3600, "/", "", false, true);
				setcookie("user_pref", "darkmode", time()+86400, "/", "", false, false);
				setcookie("visited", "true");
				
				header("Content-Type: text/plain");
				echo "Cookies have been set";
			?>`,
			expectedType:   "text/plain",
			expectedStatus: http.StatusOK,
			validationFunc: func(t *testing.T, body string, headers http.Header) {
				// Check for Set-Cookie headers
				cookies := headers["Set-Cookie"]
				if len(cookies) < 3 {
					t.Errorf("Expected at least 3 cookies, got %d", len(cookies))
				}

				// Check for specific cookie values
				cookieMap := make(map[string]bool)
				for _, cookie := range cookies {
					if strings.Contains(cookie, "session_id=abc123") {
						cookieMap["session_id"] = true
					}
					if strings.Contains(cookie, "user_pref=darkmode") {
						cookieMap["user_pref"] = true
					}
					if strings.Contains(cookie, "visited=true") {
						cookieMap["visited"] = true
					}
				}

				expectedCookies := []string{"session_id", "user_pref", "visited"}
				for _, name := range expectedCookies {
					if !cookieMap[name] {
						t.Errorf("Expected cookie '%s' not found in response headers", name)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test environment
			env := SetupTest(t, map[string]string{
				"test.php": tc.phpCode,
			})
			defer CleanupTest(env)

			// Execute request
			status, headers, body := ExecutePHP(t, env, "/test.php",
				httptest.NewRequest("GET", "/test.php", nil), nil)

			// Check status code
			if status != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, status)
			}

			// Check content type
			contentType := headers.Get("Content-Type")
			if !strings.HasPrefix(contentType, tc.expectedType) {
				t.Errorf("Expected Content-Type starting with '%s', got '%s'", tc.expectedType, contentType)
			}

			// Check for PHP errors
			AssertNoPHPErrors(t, body)

			// Run content type specific validation
			if tc.validationFunc != nil {
				switch fn := tc.validationFunc.(type) {
				case func(t *testing.T, body string):
					fn(t, body)
				case func(t *testing.T, body string, headers http.Header):
					fn(t, body, headers)
				}
			}
		})
	}
}

// TestRequestHeaderHandling tests that PHP scripts can read HTTP request headers
func TestRequestHeaderHandling(t *testing.T) {
	// Standard header test
	t.Run("Basic Headers", func(t *testing.T) {
		// PHP script that outputs request headers
		headersPHP := `<?php
			header("Content-Type: application/json");
			
			// Collect server headers
			$requestHeaders = [];
			foreach ($_SERVER as $key => $value) {
				if (strpos($key, 'HTTP_') === 0) {
					$headerName = str_replace('HTTP_', '', $key);
					$headerName = str_replace('_', '-', $headerName);
					$requestHeaders[$headerName] = $value;
				}
			}
			
			// Output as JSON
			echo json_encode([
				"original_headers" => $_SERVER,
				"parsed_headers" => $requestHeaders
			]);
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"headers.php": headersPHP,
		})
		defer CleanupTest(env)

		// Create request with custom headers
		req := httptest.NewRequest("GET", "/headers.php", nil)
		req.Header.Set("User-Agent", "FrangoTestClient/1.0")
		req.Header.Set("X-Custom-Header", "CustomValue")
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		// Execute request
		_, _, body := ExecutePHP(t, env, "/headers.php", req, nil)

		// Parse JSON response
		responseData := ParseJSON(t, body)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Get parsed headers from response
		parsedHeaders, ok := responseData["parsed_headers"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected parsed_headers to be a map, got %T", responseData["parsed_headers"])
		}

		// Verify headers were correctly passed to PHP
		expectedHeaders := map[string]string{
			"USER-AGENT":      "FrangoTestClient/1.0",
			"X-CUSTOM-HEADER": "CustomValue",
			"AUTHORIZATION":   "Bearer test-token",
			"ACCEPT-LANGUAGE": "en-US,en;q=0.9",
		}

		for name, value := range expectedHeaders {
			if parsedHeaders[name] != value {
				t.Errorf("Expected header %s: %s, got: %s", name, value, parsedHeaders[name])
			}
		}
	})

	// Multiple header values test
	t.Run("Multiple Header Values", func(t *testing.T) {
		// PHP script that outputs headers with multiple values
		multiValueHeadersPHP := `<?php
			header("Content-Type: application/json");
			
			// Get specific headers we're testing
			$acceptHeader = $_SERVER['HTTP_ACCEPT'] ?? '';
			$forwardedForHeader = $_SERVER['HTTP_X_FORWARDED_FOR'] ?? '';
			$cookieHeader = $_SERVER['HTTP_COOKIE'] ?? '';
			
			echo json_encode([
				"accept" => $acceptHeader,
				"x_forwarded_for" => $forwardedForHeader,
				"cookie" => $cookieHeader,
				// Include raw access to $_SERVER for debugging
				"server" => $_SERVER
			]);
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"multi_headers.php": multiValueHeadersPHP,
		})
		defer CleanupTest(env)

		// Create request with multi-value headers
		req := httptest.NewRequest("GET", "/multi_headers.php", nil)

		// Set Accept header with multiple values
		req.Header.Set("Accept", "text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8")

		// Set X-Forwarded-For with multiple client IPs
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")

		// Add cookies
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
		req.AddCookie(&http.Cookie{Name: "user", Value: "testuser"})
		req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})

		// Execute request
		_, _, body := ExecutePHP(t, env, "/multi_headers.php", req, nil)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Parse JSON response
		responseData := ParseJSON(t, body)

		// Check Accept header
		accept, ok := responseData["accept"].(string)
		if !ok || !strings.Contains(accept, "text/html") || !strings.Contains(accept, "application/xml") {
			t.Errorf("Accept header not properly received, got: %v", accept)
		}

		// Check X-Forwarded-For header
		forwardedFor, ok := responseData["x_forwarded_for"].(string)
		if !ok || !strings.Contains(forwardedFor, "203.0.113.195") || !strings.Contains(forwardedFor, "150.172.238.178") {
			t.Errorf("X-Forwarded-For header not properly received, got: %v", forwardedFor)
		}

		// Check Cookie header
		cookie, ok := responseData["cookie"].(string)
		if !ok || !strings.Contains(cookie, "session=abc123") || !strings.Contains(cookie, "user=testuser") || !strings.Contains(cookie, "theme=dark") {
			t.Errorf("Cookie header not properly received, got: %v", cookie)
		}
	})

	// Case sensitivity test
	t.Run("Header Case Sensitivity", func(t *testing.T) {
		// PHP script that outputs original header names and values
		caseSensitivityPHP := `<?php
			header("Content-Type: application/json");
			
			// Get all headers using getallheaders() if available
			$allHeaders = function_exists('getallheaders') ? getallheaders() : [];
			
			// Also collect from $_SERVER
			$serverHeaders = [];
			foreach ($_SERVER as $key => $value) {
				if (strpos($key, 'HTTP_') === 0) {
					$headerName = substr($key, 5); // Remove HTTP_ prefix
					$serverHeaders[$headerName] = $value;
				}
			}
			
			echo json_encode([
				"getallheaders" => $allHeaders,
				"server_headers" => $serverHeaders
			]);
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"case_headers.php": caseSensitivityPHP,
		})
		defer CleanupTest(env)

		// Create request with headers in different case formats
		req := httptest.NewRequest("GET", "/case_headers.php", nil)

		// Set headers with different cases
		req.Header.Set("content-type", "application/json") // Lowercase
		req.Header.Set("USER-AGENT", "TestClient/1.0")     // Uppercase
		req.Header.Set("X-Custom-Value", "MixedCase")      // Mixed case with hyphens
		req.Header.Set("Authorization", "Bearer token")    // Title case

		// Execute request
		_, _, body := ExecutePHP(t, env, "/case_headers.php", req, nil)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Parse JSON response
		responseData := ParseJSON(t, body)

		// Check server headers
		serverHeaders, ok := responseData["server_headers"].(map[string]interface{})
		if !ok {
			t.Fatalf("Server headers not found in response")
		}

		// In PHP's $_SERVER, header names are all uppercase with underscores
		expectedHeaders := map[string]string{
			"CONTENT_TYPE":   "application/json", // Note: content-type is special and doesn't get HTTP_ prefix
			"USER_AGENT":     "TestClient/1.0",
			"X_CUSTOM_VALUE": "MixedCase",
			"AUTHORIZATION":  "Bearer token",
		}

		// Check if headers were normalized as expected in PHP environment
		for name, expectedValue := range expectedHeaders {
			if name != "CONTENT_TYPE" { // Content-Type is handled specially
				value, exists := serverHeaders[name]
				if !exists {
					t.Errorf("Expected header %s not found in server headers", name)
					continue
				}

				if value != expectedValue {
					t.Errorf("Expected header %s to be '%s', got '%v'", name, expectedValue, value)
				}
			}
		}
	})
}

// TestRenderTemplate tests that PHP scripts can render templates with provided variables
func TestRenderTemplate(t *testing.T) {
	// Simple template test
	t.Run("Basic Template", func(t *testing.T) {
		templatePHP := `<?php
			header("Content-Type: text/html; charset=utf-8");
			echo "<!DOCTYPE html><html><body>";
			
			// Access template variables
			if (isset($title)) {
				echo "<h1>" . htmlspecialchars($title) . "</h1>";
			}
			
			if (isset($content)) {
				echo "<div class=\"content\">" . htmlspecialchars($content) . "</div>";
			}
			
			if (isset($user) && is_array($user)) {
				echo "<div class=\"user-info\">";
				echo "User: " . htmlspecialchars($user['name']) . ", ";
				echo "Role: " . htmlspecialchars($user['role']);
				echo "</div>";
			}
			
			if (isset($stats) && is_array($stats)) {
				echo "<div class=\"stats-info\">";
				echo "Stats: Visits: " . htmlspecialchars($stats['visits']) . ", ";
				echo "Logins: " . htmlspecialchars($stats['logins']);
				echo "</div>";
			}
			
			// Show all template variables in JSON
			echo "<script>var templateData = " . json_encode($_TEMPLATE) . ";</script>";
			
			echo "</body></html>";
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"template.php": templatePHP,
		})
		defer CleanupTest(env)

		// Create render function with template data
		renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"title":   "Welcome to Frango",
				"content": "This is a rendered template with variables.",
				"user": map[string]interface{}{
					"name": "John Doe",
					"role": "Admin",
				},
				"stats": map[string]interface{}{
					"visits": 42,
					"logins": 7,
				},
			}
		}

		// Execute request with render function
		_, _, body := ExecutePHP(t, env, "/template.php",
			httptest.NewRequest("GET", "/template.php", nil), renderFn)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Check that template variables are in the response
		expectedContent := []string{
			"<h1>Welcome to Frango</h1>",
			"<div class=\"content\">This is a rendered template with variables.</div>",
			"User: John Doe, Role: Admin",
			"Stats: Visits: 42, Logins: 7",
			"templateData = {",
			"\"title\":\"Welcome to Frango\"",
			"\"user\":{\"name\":\"John Doe\",\"role\":\"Admin\"}",
			"\"stats\":{\"logins\":7,\"visits\":42}",
		}

		CheckResponseContains(t, body, expectedContent)
	})

	// Complex nested object test
	t.Run("Complex Nested Objects", func(t *testing.T) {
		nestedObjectsPHP := `<?php
			header("Content-Type: application/json");
			
			// Output the nested objects as received
			echo json_encode([
				"received" => [
					"config" => $config ?? null,
					"products" => $products ?? null,
					"metadata" => $metadata ?? null
				],
				"nested_access" => [
					"db_host" => $config['database']['host'] ?? null,
					"featured_product" => $products[0]['name'] ?? null,
					"last_updated_by" => $metadata['history']['updated_by']['name'] ?? null
				]
			]);
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"nested.php": nestedObjectsPHP,
		})
		defer CleanupTest(env)

		// Create render function with nested data
		renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"config": map[string]interface{}{
					"application": map[string]interface{}{
						"name":    "Test App",
						"version": "1.0.0",
						"debug":   true,
					},
					"database": map[string]interface{}{
						"host":     "localhost",
						"port":     3306,
						"username": "testuser",
						"password": "testpass",
					},
				},
				"products": []map[string]interface{}{
					{
						"id":    1,
						"name":  "Featured Product",
						"price": 99.99,
						"categories": []string{
							"electronics", "featured", "new",
						},
					},
					{
						"id":    2,
						"name":  "Regular Product",
						"price": 49.99,
						"categories": []string{
							"household", "sale",
						},
					},
				},
				"metadata": map[string]interface{}{
					"version": "2.0",
					"history": map[string]interface{}{
						"created_at": "2023-01-01",
						"updated_at": "2023-04-15",
						"updated_by": map[string]interface{}{
							"id":   42,
							"name": "Admin User",
							"role": "administrator",
						},
					},
				},
			}
		}

		// Execute request with render function
		_, _, body := ExecutePHP(t, env, "/nested.php",
			httptest.NewRequest("GET", "/nested.php", nil), renderFn)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Parse JSON response
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(body), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		// Extract nested_access section
		nestedAccess, ok := response["nested_access"].(map[string]interface{})
		if !ok {
			t.Fatalf("Response missing expected nested_access field")
		}

		// Check deep nested values were correctly accessed
		expectedNestedValues := map[string]interface{}{
			"db_host":          "localhost",
			"featured_product": "Featured Product",
			"last_updated_by":  "Admin User",
		}

		for key, expected := range expectedNestedValues {
			actual, exists := nestedAccess[key]
			if !exists {
				t.Errorf("Missing expected nested field: %s", key)
				continue
			}
			if actual != expected {
				t.Errorf("Incorrect value for nested field %s: expected '%v', got '%v'", key, expected, actual)
			}
		}
	})

	// Array iteration test
	t.Run("Array Iteration", func(t *testing.T) {
		arrayIterationPHP := `<?php
			header("Content-Type: text/html; charset=utf-8");
			echo "<!DOCTYPE html><html><body>";
			
			// List all items using iteration
			if (isset($items) && is_array($items)) {
				echo "<h2>Items List</h2>";
				echo "<ul class=\"items-list\">";
				
				foreach ($items as $item) {
					echo "<li>";
					echo "Item #" . htmlspecialchars($item['id']) . ": ";
					echo htmlspecialchars($item['name']) . " - ";
					echo "$" . htmlspecialchars(number_format($item['price'], 2));
					
					// Show tags if available
					if (isset($item['tags']) && is_array($item['tags'])) {
						echo " [";
						echo implode(", ", array_map('htmlspecialchars', $item['tags']));
						echo "]";
					}
					
					echo "</li>";
				}
				
				echo "</ul>";
				
				// Show item count and total price
				$total = array_sum(array_column($items, 'price'));
				echo "<p>Total items: " . count($items) . "</p>";
				echo "<p>Total price: $" . number_format($total, 2) . "</p>";
			}
			
			echo "</body></html>";
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"array_iteration.php": arrayIterationPHP,
		})
		defer CleanupTest(env)

		// Create render function with array data
		renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":    101,
						"name":  "Product A",
						"price": 29.99,
						"tags":  []string{"sale", "popular"},
					},
					{
						"id":    102,
						"name":  "Product B",
						"price": 49.99,
						"tags":  []string{"new", "featured"},
					},
					{
						"id":    103,
						"name":  "Product C",
						"price": 19.99,
						"tags":  []string{"clearance"},
					},
				},
			}
		}

		// Execute request with render function
		_, _, body := ExecutePHP(t, env, "/array_iteration.php",
			httptest.NewRequest("GET", "/array_iteration.php", nil), renderFn)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Check expected rendered content
		expectedContent := []string{
			"<h2>Items List</h2>",
			"<ul class=\"items-list\">",
			"<li>Item #101: Product A - $29.99 [sale, popular]</li>",
			"<li>Item #102: Product B - $49.99 [new, featured]</li>",
			"<li>Item #103: Product C - $19.99 [clearance]</li>",
			"</ul>",
			"<p>Total items: 3</p>",
			"<p>Total price: $99.97</p>",
		}

		CheckResponseContains(t, body, expectedContent)
	})

	// HTML escaping test
	t.Run("HTML Escaping", func(t *testing.T) {
		escapingPHP := `<?php
			header("Content-Type: text/html; charset=utf-8");
			echo "<!DOCTYPE html><html><body>";
			
			// Test proper escaping of potentially dangerous content
			if (isset($user_input)) {
				echo "<div class=\"user-content\">";
				echo "<h3>Safe Output (Escaped):</h3>";
				echo "<p>" . htmlspecialchars($user_input) . "</p>";
				
				echo "<h3>Raw Output (Unsafe, for testing only):</h3>";
				echo "<p class=\"raw\">" . $user_input . "</p>";
				echo "</div>";
			}
			
			echo "</body></html>";
		?>`

		// Setup test environment
		env := SetupTest(t, map[string]string{
			"escaping.php": escapingPHP,
		})
		defer CleanupTest(env)

		// Create render function with potentially dangerous input
		renderFn := func(w http.ResponseWriter, r *http.Request) map[string]interface{} {
			return map[string]interface{}{
				"user_input": "<script>alert('XSS attack');</script><img src=\"x\" onerror=\"alert('Image XSS')\"> & < > \" '",
			}
		}

		// Execute request with render function
		_, _, body := ExecutePHP(t, env, "/escaping.php",
			httptest.NewRequest("GET", "/escaping.php", nil), renderFn)

		// Check for PHP errors
		AssertNoPHPErrors(t, body)

		// Check that the dangerous content is properly escaped in the safe output
		if !strings.Contains(body, "&lt;script&gt;alert(&#039;XSS attack&#039;);&lt;/script&gt;") {
			t.Errorf("Response does not contain properly escaped script tag")
		}

		if !strings.Contains(body, "&lt;img src=&quot;x&quot; onerror=&quot;alert(&#039;Image XSS&#039;)&quot;&gt;") {
			t.Errorf("Response does not contain properly escaped img tag")
		}

		// Verify special characters are escaped
		if !strings.Contains(body, "&amp; &lt; &gt; &quot; &#039;") {
			t.Errorf("Response does not contain properly escaped special characters")
		}

		// The raw output should contain the unescaped content for verification
		if !strings.Contains(body, "<p class=\"raw\"><script>alert('XSS attack');</script>") {
			t.Errorf("Raw output section is missing or doesn't contain the unescaped content")
		}
	})
}

// TestStreamingResponse tests PHP's streaming output capabilities
func TestStreamingResponse(t *testing.T) {
	streamingPHP := `<?php
		header("Content-Type: text/plain");
		
		// Disable output buffering
		if (ob_get_level()) ob_end_clean();
		
		// Set implicit flush
		ob_implicit_flush(true);
		
		// Output 5 chunks with flushes between them
		for ($i = 1; $i <= 5; $i++) {
			echo "Chunk $i of streaming output\n";
			flush();
			// In a real streaming context, there would be a sleep here
		}
		
		echo "Streaming complete";
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"streaming.php": streamingPHP,
	})
	defer CleanupTest(env)

	// Execute request
	_, _, body := ExecutePHP(t, env, "/streaming.php",
		httptest.NewRequest("GET", "/streaming.php", nil), nil)

	// Check for PHP errors
	AssertNoPHPErrors(t, body)

	// Check that all chunks are in the response
	expectedChunks := []string{
		"Chunk 1 of streaming output",
		"Chunk 2 of streaming output",
		"Chunk 3 of streaming output",
		"Chunk 4 of streaming output",
		"Chunk 5 of streaming output",
		"Streaming complete",
	}

	CheckResponseContains(t, body, expectedChunks)
}
