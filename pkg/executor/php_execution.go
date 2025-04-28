package executor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidroman0O/frango/pkg/php"
	"github.com/dunglas/frankenphp"
)

// executePhpWithRecorder executes a PHP script using FrankenPHP and returns the recorder, exit code, and any error result.
func executePhpWithRecorder(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) (*httptest.ResponseRecorder, int, error) {
	if logger != nil {
		logger.Printf("Executor: Executing PHP script: %s", scriptPath)
		logger.Printf("Executor: Total PHP environment variables: %d", len(env))
	}

	// Prepare PHP environment
	request, err := preparePhpRequest(ctx, scriptPath, env, r, logger)
	if err != nil {
		return nil, 1, err
	}

	// Execute and capture response
	recorder, execErr := executePhpRequest(request, logger)

	// Process PHP output and check for errors
	exitCode, resultErr := processPhpOutput(recorder, execErr, logger)

	return recorder, exitCode, resultErr
}

// preparePhpRequest prepares the PHP request with necessary environment
func preparePhpRequest(ctx context.Context, scriptPath string, env map[string]string, r *http.Request, logger *log.Logger) (*http.Request, error) {
	// Determine document root from environment
	documentRoot := env["DOCUMENT_ROOT"]

	// Add output buffering control
	addOutputBuffering(env)

	// Prepare request path
	reqClone := cloneRequest(ctx, r, env, logger)

	// Create FrankenPHP request with environment
	phpRequest, err := createFrankenPhpRequest(reqClone, documentRoot, env)
	if err != nil {
		if logger != nil {
			logger.Printf("Executor: Error creating PHP request: %v", err)
		}
		return nil, fmt.Errorf("failed to create PHP request: %w", err)
	}

	return phpRequest, nil
}

// addOutputBuffering adds output buffering control to PHP environment
func addOutputBuffering(env map[string]string) {
	// Ensure PHP's output buffer is flushed and reset
	if env["auto_prepend_text"] == "" {
		env["auto_prepend_text"] = "<?php ob_clean(); ?>"
	} else {
		env["auto_prepend_text"] = "<?php ob_clean(); ?>" + env["auto_prepend_text"]
	}
}

// cloneRequest prepares a request clone with modified path
func cloneRequest(ctx context.Context, r *http.Request, env map[string]string, logger *log.Logger) *http.Request {
	reqClone := r.Clone(ctx)
	physicalPath := env["SCRIPT_FILENAME"]
	// Align with physical path strategy: Use basename of the physical path
	reqClone.URL.Path = "/" + filepath.Base(physicalPath)

	if logger != nil {
		logger.Printf("Executor: Modified request path for FrankenPHP: %s (based on physical path: %s)", reqClone.URL.Path, physicalPath)
	}

	return reqClone
}

// createFrankenPhpRequest creates a FrankenPHP request with proper configuration
func createFrankenPhpRequest(r *http.Request, documentRoot string, env map[string]string) (*http.Request, error) {
	return frankenphp.NewRequestWithContext(
		r,
		frankenphp.WithRequestDocumentRoot(documentRoot, false),
		frankenphp.WithRequestEnv(env),
	)
}

// executePhpRequest executes the PHP script and returns the recorder
func executePhpRequest(request *http.Request, logger *log.Logger) (*httptest.ResponseRecorder, error) {
	// Create a recorder to capture output
	recorder := httptest.NewRecorder()

	// Execute the PHP script
	execErr := frankenphp.ServeHTTP(recorder, request)
	if execErr != nil && logger != nil {
		logger.Printf("Executor: Error executing PHP script: %v", execErr)
	}

	return recorder, execErr
}

// processPhpOutput processes PHP output and checks for errors
func processPhpOutput(recorder *httptest.ResponseRecorder, execErr error, logger *log.Logger) (int, error) {
	// Get response body
	respBodyBytes := recorder.Body.Bytes()
	respBodyString := string(respBodyBytes)

	// Check for PHP errors in content
	phpErrorResult := php.CheckErrors(respBodyString)

	// Check specifically for division by zero, which might be missed by simple string search
	// if it's formatted differently (e.g., Uncaught DivisionByZeroError)
	// php.CheckErrors should catch this via ErrorIndicators, but we double-check.
	if phpErrorResult == nil && strings.Contains(strings.ToLower(respBodyString), "division by zero") {
		// Ensure the indicator actually contains the expected text
		phpErrorResult = &php.ErrorResult{
			Type:      php.ErrorFatal,
			Indicator: "Division by zero", // Set the indicator to the expected string
			Context:   respBodyString,     // Provide full context
		}
	}

	// Determine exit code
	exitCode := 0
	if execErr != nil {
		exitCode = 1
	}

	// Log output in debug mode
	logPhpOutput(logger, respBodyBytes)

	// Create combined error result if needed
	var resultErr error
	if execErr != nil || phpErrorResult != nil {
		resultErr = &PHPExecutionError{
			ExecErr:        execErr,
			PHPErrorResult: phpErrorResult,
			RawOutput:      respBodyString, // Store raw output
		}
		// Log the detected PHP error type and indicator
		if logger != nil && phpErrorResult != nil {
			logger.Printf("Executor: PHP execution error detected: PHP %s: %s (Exit Code: %d)",
				phpErrorResult.Type, phpErrorResult.Indicator, exitCode)
		}
	}

	return exitCode, resultErr
}

// logPhpOutput logs PHP output for debugging
func logPhpOutput(logger *log.Logger, output []byte) {
	if logger == nil || len(output) == 0 || !strings.Contains(os.Getenv("LOG_LEVEL"), "DEBUG") {
		return
	}

	if len(output) > 200 {
		logger.Printf("Executor: PHP output (truncated): %s...", output[:200])
	} else {
		logger.Printf("Executor: PHP output: %s", output)
	}
}
