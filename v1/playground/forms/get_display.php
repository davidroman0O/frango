<?php
/**
 * GET Form Display
 * 
 * Displays the results of a GET form submission
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>GET Form Results</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        h2 { color: #3498db; }
        pre {
            background: #f0f0f0;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .method {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            margin-right: 5px;
            font-size: 0.8rem;
            font-weight: bold;
            background: #3498db;
            color: white;
        }
        .result-section {
            margin-top: 20px;
            padding-top: 15px;
            border-top: 1px solid #eee;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
        }
        table, th, td {
            border: 1px solid #ddd;
        }
        th, td {
            padding: 10px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1><span class="method">GET</span> Form Results</h1>
        
        <?php if (empty($_GET)): ?>
            <div style="background: #fff4e5; padding: 15px; border-left: 4px solid #f39c12; margin: 20px 0;">
                <strong>No GET data received.</strong> Please submit the form from the forms page.
            </div>
        <?php else: ?>
            <div style="background: #e8f8f5; padding: 15px; border-left: 4px solid #2ecc71; margin: 20px 0;">
                <strong>Success!</strong> GET data received and available in $_GET superglobal.
            </div>
            
            <div class="result-section">
                <h2>GET Parameters</h2>
                <table>
                    <tr>
                        <th>Parameter</th>
                        <th>Value</th>
                    </tr>
                    <?php foreach ($_GET as $key => $value): ?>
                        <tr>
                            <td><strong><?= htmlspecialchars($key) ?></strong></td>
                            <td><?= htmlspecialchars($value) ?></td>
                        </tr>
                    <?php endforeach; ?>
                </table>
            </div>
            
            <div class="result-section">
                <h2>Data Access Examples</h2>
                
                <h3>Standard $_GET Access:</h3>
                <pre><?php
                    $name = isset($_GET['name']) ? $_GET['name'] : 'Not provided';
                    $category = isset($_GET['category']) ? $_GET['category'] : 'Not provided';
                    $limit = isset($_GET['limit']) ? $_GET['limit'] : 'Not provided';
                    
                    echo "\$name = $_GET[name] ?? 'Not provided';\n";
                    echo "// Result: $name\n\n";
                    
                    echo "\$category = $_GET[category] ?? 'Not provided';\n";
                    echo "// Result: $category\n\n";
                    
                    echo "\$limit = $_GET[limit] ?? 'Not provided';\n";
                    echo "// Result: $limit";
                ?></pre>
                
                <h3>Frango $_QUERY Alias:</h3>
                <pre><?php
                    if (isset($_QUERY)) {
                        $name = isset($_QUERY['name']) ? $_QUERY['name'] : 'Not provided';
                        $category = isset($_QUERY['category']) ? $_QUERY['category'] : 'Not provided';
                        $limit = isset($_QUERY['limit']) ? $_QUERY['limit'] : 'Not provided';
                        
                        echo "\$name = \$_QUERY['name'] ?? 'Not provided';\n";
                        echo "// Result: $name\n\n";
                        
                        echo "\$category = \$_QUERY['category'] ?? 'Not provided';\n";
                        echo "// Result: $category\n\n";
                        
                        echo "\$limit = \$_QUERY['limit'] ?? 'Not provided';\n";
                        echo "// Result: $limit";
                    } else {
                        echo "// \$_QUERY is not available";
                    }
                ?></pre>
            </div>
            
            <div class="result-section">
                <h2>Environment Variables</h2>
                
                <h3>PHP_QUERY_* Variables (Raw Source):</h3>
                <pre><?php
                    $queryVars = [];
                    foreach ($_SERVER as $key => $value) {
                        if (strpos($key, 'PHP_QUERY_') === 0) {
                            $queryVars[$key] = $value;
                        }
                    }
                    var_export($queryVars);
                ?></pre>
            </div>
        <?php endif; ?>
    </div>
    
    <div class="card">
        <h3>Navigation:</h3>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
</body>
</html> 