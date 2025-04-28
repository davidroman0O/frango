<?php
// Start session
session_start();

// Ensure this is a POST request
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: /login');
    exit;
}

// Get login credentials
$username = trim($_POST['username'] ?? '');
$password = $_POST['password'] ?? '';

// Validate inputs
if (empty($username) || empty($password)) {
    header('Location: /login?error=empty');
    exit;
}

// Demo users for this example
$users = [
    'admin' => [
        'password' => 'password',
        'role' => 'admin',
        'full_name' => 'Administrator',
        'email' => 'admin@example.com'
    ],
    'user' => [
        'password' => 'password',
        'role' => 'user',
        'full_name' => 'Regular User',
        'email' => 'user@example.com'
    ]
];

// Check if user exists and password is correct
if (isset($users[$username]) && $users[$username]['password'] === $password) {
    // Authentication successful
    
    // Store user information in session
    $_SESSION['user'] = [
        'username' => $username,
        'role' => $users[$username]['role'],
        'full_name' => $users[$username]['full_name'],
        'email' => $users[$username]['email'],
        'login_time' => time(),
        'last_activity' => time()
    ];
    
    // Store some additional session data for demonstration
    $_SESSION['visit_count'] = ($_SESSION['visit_count'] ?? 0) + 1;
    $_SESSION['login_history'][] = [
        'time' => time(),
        'ip' => $_SERVER['REMOTE_ADDR']
    ];
    
    // Redirect to dashboard
    header('Location: /dashboard');
    exit;
} else {
    // Authentication failed
    header('Location: /login?error=invalid');
    exit;
} 