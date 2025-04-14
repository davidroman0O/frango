package executor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/davidroman0O/frango/v2/pkg/php"
	"github.com/davidroman0O/frango/v2/pkg/vfs"
)

// resolveScriptPath resolves a script path using the cache if available
func (e *Executor) resolveScriptPath(vfs *vfs.VFS, scriptPath string) (string, bool, error) {
	logger := e.config.Logger

	// Try cache first (if not in development mode)
	if !e.config.DevelopmentMode {
		if cachedPath, ok := e.getPathFromCache(scriptPath); ok {
			if logger != nil {
				logger.Printf("Executor: Using cached path for '%s' -> %s", scriptPath, cachedPath)
			}
			return cachedPath, true, nil
		}
	}

	// Perform regular path resolution
	resolvedPath, isVfsPath, err := e.resolveScriptPathInternal(vfs, scriptPath)

	// Cache the result if successful and not in development mode
	if err == nil && !e.config.DevelopmentMode {
		e.cacheResolvedPath(scriptPath, resolvedPath)
	}

	return resolvedPath, isVfsPath, err
}

// resolveScriptPathInternal performs the actual path resolution
func (e *Executor) resolveScriptPathInternal(vfs *vfs.VFS, scriptPath string) (string, bool, error) {
	logger := e.config.Logger

	if logger != nil {
		logger.Printf("Executor: Resolving script path for '%s'", scriptPath)
	}

	// Ensure VFS is available
	if vfs == nil {
		return "", false, fmt.Errorf("VFS is required to resolve script path for '%s'", scriptPath)
	}

	// Try resolving directly within the VFS
	resolvedPath, err := vfs.ResolvePath(scriptPath)
	if err == nil {
		if logger != nil {
			logger.Printf("Executor: Found '%s' in VFS -> %s", scriptPath, resolvedPath)
		}
		return resolvedPath, true, nil
	} else if logger != nil {
		logger.Printf("Executor: Script '%s' not found directly in VFS: %v", scriptPath, err)
	}

	// Try parameterized paths
	if strings.Contains(scriptPath, "{") && strings.Contains(scriptPath, "}") {
		return e.resolveParameterizedPath(vfs, scriptPath, logger)
	}

	// If not found anywhere in VFS, return the error
	if logger != nil {
		logger.Printf("Executor: Script '%s' could not be resolved in VFS.", scriptPath)
	}
	return "", false, fmt.Errorf("script '%s' not found in VFS", scriptPath)
}

// resolveParameterizedPath resolves a script path with parameters
func (e *Executor) resolveParameterizedPath(vfs *vfs.VFS, scriptPath string, logger *log.Logger) (string, bool, error) {
	// For parameterized paths, try literal resolution first
	literalPath, err := vfs.ResolvePathLiteral(scriptPath)
	if err == nil {
		if logger != nil {
			logger.Printf("Executor: Found parameterized script '%s' in VFS -> %s", scriptPath, literalPath)
		}
		return literalPath, true, nil
	} else if logger != nil {
		logger.Printf("Executor: Parameterized script '%s' not found with literal resolution: %v", scriptPath, err)
	}

	// Try to find a matching file pattern
	dirPath := filepath.Dir(scriptPath)
	fileName := filepath.Base(scriptPath)

	// List files in that directory
	files, err := vfs.ListFilesIn(dirPath)
	if err == nil && len(files) > 0 {
		// Look for a matching pattern
		for _, f := range files {
			baseF := filepath.Base(f)
			paramPattern := regexp.MustCompile(`\{[^}]+\}`)
			patternRegex := "^" + paramPattern.ReplaceAllString(regexp.QuoteMeta(fileName), ".*") + "$"
			matched, _ := regexp.MatchString(patternRegex, baseF)

			if matched {
				if logger != nil {
					logger.Printf("Executor: Found matching file pattern: %s", f)
				}
				// Try to resolve this file
				resolvedPath, err := vfs.ResolvePath(f)
				if err == nil {
					return resolvedPath, true, nil
				}
			}
		}
	}

	return "", false, fmt.Errorf("parameterized script '%s' not found in VFS", scriptPath)
}

// getPathFromCache retrieves a cached path
func (e *Executor) getPathFromCache(scriptPath string) (string, bool) {
	if val, ok := e.pathCache.Load(scriptPath); ok {
		return val.(string), true
	}
	return "", false
}

// cacheResolvedPath stores a resolved path in the cache
func (e *Executor) cacheResolvedPath(scriptPath, resolvedPath string) {
	e.pathCache.Store(scriptPath, resolvedPath)
}

// ensurePhpFileExists verifies that the PHP file exists and returns the path
func (e *Executor) ensurePhpFileExists(resolvedPath, scriptPath string) string {
	logger := e.config.Logger

	// Check if the file exists
	if _, err := os.Stat(resolvedPath); err == nil {
		if logger != nil {
			logger.Printf("Using resolved PHP file path: %s", resolvedPath)
		}
		return resolvedPath
	}

	if logger != nil {
		logger.Printf("Warning: Could not access PHP file at %s", resolvedPath)
	}

	// Check if the file exists in alternate locations
	alternativePaths := []string{
		filepath.Join(filepath.Dir(resolvedPath), filepath.Base(scriptPath)),
		scriptPath,
	}

	for _, path := range alternativePaths {
		if _, err := os.Stat(path); err == nil {
			if logger != nil {
				logger.Printf("Found PHP file at alternative path: %s", path)
			}
			return path
		}
	}

	if logger != nil {
		logger.Printf("Error: Failed to locate PHP file for script: %s", scriptPath)
	}
	return ""
}

// ensurePhpGlobals ensures that the PHP globals script is installed in the VFS
func (e *Executor) ensurePhpGlobals(v *vfs.VFS, globalsPath string) error {
	// Check if the globals file already exists
	if v.FileExists(globalsPath) {
		return nil // Already installed
	}

	// Get the PHP globals script from the php package
	provider := &php.StandardGlobalsProvider{}

	// Create a more robust globals script that properly sets up the PHP environment for direct execution
	globalsScript := `<?php
// PHP Globals script for the Go-PHP Executor
// This script sets up the PHP environment for proper script execution

// Reset the output buffer to ensure clean output
if (function_exists('ob_get_level') && ob_get_level() > 0) {
    ob_end_clean();
}
ob_start();

// If SCRIPT_DIR is set, change to that directory to ensure includes work properly
if (isset($_SERVER['SCRIPT_DIR']) && $_SERVER['SCRIPT_DIR'] !== '' && isset($_SERVER['FRANGO_DO_CHDIR']) && $_SERVER['FRANGO_DO_CHDIR'] === '1') {
    if (is_dir($_SERVER['SCRIPT_DIR'])) {
        // Change working directory to script directory for proper include resolution
        chdir($_SERVER['SCRIPT_DIR']);
        
        // For debugging
        // error_log("Changed working directory to: " . $_SERVER['SCRIPT_DIR']);
        // error_log("Current working directory: " . getcwd());
    }
}

// Define a class to enable overriding __FILE__ and __DIR__ magic constants
class FrangoConstants {
    public static function getLogicalFile() {
        return isset($_SERVER['FRANGO_LOGICAL_FILENAME']) ? $_SERVER['FRANGO_LOGICAL_FILENAME'] : __FILE__;
    }
    
    public static function getLogicalDir() {
        return isset($_SERVER['FRANGO_LOGICAL_DIR']) ? $_SERVER['FRANGO_LOGICAL_DIR'] : __DIR__;
    }
}

// Store the logical file and directory values for external access
$GLOBALS['FRANGO_LOGICAL_FILE'] = FrangoConstants::getLogicalFile();
$GLOBALS['FRANGO_LOGICAL_DIR'] = FrangoConstants::getLogicalDir();

// Set up custom error handler to capture fatal errors
set_error_handler(function($errno, $errstr, $errfile, $errline) {
    // Only handle fatal errors that would halt execution
    if ($errno == E_ERROR || $errno == E_PARSE || $errno == E_CORE_ERROR || 
        $errno == E_COMPILE_ERROR || $errno == E_USER_ERROR) {
        
        // Save error information in environment variables
        putenv("PHP_LAST_ERROR=" . $errstr);
        putenv("PHP_ERROR_TYPE=fatal");
        putenv("PHP_ERROR_CONTEXT=Error in " . $errfile . " on line " . $errline);
        
        // Log the error
        error_log("PHP Fatal Error: " . $errstr . " in " . $errfile . " on line " . $errline);
    }
    
    // Return false to allow the standard PHP error handler to run
    return false;
});

// Merge path parameters into $_GET for compatibility
if (isset($_SERVER['_PATH']) && $_SERVER['_PATH'] !== '{}') {
    $_PATH = json_decode($_SERVER['_PATH'], true);
    if (is_array($_PATH)) {
        // Create the $_PATH global for backwards compatibility
        $GLOBALS['_PATH'] = $_PATH;
        
        // Merge path parameters into $_GET (path parameters take precedence)
        $_GET = array_merge($_GET, $_PATH);
        
        // Also update $_REQUEST
        $_REQUEST = array_merge($_REQUEST, $_PATH);
    }
}

// Make additional environment data available to the script
if (isset($_SERVER['_JSON']) && $_SERVER['_JSON'] !== '{}') {
    $_JSON = json_decode($_SERVER['_JSON'], true);
    $GLOBALS['_JSON'] = $_JSON;
}

// Helper function to get current logical file path
if (!function_exists('get_current_script_path')) {
    function get_current_script_path() {
        return $GLOBALS['FRANGO_LOGICAL_FILE'];
    }
}

// Helper function to get current logical directory path
if (!function_exists('get_current_script_dir')) {
    function get_current_script_dir() {
        return $GLOBALS['FRANGO_LOGICAL_DIR'];
    }
}

// Clean up any previous script execution state
if (function_exists('session_status') && session_status() === PHP_SESSION_ACTIVE) {
    session_write_close();
}

// Reset user-defined variables to avoid leaking state between requests
unset($GLOBALS['_frango_include_state']);
unset($GLOBALS['_frango_request_state']);

// Register shutdown function to flush output
register_shutdown_function(function() {
    if (function_exists('ob_get_level') && ob_get_level() > 0) {
        while (ob_get_level() > 0) {
            ob_end_flush();
        }
    }
});

?>
// Load globals from the PHP package
` + provider.GetScript()

	// Install the globals script
	return v.CreateVirtualFile(globalsPath, []byte(globalsScript))
}
