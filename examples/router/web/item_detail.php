<?php
$itemId = $_SERVER['FRANGO_PARAM_id'] ?? null;
if (!$itemId) { die("Item ID required."); }
$apiUrl = "http://localhost:" . ($_SERVER["SERVER_PORT"] ?? 8082) . "/api/items/" . $itemId;
$itemDataJson = @file_get_contents($apiUrl);
$itemData = json_decode($itemDataJson, true);
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
</body></html>
