package php

import (
	"strings"
	"testing"
)

// ErrorType represents the type/severity of PHP error
type ErrorType string

const (
	// Error types by severity
	ErrorFatal   ErrorType = "fatal"
	ErrorWarning ErrorType = "warning"
	ErrorNotice  ErrorType = "notice"
)

// ErrorIndicators maps error types to their string indicators in output
var ErrorIndicators = map[ErrorType][]string{
	ErrorFatal: {
		"Fatal error:",
		"Parse error:",
		"Uncaught Exception",
		"Uncaught Error:",
		"Stack trace:",
		"Error:",
		"syntax error",
		"Call to undefined function",
		"Uncaught DivisionByZeroError",
		"Division by zero",
		"Uncaught TypeError",
		"Allowed memory size",
		"Maximum execution time",
	},
	ErrorWarning: {
		"Warning:",
		"include failed",
		"require failed",
	},
	ErrorNotice: {
		"Notice:",
		"Deprecated:",
		"undefined variable",
		"undefined index",
		"undefined offset",
		"undefined constant",
		"undefined array key",
		"Trying to access array offset",
	},
}

// ErrorResult contains details about a detected PHP error
type ErrorResult struct {
	Type      ErrorType
	Indicator string
	Context   string
}

// CheckErrors examines response body for PHP error conditions
// It returns details of the first detected error, or nil if no errors found
func CheckErrors(body string) *ErrorResult {
	// Convert body to lowercase for case-insensitive search
	lowerBody := strings.ToLower(body)

	// Check each category of errors
	for errorType, indicators := range ErrorIndicators {
		for _, indicator := range indicators {
			indicatorLower := strings.ToLower(indicator)
			if strings.Contains(lowerBody, indicatorLower) {
				// Extract context around the error
				index := strings.Index(lowerBody, indicatorLower)
				start := index - 50
				if start < 0 {
					start = 0
				}
				end := index + 250
				if end > len(body) {
					end = len(body)
				}

				// Return error details
				return &ErrorResult{
					Type:      errorType,
					Indicator: indicator,
					Context:   body[start:end],
				}
			}
		}
	}

	// No errors found
	return nil
}

// AssertNoErrors checks for PHP errors in the response body and fails the test if any are found
// This is a convenient wrapper for testing that should be used in all test files
func AssertNoErrors(t *testing.T, body string) {
	t.Helper() // Mark as test helper function to improve test output

	result := CheckErrors(body)
	if result != nil {
		// Log the error context
		t.Logf("Found PHP %s: %s", result.Type, result.Indicator)
		t.Logf("Error context: %s", result.Context)

		// Fail the test - PHP errors should not be present in responses
		t.Errorf("PHP %s detected in response: %s", result.Type, result.Indicator)
	}
}

// CustomErrorCheck allows for additional error patterns to be checked
// Useful for testing specific PHP warnings or notices that may not be in the standard list
func CustomErrorCheck(t *testing.T, body string, additionalPatterns map[ErrorType][]string) {
	t.Helper()

	// First run the standard check
	if result := CheckErrors(body); result != nil {
		t.Logf("Found PHP %s: %s", result.Type, result.Indicator)
		t.Logf("Error context: %s", result.Context)
		t.Errorf("PHP %s detected in response: %s", result.Type, result.Indicator)
		return
	}

	// Then check additional patterns
	lowerBody := strings.ToLower(body)
	for errorType, patterns := range additionalPatterns {
		for _, pattern := range patterns {
			patternLower := strings.ToLower(pattern)
			if strings.Contains(lowerBody, patternLower) {
				// Extract context
				index := strings.Index(lowerBody, patternLower)
				start := index - 50
				if start < 0 {
					start = 0
				}
				end := index + 250
				if end > len(body) {
					end = len(body)
				}

				// Log and fail
				t.Logf("Found custom PHP %s: %s", errorType, pattern)
				t.Logf("Error context: %s", body[start:end])
				t.Errorf("Custom PHP %s detected in response: %s", errorType, pattern)
				return
			}
		}
	}
}
