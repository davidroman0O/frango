<?php
/**
 * GET Form Test
 * 
 * Simple test focused solely on GET request handling
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>GET Form Test</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #2c3e50; }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input, textarea {
            width: 100%;
            padding: 8px;
            margin-bottom: 15px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        button {
            background: #3498db;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #2980b9;
        }
        pre {
            background: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow: auto;
        }
        .result {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #eee;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>GET Form Test</h1>
        <p>This form submits data using the GET method. The data will be visible in the URL as query parameters.</p>
        
        <form action="/forms/form-get" method="GET">
            <label for="name">Name:</label>
            <input type="text" id="name" name="name" value="<?= htmlspecialchars($_GET['name'] ?? 'Test User') ?>">
            
            <label for="category">Category:</label>
            <input type="text" id="category" name="category" value="<?= htmlspecialchars($_GET['category'] ?? 'testing') ?>">
            
            <label for="limit">Results Limit:</label>
            <input type="number" id="limit" name="limit" value="<?= htmlspecialchars($_GET['limit'] ?? '10') ?>">
            
            <button type="submit">Submit GET Form</button>
        </form>
        
        <?php if (!empty($_GET)): ?>
        <div class="result">
            <h2>GET Results</h2>
            
            <h3>$_GET Superglobal:</h3>
            <pre><?php var_export($_GET); ?></pre>
            
            <h3>$_QUERY Superglobal (Frango alias):</h3>
            <pre><?php var_export($_QUERY ?? []); ?></pre>
            
            <h3>$_REQUEST Superglobal:</h3>
            <pre><?php var_export($_REQUEST); ?></pre>
            
            <h3>PHP_QUERY_* Variables:</h3>
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
        <p><a href="/forms/form-index">Back to Form Tests</a></p>
        <p><a href="/forms">Back to Forms</a></p>
    </div>
</body>
</html> 