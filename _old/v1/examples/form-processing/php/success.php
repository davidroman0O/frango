<?php
// Start session to access stored form data
session_start();

// Check if form data exists in session
if (!isset($_SESSION['contact_form'])) {
    // If no form data, redirect to the form
    header('Location: /contact');
    exit;
}

// Get form data from session
$formData = $_SESSION['contact_form'];
$accessMethods = $_SESSION['access_methods'] ?? null;

// Set content type
header('Content-Type: text/html');
?>
<!DOCTYPE html>
<html>
<head>
    <title>Form Submission Successful - Frango Example</title>
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
        .success-banner {
            background-color: #d4edda;
            color: #155724;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        h1, h2, h3 {
            color: #333;
        }
        a {
            color: #0366d6;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .button {
            display: inline-block;
            background: #0366d6;
            color: white;
            padding: 8px 16px;
            border-radius: 4px;
            text-decoration: none;
            margin-top: 10px;
        }
        .button:hover {
            background: #0250be;
            text-decoration: none;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            text-align: left;
            padding: 8px;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f5f5f5;
        }
        .code {
            background: #f6f8fa;
            padding: 15px;
            border-radius: 4px;
            overflow: auto;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="success-banner">
        <h1>Thank You!</h1>
        <p>Your message has been successfully submitted.</p>
    </div>
    
    <div class="card">
        <h2>Submitted Information</h2>
        <table>
            <tr>
                <th>Field</th>
                <th>Value</th>
            </tr>
            <tr>
                <td>Name</td>
                <td><?= htmlspecialchars($formData['name']) ?></td>
            </tr>
            <tr>
                <td>Email</td>
                <td><?= htmlspecialchars($formData['email']) ?></td>
            </tr>
            <tr>
                <td>Subject</td>
                <td><?= htmlspecialchars($formData['subject']) ?></td>
            </tr>
            <tr>
                <td>Message</td>
                <td><?= nl2br(htmlspecialchars($formData['message'])) ?></td>
            </tr>
            <tr>
                <td>Submitted At</td>
                <td><?= htmlspecialchars($formData['submitted_at']) ?></td>
            </tr>
        </table>
    </div>
    
    <?php if ($accessMethods): ?>
    <div class="card">
        <h2>Form Data Access Methods</h2>
        <p>Frango provides multiple ways to access form data in PHP. Here's how your data was accessed:</p>
        
        <h3>1. Using $_POST Superglobal</h3>
        <pre class="code"><?php var_export($accessMethods['post']); ?></pre>
        
        <h3>2. Using $_FORM Superglobal (Frango specific)</h3>
        <p>The $_FORM superglobal works with both GET and POST requests, making it more flexible.</p>
        <pre class="code"><?php var_export($accessMethods['form']); ?></pre>
        
        <p><strong>Note:</strong> Both methods provide the same data in this case because we're using a POST form.</p>
    </div>
    <?php endif; ?>
    
    <div class="card">
        <h2>What happened behind the scenes?</h2>
        <ol>
            <li>Your form was submitted to <code>/contact/submit</code> using the POST method</li>
            <li>The Go router directed this to the <code>process-form.php</code> script</li>
            <li>PHP validated the form data</li>
            <li>The data was stored in the session</li>
            <li>You were redirected to this success page</li>
        </ol>
        <p>This pattern is common for form processing in web applications.</p>
        
        <a href="/" class="button">Return to Home</a>
        <a href="/contact" class="button">Submit Another Message</a>
    </div>
    
    <?php
    // Clear the form data from session to prevent seeing it
    // again if the user refreshes the success page
    unset($_SESSION['contact_form']);
    unset($_SESSION['access_methods']);
    ?>
</body>
</html> 