package executor

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// RequestData holds extracted information from an HTTP request, relevant for PHP execution.
type RequestData struct {
	Method       string
	FullURL      string
	Path         string
	RemoteAddr   string
	Headers      http.Header
	QueryParams  map[string][]string
	PathSegments []string
	JSONBody     map[string]interface{}
	FormData     map[string][]string
	FileUploads  map[string][]*multipart.FileHeader
}

// extractRequestData extracts all relevant data from an HTTP request into RequestData.
// This function is intended to be called within the Execute method.
func extractRequestData(r *http.Request) *RequestData {
	// Create a new request data object
	data := &RequestData{
		Method:      r.Method,
		FullURL:     r.URL.String(),
		Path:        r.URL.Path,
		RemoteAddr:  r.RemoteAddr,
		Headers:     r.Header,
		QueryParams: r.URL.Query(),
		PathSegments: func() []string {
			segments := []string{}
			for _, segment := range strings.Split(strings.Trim(r.URL.Path, "/"), "/") {
				if segment != "" {
					segments = append(segments, segment)
				}
			}
			return segments
		}(),
		JSONBody:    make(map[string]interface{}),
		FormData:    make(map[string][]string),
		FileUploads: make(map[string][]*multipart.FileHeader),
	}

	// Check if method might include a request body (most methods except GET and HEAD)
	// Include DELETE explicitly since modern APIs often use request bodies with DELETE
	if r.Method != "GET" && r.Method != "HEAD" {
		contentType := r.Header.Get("Content-Type")

		// Preserve the request body for multiple reads (form parsing and PHP execution)
		var bodyBytes []byte
		var err error
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				// Handle error reading body if necessary, maybe log it
				// For now, proceed, but PHP might not get the body
			}
			// Restore the body so it can be read again later (by ParseForm/ParseMultipartForm/FrankenPHP)
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}

		// For JSON requests, parse the preserved body
		if strings.Contains(contentType, "application/json") {
			if len(bodyBytes) > 0 {
				var jsonData map[string]interface{}
				// Use the preserved bodyBytes for unmarshalling
				if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
					data.JSONBody = jsonData
				}
			}
			// No need for r.Body = io.NopCloser(strings.NewReader(string(bodyBytes))) here, already done above

		} else if strings.Contains(contentType, "multipart/form-data") {
			// For multipart form data with file uploads
			// ParseMultipartForm reads from r.Body, which we restored
			// Use a reasonable default max memory (e.g., 32MB)
			const maxMemory = 32 << 20
			if err := r.ParseMultipartForm(maxMemory); err == nil {
				// Get form values
				if r.MultipartForm != nil {
					// Extract form values
					for key, values := range r.MultipartForm.Value {
						data.FormData[key] = values
					}

					// Extract file uploads
					for key, fileHeaders := range r.MultipartForm.File {
						data.FileUploads[key] = fileHeaders
					}
				}
			}
			// Restore body again *after* ParseMultipartForm potentially consumes it
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		} else {
			// For regular form data (application/x-www-form-urlencoded)
			// ParseForm reads from r.Body, which we restored
			if err := r.ParseForm(); err == nil {
				data.FormData = r.Form
			}
			// Restore body again *after* ParseForm potentially consumes it
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}

	return data
}
