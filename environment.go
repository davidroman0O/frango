package frango

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// phpEnvironment represents a complete PHP execution environment
type phpEnvironment struct {
	ID               string
	OriginalPath     string // Absolute path to the source script
	EndpointPath     string // Key used for cache lookup (usually OriginalPath)
	TempPath         string // Path to the isolated temp dir for this env
	ScriptRelPath    string // Relative path of the main script within the temp dir
	LastUpdated      time.Time
	OriginalFileHash string // Hash of OriginalPath content
	mutex            sync.Mutex
}

// environmentCache manages all PHP execution environments
type environmentCache struct {
	sourceDir       string                     // User's main source dir
	baseDir         string                     // Base temp dir for this frango instance
	embedDir        string                     // Subdir in baseDir for embedded files (_frango_embeds)
	globalLibraries map[string]string          // relPath in env -> abs path on disk (_frango_embeds/...)
	environments    map[string]*phpEnvironment // Keyed by EndpointPath (abs script path)
	mutex           sync.RWMutex
	logger          *log.Logger
	developmentMode bool
}

// newEnvironmentCache creates a new environment cache
func newEnvironmentCache(sourceDir string, baseDir string, logger *log.Logger, developmentMode bool) *environmentCache {
	embedDir := filepath.Join(baseDir, "_frango_embeds")
	return &environmentCache{
		sourceDir:       sourceDir,
		baseDir:         baseDir,
		embedDir:        embedDir,
		environments:    make(map[string]*phpEnvironment),
		globalLibraries: make(map[string]string),
		logger:          logger,
		developmentMode: developmentMode,
	}
}

// AddGlobalLibrary tracks an embedded library file.
func (c *environmentCache) AddGlobalLibrary(targetRelPath string, sourceDiskPath string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.globalLibraries[targetRelPath] = sourceDiskPath
	c.logger.Printf("Tracking global library: %s -> %s", targetRelPath, sourceDiskPath)
}

// GetEnvironment retrieves or creates an environment for a specific PHP script.
// endpointPath key is typically the absolute path to the script.
func (c *environmentCache) GetEnvironment(endpointPath string, originalAbsPath string) (*phpEnvironment, error) {
	// Ensure no query strings in original path
	cleanOriginalPath := originalAbsPath
	if queryIndex := strings.Index(cleanOriginalPath, "?"); queryIndex != -1 {
		cleanOriginalPath = cleanOriginalPath[:queryIndex]
	}

	c.mutex.RLock()
	env, exists := c.environments[endpointPath]
	c.mutex.RUnlock()

	if exists {
		if c.developmentMode {
			if err := c.updateEnvironmentIfNeeded(env); err != nil {
				c.logger.Printf("Warning: Failed to update environment for %s: %v", endpointPath, err)
			}
		}
		return env, nil
	}

	// Create a new environment
	env, err := c.createEnvironment(endpointPath, cleanOriginalPath)
	if err != nil {
		return nil, err
	}

	// Store the environment
	c.mutex.Lock()
	c.environments[endpointPath] = env
	c.mutex.Unlock()

	return env, nil
}

// createEnvironment creates a new PHP execution environment
func (c *environmentCache) createEnvironment(endpointPath string, originalAbsPath string) (*phpEnvironment, error) {
	// Create a unique ID based *only* on a hash of the defining path
	h := sha256.Sum256([]byte(endpointPath))
	// Use a significant portion of the hash for the directory name to avoid collisions
	id := hex.EncodeToString(h[:16]) // Use first 16 bytes (32 hex chars)

	tempPath := filepath.Join(c.baseDir, id)
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return nil, fmt.Errorf("error creating environment directory '%s': %w", tempPath, err)
	}

	// Calculate initial file hash of the main script
	initialHash, err := calculateFileHash(originalAbsPath)
	if err != nil {
		os.RemoveAll(tempPath)
		return nil, fmt.Errorf("failed to calculate initial hash for '%s': %w", originalAbsPath, err)
	}

	// Calculate relative path BEFORE creating env struct
	relScriptPath, err := c.calculateRelPath(originalAbsPath)
	if err != nil {
		os.RemoveAll(tempPath)
		return nil, fmt.Errorf("cannot determine relative path for script '%s': %w", originalAbsPath, err)
	}

	env := &phpEnvironment{
		ID:               id,
		OriginalPath:     originalAbsPath,
		EndpointPath:     endpointPath,
		TempPath:         tempPath,
		ScriptRelPath:    relScriptPath, // Store relative path
		LastUpdated:      time.Now(),
		OriginalFileHash: initialHash,
	}

	// Copy necessary files to the environment
	if err := c.populateEnvironmentFiles(env); err != nil {
		os.RemoveAll(tempPath)
		return nil, fmt.Errorf("failed to populate environment '%s': %w", env.ID, err)
	}

	c.logger.Printf("Created environment for '%s' at '%s'", endpointPath, tempPath)
	return env, nil
}

// updateEnvironmentIfNeeded checks if an environment needs to be updated.
func (c *environmentCache) updateEnvironmentIfNeeded(env *phpEnvironment) error {
	env.mutex.Lock() // Lock specific env
	defer env.mutex.Unlock()

	// Hash check on main file only for now
	currentHash, err := calculateFileHash(env.OriginalPath)
	if err != nil {
		c.logger.Printf("Warning: Could not calculate hash for '%s' during update check: %v", env.OriginalPath, err)
		return nil // Don't fail update if hash check fails temporarily
	}

	if currentHash != env.OriginalFileHash {
		c.logger.Printf("Rebuilding environment for '%s' due to file content change (hash mismatch)", env.EndpointPath)
		if err := c.populateEnvironmentFiles(env); err != nil {
			return fmt.Errorf("error rebuilding environment files for '%s': %w", env.EndpointPath, err)
		}
		env.OriginalFileHash = currentHash
		env.LastUpdated = time.Now()
	}
	return nil
}

// calculateRelPath determines the relative path of a script based on source/embed dirs
func (c *environmentCache) calculateRelPath(absScriptPath string) (string, error) {
	var relPath string
	var err error
	if strings.HasPrefix(absScriptPath, c.embedDir) {
		relPath, err = filepath.Rel(c.embedDir, absScriptPath)
	} else {
		relPath, err = filepath.Rel(c.sourceDir, absScriptPath)
	}
	if err != nil {
		return "", err // Let caller handle specific error message
	}
	relPath = filepath.Clean(relPath)
	// Handle file at root of source/embed more carefully
	if relPath == "." {
		relPath = filepath.Base(absScriptPath)
	}
	return relPath, nil
}

// populateEnvironmentFiles copies the necessary files into the environment.
// If the source is from SourceDir, mirrors the whole SourceDir.
// If the source is from EmbedDir, copies only the specific script.
// Then, overlays global libraries.
func (c *environmentCache) populateEnvironmentFiles(env *phpEnvironment) error {
	// 1. Handle main script source
	if strings.HasPrefix(env.OriginalPath, c.embedDir) {
		// Source is an embedded file - copy only this file
		relEndpointPath := env.ScriptRelPath
		if relEndpointPath == "" {
			return fmt.Errorf("internal error: ScriptRelPath empty for embed env %s", env.ID)
		}
		targetEndpointPath := filepath.Join(env.TempPath, relEndpointPath)
		if err := copyFile(env.OriginalPath, targetEndpointPath); err != nil {
			return fmt.Errorf("failed to copy embedded endpoint file '%s' to '%s': %w", env.OriginalPath, targetEndpointPath, err)
		}
		c.logger.Printf("Populated env %s with single embedded script: %s", env.ID, relEndpointPath)

	} else if strings.HasPrefix(env.OriginalPath, c.sourceDir) || !filepath.IsAbs(env.OriginalPath) {
		// Source is from user's SourceDir (or was relative, assumed to be in sourceDir)
		// Mirror the entire source directory content
		c.logger.Printf("Populating env %s by mirroring SourceDir: %s", env.ID, c.sourceDir)
		if err := c._mirrorDirectoryContent(c.sourceDir, env.TempPath); err != nil {
			return fmt.Errorf("failed to mirror sourceDir '%s' to '%s': %w", c.sourceDir, env.TempPath, err)
		}
	} else {
		// Original path is absolute but not in embed dir - how should this be handled?
		// Copy just the single file for now.
		c.logger.Printf("Warning: Handling absolute script path '%s' outside known source/embed dirs. Copying only the single file.", env.OriginalPath)
		relEndpointPath := env.ScriptRelPath
		if relEndpointPath == "" {
			return fmt.Errorf("internal error: ScriptRelPath empty for absolute env %s", env.ID)
		}
		targetEndpointPath := filepath.Join(env.TempPath, relEndpointPath)
		if err := copyFile(env.OriginalPath, targetEndpointPath); err != nil {
			return fmt.Errorf("failed to copy absolute endpoint file '%s' to '%s': %w", env.OriginalPath, targetEndpointPath, err)
		}
	}

	// 2. Copy global libraries (overlaying potentially mirrored files)
	for targetRelPath, sourceDiskPath := range c.globalLibraries {
		targetEnvPath := filepath.Join(env.TempPath, targetRelPath)
		c.logger.Printf("Copying global library '%s' to '%s'", sourceDiskPath, targetEnvPath)

		// Ensure target directory exists
		targetDir := filepath.Dir(targetEnvPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for library '%s': %w", targetDir, err)
		}

		// Copy the file
		if err := copyFile(sourceDiskPath, targetEnvPath); err != nil {
			return fmt.Errorf("failed to copy global library '%s' to '%s': %w", sourceDiskPath, targetEnvPath, err)
		}
	}

	// 3. Create path globals PHP file
	pathGlobalsContent := `<?php
// Debug helper
function _frango_debug($message, $data = null) {
    file_put_contents('php://stderr', "[PHP] $message" . ($data !== null ? ': ' . print_r($data, true) : '') . "\n");
}

// Initialize $_PATH superglobal for path parameters
if (!isset($_PATH)) {
    $_PATH = [];
    
    // Load from JSON if available
    $pathParamsJson = $_SERVER['FRANGO_PATH_PARAMS_JSON'] ?? '{}';
    _frango_debug('FRANGO_PATH_PARAMS_JSON from _SERVER', $pathParamsJson);
    
    // Decode JSON parameters
    $decodedParams = json_decode($pathParamsJson, true);
    _frango_debug('Decoded params', $decodedParams);
    
    if (is_array($decodedParams)) {
        $_PATH = $decodedParams;
        _frango_debug('Set $_PATH from JSON', $_PATH);
    } else {
        _frango_debug('JSON decode failed, empty array used');
    }
    
    // Also add any FRANGO_PARAM_ variables from $_SERVER for backward compatibility
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'FRANGO_PARAM_') === 0) {
            $paramName = substr($key, strlen('FRANGO_PARAM_'));
            _frango_debug("Found param in _SERVER: $key => $paramName=$value");
            if (!isset($_PATH[$paramName])) {
                $_PATH[$paramName] = $value;
            }
        }
    }
    
    _frango_debug('Final $_PATH contents', $_PATH);
}

// Initialize $_PATH_SEGMENTS superglobal for URL segments
if (!isset($_PATH_SEGMENTS)) {
    $_PATH_SEGMENTS = [];
    
    // Get segment count
    $segmentCount = intval($_SERVER['FRANGO_URL_SEGMENT_COUNT'] ?? 0);
    _frango_debug("URL segment count", $segmentCount);
    
    // Add segments to array
    for ($i = 0; $i < $segmentCount; $i++) {
        $segmentKey = "FRANGO_URL_SEGMENT_$i";
        if (isset($_SERVER[$segmentKey])) {
            $_PATH_SEGMENTS[] = $_SERVER[$segmentKey];
        }
    }
    
    _frango_debug('URL segments', $_PATH_SEGMENTS);
}

// Helper function to get path segments
if (!function_exists('path_segments')) {
    function path_segments() {
        global $_PATH_SEGMENTS;
        return $_PATH_SEGMENTS;
    }
}

// Ensure the variables are accessible globally
$GLOBALS['_PATH'] = $_PATH;
$GLOBALS['_PATH_SEGMENTS'] = $_PATH_SEGMENTS;

_frango_debug('Path globals initialization complete');
`

	// Write the file to the environment
	pathGlobalsPath := filepath.Join(env.TempPath, "_frango_path_globals.php")
	err := os.WriteFile(pathGlobalsPath, []byte(pathGlobalsContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write path globals file: %w", err)
	}

	return nil
}

// _mirrorDirectoryContent mirrors all files from a source directory to a destination directory.
// Used internally by populateEnvironmentFiles when dealing with SourceDir scripts.
func (c *environmentCache) _mirrorDirectoryContent(sourceDir string, destDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path from the source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("error calculating relative path during mirror: %w", err)
		}

		// Calculate the target path in the environment
		targetPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			// Create directories as needed
			// Use MkdirAll to handle nested directories properly
			if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("error creating directory during mirror '%s': %w", targetPath, err)
			}
			return nil // Don't copy directory itself, just ensure it exists
		}

		// If not a directory, copy the file
		if err := copyFile(path, targetPath); err != nil {
			return fmt.Errorf("error copying file during mirror '%s' to '%s': %w", path, targetPath, err)
		}

		return nil
	})
}

// Cleanup cleans up all environment resources.
func (c *environmentCache) Cleanup() {
	c.mutex.Lock() // Lock for modifying environments map
	defer c.mutex.Unlock()

	for key, env := range c.environments {
		c.logger.Printf("Removing environment temp dir: %s (for %s)", env.TempPath, key)
	}
	c.environments = make(map[string]*phpEnvironment) // Clear map

	c.logger.Printf("Cleanup complete (base temp dir removal handled elsewhere).")
}
