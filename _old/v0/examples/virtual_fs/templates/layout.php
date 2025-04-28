<?php
/**
 * Main layout template (embedded file)
 */
?><!DOCTYPE html>
<html>
<head>
    <title><?= isset($title) ? $title : 'Frango Demo' ?></title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- Bootstrap CSS for styling -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            line-height: 1.6;
        }
        pre {
            background-color: #f5f5f5;
            padding: 15px;
            border-radius: 5px;
            overflow: auto;
        }
        .header {
            margin-bottom: 20px;
            padding-bottom: 10px;
        }
        .footer {
            border-top: 1px solid #eee;
            margin-top: 40px;
            padding: 30px 0;
            background-color: #f8f9fa;
            text-align: center;
        }
        .nav-pills .nav-link.active {
            background-color: #563d7c;
        }
    </style>
</head>
<body>
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark">
        <div class="container">
            <a class="navbar-brand" href="/">Frango Demo</a>
            <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
                <span class="navbar-toggler-icon"></span>
            </button>
            <div class="collapse navbar-collapse" id="navbarNav">
                <ul class="navbar-nav me-auto">
                    <li class="nav-item dropdown">
                        <a class="nav-link dropdown-toggle" href="#" id="navbarDropdownPages" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                            Pages
                        </a>
                        <ul class="dropdown-menu" aria-labelledby="navbarDropdownPages">
                            <li><a class="dropdown-item" href="/pages/about">About Us</a></li>
                            <li><a class="dropdown-item" href="/pages/contact">Contact</a></li>
                            <li><a class="dropdown-item" href="/pages/features">Features</a></li>
                            <li><hr class="dropdown-divider"></li>
                            <li><a class="dropdown-item" href="/pages/pricing">Pricing</a></li>
                        </ul>
                    </li>
                    <li class="nav-item dropdown">
                        <a class="nav-link dropdown-toggle" href="#" id="navbarDropdownProducts" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                            Products
                        </a>
                        <ul class="dropdown-menu" aria-labelledby="navbarDropdownProducts">
                            <li><a class="dropdown-item" href="/products">All Products</a></li>
                            <li><a class="dropdown-item" href="/products/featured">Featured</a></li>
                            <li><a class="dropdown-item" href="/products/category/electronics">Electronics</a></li>
                            <li><a class="dropdown-item" href="/products/category/clothing">Clothing</a></li>
                        </ul>
                    </li>
                    <li class="nav-item dropdown">
                        <a class="nav-link dropdown-toggle" href="#" id="navbarDropdownBlog" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                            Blog
                        </a>
                        <ul class="dropdown-menu" aria-labelledby="navbarDropdownBlog">
                            <li><a class="dropdown-item" href="/blog">Latest Posts</a></li>
                            <li><a class="dropdown-item" href="/blog/category/tech">Technology</a></li>
                            <li><a class="dropdown-item" href="/blog/category/web">Web Development</a></li>
                        </ul>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="/api/docs">API Docs</a>
                    </li>
                </ul>
                <ul class="navbar-nav">
                    <li class="nav-item">
                        <a class="nav-link" href="/admin/dashboard">Admin</a>
                    </li>
                    <li class="nav-item">
                        <a class="btn btn-outline-light ms-2" href="/users/login">Login</a>
                    </li>
                </ul>
            </div>
        </div>
    </nav>
    
    <div class="container mt-4">
        <div class="content">
            <?= $content ?? 'No content provided' ?>
        </div>
        
        <footer class="footer">
            <div class="container">
                <div class="row">
                    <div class="col-md-4">
                        <h5>About Frango</h5>
                        <p>A powerful Go library for integrating PHP with the Go ecosystem.</p>
                    </div>
                    <div class="col-md-4">
                        <h5>Quick Links</h5>
                        <ul class="list-unstyled">
                            <li><a href="/pages/about">About</a></li>
                            <li><a href="/pages/contact">Contact</a></li>
                            <li><a href="/pages/terms">Terms of Service</a></li>
                            <li><a href="/pages/privacy">Privacy Policy</a></li>
                        </ul>
                    </div>
                    <div class="col-md-4">
                        <h5>Technical Info</h5>
                        <p>
                            <?php 
                            if (function_exists('get_app_info')) {
                                $info = get_app_info();
                                echo $info['name'] . ' v' . $info['version'];
                                echo '<br>PHP ' . PHP_VERSION;
                            }
                            ?>
                        </p>
                    </div>
                </div>
                <div class="row mt-3">
                    <div class="col-12 text-center">
                        <p class="mb-0">&copy; <?= date('Y') ?> Frango Demo - Powered by Go & PHP</p>
                    </div>
                </div>
            </div>
        </footer>
    </div>
    
    <!-- Bootstrap JavaScript bundle with Popper -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html> 