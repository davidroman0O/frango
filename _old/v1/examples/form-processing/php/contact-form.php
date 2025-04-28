<?php
header('Content-Type: text/html');

// Initialize variables for form fields
$name = '';
$email = '';
$subject = '';
$message = '';

// Check for errors passed in query string
$errors = [];
if (isset($_GET['error']) && $_GET['error'] === '1') {
    // Parse errors passed through query string
    if (isset($_GET['fields'])) {
        $errorFields = explode(',', $_GET['fields']);
        foreach ($errorFields as $field) {
            $errors[$field] = true;
        }
    }
    
    // Preserve submitted values if provided
    $name = $_GET['name'] ?? '';
    $email = $_GET['email'] ?? '';
    $subject = $_GET['subject'] ?? '';
    $message = $_GET['message'] ?? '';
}
?>
<!DOCTYPE html>
<html>
<head>
    <title>Contact Form - Frango Example</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .card {
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 15px;
            margin-bottom: 20px;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input, textarea, select {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
            font-family: inherit;
            font-size: inherit;
        }
        .error-message {
            color: #e53e3e;
            font-size: 0.9em;
            margin-top: 5px;
        }
        .error-field {
            border-color: #e53e3e;
        }
        button {
            background: #0366d6;
            color: white;
            border: none;
            padding: 10px 15px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 1em;
        }
        button:hover {
            background: #0250be;
        }
        .nav-link {
            display: inline-block;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <h1>Contact Form</h1>
    
    <div class="card">
        <?php if (!empty($errors)): ?>
            <div style="background: #fff5f5; color: #e53e3e; padding: 10px; margin-bottom: 15px; border-radius: 4px;">
                <p><strong>Please fix the following errors:</strong></p>
                <ul>
                    <?php if (isset($errors['name'])): ?>
                        <li>Name is required</li>
                    <?php endif; ?>
                    
                    <?php if (isset($errors['email'])): ?>
                        <li>Valid email address is required</li>
                    <?php endif; ?>
                    
                    <?php if (isset($errors['message'])): ?>
                        <li>Message is required</li>
                    <?php endif; ?>
                </ul>
            </div>
        <?php endif; ?>
        
        <form action="/contact/submit" method="POST">
            <div class="form-group">
                <label for="name">Name</label>
                <input 
                    type="text" 
                    id="name" 
                    name="name" 
                    value="<?= htmlspecialchars($name) ?>"
                    class="<?= isset($errors['name']) ? 'error-field' : '' ?>"
                    required
                >
                <?php if (isset($errors['name'])): ?>
                    <div class="error-message">Name is required</div>
                <?php endif; ?>
            </div>
            
            <div class="form-group">
                <label for="email">Email</label>
                <input 
                    type="email" 
                    id="email" 
                    name="email" 
                    value="<?= htmlspecialchars($email) ?>"
                    class="<?= isset($errors['email']) ? 'error-field' : '' ?>"
                    required
                >
                <?php if (isset($errors['email'])): ?>
                    <div class="error-message">Valid email address is required</div>
                <?php endif; ?>
            </div>
            
            <div class="form-group">
                <label for="subject">Subject</label>
                <select id="subject" name="subject">
                    <option value="general" <?= $subject === 'general' ? 'selected' : '' ?>>General Inquiry</option>
                    <option value="support" <?= $subject === 'support' ? 'selected' : '' ?>>Technical Support</option>
                    <option value="feedback" <?= $subject === 'feedback' ? 'selected' : '' ?>>Feedback</option>
                    <option value="other" <?= $subject === 'other' ? 'selected' : '' ?>>Other</option>
                </select>
            </div>
            
            <div class="form-group">
                <label for="message">Message</label>
                <textarea 
                    id="message" 
                    name="message" 
                    rows="5"
                    class="<?= isset($errors['message']) ? 'error-field' : '' ?>"
                    required
                ><?= htmlspecialchars($message) ?></textarea>
                <?php if (isset($errors['message'])): ?>
                    <div class="error-message">Message is required</div>
                <?php endif; ?>
            </div>
            
            <button type="submit">Send Message</button>
        </form>
        
        <a href="/" class="nav-link">Back to Home</a>
    </div>
</body>
</html> 