package php

import (
	"strings"
	"testing"
)

// Test the PHP error detection functionality
func TestErrorDetection(t *testing.T) {
	testCases := []struct {
		name         string
		errorContent string
		expectedType ErrorType
		shouldDetect bool
	}{
		{
			name:         "Fatal Error",
			errorContent: "PHP Fatal error:  Uncaught Error: Call to undefined function non_existent_function()",
			expectedType: ErrorFatal,
			shouldDetect: true,
		},
		{
			name:         "Parse Error",
			errorContent: "PHP Parse error:  syntax error, unexpected 'echo' (T_ECHO)",
			expectedType: ErrorFatal,
			shouldDetect: true,
		},
		{
			name:         "Warning",
			errorContent: "PHP Warning:  include(non_existent_file.php): failed to open stream",
			expectedType: ErrorWarning,
			shouldDetect: true,
		},
		{
			name:         "Notice",
			errorContent: "PHP Notice:  Undefined variable: undefined_var",
			expectedType: ErrorNotice,
			shouldDetect: true,
		},
		{
			name:         "Deprecated",
			errorContent: "PHP Deprecated:  The each() function is deprecated in PHP 8.0",
			expectedType: ErrorNotice, // In our system, Deprecated is considered a Notice
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
			expectedType: ErrorFatal,
			shouldDetect: true,
		},
		{
			name:         "FrankenPHP Error Format",
			errorContent: "<br />\n<b>Fatal error</b>:  Uncaught Error: Call to undefined function",
			expectedType: ErrorFatal,
			shouldDetect: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run the error detection function
			errorResult := CheckErrors(tc.errorContent)

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

// TestExtractErrorDetails tests the extraction of details from PHP error messages
func TestExtractErrorDetails(t *testing.T) {
	testCases := []struct {
		name          string
		errorContent  string
		expectedMsg   string
		shouldExtract bool
	}{
		{
			name:          "Standard PHP Error",
			errorContent:  "PHP Fatal error:  Uncaught Error: Call to undefined function non_existent_function() in /var/www/html/test.php on line 10",
			expectedMsg:   "Uncaught Error: Call to undefined function non_existent_function()",
			shouldExtract: true,
		},
		{
			name:          "HTML Formatted Error",
			errorContent:  "<br />\n<b>Fatal error</b>:  Uncaught Error: Division by zero in <b>/var/www/html/division.php</b> on line <b>5</b><br />",
			expectedMsg:   "Uncaught Error: Division by zero",
			shouldExtract: true,
		},
		{
			name:          "No Line Number",
			errorContent:  "PHP Parse error: syntax error, unexpected end of file",
			expectedMsg:   "syntax error, unexpected end of file",
			shouldExtract: true,
		},
		{
			name:          "No Error",
			errorContent:  "This is regular output with no errors",
			shouldExtract: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Detect the error
			errorResult := CheckErrors(tc.errorContent)

			// Check if extraction should happen
			if !tc.shouldExtract {
				if errorResult != nil {
					t.Errorf("Extracted details from non-error content: %v", errorResult)
				}
				return
			}

			// Verify we have an error result
			if errorResult == nil {
				t.Fatalf("Failed to detect error in: %s", tc.errorContent)
			}

			// Check error message/indicator
			if tc.expectedMsg != "" && !strings.Contains(errorResult.Indicator, tc.expectedMsg) {
				t.Errorf("Expected message containing '%s', got '%s'", tc.expectedMsg, errorResult.Indicator)
			}
		})
	}
}
