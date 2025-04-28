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

### Form Handling Tests

The following tests should be implemented to ensure form handling works correctly:

1. **GET Parameters**
   - Verify query parameters are available through `$_SERVER['PHP_QUERY_*']` variables
   - Test simple GET forms with various parameter types (text, numbers)
   - Test URL encoding/decoding of special characters
   - Test array parameters (e.g., `?items[]=1&items[]=2`)

2. **POST Form Data**
   - Verify form fields are available through `$_SERVER['PHP_FORM_*']` variables 
   - Test application/x-www-form-urlencoded submissions
   - Test different field types (text, numbers, booleans)
   - Test empty submissions

3. **Multipart Form Data**
   - Test file uploads
   - Test mixed form fields and file uploads
   - Verify file metadata (name, size, type)
   - Test large file handling

4. **JSON Request Bodies**
   - Test JSON request body parsing via `$_SERVER['PHP_JSON']`
   - Test nested JSON structures
   - Verify individual JSON fields are accessible via `$_SERVER['PHP_JSON_*']`
   - Test error handling for malformed JSON

### Important Note on Form Processing

Due to the implementation of FrankenPHP, form data is not available through the standard PHP superglobal arrays (`$_GET`, `$_POST`). Instead, form data is exposed through `$_SERVER` variables with different prefixes:

- GET parameters: `$_SERVER['PHP_QUERY_paramname']`
- POST form fields: `$_SERVER['PHP_FORM_fieldname']`
- JSON data: `$_SERVER['PHP_JSON']` and `$_SERVER['PHP_JSON_propname']`

All tests must check for form data in these environment variables rather than the standard PHP superglobals. 