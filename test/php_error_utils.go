package test

import (
	"strings"
	"testing"
)

// PHPErrorType represents the type/severity of PHP error
type PHPErrorType string

const (
	// PHP error types by severity
	PHPErrorFatal   PHPErrorType = "fatal"
	PHPErrorWarning PHPErrorType = "warning"
	PHPErrorNotice  PHPErrorType = "notice"
)

// PHPErrorIndicators maps error types to their string indicators in output
var PHPErrorIndicators = map[PHPErrorType][]string{
	PHPErrorFatal: {
		"Fatal error:",
		"Parse error:",
		"Uncaught Exception",
		"Stack trace:",
		"Error:",
		"syntax error",
	},
	PHPErrorWarning: {
		"Warning:",
		"include failed",
		"require failed",
	},
	PHPErrorNotice: {
		"Notice:",
		"Deprecated:",
		"undefined variable",
		"undefined index",
		"undefined offset",
		"undefined constant",
		"undefined array key",
	},
}

// PHPErrorResult contains details about a detected PHP error
type PHPErrorResult struct {
	Type      PHPErrorType
	Indicator string
	Context   string
}

// CheckPHPErrors examines response body for PHP error conditions
// It returns details of the first detected error, or nil if no errors found
func CheckPHPErrors(body string) *PHPErrorResult {
	// Convert body to lowercase for case-insensitive search
	lowerBody := strings.ToLower(body)

	// Check each category of errors
	for errorType, indicators := range PHPErrorIndicators {
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
				return &PHPErrorResult{
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

// AssertNoPHPErrors checks for PHP errors in the response body and fails the test if any are found
// This is a convenient wrapper for testing that should be used in all test files
func AssertNoPHPErrors(t *testing.T, body string) {
	t.Helper() // Mark as test helper function to improve test output

	result := CheckPHPErrors(body)
	if result != nil {
		// Log the error context
		t.Logf("Found PHP %s: %s", result.Type, result.Indicator)
		t.Logf("Error context: %s", result.Context)

		// Fail the test - PHP errors should not be present in responses
		t.Errorf("PHP %s detected in response: %s", result.Type, result.Indicator)
	}
}

// CustomPHPErrorCheck allows for additional error patterns to be checked
// Useful for testing specific PHP warnings or notices that may not be in the standard list
func CustomPHPErrorCheck(t *testing.T, body string, additionalPatterns map[PHPErrorType][]string) {
	t.Helper()

	// First run the standard check
	if result := CheckPHPErrors(body); result != nil {
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
