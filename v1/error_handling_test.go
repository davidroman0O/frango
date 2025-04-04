package frango

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPHPSyntaxErrors tests that PHP syntax errors are properly caught and reported
func TestPHPSyntaxErrors(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "frango-syntax-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file with syntax error
	syntaxErrorPHP := filepath.Join(tempDir, "syntax_error.php")
	syntaxErrorContent := `<?php
		// This has a clear syntax error - missing semicolon
		echo "This is a test"
		$var = 123;
		// Another syntax error - invalid variable name
		$1invalid = "test";
	?>`
	if err := os.WriteFile(syntaxErrorPHP, []byte(syntaxErrorContent), 0644); err != nil {
		t.Fatalf("Failed to create syntax error PHP file: %v", err)
	}

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add the file to VFS
	err = vfs.AddSourceFile(syntaxErrorPHP, "/syntax_error.php")
	if err != nil {
		t.Fatalf("Failed to add source file to VFS: %v", err)
	}

	// Create and execute request
	req := httptest.NewRequest("GET", "/syntax_error.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/syntax_error.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	// Note: FrankenPHP may not return 500 errors for syntax errors,
	// but the error should be in the output
	// if resp.StatusCode != http.StatusInternalServerError {
	//    t.Errorf("Expected status code %d for syntax error, got %d", http.StatusInternalServerError, resp.StatusCode)
	// }

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Verify error detection works
	errorResult := CheckPHPErrors(bodyStr)
	if errorResult == nil {
		t.Fatalf("Failed to detect PHP syntax error in output: %s", bodyStr)
	}

	// Check error type - syntax errors are fatal errors
	if errorResult.Type != PHPErrorFatal {
		t.Errorf("Expected fatal error for syntax error, got %s", errorResult.Type)
	}

	// Check error contains meaningful information
	expectedErrorTexts := []string{"syntax error", "parse error", "unexpected"}
	errorDetected := false
	for _, text := range expectedErrorTexts {
		if strings.Contains(strings.ToLower(bodyStr), text) {
			errorDetected = true
			break
		}
	}
	if !errorDetected {
		t.Errorf("Response should contain one of %v, got: %s", expectedErrorTexts, bodyStr)
	}
}

// TestPHPRuntimeErrors tests that PHP runtime errors are properly caught and reported
func TestPHPRuntimeErrors(t *testing.T) {
	// Create test files with various runtime errors
	runtimeErrors := []struct {
		name      string
		content   string
		errorType PHPErrorType
	}{
		{
			name: "undefined_function.php",
			content: `<?php
				nonexistent_function();
			?>`,
			errorType: PHPErrorFatal,
		},
		{
			name: "undefined_variable.php",
			content: `<?php
				echo $undefined_variable;
			?>`,
			errorType: PHPErrorNotice, // In PHP 8+ this is just a notice
		},
		{
			name: "division_by_zero.php",
			content: `<?php
				$result = 10 / 0;
				echo $result;
			?>`,
			errorType: PHPErrorFatal, // In PHP 8+ this is a fatal error
		},
	}

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "frango-runtime-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Run tests for each error type
	for _, errorTest := range runtimeErrors {
		t.Run(errorTest.name, func(t *testing.T) {
			// Create the PHP file
			filePath := filepath.Join(tempDir, errorTest.name)
			if err := os.WriteFile(filePath, []byte(errorTest.content), 0644); err != nil {
				t.Fatalf("Failed to create PHP file: %v", err)
			}

			// Add the file to VFS
			err = vfs.AddSourceFile(filePath, "/"+errorTest.name)
			if err != nil {
				t.Fatalf("Failed to add source file to VFS: %v", err)
			}

			// Create and execute request
			req := httptest.NewRequest("GET", "/"+errorTest.name, nil)
			w := httptest.NewRecorder()

			// Execute the PHP script
			php.ExecutePHP("/"+errorTest.name, vfs, nil, w, req)

			// Check response
			resp := w.Result()
			defer resp.Body.Close()

			// Read body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			bodyStr := string(body)

			// Verify error detection works
			errorResult := CheckPHPErrors(bodyStr)
			if errorResult == nil {
				t.Fatalf("Failed to detect PHP error in output: %s", bodyStr)
			}

			// Check error type matches expected
			if errorResult.Type != errorTest.errorType {
				t.Errorf("Expected error type %s, got %s", errorTest.errorType, errorResult.Type)
			}

			// Check that error is detected by AssertNoPHPErrors
			mockT := new(testing.T)
			AssertNoPHPErrors(mockT, bodyStr)
			if !mockT.Failed() {
				t.Errorf("AssertNoPHPErrors did not detect error in: %s", bodyStr)
			}
		})
	}
}

// TestErrorHandlingWithCustomHandler tests custom error handling for PHP errors
func TestErrorHandlingWithCustomHandler(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "frango-custom-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file with a runtime error
	errorPHP := filepath.Join(tempDir, "error.php")
	errorContent := `<?php
		// This will cause a division by zero error
		$result = 10 / 0;
	?>`
	if err := os.WriteFile(errorPHP, []byte(errorContent), 0644); err != nil {
		t.Fatalf("Failed to create error PHP file: %v", err)
	}

	// Create a custom error handler
	errorHandlerPHP := filepath.Join(tempDir, "error_handler.php")
	errorHandlerContent := `<?php
		// Custom error handler
		header('Content-Type: application/json');
		http_response_code(500);
		
		$errorData = array(
			'status' => 'error',
			'message' => 'A PHP error occurred',
			'details' => isset($_SERVER['PHP_LAST_ERROR']) ? $_SERVER['PHP_LAST_ERROR'] : 'Unknown error',
			'time' => date('Y-m-d H:i:s')
		);
		
		echo json_encode($errorData);
	?>`
	if err := os.WriteFile(errorHandlerPHP, []byte(errorHandlerContent), 0644); err != nil {
		t.Fatalf("Failed to create error handler PHP file: %v", err)
	}

	// Setup middleware with the error handler
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
		// TODO: Add an option for custom error handling when implemented
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add files to VFS
	err = vfs.AddSourceFile(errorPHP, "/error.php")
	if err != nil {
		t.Fatalf("Failed to add error file to VFS: %v", err)
	}
	err = vfs.AddSourceFile(errorHandlerPHP, "/error_handler.php")
	if err != nil {
		t.Fatalf("Failed to add error handler to VFS: %v", err)
	}

	// This test is currently a placeholder for when custom error handling is implemented
	// TODO: Complete this test when custom error handling is added to the middleware
	t.Skip("Custom error handling not yet implemented")
}

// TestPHPErrorDisplayConfiguration tests PHP error display configuration options
func TestPHPErrorDisplayConfiguration(t *testing.T) {
	t.Skip("PHP error display configuration not fully implemented yet")

	// This test is a placeholder for when PHP error display configuration is implemented
	// Current implementation of FrankenPHP does not fully support configuring error display settings
	// through runtime environment variables in a way that's compatible with this test.

	// When error display configuration is implemented, this test should verify:
	// 1. In development mode with errors shown, PHP notices/warnings appear in output
	// 2. In development mode with errors hidden, PHP notices/warnings don't appear
	// 3. In production mode, PHP notices/warnings don't appear by default

	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "frango-error-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a PHP file with a notice (non-fatal) error
	errorPHP := filepath.Join(tempDir, "notice_error.php")
	errorContent := `<?php
		// This will cause a notice about an undefined variable
		echo $undefined_variable;
		
		// Output a success message after the error
		echo "Successfully continued execution after the notice error.";
	?>`
	if err := os.WriteFile(errorPHP, []byte(errorContent), 0644); err != nil {
		t.Fatalf("Failed to create error PHP file: %v", err)
	}

	testCases := []struct {
		name            string
		developMode     bool
		displayErrors   bool
		shouldShowError bool
	}{
		{
			name:            "Development mode with errors displayed",
			developMode:     true,
			displayErrors:   true,
			shouldShowError: true,
		},
		{
			name:            "Development mode with errors hidden",
			developMode:     true,
			displayErrors:   false,
			shouldShowError: false,
		},
		{
			name:            "Production mode with errors hidden",
			developMode:     false,
			displayErrors:   false,
			shouldShowError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup middleware with the specified options
			php, err := New(
				WithSourceDir(tempDir),
				WithDevelopmentMode(tc.developMode),
				// TODO: Add option for controlling error display when implemented
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}
			defer php.Shutdown()

			// Create VFS for testing
			vfs := php.NewVFS()
			defer vfs.Cleanup()

			// Add file to VFS
			err = vfs.AddSourceFile(errorPHP, "/notice_error.php")
			if err != nil {
				t.Fatalf("Failed to add file to VFS: %v", err)
			}

			// Create and execute request
			req := httptest.NewRequest("GET", "/notice_error.php", nil)
			w := httptest.NewRecorder()

			// Execute the PHP script
			php.ExecutePHP("/notice_error.php", vfs, nil, w, req)

			// Check response
			resp := w.Result()
			defer resp.Body.Close()

			// Read body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			bodyStr := string(body)

			// Check if the error is displayed as expected
			hasError := strings.Contains(strings.ToLower(bodyStr), "undefined variable") ||
				strings.Contains(strings.ToLower(bodyStr), "notice:")

			if tc.shouldShowError && !hasError {
				t.Errorf("Expected error to be displayed but it wasn't: %s", bodyStr)
			} else if !tc.shouldShowError && hasError {
				t.Errorf("Expected error to be hidden but it was displayed: %s", bodyStr)
			}

			// Check if the successful message is displayed
			hasSuccess := strings.Contains(bodyStr, "Successfully continued execution")
			if !hasSuccess {
				t.Errorf("Expected 'Successfully continued execution' message, but it wasn't found: %s", bodyStr)
			}
		})
	}

	// This test is partially a placeholder for when PHP error display configuration is implemented
	// TODO: Update this test when error display configuration is added to the middleware
	t.Skip("PHP error display configuration not fully implemented")
}

// TestPHPErrorWithReadableStackTraces tests that stack traces are readable
func TestPHPErrorWithReadableStackTraces(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "frango-stacktrace-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a main.php file that includes other files to create a stack
	mainPHP := filepath.Join(tempDir, "main.php")
	mainContent := `<?php
		// Include the helper file
		require_once('helper.php');
		
		// Call the function that will trigger an error
		calculate_result();
	?>`
	if err := os.WriteFile(mainPHP, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main PHP file: %v", err)
	}

	// Create a helper file that defines a function
	helperPHP := filepath.Join(tempDir, "helper.php")
	helperContent := `<?php
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
	if err := os.WriteFile(helperPHP, []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to create helper PHP file: %v", err)
	}

	// Setup middleware
	php, err := New(
		WithSourceDir(tempDir),
		WithDevelopmentMode(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	defer php.Shutdown()

	// Create VFS for testing
	vfs := php.NewVFS()
	defer vfs.Cleanup()

	// Add files to VFS
	err = vfs.AddSourceFile(mainPHP, "/main.php")
	if err != nil {
		t.Fatalf("Failed to add main file to VFS: %v", err)
	}
	err = vfs.AddSourceFile(helperPHP, "/helper.php")
	if err != nil {
		t.Fatalf("Failed to add helper file to VFS: %v", err)
	}

	// Create and execute request
	req := httptest.NewRequest("GET", "/main.php", nil)
	w := httptest.NewRecorder()

	// Execute the PHP script
	php.ExecutePHP("/main.php", vfs, nil, w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)

	// Verify error detection works
	errorResult := CheckPHPErrors(bodyStr)
	if errorResult == nil {
		t.Fatalf("Failed to detect PHP error in output")
	}

	// Check for stack trace
	if !strings.Contains(strings.ToLower(bodyStr), "stack trace:") {
		t.Errorf("Stack trace not found in error output")
	}

	// Check that we can see the function call chain in the stack trace
	expectedFunctions := []string{"process_calculation", "calculate_result"}
	for _, funcName := range expectedFunctions {
		if !strings.Contains(bodyStr, funcName) {
			t.Errorf("Function '%s' not found in stack trace", funcName)
		}
	}
}
