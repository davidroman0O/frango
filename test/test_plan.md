# Complete Test Plan for Frango

## Core Request/Response Tests
1. Basic PHP script execution with plain text response
2. PHP script generating HTML response
3. PHP script handling query parameters
4. PHP script accessing request headers
5. PHP script setting response headers
6. PHP script generating and returning JSON
7. PHP script generating and returning XML
8. Binary data (image/PDF) generation and serving

## Form Handling Tests
9. GET form parameter handling
10. POST form handling (application/x-www-form-urlencoded)
11. Multipart form data handling with file uploads
12. PHP script handling JSON request body

## Routing Tests
13. Path parameter extraction in routes
14. Direct PHP file access blocking
15. File system router with clean URLs
16. File system router with index.php handling
17. Method detection in filesystem routing
18. Nested route parameters
19. Optional parameters
20. Wildcard routes
21. Route priority handling

## PHP Environment Tests
22. PHP script including/requiring other PHP files
23. PHP environment variable access
24. Server variable access from PHP
25. PHP embedded files execution
26. Virtual filesystem file operations

## Error Handling Tests
27. Error handling for PHP syntax errors
28. Error handling for PHP runtime errors
29. Custom 404 handler with PHP template
30. Custom 500 handler with error details in dev mode
31. Error page template with layout
32. PHP error display configuration
33. Custom error handling with readable stack traces

## State & Session Tests
34. PHP session state persistence
35. Basic session-based authentication flow
36. JWT token handling between PHP and Go
37. Redis connecting for session storage
38. Sharing data between Go and PHP via Redis

## AJAX & WebSocket Tests
39. PHP making Ajax requests to another PHP endpoint
40. PHP making Ajax requests to Go endpoint
41. Go serving PHP that returns AJAX-consumable data
42. PHP triggering WebSocket messages from Go server
43. WebSocket connection handling in PHP
44. Real-time updates from Redis to WebSocket clients

## Security Tests
45. CSRF token generation and validation
46. XSS prevention techniques
47. CORS header configuration
48. Content Security Policy implementation
49. Automated OAuth2 flow with authorization code exchange

## Developer Experience Tests
50. Development mode file change detection
51. Request/response logging between Go and PHP
52. PHP file change detection and browser refresh
53. Asset change detection
54. Development workflow optimizations 