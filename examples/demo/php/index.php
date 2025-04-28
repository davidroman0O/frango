<?php
/**
 * Frango v2 Demo - Main Dashboard
 */
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Frango v2 Demo</title>
    <style>
        :root {
            --primary-color: #007bff;
            --secondary-color: #6c757d;
            --background-color: #f8f9fa;
            --card-background: #ffffff;
            --text-color: #333;
            --link-color: #0056b3;
            --border-color: #dee2e6;
            --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            --border-radius: 4px;
            --box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        body {
            font-family: var(--font-family);
            background-color: var(--background-color);
            color: var(--text-color);
            margin: 0;
            padding: 20px;
            line-height: 1.5;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
        }
        h1 {
            color: var(--primary-color);
            border-bottom: 2px solid var(--primary-color);
            padding-bottom: 10px;
            margin-bottom: 20px;
        }
        h2 {
            color: var(--secondary-color);
            margin-top: 30px;
            margin-bottom: 15px;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 5px;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        .card {
            background-color: var(--card-background);
            border: 1px solid var(--border-color);
            border-radius: var(--border-radius);
            padding: 15px;
            box-shadow: var(--box-shadow);
        }
        .card h3 {
            margin-top: 0;
            color: var(--primary-color);
            font-size: 1.1em;
        }
        .card ul {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .card li {
            margin-bottom: 8px;
        }
        .card a {
            color: var(--link-color);
            text-decoration: none;
            display: block;
            padding: 5px 0;
            transition: background-color 0.2s ease;
        }
        .card a:hover {
            background-color: #f0f0f0;
            text-decoration: underline;
        }
        .card p {
            font-size: 0.9em;
            color: #555;
            margin-top: 5px;
        }
        code {
            background-color: #e9ecef;
            padding: 2px 4px;
            border-radius: var(--border-radius);
            font-size: 0.9em;
        }
        .debug-link {
             position: fixed;
             bottom: 10px;
             right: 10px;
             background: var(--secondary-color);
             color: white;
             padding: 8px 12px;
             border-radius: var(--border-radius);
             text-decoration: none;
             box-shadow: 0 2px 5px rgba(0,0,0,0.2);
             font-size: 0.9em;
             z-index: 1000; /* Ensure it's above debug panel */
        }
        .debug-link:hover {
             background: #5a6268;
             color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Frango v2 Demo Dashboard</h1>
        <p>Welcome! This demo showcases various features of the Frango v2 Go-PHP middleware.</p>

        <h2>Checklist Compliance</h2>
        <p>These pages demonstrate compliance with standard PHP expectations (from <code>checklist.md</code>).</p>
        <div class="grid">
            <div class="card">
                <h3>Superglobals & Forms</h3>
                <ul>
                    <li><a href="/checklist/superglobals"><code>$_GET</code>, <code>$_POST</code>, <code>$_FILES</code>, <code>$_COOKIE</code>, <code>$_REQUEST</code></a></li>
                    <li><p>Demonstrates basic form handling and superglobal population.</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Server Variables</h3>
                <ul>
                    <li><a href="/checklist/server"><code>$_SERVER</code> Key Variables</a></li>
                    <li><p>Shows important <code>$_SERVER</code> values like <code>REQUEST_URI</code>, <code>SCRIPT_NAME</code>, etc.</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Input Stream</h3>
                <ul>
                    <li><a href="/checklist/php_input"><code>php://input</code></a></li>
                    <li><p>Tests reading raw request body (use with PUT/POST).</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Sessions</h3>
                <ul>
                    <li><a href="/checklist/sessions"><code>$_SESSION</code> Handling</a></li>
                    <li><p>Verifies session start and variable persistence.</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Paths & Includes</h3>
                <ul>
                    <li><a href="/checklist/paths">Paths, Includes, <code>__DIR__</code></a></li>
                    <li><p>Checks <code>__FILE__</code>, <code>__DIR__</code>, <code>getcwd()</code> and relative/absolute includes.</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Environment</h3>
                <ul>
                    <li><a href="/checklist/env"><code>$_ENV</code> / <code>getenv()</code></a></li>
                    <li><p>Shows access to environment variables set by Go.</p></li>
                </ul>
            </div>
        </div>

        <h2>Routing Features</h2>
        <p>Examples of URL routing and parameter extraction.</p>
        <div class="grid">
            <div class="card">
                <h3>Path Parameters</h3>
                <ul>
                    <li><a href="/users/123">User Profile (<code>/users/{userID}</code>)</a></li>
                    <li><a href="/products/abc?color=blue&size=L">Product Detail (<code>/products/{productID}</code>)</a></li>
                    <li><a href="/categories/electronics/smartphones">Category Page (<code>/categories/{cat}/{subcat}</code>)</a></li>
                    <li><p>Demonstrates accessing named parameters from the URL path via <code>$_GET</code> (merged).</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Query Parameters</h3>
                <ul>
                    <li><a href="/search?q=frango&sort=date&filters[]=php&filters[]=go">Search Results (<code>/search</code>)</a></li>
                    <li><p>Shows handling of standard query string parameters via <code>$_GET</code>.</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Path Segments</h3>
                <ul>
                    <li><a href="/segments/show/one/two/three">Path Segments (<code>/segments/show/*</code>)</a></li>
                    <li><p>Shows access to raw URL path segments via <code>$GLOBALS['_PATH_SEGMENTS']</code>.</p></li>
                </ul>
            </div>
        </div>

        <h2>Form Handling</h2>
        <p>Demonstrations of processing different types of form submissions.</p>
        <div class="grid">
            <div class="card">
                <h3>Forms Index</h3>
                <ul>
                    <li><a href="/forms">All Form Types</a></li>
                    <li><p>A page containing various forms to test different handlers.</p></li>
                </ul>
            </div>
             <div class="card">
                <h3>Go vs PHP Upload</h3>
                <ul>
                     <li><a href="/forms">PHP Upload Handling (via Forms Index)</a></li>
                     <li><a href="/forms">Go Upload Handling (via Forms Index)</a></li>
                    <li><p>Compare file uploads processed by PHP (<code>$_FILES</code>) vs. a direct Go handler.</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>JSON Handling</h3>
                 <ul>
                     <li><a href="/forms">JSON Request (via Forms Index)</a></li>
                     <li><p>Shows processing of <code>application/json</code> request bodies via <code>$_JSON</code>.</p></li>
                 </ul>
            </div>
        </div>

        <h2>Integration & Dev Features</h2>
        <p>Examples of Go-PHP interaction and development utilities.</p>
        <div class="grid">
            <div class="card">
                <h3>Go -> PHP Rendering</h3>
                <ul>
                    <li><a href="/render">Render with Go Data</a></li>
                    <li><p>Demonstrates injecting data from Go into a PHP template via <code>middleware.Render()</code> (access via <code>$GLOBALS['_TEMPLATE']</code>).</p></li>
                </ul>
            </div>
            <div class="card">
                <h3>Go API Endpoint</h3>
                <ul>
                    <li><a href="/api/time" target="_blank">Simple Go API</a></li>
                    <li><p>A basic API endpoint implemented entirely in Go.</p></li>
                </ul>
            </div>
             <div class="card">
                <h3>Debug Panel</h3>
                <ul>
                    <li><a href="/debug-panel">Test Debug Panel</a></li>
                    <li><p>Shows the reusable debug panel displaying Frango-specific variables.</p></li>
                </ul>
            </div>
        </div>

    </div>

    <!-- Link to open debug panel (if included separately) -->
    <!-- <a href="/debug-panel" class="debug-link" target="_blank">Debug</a> -->

    <!-- Auto-reload script will be injected here by Frango if enabled -->
</body>
</html> 