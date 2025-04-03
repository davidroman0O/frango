<?php
/**
 * API Documentation page
 */

include_once($_SERVER['DOCUMENT_ROOT'] . '/lib/utils.php');

// Start the page with the layout header
page_header('API Documentation');
?>

<div class="container py-5">
    <div class="row">
        <div class="col-md-3">
            <!-- Sidebar navigation -->
            <div class="sticky-top" style="top: 2rem;">
                <div class="list-group">
                    <a href="#introduction" class="list-group-item list-group-item-action">Introduction</a>
                    <a href="#conventional-routing" class="list-group-item list-group-item-action">Conventional Routing</a>
                    <a href="#endpoints" class="list-group-item list-group-item-action">API Endpoints</a>
                    <a href="#products-api" class="list-group-item list-group-item-action">Products API</a>
                    <a href="#users-api" class="list-group-item list-group-item-action">Users API</a>
                    <a href="#authentication" class="list-group-item list-group-item-action">Authentication</a>
                    <a href="#error-handling" class="list-group-item list-group-item-action">Error Handling</a>
                </div>
            </div>
        </div>
        
        <div class="col-md-9">
            <!-- API Documentation content -->
            <div id="introduction" class="mb-5">
                <h2>Introduction</h2>
                <p class="lead">Welcome to the Frango API Documentation</p>
                <p>
                    This documentation describes the RESTful API endpoints available in this demo application.
                    All API endpoints follow RESTful conventions and return JSON responses.
                </p>
                <p>
                    Base URL: <code>http://localhost:8080/api</code>
                </p>
            </div>
            
            <div id="conventional-routing" class="mb-5">
                <h2>Conventional Routing</h2>
                <p>
                    Frango uses a conventional routing approach where the filesystem structure and file naming patterns
                    determine the routes and HTTP methods supported.
                </p>
                
                <div class="card mb-4">
                    <div class="card-header">File Naming Conventions</div>
                    <div class="card-body">
                        <p>Files are named according to these patterns:</p>
                        <ul>
                            <li><code>resource.{method}.php</code> - For method-specific endpoints (GET, POST, PUT, DELETE)</li>
                            <li><code>resource/{id}.php</code> - For parameterized routes</li>
                            <li><code>resource/index.php</code> - For resource collection endpoints</li>
                        </ul>
                        
                        <p>Examples:</p>
                        <ul>
                            <li><code>/api/products.get.php</code> → Responds to <code>GET /api/products</code></li>
                            <li><code>/api/products.post.php</code> → Responds to <code>POST /api/products</code></li>
                            <li><code>/api/products/{id}.php</code> → Responds to <code>ANY /api/products/123</code></li>
                            <li><code>/api/products/{id}.delete.php</code> → Responds to <code>DELETE /api/products/123</code></li>
                        </ul>
                    </div>
                </div>
            </div>
            
            <div id="endpoints" class="mb-5">
                <h2>API Endpoints</h2>
                <p>The following API endpoints are available in this demo:</p>
                
                <table class="table table-striped">
                    <thead>
                        <tr>
                            <th>Method</th>
                            <th>Endpoint</th>
                            <th>Description</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td><span class="badge bg-success">GET</span></td>
                            <td><code>/api/products</code></td>
                            <td>Get all products</td>
                        </tr>
                        <tr>
                            <td><span class="badge bg-primary">POST</span></td>
                            <td><code>/api/products</code></td>
                            <td>Create a new product</td>
                        </tr>
                        <tr>
                            <td><span class="badge bg-success">GET</span></td>
                            <td><code>/api/users</code></td>
                            <td>Get all users</td>
                        </tr>
                        <tr>
                            <td><span class="badge bg-primary">POST</span></td>
                            <td><code>/api/users</code></td>
                            <td>Create a new user</td>
                        </tr>
                        <tr>
                            <td><span class="badge bg-success">GET</span></td>
                            <td><code>/api/hello</code></td>
                            <td>Simple hello world endpoint</td>
                        </tr>
                    </tbody>
                </table>
            </div>
            
            <div id="products-api" class="mb-5">
                <h2>Products API</h2>
                
                <div class="card mb-4">
                    <div class="card-header bg-success text-white">
                        <strong>GET</strong> /api/products
                    </div>
                    <div class="card-body">
                        <h5>Description</h5>
                        <p>Retrieves a list of products with optional filtering</p>
                        
                        <h5>Query Parameters</h5>
                        <ul>
                            <li><code>category</code> - Filter by product category</li>
                            <li><code>limit</code> - Limit the number of results (default: 10, max: 50)</li>
                        </ul>
                        
                        <h5>Example Request</h5>
                        <pre><code>GET /api/products?category=clothing&limit=5</code></pre>
                        
                        <h5>Example Response</h5>
                        <pre><code>{
  "success": true,
  "products": [
    {
      "id": 1,
      "name": "Frango T-Shirt",
      "price": 24.99,
      "category": "clothing",
      "description": "Show your love for Frango with this comfortable cotton t-shirt",
      "image": "https://via.placeholder.com/300x300?text=Frango+Tshirt"
    },
    ...
  ],
  "metadata": {
    "total": 2,
    "filtered": true,
    "filter_category": "clothing",
    "limit": 5
  }
}</code></pre>
                        
                        <div class="mt-3">
                            <a href="/api/products" target="_blank" class="btn btn-sm btn-outline-success">Try it</a>
                            <a href="/api/products?category=clothing" target="_blank" class="btn btn-sm btn-outline-success">Try with category filter</a>
                        </div>
                    </div>
                </div>
                
                <div class="card mb-4">
                    <div class="card-header bg-primary text-white">
                        <strong>POST</strong> /api/products
                    </div>
                    <div class="card-body">
                        <h5>Description</h5>
                        <p>Creates a new product</p>
                        
                        <h5>Request Body</h5>
                        <pre><code>{
  "name": "New Product",
  "price": 29.99,
  "category": "electronics",
  "description": "This is a new product"
}</code></pre>
                        
                        <h5>Response</h5>
                        <pre><code>{
  "success": true,
  "message": "Product created successfully",
  "product": {
    "id": 123,
    "name": "New Product",
    "price": 29.99,
    "category": "electronics",
    "description": "This is a new product",
    "image": "https://via.placeholder.com/300x300?text=Product+Image",
    "created_at": "2023-10-15T14:30:00+00:00"
  }
}</code></pre>
                    </div>
                </div>
            </div>
            
            <div id="error-handling" class="mb-5">
                <h2>Error Handling</h2>
                <p>
                    All API endpoints return appropriate HTTP status codes and a consistent error format:
                </p>
                
                <pre><code>{
  "success": false,
  "message": "Error message",
  "errors": [
    "Detailed error 1",
    "Detailed error 2"
  ]
}</code></pre>
                
                <h5>Common HTTP Status Codes</h5>
                <ul>
                    <li><strong>200</strong> - Success</li>
                    <li><strong>201</strong> - Created</li>
                    <li><strong>400</strong> - Bad Request</li>
                    <li><strong>401</strong> - Unauthorized</li>
                    <li><strong>404</strong> - Not Found</li>
                    <li><strong>500</strong> - Server Error</li>
                </ul>
            </div>
            
            <div class="alert alert-info">
                <h4>Try the API</h4>
                <p>You can try the API endpoints directly in your browser or using tools like curl, Postman, or Insomnia.</p>
                <p>Example curl command for getting products:</p>
                <pre><code>curl -X GET "http://localhost:8080/api/products?category=electronics"</code></pre>
                
                <p>Example curl command for creating a product:</p>
                <pre><code>curl -X POST "http://localhost:8080/api/products" \
     -H "Content-Type: application/json" \
     -d '{"name":"Test Product","price":19.99,"category":"books","description":"Testing API"}'</code></pre>
            </div>
        </div>
    </div>
</div>

<?php
// End the page with the layout footer
page_footer();
?> 