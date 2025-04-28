<?php
/**
 * Blog index page demonstrating conventional routing
 * This file matches the route pattern /blog
 */

include_once($_SERVER['DOCUMENT_ROOT'] . '/lib/utils.php');

// Start the page with the layout header
page_header('Blog - Latest Posts');

// In a real app, these would come from a database
$blogPosts = [
    [
        'id' => 1,
        'title' => 'Getting Started with Frango',
        'slug' => 'getting-started-with-frango',
        'excerpt' => 'Learn how to set up your first Frango project and build a PHP application powered by Go.',
        'date' => '2023-10-15',
        'author' => 'John Doe',
        'category' => 'tech',
        'image' => 'https://images.unsplash.com/photo-1661956602868-6ae368943878?ixlib=rb-4.0.3&ixid=MnwxMjA3fDF8MHxlZGl0b3JpYWwtZmVlZHwxfHx8ZW58MHx8fHw%3D&auto=format&fit=crop&w=500&q=60',
    ],
    [
        'id' => 2,
        'title' => 'Building RESTful APIs with Conventional Routing',
        'slug' => 'building-restful-apis',
        'excerpt' => 'Discover how to create clean, maintainable RESTful APIs using Frango\'s conventional routing patterns.',
        'date' => '2023-09-28',
        'author' => 'Jane Smith',
        'category' => 'web',
        'image' => 'https://images.unsplash.com/photo-1516116216624-53e697fedbea?ixlib=rb-4.0.3&ixid=MnwxMjA3fDB8MHxzZWFyY2h8NXx8d2ViJTIwZGV2ZWxvcG1lbnR8ZW58MHx8MHx8&auto=format&fit=crop&w=500&q=60',
    ],
    [
        'id' => 3,
        'title' => 'PHP and Go: The Best of Both Worlds',
        'slug' => 'php-and-go-best-of-both-worlds',
        'excerpt' => 'Why combining PHP and Go gives you the perfect balance of development speed and runtime performance.',
        'date' => '2023-09-10',
        'author' => 'Robert Johnson',
        'category' => 'tech',
        'image' => 'https://images.unsplash.com/photo-1537432376769-00f5c2f4c8d2?ixlib=rb-4.0.3&ixid=MnwxMjA3fDB8MHxzZWFyY2h8MTB8fHRlY2h8ZW58MHx8MHx8&auto=format&fit=crop&w=500&q=60',
    ],
];
?>

<div class="container py-4">
    <div class="p-4 p-md-5 mb-4 text-white rounded bg-dark">
        <div class="col-md-8 px-0">
            <h1 class="display-4 fst-italic">Latest from our Blog</h1>
            <p class="lead my-3">Explore articles about PHP, Go, web development, and the powerful combination of these technologies.</p>
            <p class="lead mb-0"><a href="/blog/category/tech" class="text-white fw-bold">Continue reading...</a></p>
        </div>
    </div>

    <div class="row mb-2">
        <?php foreach ($blogPosts as $post): ?>
        <div class="col-md-6">
            <div class="row g-0 border rounded overflow-hidden flex-md-row mb-4 shadow-sm h-md-250 position-relative">
                <div class="col p-4 d-flex flex-column position-static">
                    <strong class="d-inline-block mb-2 text-primary"><?= htmlspecialchars(ucfirst($post['category'])) ?></strong>
                    <h3 class="mb-0"><?= htmlspecialchars($post['title']) ?></h3>
                    <div class="mb-1 text-muted"><?= htmlspecialchars($post['date']) ?></div>
                    <p class="card-text mb-auto"><?= htmlspecialchars($post['excerpt']) ?></p>
                    <a href="/blog/post/<?= htmlspecialchars($post['slug']) ?>" class="stretched-link">Continue reading</a>
                </div>
                <div class="col-auto d-none d-lg-block">
                    <img src="<?= htmlspecialchars($post['image']) ?>" width="200" height="250" class="bd-placeholder-img" alt="<?= htmlspecialchars($post['title']) ?>" style="object-fit: cover;">
                </div>
            </div>
        </div>
        <?php endforeach; ?>
    </div>

    <div class="row g-5">
        <div class="col-md-8">
            <h3 class="pb-4 mb-4 fst-italic border-bottom">
                From the Frango Community
            </h3>

            <article class="blog-post">
                <h2 class="blog-post-title">Conventional Routing Explained</h2>
                <p class="blog-post-meta">January 1, 2023 by <a href="#">Mark</a></p>

                <p>Frango uses a simple, convention-based approach to routing. By following naming conventions, your PHP files automatically map to routes without additional configuration.</p>
                <hr>
                <p>For example, here are how files map to routes:</p>
                <ul>
                    <li><code>/blog/index.php</code> → <code>/blog</code></li>
                    <li><code>/blog/post/{slug}.php</code> → <code>/blog/post/getting-started</code></li>
                    <li><code>/api/users.get.php</code> → <code>GET /api/users</code></li>
                    <li><code>/api/users.post.php</code> → <code>POST /api/users</code></li>
                </ul>
                <p>This approach results in clean, maintainable code with a clear structure.</p>
            </article>

            <nav class="blog-pagination" aria-label="Pagination">
                <a class="btn btn-outline-primary" href="#">Older</a>
                <a class="btn btn-outline-secondary disabled" tabindex="-1" aria-disabled="true">Newer</a>
            </nav>
        </div>

        <div class="col-md-4">
            <div class="position-sticky" style="top: 2rem;">
                <div class="p-4 mb-3 bg-light rounded">
                    <h4 class="fst-italic">About</h4>
                    <p class="mb-0">This blog demonstrates Frango's conventional routing system. Notice how the URL structure maps cleanly to the filesystem structure.</p>
                </div>

                <div class="p-4">
                    <h4 class="fst-italic">Categories</h4>
                    <ol class="list-unstyled mb-0">
                        <li><a href="/blog/category/tech">Technology</a></li>
                        <li><a href="/blog/category/web">Web Development</a></li>
                        <li><a href="/blog/category/go">Go</a></li>
                        <li><a href="/blog/category/php">PHP</a></li>
                    </ol>
                </div>

                <div class="p-4">
                    <h4 class="fst-italic">Archives</h4>
                    <ol class="list-unstyled mb-0">
                        <li><a href="#">March 2023</a></li>
                        <li><a href="#">February 2023</a></li>
                        <li><a href="#">January 2023</a></li>
                        <li><a href="#">December 2022</a></li>
                    </ol>
                </div>
            </div>
        </div>
    </div>
</div>

<?php
// End the page with the layout footer
page_footer();
?> 