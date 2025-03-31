<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Embedded Dashboard</title>
    <?php
    // Include the utility library
    include_once($_SERVER['DOCUMENT_ROOT'] . '/lib/utils.php');
    
    /**
     * Gets a variable injected from Go
     * 
     * @param string $name The variable name
     * @param mixed $default The default value to return if the variable is not found
     * @return mixed The variable value
     */
    function go_var($name, $default = null) {
        $envKey = "frango_VAR_" . $name;
        
        if (!isset($_SERVER[$envKey])) {
            return $default;
        }
        
        $jsonValue = $_SERVER[$envKey];
        $value = json_decode($jsonValue, true);
        
        // Handle JSON decode errors
        if ($value === null && json_last_error() !== JSON_ERROR_NONE) {
            error_log("Error decoding Go variable {$name}: " . json_last_error_msg());
            return $default;
        }
        
        return $value;
    }

    /**
     * Gets all variables injected from Go
     * 
     * @return array An associative array of all injected variables
     */
    function go_vars() {
        $vars = [];
        $prefix = "frango_VAR_";
        $prefixLen = strlen($prefix);
        
        foreach ($_SERVER as $key => $value) {
            if (strpos($key, $prefix) === 0) {
                $name = substr($key, $prefixLen);
                $jsonValue = $value;
                
                $decodedValue = json_decode($jsonValue, true);
                
                // Handle JSON decode errors
                if ($decodedValue === null && json_last_error() !== JSON_ERROR_NONE) {
                    error_log("Error decoding Go variable {$name}: " . json_last_error_msg());
                    continue;
                }
                
                $vars[$name] = $decodedValue;
            }
        }
        
        return $vars;
    }
    ?>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f7;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 20px;
        }
        header {
            border-bottom: 1px solid #eee;
            padding-bottom: 20px;
            margin-bottom: 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .user-info {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .user-avatar {
            width: 40px;
            height: 40px;
            border-radius: 50%;
            background-color: #007bff;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background-color: #f8f9fa;
            border-radius: 6px;
            padding: 15px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            margin: 5px 0;
            color: #007bff;
        }
        .stat-label {
            color: #6c757d;
            font-size: 14px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th, td {
            text-align: left;
            padding: 12px 15px;
            border-bottom: 1px solid #eee;
        }
        th {
            background-color: #f8f9fa;
            font-weight: bold;
        }
        tr:hover {
            background-color: #f5f5f7;
        }
        .badge {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
        }
        .badge-success {
            background-color: #d1e7dd;
            color: #0f5132;
        }
        .badge-warning {
            background-color: #fff3cd;
            color: #664d03;
        }
        .badge-danger {
            background-color: #f8d7da;
            color: #842029;
        }
        .badge-info {
            background-color: #cff4fc;
            color: #055160;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1><?= htmlspecialchars(go_var('title', 'Embedded Dashboard')) ?></h1>
            <?php $user = go_var('user', []); ?>
            <div class="user-info">
                <div class="user-avatar"><?= strtoupper(substr($user['name'] ?? 'U', 0, 1)) ?></div>
                <div>
                    <div><?= htmlspecialchars($user['name'] ?? 'Unknown User') ?></div>
                    <div style="font-size: 12px; color: #6c757d;"><?= htmlspecialchars($user['role'] ?? 'Guest') ?></div>
                </div>
            </div>
        </header>

        <section>
            <h2>Statistics</h2>
            <?php $stats = go_var('stats', []); ?>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Total Users</div>
                    <div class="stat-value"><?= htmlspecialchars($stats['total_users'] ?? '0') ?></div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Active Users</div>
                    <div class="stat-value"><?= htmlspecialchars($stats['active_users'] ?? '0') ?></div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Total Products</div>
                    <div class="stat-value"><?= htmlspecialchars($stats['total_products'] ?? '0') ?></div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Revenue</div>
                    <div class="stat-value"><?= format_currency($stats['revenue'] ?? 0) ?></div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Conversion Rate</div>
                    <div class="stat-value"><?= htmlspecialchars($stats['conversion_rate'] ?? '0%') ?></div>
                </div>
            </div>
        </section>

        <section>
            <h2>Recent Items</h2>
            <?php $items = go_var('items', []); ?>
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Name</th>
                        <th>Description</th>
                        <th>Price</th>
                    </tr>
                </thead>
                <tbody>
                    <?php foreach ($items as $item): ?>
                    <tr>
                        <td><?= htmlspecialchars($item['id'] ?? '') ?></td>
                        <td><?= htmlspecialchars($item['name'] ?? '') ?></td>
                        <td><?= truncate($item['description'] ?? '', 50) ?></td>
                        <td><?= format_currency($item['price'] ?? 0) ?></td>
                    </tr>
                    <?php endforeach; ?>
                    <?php if (empty($items)): ?>
                    <tr>
                        <td colspan="4" style="text-align: center;">No items found</td>
                    </tr>
                    <?php endif; ?>
                </tbody>
            </table>
        </section>

        <footer style="margin-top: 40px; border-top: 1px solid #eee; padding-top: 20px; color: #6c757d; font-size: 14px;">
            <?php $debug = go_var('debug_info', []); ?>
            <p>Generated at: <?= format_date($debug['timestamp'] ?? time()) ?></p>
        </footer>
    </div>
</body>
</html> 