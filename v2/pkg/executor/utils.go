package executor

import (
	"net"
	"sort"
	"strings"

	"github.com/davidroman0O/frango/v2/pkg/php"
)

// calculateScriptPathHash generates a hash for a given script path.
// Used for creating unique temporary file names.
func calculateScriptPathHash(scriptPath string) string {
	return php.CalculatePathHash(scriptPath)
}

// getMapKeys returns a sorted list of keys from a map[string]interface{}.
// Useful for logging or consistent iteration.
func getMapKeys(m map[string]interface{}) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// extractQueryString extracts the query string part from a full URL.
func extractQueryString(fullURL string) string {
	queryIndex := strings.Index(fullURL, "?")
	if queryIndex != -1 {
		return fullURL[queryIndex+1:]
	}
	return ""
}

// extractHostOnly extracts the host part from a remote address with format "host:port".
func extractHostOnly(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If splitting fails, it might be just a host without a port
		return remoteAddr
	}
	return host
}

// extractPortOnly extracts the port part from a remote address with format "host:port".
func extractPortOnly(remoteAddr string) string {
	_, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If splitting fails, assume default port based on context
		return ""
	}
	return port
}
