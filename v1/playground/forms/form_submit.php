<?php
/**
 * Form Submission Test
 * 
 * HTML form that submits to form_test.php to verify form handling
 */
?>
<!DOCTYPE html>
<html>
<head>
    <title>Form Test</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
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
            font-size: 16px;
        }
        button:hover {
            background: #2980b9;
        }
        .method-toggle {
            margin-bottom: 15px;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>Form Submission Test</h1>
        <p>This form will submit to form_test.php to verify form data handling.</p>
        
        <div class="method-toggle">
            <label><input type="radio" name="method_choice" value="post" checked> POST Method</label>
            <label><input type="radio" name="method_choice" value="get"> GET Method</label>
        </div>
        
        <!-- POST Form -->
        <form id="post-form" action="form_test.php" method="POST">
            <h2>POST Form</h2>
            
            <label for="name">Name:</label>
            <input type="text" id="name" name="name" value="Test User">
            
            <label for="email">Email:</label>
            <input type="email" id="email" name="email" value="test@example.com">
            
            <label for="message">Message:</label>
            <textarea id="message" name="message" rows="4">This is a test message to verify form handling</textarea>
            
            <button type="submit">Submit Form (POST)</button>
        </form>
        
        <!-- GET Form -->
        <form id="get-form" action="form_test.php" method="GET" style="display: none;">
            <h2>GET Form</h2>
            
            <label for="query">Search Query:</label>
            <input type="text" id="query" name="query" value="test search">
            
            <label for="category">Category:</label>
            <input type="text" id="category" name="category" value="testing">
            
            <label for="limit">Result Limit:</label>
            <input type="number" id="limit" name="limit" value="10">
            
            <button type="submit">Submit Form (GET)</button>
        </form>
    </div>
    
    <script>
        // Toggle between POST and GET forms
        const methodRadios = document.getElementsByName('method_choice');
        const postForm = document.getElementById('post-form');
        const getForm = document.getElementById('get-form');
        
        for (const radio of methodRadios) {
            radio.addEventListener('change', function() {
                if (this.value === 'post') {
                    postForm.style.display = 'block';
                    getForm.style.display = 'none';
                } else {
                    postForm.style.display = 'none';
                    getForm.style.display = 'block';
                }
            });
        }
    </script>
</body>
</html> 