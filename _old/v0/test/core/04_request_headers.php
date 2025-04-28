<?php
// Request headers test
header('Content-Type: text/html; charset=UTF-8');

// Get all headers via getallheaders() if available
$allHeaders = function_exists('getallheaders') ? getallheaders() : [];

// Get specific important headers directly from $_SERVER
$userAgent = $_SERVER['HTTP_USER_AGENT'] ?? 'Not set';
$acceptHeader = $_SERVER['HTTP_ACCEPT'] ?? 'Not set';
$contentType = $_SERVER['CONTENT_TYPE'] ?? $_SERVER['HTTP_CONTENT_TYPE'] ?? 'Not set';

// Check FRANGO_HEADER variables
$frangoUserAgent = $_SERVER['FRANGO_HEADER_USER_AGENT'] ?? 'Not set in FRANGO';
?>
<!DOCTYPE html>
<html>
<head>
    <title>Request Headers Test</title>
    <style>
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
    </style>
</head>
<body>
    <h1>Request Headers Test</h1>
    
    <h2>Common Headers via $_SERVER:</h2>
    <ul>
        <li>User-Agent: <?= htmlspecialchars($userAgent) ?></li>
        <li>Accept: <?= htmlspecialchars($acceptHeader) ?></li>
        <li>Content-Type: <?= htmlspecialchars($contentType) ?></li>
    </ul>
    
    <h2>FRANGO Headers:</h2>
    <ul>
        <li>User-Agent via FRANGO: <?= htmlspecialchars($frangoUserAgent) ?></li>
    </ul>
    
    <h2>All Request Headers:</h2>
    <?php if (empty($allHeaders)): ?>
        <p>getallheaders() function not available</p>
    <?php else: ?>
        <table>
            <tr>
                <th>Header Name</th>
                <th>Value</th>
            </tr>
            <?php foreach ($allHeaders as $name => $value): ?>
                <tr>
                    <td><?= htmlspecialchars($name) ?></td>
                    <td><?= htmlspecialchars($value) ?></td>
                </tr>
            <?php endforeach; ?>
        </table>
    <?php endif; ?>
    
    <h2>All $_SERVER Variables:</h2>
    <table>
        <tr>
            <th>Name</th>
            <th>Value</th>
        </tr>
        <?php foreach ($_SERVER as $name => $value): 
            // Skip very long values
            if (is_string($value) && strlen($value) > 100) {
                $value = substr($value, 0, 100) . '... (truncated)';
            }
            if (is_array($value)) {
                $value = 'Array(' . count($value) . ')';
            }
        ?>
            <tr>
                <td><?= htmlspecialchars($name) ?></td>
                <td><?= htmlspecialchars($value) ?></td>
            </tr>
        <?php endforeach; ?>
    </table>
</body>
</html> 