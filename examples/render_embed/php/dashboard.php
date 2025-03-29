<?php
// Simple direct access to variables
$title = "Dashboard";

// DEBUG: Show all available data sources
echo "<pre style='background:#f5f5f5;padding:10px;border:1px solid #ddd;margin:10px 0;overflow:auto;'>";
echo "<h3>All Environment Variables:</h3>";
print_r($_ENV);
echo "<h3>Server Variables:</h3>";
print_r($_SERVER);
echo "<h3>GET Variables:</h3>";
print_r($_GET);
echo "<h3>Debugging Markers:</h3>";
echo "ENV Marker: " . ($_ENV['DEBUG_FRANGO_MARKER'] ?? 'Not found') . "<br>";
echo "SERVER Marker: " . ($_SERVER['DEBUG_FRANGO_MARKER'] ?? 'Not found') . "<br>";
echo "HTTP_ Marker: " . ($_SERVER['HTTP_DEBUG_FRANGO_MARKER'] ?? 'Not found') . "<br>";
echo "PHP_ Marker: " . ($_ENV['PHP_DEBUG_FRANGO_MARKER'] ?? 'Not found') . "<br>";
echo "</pre>";

// Try multiple approaches to get variables
function getVar($name, $default = []) {
    // Try multiple places where variables could be
    if (isset($_ENV['frango_VAR_' . $name])) {
        return json_decode($_ENV['frango_VAR_' . $name], true);
    } 
    if (isset($_SERVER['frango_VAR_' . $name])) {
        return json_decode($_SERVER['frango_VAR_' . $name], true);
    }
    if (isset($_SERVER['HTTP_FRANGO_VAR_' . $name])) {
        return json_decode($_SERVER['HTTP_FRANGO_VAR_' . $name], true);
    }
    if (isset($_ENV['PHP_frango_VAR_' . $name])) {
        return json_decode($_ENV['PHP_frango_VAR_' . $name], true);
    }
    if (isset($_SERVER[$name])) {
        return json_decode($_SERVER[$name], true);
    }
    
    // Try uppercase keys
    $name_upper = strtoupper($name);
    if (isset($_ENV[$name_upper])) {
        return json_decode($_ENV[$name_upper], true);
    }
    if (isset($_SERVER[$name_upper])) {
        return json_decode($_SERVER[$name_upper], true);
    }
    
    return $default;
}

// Get data using our improved function
$userData = getVar('user', []);
$itemsData = getVar('items', []);
$statsData = getVar('stats', []);

// Get username or default
$username = htmlspecialchars($userData['name'] ?? 'Guest');
$userRole = htmlspecialchars($userData['role'] ?? 'Visitor');

// Show which method succeeded
echo "<div style='background:#e1f5fe;padding:10px;margin:10px 0;border:1px solid #b3e5fc;'>";
echo "<h3>Data Sources Found:</h3>";
echo "userData: " . (empty($userData) ? 'NOT FOUND' : 'FOUND (' . count($userData) . ' items)') . "<br>";
echo "itemsData: " . (empty($itemsData) ? 'NOT FOUND' : 'FOUND (' . count($itemsData) . ' items)') . "<br>";
echo "statsData: " . (empty($statsData) ? 'NOT FOUND' : 'FOUND (' . count($statsData) . ' items)') . "<br>";
echo "</div>";
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
    </style>
</head>
<body>
    <div style="display:flex;justify-content:space-between;border-bottom:1px solid #eee;padding-bottom:10px;">
        <h1><?= $title ?></h1>
        <div style="text-align:right;">
            <p>Welcome, <strong><?= $username ?></strong></p>
            <p>Role: <?= $userRole ?></p>
        </div>
    </div>

    <?php if (!empty($statsData)): ?>
    <div style="display:grid;grid-template-columns:repeat(3, 1fr);gap:20px;margin:20px 0;">
        <?php foreach ($statsData as $key => $value): ?>
            <div style="background:#f5f5f5;padding:15px;border-radius:5px;">
                <div><?= htmlspecialchars(ucwords(str_replace('_', ' ', $key))) ?></div>
                <div style="font-size:24px;font-weight:bold;">
                    <?php if (is_numeric($value)): ?>
                        <?= is_float($value) ? '$' . number_format($value, 2) : number_format($value) ?>
                    <?php else: ?>
                        <?= htmlspecialchars($value) ?>
                    <?php endif; ?>
                </div>
            </div>
        <?php endforeach; ?>
    </div>
    <?php endif; ?>

    <h2>Recent Items</h2>
    <?php if (empty($itemsData)): ?>
        <div style="padding:10px;background:#fff8e1;border:1px solid #ffe0b2;">
            <p>No items to display.</p>
            <p>Debug info: itemsData is <?= var_export($itemsData, true) ?></p>
            <p>ENV['frango_VAR_items'] = <?= var_export($_ENV['frango_VAR_items'] ?? 'not set', true) ?></p>
            <p>SERVER['frango_VAR_items'] = <?= var_export($_SERVER['frango_VAR_items'] ?? 'not set', true) ?></p>
            <p>SERVER['HTTP_FRANGO_VAR_items'] = <?= var_export($_SERVER['HTTP_FRANGO_VAR_items'] ?? 'not set', true) ?></p>
        </div>
    <?php else: ?>
        <table style="width:100%;border-collapse:collapse;">
            <thead>
                <tr style="background:#f5f5f5;">
                    <th style="text-align:left;padding:8px;">ID</th>
                    <th style="text-align:left;padding:8px;">Name</th>
                    <th style="text-align:left;padding:8px;">Description</th>
                    <th style="text-align:left;padding:8px;">Price</th>
                </tr>
            </thead>
            <tbody>
                <?php foreach ($itemsData as $item): ?>
                <tr style="border-bottom:1px solid #eee;">
                    <td style="padding:8px;"><?= htmlspecialchars($item['id'] ?? 'N/A') ?></td>
                    <td style="padding:8px;"><?= htmlspecialchars($item['name'] ?? 'Unnamed') ?></td>
                    <td style="padding:8px;"><?= htmlspecialchars($item['description'] ?? '') ?></td>
                    <td style="padding:8px;">
                        <?php if (isset($item['price'])): ?>
                            $<?= number_format((float)$item['price'], 2) ?>
                        <?php else: ?>
                            N/A
                        <?php endif; ?>
                    </td>
                </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    <?php endif; ?>

    <div style="margin-top:40px;text-align:center;color:#666;font-size:14px;">
        <p>Generated with Frango - Rendered at <?= date('Y-m-d H:i:s') ?></p>
        <p>PHP Version: <?= PHP_VERSION ?></p>
    </div>
</body>
</html> 