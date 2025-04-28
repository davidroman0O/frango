<?php
// Get parameters from URL
$name = $_GET['name'] ?? 'Guest';
$color = $_GET['color'] ?? 'black';
$count = (int)($_GET['count'] ?? 1);
?>
<!DOCTYPE html>
<html>
<head>
    <title>Dynamic Page - PHP Multi-Page Example</title>
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
            color: <?php echo htmlspecialchars($color); ?>;
        }
        .dynamic-content {
            background-color: #f9f9f9;
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
        }
        .box {
            margin: 10px 0;
            padding: 10px;
            background-color: #e9e9e9;
            border-radius: 4px;
        }
        .nav-links {
            margin-top: 30px;
            display: flex;
            gap: 10px;
        }
        a {
            display: inline-block;
            padding: 10px 15px;
            background-color: #2196F3;
            color: white;
            text-decoration: none;
            border-radius: 4px;
        }
        a:hover {
            background-color: #0b7dda;
        }
        form {
            margin-top: 20px;
            padding: 15px;
            background-color: #f0f0f0;
            border-radius: 4px;
        }
        label {
            display: block;
            margin: 10px 0 5px;
        }
        input, select {
            padding: 8px;
            margin-bottom: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            width: 100%;
            max-width: 300px;
        }
        button {
            padding: 10px 15px;
            background-color: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #45a049;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Dynamic Page for <?php echo htmlspecialchars($name); ?></h1>
        
        <div class="dynamic-content">
            <h2>Dynamic Content</h2>
            <p>This page demonstrates dynamic content generation with PHP.</p>
            <p>Current parameters:</p>
            <ul>
                <li>Name: <?php echo htmlspecialchars($name); ?></li>
                <li>Color: <?php echo htmlspecialchars($color); ?></li>
                <li>Count: <?php echo $count; ?></li>
            </ul>
            
            <?php for($i = 0; $i < $count; $i++): ?>
                <div class="box">
                    Box #<?php echo $i + 1; ?> for <?php echo htmlspecialchars($name); ?>
                </div>
            <?php endfor; ?>
        </div>
        
        <form action="/dynamic" method="GET">
            <h3>Change Parameters</h3>
            <label for="name">Name:</label>
            <input type="text" id="name" name="name" value="<?php echo htmlspecialchars($name); ?>">
            
            <label for="color">Color:</label>
            <select id="color" name="color">
                <option value="black" <?php if($color == 'black') echo 'selected'; ?>>Black</option>
                <option value="red" <?php if($color == 'red') echo 'selected'; ?>>Red</option>
                <option value="blue" <?php if($color == 'blue') echo 'selected'; ?>>Blue</option>
                <option value="green" <?php if($color == 'green') echo 'selected'; ?>>Green</option>
                <option value="purple" <?php if($color == 'purple') echo 'selected'; ?>>Purple</option>
            </select>
            
            <label for="count">Count:</label>
            <input type="number" id="count" name="count" min="1" max="10" value="<?php echo $count; ?>">
            
            <button type="submit">Update</button>
        </form>
        
        <div class="nav-links">
            <a href="/">Back to Home</a>
            <a href="/demo">Go to Demo Page</a>
        </div>
    </div>
</body>
</html> 