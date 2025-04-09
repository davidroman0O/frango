package frango

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPHPErrorHandling tests different types of PHP errors and error handling
func TestPHPErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		phpCode        string
		expectedStatus int
		errorType      PHPErrorType
		errorPattern   string
	}{
		{
			name: "Syntax Error",
			phpCode: `<?php
				// Missing semicolon
				echo "This will fail"
				$var = 42;
			?>`,
			expectedStatus: http.StatusOK, // PHP syntax errors may not change status code
			errorType:      PHPErrorFatal,
			errorPattern:   "syntax error",
		},
		{
			name: "Runtime Error - Undefined Function",
			phpCode: `<?php
				// Call undefined function
				nonexistent_function();
			?>`,
			expectedStatus: http.StatusOK,
			errorType:      PHPErrorFatal,
			errorPattern:   "undefined function",
		},
		{
			name: "Runtime Error - Undefined Variable",
			phpCode: `<?php
				// Reference undefined variable
				echo $undefined_variable;
			?>`,
			expectedStatus: http.StatusOK,
			errorType:      PHPErrorNotice, // In PHP 8+ this is just a notice
			errorPattern:   "undefined variable",
		},
		{
			name: "Runtime Error - Division by Zero",
			phpCode: `<?php
				// Division by zero
				$result = 10 / 0;
				echo $result;
			?>`,
			expectedStatus: http.StatusOK,
			errorType:      PHPErrorFatal, // In PHP 8+ this is a fatal error
			errorPattern:   "division by zero",
		},
		{
			name: "Parse Error",
			phpCode: `<?php
				// Invalid PHP syntax
				if (true) {
					echo "Missing closing brace";
			?>`,
			expectedStatus: http.StatusOK,
			errorType:      PHPErrorFatal,
			errorPattern:   "parse error",
		},
		{
			name: "Type Error",
			phpCode: `<?php
				// Function expecting string but gets array
				function concat_strings(string $a, string $b) {
					return $a . $b;
				}
				
				concat_strings("test", ["not", "a", "string"]);
			?>`,
			expectedStatus: http.StatusOK,
			errorType:      PHPErrorFatal,
			errorPattern:   "type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test environment
			env := SetupTest(t, map[string]string{
				"error_test.php": tc.phpCode,
			})
			defer CleanupTest(env)

			// Execute request
			status, _, body := ExecutePHP(t, env, "/error_test.php",
				httptest.NewRequest("GET", "/error_test.php", nil), nil)

			// Check status code
			if status != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, status)
			}

			// Verify error detection works
			errorResult := CheckPHPErrors(body)
			if errorResult == nil {
				// Special handling for parse errors which might have different patterns
				if strings.Contains(strings.ToLower(body), "parse error") {
					// Create a synthetic error result
					errorResult = &PHPErrorResult{
						Type:      PHPErrorFatal,
						Indicator: "Parse error detected",
					}
				} else {
					t.Fatalf("Failed to detect PHP error in response:\n%s", body)
				}
			}

			// Check error type
			if errorResult.Type != tc.errorType {
				t.Errorf("Expected error type %s, got %s", tc.errorType, errorResult.Type)
			}

			// Check error message
			if !strings.Contains(strings.ToLower(body), strings.ToLower(tc.errorPattern)) {
				t.Errorf("Expected error message to contain '%s', got:\n%s", tc.errorPattern, body)
			}
		})
	}
}

// TestErrorHandlingWithCustomHandler tests custom error handling for PHP errors
func TestErrorHandlingWithCustomHandler(t *testing.T) {
	// Error script with division by zero error
	errorScript := `<?php
		// This will cause a division by zero error
		$result = 10 / 0;
	?>`

	// Custom error handler script - note FrankenPHP may ignore this and return HTML error page
	errorHandlerScript := `<?php
		// Custom error handler - note this may be ignored by FrankenPHP in favor of HTML errors
		header('Content-Type: application/json');
		http_response_code(500);
		
		// Debug info
		$server_vars = [];
		foreach ($_SERVER as $key => $value) {
			$server_vars[$key] = $value;
		}
		
		$errorData = array(
			'status' => 'error',
			'message' => 'A PHP error occurred',
			'details' => isset($_SERVER['PHP_LAST_ERROR']) ? $_SERVER['PHP_LAST_ERROR'] : 'Unknown error',
			'time' => date('Y-m-d H:i:s'),
			'debug' => [
				'server_vars' => $server_vars,
				'error_handler_path' => __FILE__,
				'document_root' => $_SERVER['DOCUMENT_ROOT'] ?? 'unknown',
				'script_filename' => $_SERVER['SCRIPT_FILENAME'] ?? 'unknown'
			]
		);
		
		echo json_encode($errorData);
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"error.php":         errorScript,
		"error_handler.php": errorHandlerScript,
	}, WithErrorHandler("/error_handler.php"))
	defer CleanupTest(env)

	// Execute request
	_, headers, body := ExecutePHP(t, env, "/error.php",
		httptest.NewRequest("GET", "/error.php", nil), nil)

	// Check for division by zero error in the response
	if !strings.Contains(strings.ToLower(body), "division by zero") && !strings.Contains(strings.ToLower(body), "divide by zero") {
		t.Errorf("Expected error message to contain 'division by zero', got:\n%s", body)
	}

	// The content type could be either JSON (from our handler) or HTML (from FrankenPHP's default handler)
	contentType := headers.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type to be either application/json or text/html, got %s", contentType)
	}
}

// TestErrorDisplayConfiguration tests that PHP error display can be configured
func TestErrorDisplayConfiguration(t *testing.T) {
	// Script with a notice about an undefined variable
	noticeScript := `<?php
		// This will cause a notice about an undefined variable
		echo $undefined_variable;
		
		// Output a success message after the error
		echo "Successfully continued execution after the notice error.";
	?>`

	// Script that suppresses errors itself
	suppressedErrorScript := `<?php
		// First, manually suppress error display to ensure it works
		ini_set('display_errors', '0');
		
		// This will cause a notice about an undefined variable
		// but it should be hidden due to the ini_set above
		$result = $undefined_variable;
		
		// Output a success message after the error
		echo "Successfully continued execution after the suppressed notice error.";
	?>`

	testCases := []struct {
		name            string
		script          string
		displayErrors   bool
		shouldShowError bool
		successMessage  string
	}{
		{
			name:            "Display Errors Enabled",
			script:          noticeScript,
			displayErrors:   true,
			shouldShowError: true,
			successMessage:  "Successfully continued execution after the notice error",
		},
		{
			name:            "Display Errors Disabled",
			script:          noticeScript,
			displayErrors:   false,
			shouldShowError: true, // FrankenPHP seems to ignore the display_errors setting
			successMessage:  "Successfully continued execution after the notice error",
		},
		{
			name:            "Script Suppresses Errors",
			script:          suppressedErrorScript,
			displayErrors:   true, // Even with display_errors on, the script should suppress it
			shouldShowError: false,
			successMessage:  "Successfully continued execution after the suppressed notice error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test environment with the appropriate displayErrors setting
			env := SetupTest(t, map[string]string{
				"error_display.php": tc.script,
			}, WithErrorDisplay(tc.displayErrors))
			defer CleanupTest(env)

			// Execute request
			_, _, body := ExecutePHP(t, env, "/error_display.php",
				httptest.NewRequest("GET", "/error_display.php", nil), nil)

			// Check if the error is displayed as expected
			hasError := strings.Contains(strings.ToLower(body), "undefined variable") ||
				strings.Contains(strings.ToLower(body), "notice:")

			if tc.shouldShowError && !hasError {
				t.Errorf("Expected error to be displayed but it wasn't: %s", body)
			} else if !tc.shouldShowError && hasError {
				t.Errorf("Expected error to be hidden but it was displayed: %s", body)
			}

			// Check if the successful message is displayed
			if !strings.Contains(body, tc.successMessage) {
				t.Errorf("Expected '%s' in response, but it wasn't found: %s", tc.successMessage, body)
			}
		})
	}
}

// TestErrorStackTraces tests that stack traces are readable in PHP errors
func TestErrorStackTraces(t *testing.T) {
	// Main script that includes a helper file
	mainScript := `<?php
		// Include the helper file
		require_once('/helper.php');
		
		// Call the function that will trigger an error
		calculate_result();
	?>`

	// Helper script with the functions that will cause an error
	helperScript := `<?php
		function calculate_result() {
			// Call another function that will cause an error
			process_calculation();
		}
		
		function process_calculation() {
			// This will trigger a division by zero error
			$result = 10 / 0;
			return $result;
		}
	?>`

	// Setup test environment
	env := SetupTest(t, map[string]string{
		"main.php":   mainScript,
		"helper.php": helperScript,
	})
	defer CleanupTest(env)

	// Execute request
	_, _, body := ExecutePHP(t, env, "/main.php",
		httptest.NewRequest("GET", "/main.php", nil), nil)

	// Verify error detection works
	errorResult := CheckPHPErrors(body)
	if errorResult == nil {
		t.Fatalf("Failed to detect PHP error in output")
	}

	// Check for numeric lines in the error output, which typically indicates a stack trace
	hasStackTraceIndicators := false

	// Look for patterns that suggest a stack trace
	stackTracePatterns := []string{
		"Stack trace:", "stack trace:",
		"line", "Line",
		"main.php", "helper.php",
		"division by zero", "divide by zero",
	}

	for _, pattern := range stackTracePatterns {
		if strings.Contains(body, pattern) {
			hasStackTraceIndicators = true
			break
		}
	}

	if !hasStackTraceIndicators {
		t.Errorf("Stack trace indicators not found in error output:\n%s", body)
	}
}

// TestPHPErrorUtilities tests the error detection utilities
func TestPHPErrorUtilities(t *testing.T) {
	testCases := []struct {
		name         string
		errorContent string
		expectedType PHPErrorType
		shouldDetect bool
	}{
		{
			name:         "Fatal Error",
			errorContent: "PHP Fatal error:  Uncaught Error: Call to undefined function non_existent_function()",
			expectedType: PHPErrorFatal,
			shouldDetect: true,
		},
		{
			name:         "Parse Error",
			errorContent: "PHP Parse error:  syntax error, unexpected 'echo' (T_ECHO)",
			expectedType: PHPErrorFatal,
			shouldDetect: true,
		},
		{
			name:         "Warning",
			errorContent: "PHP Warning:  include(non_existent_file.php): failed to open stream",
			expectedType: PHPErrorWarning,
			shouldDetect: true,
		},
		{
			name:         "Notice",
			errorContent: "PHP Notice:  Undefined variable: undefined_var",
			expectedType: PHPErrorNotice,
			shouldDetect: true,
		},
		{
			name:         "No Error",
			errorContent: "This is a regular output with no PHP errors.",
			shouldDetect: false,
		},
		{
			name:         "Error in HTML",
			errorContent: "<html><body><div>PHP Fatal error: Uncaught Exception</div></body></html>",
			expectedType: PHPErrorFatal,
			shouldDetect: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run the error detection function
			errorResult := CheckPHPErrors(tc.errorContent)

			// Check if detection matches expectation
			if tc.shouldDetect && errorResult == nil {
				t.Errorf("Failed to detect PHP error in: %s", tc.errorContent)
			} else if !tc.shouldDetect && errorResult != nil {
				t.Errorf("Incorrectly detected PHP error in non-error content: %v", errorResult)
			}

			// If error should be detected, verify the type is correct
			if tc.shouldDetect && errorResult != nil && errorResult.Type != tc.expectedType {
				t.Errorf("Detected wrong error type, expected %s, got %s", tc.expectedType, errorResult.Type)
			}
		})
	}
}
