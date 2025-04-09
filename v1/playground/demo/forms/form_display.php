<?php
/**
 * General Form Data Display
 * 
 * Demonstrates accessing form data through the $_FORM superglobal
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>$_FORM Data Results</title>
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
            background: #f39c12;
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
        <h1><span class="method">FORM</span> Data Results</h1>
        
        <?php if ($_SERVER['REQUEST_METHOD'] !== 'POST' || (empty($_POST) && empty($_FORM))): ?>
            <div style="background: #fff4e5; padding: 15px; border-left: 4px solid #f39c12; margin: 20px 0;">
                <strong>No form data received.</strong> Please submit the form from the forms page.
            </div>
        <?php else: ?>
            <div style="background: #e8f8f5; padding: 15px; border-left: 4px solid #2ecc71; margin: 20px 0;">
                <strong>Success!</strong> Form data received and available in $_FORM superglobal.
            </div>
            
            <div class="result-section">
                <h2>Form Data Parameters</h2>
                
                <h3>$_FORM Superglobal:</h3>
                <table>
                    <tr>
                        <th>Parameter</th>
                        <th>Value</th>
                    </tr>
                    <?php if (isset($_FORM) && is_array($_FORM)): ?>
                        <?php foreach ($_FORM as $key => $value): ?>
                            <tr>
                                <td><strong><?= htmlspecialchars($key) ?></strong></td>
                                <td><?= htmlspecialchars($value) ?></td>
                            </tr>
                        <?php endforeach; ?>
                    <?php else: ?>
                        <tr>
                            <td colspan="2"><em>$_FORM superglobal is not available</em></td>
                        </tr>
                    <?php endif; ?>
                </table>
                
                <h3>$_POST Superglobal (for comparison):</h3>
                <table>
                    <tr>
                        <th>Parameter</th>
                        <th>Value</th>
                    </tr>
                    <?php if (!empty($_POST)): ?>
                        <?php foreach ($_POST as $key => $value): ?>
                            <tr>
                                <td><strong><?= htmlspecialchars($key) ?></strong></td>
                                <td><?= htmlspecialchars($value) ?></td>
                            </tr>
                        <?php endforeach; ?>
                    <?php else: ?>
                        <tr>
                            <td colspan="2"><em>No data in $_POST</em></td>
                        </tr>
                    <?php endif; ?>
                </table>
            </div>
            
            <div class="result-section">
                <h2>Data Access Examples</h2>
                
                <h3>Using $_FORM Superglobal:</h3>
                <pre><?php
                    if (isset($_FORM) && is_array($_FORM)) {
                        $product = isset($_FORM['product']) ? $_FORM['product'] : 'Not provided';
                        $quantity = isset($_FORM['quantity']) ? $_FORM['quantity'] : 'Not provided';
                        $notes = isset($_FORM['notes']) ? $_FORM['notes'] : 'Not provided';
                        
                        echo "\$product = \$_FORM['product'] ?? 'Not provided';\n";
                        echo "// Result: $product\n\n";
                        
                        echo "\$quantity = \$_FORM['quantity'] ?? 'Not provided';\n";
                        echo "// Result: $quantity\n\n";
                        
                        echo "\$notes = \$_FORM['notes'] ?? 'Not provided';\n";
                        echo "// Result: " . substr($notes, 0, 30) . (strlen($notes) > 30 ? '...' : '');
                    } else {
                        echo "// \$_FORM is not available";
                    }
                ?></pre>
            </div>
            
            <div class="result-section">
                <h2>Environment Variables</h2>
                
                <h3>PHP_FORM_* Variables (Raw Source):</h3>
                <pre><?php
                    $formVars = [];
                    foreach ($_SERVER as $key => $value) {
                        if (strpos($key, 'PHP_FORM_') === 0) {
                            $formVars[$key] = $value;
                        }
                    }
                    var_export($formVars);
                ?></pre>
                
                <h3>Is $_FORM the same as $_POST?</h3>
                <pre><?php
                    if (isset($_FORM) && isset($_POST)) {
                        $isIdentical = $_FORM === $_POST ? 'YES - Arrays are identical' : 'NO - Arrays are different';
                        echo $isIdentical;
                        
                        if ($_FORM !== $_POST) {
                            echo "\n\nDifferences:";
                            $inFormNotPost = array_diff_assoc($_FORM, $_POST);
                            $inPostNotForm = array_diff_assoc($_POST, $_FORM);
                            
                            if (!empty($inFormNotPost)) {
                                echo "\nIn \$_FORM but not in \$_POST:\n";
                                var_export($inFormNotPost);
                            }
                            
                            if (!empty($inPostNotForm)) {
                                echo "\nIn \$_POST but not in \$_FORM:\n";
                                var_export($inPostNotForm);
                            }
                        }
                    } else {
                        echo "Cannot compare - One or both superglobals are not available";
                    }
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