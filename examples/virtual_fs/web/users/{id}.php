<?php
/**
 * User profile page demonstrating URL parameters
 * This file matches the route pattern /users/{id}
 */

include_once($_SERVER['DOCUMENT_ROOT'] . '/lib/utils.php');

// Start the page with the layout header
page_header('User Profile');

// Get the user ID from the URL parameter
// In a conventional router setup, the {id} in the filename becomes a URL parameter
$userId = $_PATH['id'] ?? 'unknown';

// In a real app, you would fetch user data from a database
// For demo, we'll simulate different users
$users = [
    '42' => [
        'id' => 42,
        'name' => 'John Doe',
        'email' => 'john@example.com',
        'role' => 'Administrator',
        'joined' => '2022-01-15',
        'avatar' => 'https://randomuser.me/api/portraits/men/32.jpg',
    ],
    '100' => [
        'id' => 100,
        'name' => 'Jane Smith',
        'email' => 'jane@example.com',
        'role' => 'Editor',
        'joined' => '2022-03-22',
        'avatar' => 'https://randomuser.me/api/portraits/women/44.jpg',
    ],
];

// Get the user data or use a default
$user = $users[$userId] ?? [
    'id' => $userId,
    'name' => 'Example User',
    'email' => 'user@example.com',
    'role' => 'Member',
    'joined' => 'Unknown',
    'avatar' => 'https://randomuser.me/api/portraits/lego/1.jpg',
];
?>

<div class="container py-5">
    <div class="row">
        <div class="col-lg-4">
            <div class="card mb-4">
                <div class="card-body text-center">
                    <img src="<?= htmlspecialchars($user['avatar']) ?>" alt="avatar" class="rounded-circle img-fluid" style="width: 150px;">
                    <h5 class="my-3"><?= htmlspecialchars($user['name']) ?></h5>
                    <p class="text-muted mb-1"><?= htmlspecialchars($user['role']) ?></p>
                    <p class="text-muted mb-4">Member since: <?= htmlspecialchars($user['joined']) ?></p>
                    <div class="d-flex justify-content-center mb-2">
                        <a href="/messages/new?to=<?= urlencode($userId) ?>" class="btn btn-primary">Message</a>
                        <a href="/users" class="btn btn-outline-primary ms-1">Back to Users</a>
                    </div>
                </div>
            </div>
        </div>
        <div class="col-lg-8">
            <div class="card mb-4">
                <div class="card-body">
                    <div class="row">
                        <div class="col-sm-3">
                            <p class="mb-0">User ID</p>
                        </div>
                        <div class="col-sm-9">
                            <p class="text-muted mb-0"><?= htmlspecialchars($user['id']) ?></p>
                        </div>
                    </div>
                    <hr>
                    <div class="row">
                        <div class="col-sm-3">
                            <p class="mb-0">Full Name</p>
                        </div>
                        <div class="col-sm-9">
                            <p class="text-muted mb-0"><?= htmlspecialchars($user['name']) ?></p>
                        </div>
                    </div>
                    <hr>
                    <div class="row">
                        <div class="col-sm-3">
                            <p class="mb-0">Email</p>
                        </div>
                        <div class="col-sm-9">
                            <p class="text-muted mb-0"><?= htmlspecialchars($user['email']) ?></p>
                        </div>
                    </div>
                    <hr>
                    <div class="row">
                        <div class="col-sm-3">
                            <p class="mb-0">Role</p>
                        </div>
                        <div class="col-sm-9">
                            <p class="text-muted mb-0"><?= htmlspecialchars($user['role']) ?></p>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="alert alert-info">
                <h4>URL Parameter Demo</h4>
                <p>This page demonstrates URL parameters in action. The URL pattern <code>/users/{id}</code> extracts the ID and makes it available in the <code>$_PATH</code> superglobal.</p>
                <p>Try visiting different user profiles:</p>
                <ul>
                    <li><a href="/users/42">User #42</a></li>
                    <li><a href="/users/100">User #100</a></li>
                    <li><a href="/users/999">Non-existent User</a></li>
                </ul>
                <p>The parameter value is: <code>$_PATH['id'] = '<?= htmlspecialchars($userId) ?>'</code></p>
            </div>
        </div>
    </div>
</div>

<?php
// End the page with the layout footer
page_footer();
?> 