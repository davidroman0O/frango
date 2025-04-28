<?php
// Contact page

// Process form submission
$message = '';
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $name = $_POST['name'] ?? '';
    $email = $_POST['email'] ?? '';
    $comments = $_POST['comments'] ?? '';
    
    if (!empty($name) && !empty($email) && !empty($comments)) {
        $message = "Thank you for your message, {$name}! This is a demo, so no email has been sent.";
    } else {
        $message = "Please fill in all fields.";
    }
}
?>
<!DOCTYPE html>
<html>
<head>
    <title>Contact - Static Files PHP Example</title>
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
        }
        .form-container {
            margin-top: 20px;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 4px;
        }
        .message {
            margin: 15px 0;
            padding: 10px;
            border-radius: 4px;
            background-color: #d4edda;
        }
        label {
            display: block;
            margin: 15px 0 5px;
            font-weight: bold;
        }
        input, textarea {
            width: 100%;
            padding: 8px;
            margin-bottom: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        textarea {
            min-height: 100px;
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
        .nav {
            margin-top: 30px;
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
    </style>
</head>
<body>
    <div class="container">
        <h1>Contact Us</h1>
        
        <?php if (!empty($message)): ?>
            <div class="message">
                <?php echo htmlspecialchars($message); ?>
            </div>
        <?php endif; ?>
        
        <div class="form-container">
            <form method="POST">
                <label for="name">Name:</label>
                <input type="text" id="name" name="name" required>
                
                <label for="email">Email:</label>
                <input type="email" id="email" name="email" required>
                
                <label for="comments">Message:</label>
                <textarea id="comments" name="comments" required></textarea>
                
                <button type="submit">Submit</button>
            </form>
        </div>
        
        <div class="nav">
            <a href="/">Back to Home</a>
        </div>
    </div>
</body>
</html> 