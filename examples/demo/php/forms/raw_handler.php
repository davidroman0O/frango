<?php
/**
 * Frango v2 Demo - Raw Request Body Handler
 * 
 * Demonstrates processing raw request body data via php://input,
 * useful for PUT, PATCH, DELETE requests or custom formats.
 */

// Check if this is a supported HTTP method for showing request body
$supportedMethods = ['POST', 'PUT', 'PATCH', 'DELETE'];
$method = $_SERVER['REQUEST_METHOD'];
$hasBody = in_array($method, $supportedMethods);

// Get the raw input data
$rawInput = file_get_contents('php://input');
$contentLength = intval($_SERVER['CONTENT_LENGTH'] ?? 0);
$contentType = $_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'Not specified';

// Attempt to handle different content types intelligently
$processedData = null;
$processedDataFormat = 'text';
$processError = null;

if (!empty($rawInput)) {
    // Try to process based on content type
    if (strpos($contentType, 'application/json') === 0) {
        // JSON data
        $processedData = json_decode($rawInput, true);
        $processedDataFormat = 'json';
        
        if (json_last_error() !== JSON_ERROR_NONE) {
            $processError = 'Invalid JSON: ' . json_last_error_msg();
        }
    } elseif (strpos($contentType, 'application/xml') === 0 || strpos($contentType, 'text/xml') === 0) {
        // XML data
        libxml_use_internal_errors(true);
        $processedData = simplexml_load_string($rawInput);
        $processedDataFormat = 'xml';
        
        if ($processedData === false) {
            $errors = libxml_get_errors();
            $processError = 'XML parsing error: ' . (count($errors) > 0 ? $errors[0]->message : 'Unknown error');
            libxml_clear_errors();
        }
    } elseif (strpos($contentType, 'text/plain') === 0) {
        // Plain text, keep as is
        $processedData = $rawInput;
        $processedDataFormat = 'text';
    } else {
        // Try as JSON first (common for API clients that don't set proper headers)
        $jsonAttempt = json_decode($rawInput, true);
        if (json_last_error() === JSON_ERROR_NONE) {
            $processedData = $jsonAttempt;
            $processedDataFormat = 'json';
        } else {
            // Just treat as plain text
            $processedData = $rawInput;
            $processedDataFormat = 'text';
        }
    }
}

// Prepare response (text by default)
header('Content-Type: text/plain');

// Start response with request information
$response = "=== RAW REQUEST BODY HANDLER ===\n\n";
$response .= "Request Method: {$method}\n";
$response .= "Content-Type: {$contentType}\n";
$response .= "Content-Length: {$contentLength} bytes\n\n";

if (!$hasBody) {
    $response .= "NOTE: {$method} requests typically don't include a request body.\n\n";
}

$response .= "=== RAW INPUT (php://input) ===\n";
if (empty($rawInput)) {
    $response .= "(empty)\n";
} else {
    // Limit output to reasonable size
    $maxOutputLength = 1024;
    $truncated = strlen($rawInput) > $maxOutputLength;
    $output = $truncated ? substr($rawInput, 0, $maxOutputLength) : $rawInput;
    
    $response .= $output;
    if ($truncated) {
        $response .= "\n\n...output truncated, total length: " . strlen($rawInput) . " bytes";
    }
}

$response .= "\n\n=== PROCESSED DATA ===\n";
if ($processError) {
    $response .= "ERROR: {$processError}\n";
} elseif ($processedData === null) {
    $response .= "(none)\n";
} else {
    if ($processedDataFormat === 'json') {
        // Format the data as JSON for display
        $response .= "Parsed as JSON:\n";
        $response .= json_encode($processedData, JSON_PRETTY_PRINT);
    } elseif ($processedDataFormat === 'xml') {
        // Format the XML data for display
        $response .= "Parsed as XML:\n";
        $dom = new DOMDocument('1.0');
        $dom->preserveWhiteSpace = false;
        $dom->formatOutput = true;
        $dom->loadXML($processedData->asXML());
        $response .= $dom->saveXML();
    } else {
        // Just show the text
        $response .= "Treated as plain text\n";
        $response .= $processedData;
    }
}

$response .= "\n\n=== PHP CODE EXAMPLE ===\n";
$response .= <<<'EOT'
// Reading raw input data
$rawData = file_get_contents('php://input');

// For JSON data
$jsonData = json_decode($rawData, true);
if (json_last_error() === JSON_ERROR_NONE) {
    // Process JSON data
    $userId = $jsonData['user_id'] ?? null;
}

// For XML data
$xml = simplexml_load_string($rawData);
if ($xml !== false) {
    // Process XML data
    $userId = (string)$xml->user_id;
}

// For custom text formats
$lines = explode("\n", $rawData);
foreach ($lines as $line) {
    // Process each line
}
EOT;

$response .= "\n\n=== END OF RESPONSE ===\n";

// Output the response
echo $response; 