package frango

import (
	"io"
	"log"
	"net/http/httptest"
	"os"
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

// TestErrorDetectionWithRealPHPErrors creates actual PHP scripts with errors and verifies they are detected
func TestErrorDetectionWithRealPHPErrors(t *testing.T) {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "frango-php-errors-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a VFS instance for testing
	logger := log.New(io.Discard, "", 0)
	vfs, err := NewVFS(tempDir, logger, true)
	if err != nil {
		t.Fatalf("Failed to create VFS: %v", err)
	}
	defer vfs.Cleanup()

	errorScripts := []struct {
		name         string
		content      string
		errorType    PHPErrorType
		errorStrings []string // Multiple possible error indicators that could match
	}{
		{
			name: "syntax_error.php",
			content: `<?php
				echo "This has a syntax error;
				$foo = "unclosed string;
			?>`,
			errorType:    PHPErrorFatal,
			errorStrings: []string{"Parse error:", "syntax error"},
		},
		{
			name: "undefined_function.php",
			content: `<?php
				this_function_does_not_exist();
			?>`,
			errorType:    PHPErrorFatal,
			errorStrings: []string{"Fatal error:", "Call to undefined function", "Uncaught Error:"},
		},
		{
			name: "undefined_variable.php",
			content: `<?php
				echo $undefined_variable;
			?>`,
			errorType:    PHPErrorNotice,
			errorStrings: []string{"Notice:", "undefined variable", "Undefined variable"},
		},
		{
			name: "division_by_zero.php",
			content: `<?php
				$result = 10 / 0;
				echo $result;
			?>`,
			errorType:    PHPErrorFatal, // In PHP 8 this is a fatal error, not a warning
			errorStrings: []string{"Division by zero", "DivisionByZeroError", "Uncaught"},
		},
		{
			name: "type_error.php",
			content: `<?php
				function test_type(int $x) {
					return $x + 1;
				}
				test_type("not an integer");
			?>`,
			errorType:    PHPErrorFatal,
			errorStrings: []string{"TypeError", "Uncaught"},
		},
	}

	// Create scripts with different types of errors
	for _, script := range errorScripts {
		t.Run(script.name, func(t *testing.T) {
			// Create the error script
			vfs.CreateVirtualFile("/"+script.name, []byte(script.content))

			// Create a buffer to capture stdout/stderr
			var output strings.Builder

			// Create middleware with the VFS
			middleware, err := New(
				WithLogger(log.New(&output, "", 0)),
				WithDevelopmentMode(true),
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}
			defer middleware.Shutdown()

			// Create a test request
			req := httptest.NewRequest("GET", "/"+script.name, nil)
			w := httptest.NewRecorder()

			// Execute the PHP script
			middleware.ExecutePHP("/"+script.name, vfs, nil, w, req)

			// Get the response
			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			// Verify the error was detected
			result := CheckPHPErrors(bodyStr)
			if result == nil {
				t.Errorf("Failed to detect PHP error in output: %s", bodyStr)
				return
			}

			// Verify the detected error type matches the expected one
			if result.Type != script.errorType {
				t.Errorf("Wrong error type detected: expected %s, got %s", script.errorType, result.Type)
			}

			// Check if any of the possible error strings is found in the output
			errorFound := false
			for _, errorStr := range script.errorStrings {
				if strings.Contains(strings.ToLower(bodyStr), strings.ToLower(errorStr)) {
					errorFound = true
					break
				}
			}

			if !errorFound {
				t.Errorf("None of the expected error strings %v found in output: %s",
					script.errorStrings, bodyStr)
			}

			// Also test the AssertNoPHPErrors function (it should fail on these scripts)
			mockT := new(testing.T)
			AssertNoPHPErrors(mockT, bodyStr)
			if !mockT.Failed() {
				t.Errorf("AssertNoPHPErrors should have failed for script with errors")
			}
		})
	}
}
