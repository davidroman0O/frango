package frango

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckPHPErrors(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		shouldFind  bool
		errorType   PHPErrorType
		errorString string
	}{
		{
			name:       "No errors",
			body:       "This is a valid response with no PHP errors in it.",
			shouldFind: false,
		},
		{
			name:        "Fatal error",
			body:        "Fatal error: Uncaught Error: Call to undefined function non_existent_function()",
			shouldFind:  true,
			errorType:   PHPErrorFatal,
			errorString: "Fatal error:",
		},
		{
			name:        "Parse error",
			body:        "Parse error: syntax error, unexpected '}' in /var/www/html/broken.php on line 5",
			shouldFind:  true,
			errorType:   PHPErrorFatal,
			errorString: "Parse error:",
		},
		{
			name:        "Warning",
			body:        "Warning: include(missing_file.php): Failed to open stream: No such file or directory",
			shouldFind:  true,
			errorType:   PHPErrorWarning,
			errorString: "Warning:",
		},
		{
			name:        "Notice",
			body:        "Notice: Undefined variable: foo in /var/www/html/index.php on line 10",
			shouldFind:  true,
			errorType:   PHPErrorNotice,
			errorString: "Notice:",
		},
		{
			name:        "Undefined index",
			body:        "Some HTML content<br>Notice: Undefined index: missing in /var/www/html/index.php on line 15<br>More content",
			shouldFind:  true,
			errorType:   PHPErrorNotice,
			errorString: "Notice:",
		},
		{
			name:        "Case insensitive",
			body:        "FATAL ERROR: Something went wrong",
			shouldFind:  true,
			errorType:   PHPErrorFatal,
			errorString: "Fatal error:",
		},
		{
			name:        "Error with context",
			body:        "<html><body><h1>Page Title</h1><p>Some content</p><div>Warning: include failed in file.php</div></body></html>",
			shouldFind:  true,
			errorType:   PHPErrorWarning,
			errorString: "Warning:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPHPErrors(tt.body)

			if tt.shouldFind {
				assert.NotNil(t, result, "Should have found an error")
				if result != nil {
					assert.Equal(t, tt.errorType, result.Type, "Wrong error type detected")
					assert.Equal(t, tt.errorString, result.Indicator, "Wrong error indicator detected")
					if result.Indicator == "Fatal error:" && result.Context == "FATAL ERROR: Something went wrong" {
						assert.Contains(t, strings.ToLower(result.Context), strings.ToLower(result.Indicator),
							"Context should contain the error string (case-insensitive)")
					} else {
						assert.Contains(t, result.Context, result.Indicator, "Context should contain the error string")
					}
				}
			} else {
				assert.Nil(t, result, "Should not have found an error")
			}
		})
	}
}

func TestAssertNoPHPErrors(t *testing.T) {
	// Create a mock testing.T to capture failures
	mockT := new(testing.T)

	// Test with no errors - should pass
	AssertNoPHPErrors(mockT, "Valid response with no errors")
	assert.False(t, mockT.Failed(), "Should not fail when no errors present")

	// Test with errors - should fail
	// Create a new mock T since the previous one might be in failed state
	mockT = new(testing.T)
	AssertNoPHPErrors(mockT, "Fatal error: Something went wrong")
	assert.True(t, mockT.Failed(), "Should fail when errors are present")
}

func TestCustomPHPErrorCheck(t *testing.T) {
	// Create custom error patterns
	customPatterns := map[PHPErrorType][]string{
		PHPErrorFatal: {
			"custom fatal error",
		},
		PHPErrorWarning: {
			"custom warning pattern",
		},
	}

	// Create a mock testing.T to capture failures
	mockT := new(testing.T)

	// Test with no errors - should pass
	CustomPHPErrorCheck(mockT, "Valid response with no errors", customPatterns)
	assert.False(t, mockT.Failed(), "Should not fail when no errors present")

	// Test with standard error - should fail
	mockT = new(testing.T)
	CustomPHPErrorCheck(mockT, "Fatal error: Something went wrong", customPatterns)
	assert.True(t, mockT.Failed(), "Should fail when standard errors are present")

	// Test with custom error pattern - should fail
	mockT = new(testing.T)
	CustomPHPErrorCheck(mockT, "This contains a custom warning pattern that should be detected", customPatterns)
	assert.True(t, mockT.Failed(), "Should fail when custom errors are present")
}
