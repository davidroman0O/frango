<?php
// Get the item ID from URL segment (items/123 -> segment 1 is "123")
$itemId = $_SERVER['FRANGO_URL_SEGMENT_1'] ?? null;
if (!$itemId) { die("Item ID required."); }

// Debug info
$debug = "URL Segments: ";
for ($i = 0; $i < ($_SERVER['FRANGO_URL_SEGMENT_COUNT'] ?? 0); $i++) {
    $debug .= "[$i]=" . ($_SERVER["FRANGO_URL_SEGMENT_$i"] ?? 'none') . " ";
}

// Get item data from API
$apiUrl = "http://localhost:" . ($_SERVER["SERVER_PORT"] ?? 8082) . "/api/items/" . $itemId;

// Add debug information about the API call
$apiDebug = "API URL: " . $apiUrl . "\n";

// Make API request with better error handling 
$itemDataJson = @file_get_contents($apiUrl);
$apiDebug .= "API Response Success: " . ($itemDataJson !== false ? "Yes" : "No") . "\n";

if ($itemDataJson === false) {
    $apiDebug .= "Error: " . error_get_last()['message'] . "\n";
    $itemData = null;
} else {
    // Parse the response
    $itemData = json_decode($itemDataJson, true);
    $apiDebug .= "JSON Decode Result: " . (json_last_error() === JSON_ERROR_NONE ? "Success" : json_last_error_msg()) . "\n";
    $apiDebug .= "Result contains 'item': " . (isset($itemData['item']) ? "Yes" : "No") . "\n";
}
?>
<!DOCTYPE html><html><head><title>Item Details</title></head><body>
<h1>Item Detail</h1>
<?php if ($itemData && isset($itemData["item"])): ?>
    <?php $item = $itemData["item"]; ?>
    <p><b>ID:</b> <?= htmlspecialchars($item["id"]) ?></p>
    <p><b>Name:</b> <?= htmlspecialchars($item["name"]) ?></p>
    <p><b>Description:</b> <?= htmlspecialchars($item["description"]) ?></p>
    <p><b>Created At:</b> <?= htmlspecialchars($item["created_at"]) ?></p>
<?php else: ?>
    <p>Item not found or API error.</p>
    <pre><?= htmlspecialchars($itemDataJson) ?></pre>
<?php endif; ?>
<br/><a href="/">Back to list</a>

<!-- Debug info -->
<div style="margin-top: 30px; padding: 15px; background: #f5f5f5; border: 1px solid #ddd; border-radius: 5px; font-family: monospace; font-size: 12px;">
    <h3>Debug Information</h3>
    <p><?= htmlspecialchars($debug ?? '') ?></p>
    <p>Raw URL Path: <?= htmlspecialchars($_SERVER['FRANGO_URL_PATH'] ?? 'not available') ?></p>
    <p><?= htmlspecialchars($apiDebug ?? '') ?></p>
</div>
</body></html>
