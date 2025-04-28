<?php
/**
 * Blog post page demonstrating URL parameters with slugs
 * This file matches the route pattern /blog/post/{slug}
 */

include_once($_SERVER['DOCUMENT_ROOT'] . '/lib/utils.php');

// Get the slug parameter from the URL
$slug = $_PATH['slug'] ?? '';

// In a real app, you would fetch the post from a database using the slug
// For demo, we'll have a few predefined posts
$posts = [
    'getting-started-with-frango' => [
        'id' => 1,
        'title' => 'Getting Started with Frango',
        'slug' => 'getting-started-with-frango',
        'content' => <<<HTML
<p class="lead">Frango brings together the ease of PHP development with the performance and concurrency of Go.</p>

<p>In this tutorial, we'll walk through setting up your first Frango project.</p>

<h2>Prerequisites</h2>
<ul>
    <li>Go 1.18 or higher</li>
    <li>PHP 7.4 or higher</li>
</ul>

<h2>Step 1: Install Frango</h2>
<pre><code>go get github.com/username/frango</code></pre>

<h2>Step 2: Create a Basic Project</h2>
<p>Create a new directory for your project:</p>
<pre><code>mkdir myproject
cd myproject</code></pre>

<p>Create a main.go file with the following content:</p>
<pre><code>package main

import (
    "log"
    "net/http"
    
    "github.com/username/frango"
)

func main() {
    // Initialize Frango with your PHP source directory
    php, err := frango.New(
        frango.WithSourceDir("./web"),
        frango.WithDevelopmentMode(true),
    )
    if err != nil {
        log.Fatalf("Error initializing Frango: %v", err)
    }
    defer php.Shutdown()
    
    // Create a router
    router := php.NewConventionalRouter(nil)
    
    // Start the server
    log.Println("Server starting on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", router.Handler()))
}</code></pre>

<h2>Step 3: Create Your First PHP Page</h2>
<p>Create a directory for your PHP files:</p>
<pre><code>mkdir -p web/pages</code></pre>

<p>Create a simple PHP file at web/pages/hello.php:</p>
<pre><code>&lt;?php
echo "Hello from Frango!";
?&gt;</code></pre>

<h2>Step 4: Run Your Application</h2>
<pre><code>go run main.go</code></pre>

<p>Now visit <a href="http://localhost:8080/pages/hello">http://localhost:8080/pages/hello</a> in your browser. You should see your PHP page rendered!</p>

<h2>What's Next?</h2>
<p>Now that you have a basic Frango application running, you can explore more features like:</p>
<ul>
    <li>RESTful API patterns with HTTP method suffixes</li>
    <li>URL parameters with {param} syntax</li>
    <li>Virtual filesystem for embedded resources</li>
    <li>Advanced routing strategies</li>
</ul>
HTML,
        'date' => '2023-10-15',
        'author' => 'John Doe',
        'category' => 'tech',
        'image' => 'https://images.unsplash.com/photo-1661956602868-6ae368943878?ixlib=rb-4.0.3&ixid=MnwxMjA3fDF8MHxlZGl0b3JpYWwtZmVlZHwxfHx8ZW58MHx8fHw%3D&auto=format&fit=crop&w=800&q=60',
    ],
    'building-restful-apis' => [
        'id' => 2,
        'title' => 'Building RESTful APIs with Conventional Routing',
        'slug' => 'building-restful-apis',
        'content' => '<p class="lead">Frango makes it easy to build RESTful APIs using conventional file naming patterns.</p><p>For example, a file named users.get.php will handle GET requests to /users, while users.post.php will handle POST requests.</p><p>This makes your API structure clear and easy to maintain.</p>',
        'date' => '2023-09-28',
        'author' => 'Jane Smith',
        'category' => 'web',
        'image' => 'https://images.unsplash.com/photo-1516116216624-53e697fedbea?ixlib=rb-4.0.3&ixid=MnwxMjA3fDB8MHxzZWFyY2h8NXx8d2ViJTIwZGV2ZWxvcG1lbnR8ZW58MHx8MHx8&auto=format&fit=crop&w=800&q=60',
    ],
    'php-and-go-best-of-both-worlds' => [
        'id' => 3,
        'title' => 'PHP and Go: The Best of Both Worlds',
        'slug' => 'php-and-go-best-of-both-worlds',
        'content' => '<p class="lead">PHP is known for its rapid development capabilities, while Go excels at performance and concurrency.</p><p>Frango combines these strengths, letting you write your application logic in familiar PHP while leveraging Go\'s performance for the underlying server infrastructure.</p>',
        'date' => '2023-09-10',
        'author' => 'Robert Johnson',
        'category' => 'tech',
        'image' => 'https://images.unsplash.com/photo-1537432376769-00f5c2f4c8d2?ixlib=rb-4.0.3&ixid=MnwxMjA3fDB8MHxzZWFyY2h8MTB8fHRlY2h8ZW58MHx8MHx8&auto=format&fit=crop&w=800&q=60',
    ],
];

// Get the post or show a 404 message
$post = $posts[$slug] ?? null;

if ($post) {
    page_header($post['title']);
} else {
    page_header('Post Not Found');
}
?>

<div class="container py-5">
    <?php if ($post): ?>
        <div class="row">
            <div class="col-lg-8 mx-auto">
                <img src="<?= htmlspecialchars($post['image']) ?>" alt="<?= htmlspecialchars($post['title']) ?>" class="img-fluid rounded mb-4">
                
                <h1 class="display-5 fw-bold"><?= htmlspecialchars($post['title']) ?></h1>
                
                <div class="text-muted mb-4">
                    <span>Posted on <?= htmlspecialchars($post['date']) ?></span> • 
                    <span>By <?= htmlspecialchars($post['author']) ?></span> • 
                    <span>Category: <a href="/blog/category/<?= htmlspecialchars($post['category']) ?>"><?= htmlspecialchars(ucfirst($post['category'])) ?></a></span>
                </div>
                
                <div class="blog-content">
                    <?= $post['content'] ?>
                </div>
                
                <hr class="my-5">
                
                <div class="alert alert-info">
                    <h4>URL Parameter Demo</h4>
                    <p>This page demonstrates URL parameters with slugs. The URL pattern <code>/blog/post/{slug}</code> extracts the slug and makes it available in the <code>$_PATH</code> superglobal.</p>
                    <p>Try different posts:</p>
                    <ul>
                        <li><a href="/blog/post/getting-started-with-frango">Getting Started with Frango</a></li>
                        <li><a href="/blog/post/building-restful-apis">Building RESTful APIs</a></li>
                        <li><a href="/blog/post/php-and-go-best-of-both-worlds">PHP and Go: Best of Both Worlds</a></li>
                        <li><a href="/blog/post/non-existent-post">Non-existent Post</a></li>
                    </ul>
                    <p>The parameter value is: <code>$_PATH['slug'] = '<?= htmlspecialchars($slug) ?>'</code></p>
                </div>
                
                <div class="mt-5">
                    <a href="/blog" class="btn btn-outline-primary">&larr; Back to Blog</a>
                </div>
            </div>
        </div>
    <?php else: ?>
        <div class="row">
            <div class="col-md-8 mx-auto text-center">
                <div class="alert alert-warning">
                    <h2>Post Not Found</h2>
                    <p>Sorry, we couldn't find a blog post with the slug "<?= htmlspecialchars($slug) ?>".</p>
                    <p>The URL parameter value is: <code>$_PATH['slug'] = '<?= htmlspecialchars($slug) ?>'</code></p>
                    <p>Try one of these posts instead:</p>
                    <ul class="list-unstyled">
                        <li><a href="/blog/post/getting-started-with-frango">Getting Started with Frango</a></li>
                        <li><a href="/blog/post/building-restful-apis">Building RESTful APIs</a></li>
                        <li><a href="/blog/post/php-and-go-best-of-both-worlds">PHP and Go: Best of Both Worlds</a></li>
                    </ul>
                </div>
                <a href="/blog" class="btn btn-primary mt-3">Back to Blog</a>
            </div>
        </div>
    <?php endif; ?>
</div>

<?php
// End the page with the layout footer
page_footer();
?> 