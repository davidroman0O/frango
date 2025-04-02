<?php
// Display PHP environment information
// Note: On a production server, it's not recommended to expose phpinfo() like this
?>
<!DOCTYPE html>
<html>
<head>
    <title>PHP Info - Go-PHP Middleware Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }
        h1 {
            color: #333;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        a.back {
            display: inline-block;
            padding: 8px 16px;
            background-color: #2196F3;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            margin-top: 20px;
        }
        a.back:hover {
            background-color: #0b7dda;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>PHP Information</h1>
        
        <h2>PHP Environment</h2>
        <table>
            <tr>
                <th>PHP Version</th>
                <td><?php echo phpversion(); ?></td>
            </tr>
            <tr>
                <th>Server Software</th>
                <td><?php echo $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown'; ?></td>
            </tr>
            <tr>
                <th>Server Name</th>
                <td><?php echo $_SERVER['SERVER_NAME'] ?? 'Unknown'; ?></td>
            </tr>
            <tr>
                <th>Document Root</th>
                <td><?php echo $_SERVER['DOCUMENT_ROOT']; ?></td>
            </tr>
            <tr>
                <th>Request URI</th>
                <td><?php echo $_SERVER['REQUEST_URI']; ?></td>
            </tr>
            <tr>
                <th>Script Filename</th>
                <td><?php echo $_SERVER['SCRIPT_FILENAME']; ?></td>
            </tr>
            <tr>
                <th>Source File</th>
                <td><?php echo $_SERVER['GO_PHP_SOURCE_FILE'] ?? 'Unknown'; ?></td>
            </tr>
            <tr>
                <th>Production Mode</th>
                <td><?php echo $_SERVER['PHP_PRODUCTION'] ?? 'Off'; ?></td>
            </tr>
        </table>
        
        <h2>PHP Extensions</h2>
        <table>
            <tr>
                <th>Extension</th>
                <th>Version</th>
            </tr>
            <?php foreach(get_loaded_extensions() as $extension): ?>
            <tr>
                <td><?php echo $extension; ?></td>
                <td><?php echo phpversion($extension) ?: 'Unknown'; ?></td>
            </tr>
            <?php endforeach; ?>
        </table>
        
        <a href="/php/" class="back">Back to Home (Will 404)</a>
        <a href="/" class="back" style="margin-left: 10px; background-color: #4CAF50;">Back to Home (Correct)</a>
    </div>
</body>
</html> 