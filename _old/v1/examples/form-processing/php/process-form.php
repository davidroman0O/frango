<?php
// This script processes the contact form submission

// Validate that this is a POST request
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    // If not POST, redirect back to the form
    header('Location: /contact');
    exit;
}

// Initialize errors array
$errors = [];
$errorFields = [];

// Get and validate form data
$name = trim($_POST['name'] ?? '');
if (empty($name)) {
    $errors[] = 'Name is required';
    $errorFields[] = 'name';
}

$email = trim($_POST['email'] ?? '');
if (empty($email)) {
    $errors[] = 'Email is required';
    $errorFields[] = 'email';
} elseif (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
    $errors[] = 'Email is invalid';
    $errorFields[] = 'email';
}

$subject = $_POST['subject'] ?? 'general';
// Validate subject is one of the allowed values
$allowedSubjects = ['general', 'support', 'feedback', 'other'];
if (!in_array($subject, $allowedSubjects)) {
    $subject = 'general';
}

$message = trim($_POST['message'] ?? '');
if (empty($message)) {
    $errors[] = 'Message is required';
    $errorFields[] = 'message';
}

// Check if there are any errors
if (!empty($errors)) {
    // If there are errors, redirect back to the form with error information
    $queryParams = http_build_query([
        'error' => 1,
        'fields' => implode(',', $errorFields),
        'name' => $name,
        'email' => $email,
        'subject' => $subject,
        'message' => $message
    ]);
    
    header("Location: /contact?$queryParams");
    exit;
}

// No errors, process the form data
// In a real application, this might save to a database, send an email, etc.

// Store form data in session for display on success page
session_start();
$_SESSION['contact_form'] = [
    'name' => $name,
    'email' => $email,
    'subject' => $subject,
    'message' => $message,
    'submitted_at' => date('Y-m-d H:i:s')
];

// Demonstrate accessing form data through different methods

// Method 1: Direct access through $_POST
$post_data = [
    'name' => $_POST['name'] ?? 'Not provided',
    'email' => $_POST['email'] ?? 'Not provided',
    'subject' => $_POST['subject'] ?? 'Not provided',
    'message' => $_POST['message'] ?? 'Not provided'
];

// Method 2: Access through $_FORM (Frango specific)
$form_data = [
    'name' => $_FORM['name'] ?? 'Not provided',
    'email' => $_FORM['email'] ?? 'Not provided',
    'subject' => $_FORM['subject'] ?? 'Not provided',
    'message' => $_FORM['message'] ?? 'Not provided'
];

// Store both for comparison on the success page
$_SESSION['access_methods'] = [
    'post' => $post_data,
    'form' => $form_data
];

// Redirect to success page
header('Location: /success');
exit; 