# Guide to Coding with FrankenPHP (No Frameworks Required)

**FrankenPHP** is a modern PHP application server that integrates PHP directly into a Caddy web server environment. It aims to streamline PHP deployment and boost performance while letting you use familiar PHP code. This guide will help you transition from classic PHP (e.g., Apache with mod_php or PHP-FPM) to FrankenPHP, covering how it handles requests, superglobals, forms, file uploads, and AJAX – all without any frameworks. We’ll also discuss code organization, special considerations, and best practices for building APIs and dynamic pages.

## FrankenPHP vs Traditional PHP: What’s the Difference?

FrankenPHP can operate in two modes: **classic (traditional) mode** and **worker mode**. In classic mode, FrankenPHP behaves much like a traditional PHP setup – each HTTP request is handled by invoking a PHP script (like an `index.php` or the requested file) in isolation ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=The%20default%20mode%20with%20FrankenPHP,PHP%20in%20a%20sense%2C%20where)). This makes FrankenPHP a *drop-in replacement* for servers like Apache+PHP or PHP-FPM, meaning you can serve PHP files directly without changing your code structure ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/docs/classic/#:~:text=PHP%20files,FPM%20or%20Apache%20with%20mod_php)). In fact, by default FrankenPHP runs in classic mode, serving PHP files as-is, which is why it’s described as a seamless replacement for PHP-FPM or Apache mod_php.

Worker mode is where FrankenPHP truly innovates. In worker mode, you **boot your application once and keep it in memory**, then handle multiple requests on that single PHP instance. This can **greatly improve performance** (often 3× faster than PHP-FPM for typical apps ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/#:~:text=))) because your app’s initialization happens only once. Each incoming HTTP request is fed directly into the running PHP application in milliseconds. Worker mode is optional but powerful – if your app isn’t compatible with long-running processes, you can stick with classic mode.

**Key differences at a glance:**

- **Process model:** Traditional PHP (with FPM) spawns a new process or reuse processes per request; FrankenPHP (classic mode) uses threads in a pool to handle requests, and in worker mode uses persistent PHP workers for multiple requests.
- **Server integration:** FrankenPHP uses the Caddy web server under the hood. It embeds PHP inside Caddy, eliminating the FastCGI interface. This direct integration means no separate PHP-FPM service – **PHP runs in-process with the web server**.
- **Compatibility:** FrankenPHP doesn’t require any special frameworks or request objects. It **“uses plain old superglobals: no need for PSR-7”** to handle HTTP data ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/#:~:text=)). This design makes FrankenPHP compatible with any PHP application (even legacy ones), not just those built for modern PSR-7 request/response objects ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=request%20directly%20to%20the%20superglobals,7%20compatible)). For instance, WordPress, Drupal, or a custom PHP script all run fine since FrankenPHP populates `$_GET`, `$_POST`, `$_SERVER`, etc., just like a normal PHP environment.

In summary, if you deploy FrankenPHP in classic mode, you can continue coding exactly as you would on a traditional LAMP stack. If you opt for worker mode, you’ll need to structure your code to run as a long-lived worker (more on that below), but you gain speed and efficiency. Either way, the goal is to preserve PHP’s simplicity while enhancing performance.

## Request Handling in FrankenPHP

**Classic Mode Request Handling:** In classic mode, each HTTP request is handled separately, very much like on Apache or Nginx with PHP-FPM. When a request comes in, FrankenPHP (with Caddy) determines which PHP file should handle it, then invokes that script. By default, FrankenPHP’s Caddy integration will try to serve static files, but if a `.php` file is requested (or if URL rewriting directs to a PHP script), it will execute that PHP file and return the output ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=The%20default%20mode%20with%20FrankenPHP,PHP%20in%20a%20sense%2C%20where)). The `php_server` directive in the Caddy configuration automatically sets up sensible defaults: for example, it will add a trailing slash to directory requests, and if a requested file doesn’t exist, it will try an `index.php` (similar to `mod_rewrite` behavior in Apache). In practice, this means that pretty URLs can be supported by routing all unknown requests to `index.php` (which is common in many PHP apps). Classic mode **spawns a fresh PHP request context each time**, so there’s no state sharing or leakage between requests ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=The%20default%20mode%20with%20FrankenPHP,PHP%20in%20a%20sense%2C%20where)). You can use `header()` functions to set HTTP headers, output content normally with `echo` or templates, and call `http_response_code()` as needed – it all works the same as standard PHP.

**Worker Mode Request Handling:** In worker mode, you write a *worker script* (often an enhanced `index.php`) that initializes your application, then enters a loop to handle incoming requests one after another without exiting. FrankenPHP provides a function `frankenphp_handle_request($handler)` for this. You pass it a PHP closure (your request handler) and it will block until an HTTP request comes in, populate the PHP superglobals for that request, and then call your handler. After your handler echoes or returns a response, FrankenPHP sends that response back to the client, and `frankenphp_handle_request` returns a boolean indicating if it should continue listening. This design is similar to how workers in RoadRunner or Swoole work, but FrankenPHP’s handler uses native PHP superglobals instead of PSR-7 objects.

**Example (Worker Mode):** The documentation provides an example of a custom worker loop:

```php
<?php
ignore_user_abort(true); // Don’t terminate if client disconnects
// Boot (initialize) your app here
require __DIR__ . '/vendor/autoload.php';
// ... e.g., set up database connection, load configs, etc.

// Define a request handler closure
$handler = static function () {
    // This will run for each request:
    // superglobals like $_GET, $_POST, $_SERVER are already set for the request
    if ($_SERVER['REQUEST_METHOD'] === 'GET') {
        echo "Hello from FrankenPHP!";
    } else {
        // Handle other HTTP methods...
        echo "Received " . $_SERVER['REQUEST_METHOD'];
    }
};

// Enter the request loop
while (\frankenphp_handle_request($handler)) {
    // This loop will keep running, handling one request at a time.
    // You can perform cleanup or logging after each request if needed.
    gc_collect_cycles(); // good practice to run GC periodically
}
```

In this loop, `frankenphp_handle_request` will wait for a new incoming request, set up the environment, and invoke `$handler`. The call returns `true` as long as the server should keep running (it returns `false` if the server is shutting down or the worker is asked to stop), so the loop continues until then. Inside the handler, you use `$_SERVER`, `$_GET`, `$_POST`, etc., directly – just like you would in a normal PHP script. This is a key feature: *FrankenPHP injects the incoming HTTP request directly into PHP’s superglobals*, so your existing code that reads `$_GET['param']` or checks `$_SERVER['REQUEST_URI']` will continue to work.

**Sending Responses:** Whether you’re in classic or worker mode, sending output to the client is done as usual: by printing/echoing content or using functions like `printf`, including templates, etc. Buffering and header functions operate normally. For example, to send JSON from a FrankenPHP script, you might do:

```php
header('Content-Type: application/json');
echo json_encode(["status" => "ok", "data" => $responseData]);
```

This will send the appropriate headers and JSON body. In FrankenPHP, you don’t need any special commands to finalize the response – once the script finishes execution (classic mode) or once your handler returns (worker mode), FrankenPHP knows to flush the output to the browser. If you need to flush output mid-processing (for streaming or SSE), you can still call `flush()` as usual. Note that in **HTTP/1.1** you might need to enable full-duplex mode in Caddy to truly stream events (e.g., for WebSockets or server-sent events), but by default FrankenPHP will buffer output until the request is done unless explicitly configured otherwise.

**HTTP Status Codes and Headers:** Use PHP’s `http_response_code()` to set status if you’re not using the default 200 OK, or simply send a header: `header("HTTP/1.1 404 Not Found");` which works as expected. FrankenPHP respects these just like a normal PHP environment. Since it’s integrated in Caddy, you also get automatic features like HTTP/2 and HTTP/3 support out of the box, and HTTPS (TLS) termination by Caddy with automatic Let’s Encrypt certificates ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/#:~:text=,and%20XDebug%2C%20are%20natively%20supported)) – none of which you have to manage in your PHP code, but it’s good to know they’re there.

## Understanding Superglobals in FrankenPHP

PHP superglobals (`$_SERVER`, `$_GET`, `$_POST`, `$_COOKIE`, `$_FILES`, etc.) are the main way you interact with request data and the environment. **In FrankenPHP’s classic mode**, these superglobals work exactly as they do in any PHP environment – they’re populated for each request and cleaned up afterward. For example, `$_GET` contains query string parameters, `$_POST` has form data, `$_FILES` holds uploaded file info, and `$_SERVER` has server and request metadata (like headers, script paths, client IP, etc.). There’s no change to how you use them.

**In worker mode**, superglobals behave a bit differently under the hood, but FrankenPHP makes it smooth for you. When your worker script first starts (before handling any request), `$_SERVER` and friends contain information about the *script environment itself*. Once you enter the loop and call `frankenphp_handle_request()`, FrankenPHP will reset and populate all the superglobals with the incoming request’s data. Each new request *overwrites the superglobals* with the new request’s info. This means within your handler you just use `$_GET`, `$_POST`, etc., as if it were a brand new script execution.

One important nuance: If you need to access the original environment variables of your worker (for example, configuration or constants you set at startup) *inside your request handler*, you should capture them before entering the loop. For instance, you might do:

```php
$startupEnv = $_SERVER;  // copy initial $_SERVER (which includes env vars)
$handler = function() use ($startupEnv) {
    // Inside handler, you can access $startupEnv if needed
    // while $_SERVER now contains request-specific data.
};
```

This technique is recommended because once `frankenphp_handle_request` runs, `$_SERVER` is replaced with request data. The FrankenPHP docs show an example of copying `$_SERVER` to a `$workerServer` variable before the first request, so you can still reference the original worker script’s `$_SERVER` inside your handler if needed.

**$_SERVER Differences:** In FrankenPHP, the `$_SERVER` array is populated in a similar way to PHP-FPM or CLI. Notably, *environment variables* are made available in `$_SERVER` by default. For example, if you set an environment variable in Caddy or your system (like `DATABASE_URL`), it will appear as `$_SERVER['DATABASE_URL']` when your PHP code runs. This is because FrankenPHP ensures the PHP config `variables_order` always includes "E" (Environment) and "S" (Server). If you rely on `$_ENV`, be aware that in worker mode `$_ENV` might be overwritten per request in some cases (populated with Caddy’s env for each request). A safe practice is to load any required environment values at startup and perhaps store them in constants or separate variables as shown above.

**Key elements in `$_SERVER`** that you’ll use:
- `$_SERVER['REQUEST_METHOD']` – GET, POST, etc.
- `$_SERVER['REQUEST_URI']` – the full URI requested (path and query).
- `$_SERVER['QUERY_STRING']` – the query string (if any).
- `$_SERVER['SCRIPT_NAME']` and `$_SERVER['PHP_SELF']` – the script executing.
- `$_SERVER['REMOTE_ADDR']` – client IP address.
- `$_SERVER['HTTP_HEADERNAME']` – for each incoming HTTP header, e.g. `$_SERVER['HTTP_USER_AGENT']` or `$_SERVER['HTTP_AUTHORIZATION']`. (Caddy/FrankenPHP maps incoming headers to these entries just like Apache would.)

These all work as expected. In fact, FrankenPHP “uses the normal `$_SERVER` variables to bootstrap the request” when communicating internally between Caddy and PHP. So if your old PHP code checks `$_SERVER['REQUEST_METHOD']` to decide how to handle a request, you don’t need to change anything.

**$_GET and $_POST:** Query parameters and POST data are available in `$_GET` and `$_POST` respectively, as usual. In worker mode, after each `frankenphp_handle_request`, those arrays are refreshed for the new request, so you’ll always see only the current request’s data.

**$_FILES:** Similarly, file uploads arrive in the `$_FILES` superglobal. Each entry of `$_FILES` will contain the typical associative array with keys like `name`, `type`, `tmp_name`, `error`, `size` for the uploaded file. FrankenPHP handles the file upload process under the hood (via Caddy’s internals) and populates `$_FILES` for you. The temporary file will be stored (by default in `/tmp` inside the container or system) just as in a normal PHP environment. You can move or copy the uploaded file using `move_uploaded_file()` in the same way.

**$_COOKIE and $_SESSION:** Cookies are passed in via `$_COOKIE`. You can set cookies by outputting `Set-Cookie` headers (using `setcookie()` or `header()` calls). Session handling in FrankenPHP should behave the same as standard PHP. If you call `session_start()`, PHP will use a file (or your configured session handler) to manage session data. One thing to note: because FrankenPHP uses threads to handle requests, concurrent requests from the same session might try to access the session file simultaneously. PHP’s default file-based sessions lock the session file, so one request will wait until the other finishes `session_write_close()`. This is the same behavior as with PHP-FPM (multiple processes) – just keep it in mind if you disable session locking or use a custom session handler. No special FrankenPHP changes are needed for sessions beyond that.

In summary, **superglobals in FrankenPHP work as you’re used to**, with the one caveat that in worker mode they get reused across requests (with fresh values each time). This approach means you don’t have to learn a new way to get request data – no PSR-7 `Request` objects or special server APIs. Just use `$_GET`, `$_POST`, `$_SERVER`, etc., and FrankenPHP will supply the correct values for each request ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=request%20directly%20to%20the%20superglobals,7%20compatible)).

## Handling GET Requests and Query Parameters

Handling GET requests in FrankenPHP is straightforward. **Query parameters** (the part of the URL after the `?`) are available in the `$_GET` array. For example, if a user accesses `https://example.com/search.php?term=hello&page=2`, then in your `search.php` script (or within your handler) you can retrieve those values:

```php
$term = $_GET['term'] ?? '';
$page = $_GET['page'] ?? 1;
```

This works exactly as it would in any PHP environment. If you’re in classic mode, each page (like `search.php`) can be a separate script that is invoked when that path is requested. In worker mode, you might have one entry point (say, `index.php`) that parses `$_SERVER['REQUEST_URI']` or `$_GET` parameters to route the request internally.

**Example – Simple Router (No Framework):** Suppose you want a single `index.php` to handle all requests (perhaps you’ve configured Caddy to direct all requests to it, or you’re using PATH_INFO). You could do something like:

```php
// In index.php (front controller)
$request = $_SERVER['REQUEST_URI'] ?? '/';
if ($request === '/' || $request === '/index.php') {
    require __DIR__ . '/pages/home.php';
} elseif ($request === '/search') {
    require __DIR__ . '/pages/search.php';
} else {
    // Handle 404
    header("HTTP/1.1 404 Not Found");
    echo "Page not found";
}
```

In this snippet, `$_SERVER['REQUEST_URI']` gives the path requested. We use it to decide which page logic to execute. This is a simplistic approach (not considering query strings), but you could also use `parse_url` to separate the path and then still use `$_GET` for query parameters. If you have something like `/search?term=abc`, the routing above would treat `/search` as the route, then in `pages/search.php` you’d use `$_GET['term']` to get "abc".

**Reading Request Data in GET:** Use `$_SERVER` values for metadata if needed:
- `$_SERVER['REQUEST_METHOD']` will be `"GET"`.
- `$_SERVER['REQUEST_URI']` includes the `?query` part. For just the path without query, you could use `parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH)`.

One thing to highlight is that **FrankenPHP (via Caddy) automatically handles URL rewriting for you in many cases** if you use `php_server`. It will, by default, try to match actual files. If a file exists (like `/about.php`), it will serve it. If a directory is requested, it might redirect to add a slash or serve an index file. If nothing is found, it falls back to `index.php` (assuming you have an `index.php` in the site root). This behavior mimics common setups where you use an `.htaccess` with `FallbackResource` or front controller. So, if you plan to use a single entry point for a fancy single-page app or API, ensure `index.php` exists and let `php_server` route to it.

**Example – Accessing Query Params:** Let’s say you want to handle an API endpoint `/api.php?user=john`. In `api.php`:

```php
if ($_SERVER['REQUEST_METHOD'] === 'GET') {
    $user = $_GET['user'] ?? null;
    if ($user) {
        // Fetch user data...
        echo "Profile of user: " . htmlspecialchars($user);
    } else {
        echo "No user specified.";
    }
}
```

This works in FrankenPHP just as in any PHP. There are no special steps needed for GET. The main difference is just performance: FrankenPHP might handle it faster behind the scenes, but you don’t have to do anything differently in code.

## Handling POST Requests and Form Submissions

Processing POST requests in FrankenPHP is also done the standard PHP way via `$_POST` and `$_FILES`. If you have an HTML form that submits data (e.g., via POST method), the form fields will appear in the `$_POST` array on the server side.

**Example – Simple Form (HTML):**  
```html
<form action="/submit.php" method="POST">
  <input type="text" name="username">
  <input type="password" name="password">
  <button type="submit">Log In</button>
</form>
```

When the user submits this form, Caddy/FrankenPHP will route the request to `submit.php` (assuming it’s in your public directory). In `submit.php`, you could handle it like:

```php
<?php
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $user = $_POST['username'] ?? '';
    $pass = $_POST['password'] ?? '';
    // Validate credentials (for example purposes, just output them)
    echo "You posted Username = $user and Password = $pass";
} else {
    // If someone accessed this page via GET or other method, you might redirect or show an error
    header("HTTP/1.1 405 Method Not Allowed");
}
```

This is identical to how you’d handle it in a non-FrankenPHP environment. FrankenPHP will have populated `$_POST['username']` and `$_POST['password']` with the form data. 

**File Uploads (via POST forms):** If your form included a file input (with `enctype="multipart/form-data"`), PHP would parse the incoming request and store file info in `$_FILES`. For example:

```html
<form action="/upload.php" method="POST" enctype="multipart/form-data">
  <input type="file" name="profile_pic">
  <button type="submit">Upload</button>
</form>
```

On the server side, `$_FILES['profile_pic']` would be an array with details about the uploaded file. The code might be:

```php
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (isset($_FILES['profile_pic']) && $_FILES['profile_pic']['error'] === UPLOAD_ERR_OK) {
        $tmpPath = $_FILES['profile_pic']['tmp_name'];
        $name = basename($_FILES['profile_pic']['name']);
        move_uploaded_file($tmpPath, __DIR__ . "/uploads/$name");
        echo "File uploaded successfully!";
    } else {
        echo "Upload failed with error code: " . ($_FILES['profile_pic']['error'] ?? 'unknown');
    }
}
```

No FrankenPHP-specific code is needed for this; it uses PHP’s normal file upload mechanism. Behind the scenes, FrankenPHP’s SAPI handles the multipart parsing just like PHP-FPM would. Ensure you have appropriate `upload_max_filesize` and `post_max_size` settings in php.ini if you expect large files. If an upload is too large, PHP will throw a warning and `$_FILES` might have an error code (or `$_POST` may be empty if the entire request body was discarded). In fact, if a POST body exceeds the limit, PHP sets `$_POST` empty and triggers a warning about content length, so always check `$_FILES['yourfile']['error']` for `UPLOAD_ERR_INI_SIZE` in production.

**Handling POST in Worker Mode:** The only difference in worker mode is that your code handling POST will be inside a handler closure. But you still use `$_POST`. For example, inside the `$handler`:

```php
$handler = function () {
    if ($_SERVER['REQUEST_METHOD'] === 'POST') {
        // ... process $_POST
    }
};
```

Each time a POST request comes in, that code runs with fresh `$_POST` data.

**Reading Raw POST Data:** If you’re dealing with JSON payloads or other non-form submissions (like an API client sending JSON or XML in the request body), remember that such data won’t appear in `$_POST`. You’ll need to read the raw input from PHP’s input stream. You can do this in FrankenPHP exactly like in normal PHP:

```php
$data = file_get_contents('php://input');
$json = json_decode($data, true);
```

This is useful for AJAX or API requests (discussed more below). Also, `$_SERVER['CONTENT_TYPE']` will tell you the MIME type of the request (e.g., `application/json` or `application/x-www-form-urlencoded` for typical form posts).

**Redirects After POST:** It’s common to redirect after processing a form (Post/Redirect/Get pattern). You can use PHP’s `header("Location: /thank-you.php")` followed by an `exit;`. FrankenPHP will send that redirect status (defaults to 302 Found) and location header to the client.

## Handling File Uploads in FrankenPHP

File uploads are a special case of POST requests, but let’s highlight them. When a user uploads a file, PHP handles it by writing the file to a temporary directory and populating the `$_FILES` superglobal. FrankenPHP does the same. By default, in the official FrankenPHP Docker image, the php.ini settings for `upload_tmp_dir` might point to `/tmp` (or inherit the system default). Ensure that the location is writable.

**Processing an Upload:**

1. **HTML Form:** Must have `enctype="multipart/form-data"`. Each `<input type="file" name="...">` results in an entry in `$_FILES` on the server.
2. **Checking for Errors:** Always check `$_FILES['yourfield']['error']`. Common values:
   - `UPLOAD_ERR_OK` (0) means successful.
   - `UPLOAD_ERR_NO_FILE` means no file was uploaded.
   - `UPLOAD_ERR_INI_SIZE` or `UPLOAD_ERR_FORM_SIZE` means the file was too large (exceeded server or form limit).
3. **Moving the File:** Use `move_uploaded_file($_FILES['field']['tmp_name'], $destination)` to save it. Do this promptly if you need to keep the file, because the temp file might be cleaned up after the request.

**Example – Upload Handler (complete):**  
```php
<?php
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (!empty($_FILES['upload'])) {
        $f = $_FILES['upload'];
        if ($f['error'] === UPLOAD_ERR_OK) {
            $dest = __DIR__ . "/uploads/" . basename($f['name']);
            if (move_uploaded_file($f['tmp_name'], $dest)) {
                echo "Uploaded " . htmlspecialchars($f['name']) . " successfully.";
            } else {
                echo "Failed to move uploaded file.";
            }
        } else {
            echo "File upload error code: " . (int)$f['error'];
        }
    } else {
        echo "No file uploaded.";
    }
}
```

This code doesn’t change for FrankenPHP. You might wonder if the persistent worker mode affects uploads. In worker mode, PHP still handles each upload per request and cleans up. The `$_FILES` array will be fresh for each upload request just like `$_POST`. The only thing to be careful about is memory: extremely large file uploads that exceed limits might produce warnings (as noted above) and FrankenPHP’s worker thread could encounter an error if a client disconnects mid-upload. But FrankenPHP’s default behavior should be to handle it as PHP normally would. If a worker script crashes (say, due to a fatal error triggered by an upload), FrankenPHP will restart that worker automatically. This is generally transparent, but you’d obviously avoid letting an error happen by checking sizes and using proper limits.

**Multiple Files:** If you have multiple file inputs with the same name (array syntax like `name="photos[]"`), PHP will structure `$_FILES['photos']` with subarrays (`name`, `type`, etc. each being an array of values). That’s standard and unchanged.

**Temporary Files and Cleanup:** PHP (and thus FrankenPHP) will delete the temp file (`tmp_name`) when the request is done *if* you haven’t moved it. So always use `move_uploaded_file` during the request. In worker mode, after `frankenphp_handle_request` completes, it cleans up the request context including temp files, before waiting for the next request. So you don’t need to manually delete them on success (though you might if something fails and you want to ensure no leftover – but usually PHP cleans them).

**Upload Limits:** Configure `upload_max_filesize` and `post_max_size` in php.ini (or via `frankenphp php_ini` directive in Caddyfile if you want to set these in FrankenPHP config). Also consider `max_execution_time` and `max_input_time` for very large uploads.

## Handling AJAX and JSON Requests

**AJAX requests** (like those made with `fetch()` in JavaScript or jQuery’s `$.ajax`) are simply HTTP requests, often carrying JSON data or expecting JSON responses. FrankenPHP doesn’t treat these specially – they are handled like any other request.

If an AJAX call sends form data (e.g., using `FormData` or jQuery with traditional form encoding), it will show up in `$_POST` and `$_FILES` if applicable. If it sends JSON (common in single-page applications or modern front-ends), you will need to read the raw body:

- Check `$_SERVER['CONTENT_TYPE']` – if it’s `application/json`, you know the body is JSON.
- Use `file_get_contents('php://input')` to get the raw JSON string.
- Use `json_decode($jsonString, true)` to decode it into a PHP array.

**Example – Handling JSON POST (API endpoint):**  
```php
// In an API script or route
if ($_SERVER['CONTENT_TYPE'] === 'application/json' && $_SERVER['REQUEST_METHOD'] === 'POST') {
    $json = file_get_contents('php://input');
    $data = json_decode($json, true);
    if (!is_array($data)) {
        http_response_code(400);
        echo json_encode(["error" => "Invalid JSON"]);
        exit;
    }
    // Now $data might be ["name" => "Alice", "age" => 30] for example
    // Process the data...
    $name = $data['name'] ?? 'guest';
    $response = ["message" => "Hello, $name!"];
    header("Content-Type: application/json");
    echo json_encode($response);
}
```

From FrankenPHP’s perspective, this is no different than normal PHP. The only difference is performance: FrankenPHP’s tight integration and optional worker mode can handle many concurrent AJAX requests efficiently using its thread pool (and HTTP/2 means multiple AJAX calls can reuse one connection).

**CORS (Cross-Origin Resource Sharing):** If your FrankenPHP app serves an API that might be called via AJAX from a different origin, remember to output appropriate CORS headers (e.g., `header("Access-Control-Allow-Origin: *")` or more restrictive as needed). This is not unique to FrankenPHP, but since Caddy is the webserver, you could also configure CORS at the Caddy level. However, handling it in PHP is fine for simplicity: just ensure OPTIONS requests (preflight) are responded to with proper headers too.

**AJAX in Worker Mode:** When using worker mode, handling many quick AJAX calls is actually a strong benefit of FrankenPHP. Because your app stays in memory, repeated XHR or fetch calls can reuse the same initialized resources (like database connections or loaded config). Just be mindful of thread concurrency: FrankenPHP by default starts multiple worker threads (2 per CPU by default), so it can handle concurrent requests. If you share any resources in memory, ensure they are safe for concurrent access (e.g., avoid global variables that are modified unsafely). But if each request uses local variables and your app is mostly stateless between requests (aside from something like a DB connection per request), you’ll be fine.

**Sending Data Back to Client:** For AJAX/JSON, you typically send a JSON response. We showed an example above. Always set the `Content-Type: application/json` header when returning JSON. Use `json_encode` for the data. For error statuses, you can set an HTTP code (like 400 or 500) and perhaps include an error message in JSON.

**File Uploads via AJAX:** If you upload files via AJAX (using `XMLHttpRequest` or fetch with FormData), it’s the same as a form post on the server side. `$_FILES` will be populated. The difference might be that the client expects a JSON or text response instead of a new HTML page. So just ensure you respond accordingly (possibly with JSON confirming success).

## Organizing Your Code and Pages

When not using a framework, you have flexibility in how to organize a FrankenPHP project. Here are some approaches and considerations:

- **Classic Multiple Scripts (Simple Sites):** You can continue to use separate PHP files for separate pages (e.g., `index.php`, `about.php`, `contact.php`). FrankenPHP will serve these directly in classic mode. You might share common code by including a header or configuration file at the top of each script. This is the traditional approach and works out of the box.

- **Front Controller (Single Entry):** Alternatively, you can route all requests through a single `index.php` and dispatch internally (like the router example shown earlier). This is essentially what frameworks do, but you can implement a basic version yourself. It’s useful for building an API or any app where you don’t want to expose multiple PHP files publicly. If you do this, you may need to adjust the Caddy configuration: the default behavior tries to find files. If you want *everything* to go to `index.php`, you can set up a custom Caddyfile rule to rewrite requests. However, using the default `php_server` with its `try_files {path} {path}/index.php index.php` rule already covers most cases – it falls back to `index.php`. So as long as you have an `index.php` in the document root, any request for a non-existent file will end up hitting `index.php` with the requested path in `$_SERVER['REQUEST_URI']`. Your script can then parse that and act accordingly.

- **Directory Structure:** A typical layout for a FrankenPHP (or any PHP) project might be:

```
/app
├─ /public        <-- Document root (this is what Caddy serves)
│   ├─ index.php
│   ├─ about.php
│   ├─ submit.php
│   └─ ... other public-facing PHP files ...
├─ /src           <-- PHP source code (classes, functions)
├─ /vendor        <-- Composer dependencies if any
└─ /templates     <-- HTML templates or partials
```

FrankenPHP by default uses `/app/public` as the web root in the Docker image. You can adjust that, but it’s a good practice to keep your entry scripts in a `public` folder and the rest of your code outside of it (so that they can’t be accessed directly over the web, for security). In classic mode, if someone requests a file under `public`, it’s either served or executed; anything outside isn’t directly reachable unless included by your code.

- **Autoloading:** Even without a framework, you can use Composer to autoload classes. This is recommended if your project grows beyond a few files. FrankenPHP supports all normal PHP features, so `require` and `include` work, but autoloading via Composer’s `vendor/autoload.php` is convenient. Just ensure you include `require __DIR__ . '/../vendor/autoload.php';` at the top of your `index.php` or worker script to load classes automatically.

- **Reusing Code:** If you have common logic (like connecting to a database, or header/footer HTML for pages), structure your code to reuse it. For example, you might have a `bootstrap.php` that initializes config, and include that at the top of every script (or use worker mode to initialize once). Similarly, have a `header.php` and `footer.php` that output common HTML, and include those in each page script.

- **Worker Mode Organization:** If using worker mode for a plain PHP app (no frameworks), you’ll likely still have one main `index.php` (the worker script). Inside the loop, you can implement routing as shown, or even instantiate a small router object. Essentially, you are writing your own micro-framework in that handler. One approach is:
  - Preload any resources (DB connection, config) before the loop so they persist.
  - In the handler, use something like `switch($_SERVER['REQUEST_URI'])` or a custom function to route to an appropriate function that handles the request.
  - That function can generate the response (perhaps by including a template or just echoing).

- **Example – Minimal Router in Worker Mode:**  
  ```php
  // Before loop:
  $db = new PDO(...); // database connection
  // ... other setup ...
  
  $handler = function() use ($db) {
      $uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
      if ($uri === '/' || $uri === '/index.php') {
          echo "<h1>Welcome to my site</h1>";
      } elseif ($uri === '/data') {
          header("Content-Type: application/json");
          // Fetch something from $db
          $result = $db->query("SELECT ...")->fetchAll(PDO::FETCH_ASSOC);
          echo json_encode($result);
      } else {
          header("HTTP/1.0 404 Not Found");
          echo "404 Not Found";
      }
  };
  while (frankenphp_handle_request($handler)) {
      // After each request, you could reset something if needed
  }
  ```
  This illustrates a structure where persistent resources ($db) are kept and reused. Notice we use `$db` inside the handler via a `use` in the closure. In traditional PHP, you might open a DB connection on each page load; here you can keep it around if you want (just ensure the PDO or driver is okay with reuse across requests – most are, but if not, you can reconnect per request too).

- **Static Assets and Other File Types:** FrankenPHP (Caddy) will serve static files (CSS, JS, images) from the `public` directory automatically via Caddy’s `file_server`. If you use `php_server` directive, by default it enables a `file_server` for non-PHP files. So you can organize your static files in `public` as well, and they’ll be served efficiently by Caddy (bypassing PHP entirely).

## Limitations and Special Considerations

While FrankenPHP is largely compatible with traditional PHP code, there are some important considerations:

- **Thread-Safety:** FrankenPHP uses threads for concurrency (especially in worker mode, where multiple requests may be handled by separate threads in the same process). Most PHP code is fine with this, but certain PHP extensions are *not thread-safe*. For example, the PHP `imap` extension is known to be incompatible with FrankenPHP because it isn't thread-safe. Another example is the New Relic extension. If your application uses an extension that has global state or isn't certified for ZTS (Zend Thread Safe) environments, it may cause issues. Always check the FrankenPHP documentation’s *Known Issues* section for any extensions that might be problematic. Fortunately, the vast majority of built-in extensions (MySQLi/PDO, cURL, GD, etc.) are fine in a threaded environment.

- **Long-Running Pitfalls:** In worker mode, because your PHP script doesn’t terminate after each request, you need to be mindful of memory leaks and state persistence:
  - **Memory Leaks:** Any memory allocated will accumulate if not freed. PHP’s garbage collector handles most of it, but some static caches or truly leaked memory in extensions can bloat over time. FrankenPHP provides a way to mitigate this: you can set an environment variable (e.g., `MAX_REQUESTS`) to automatically restart the worker after a certain number of requests. In the example earlier, the worker loop reads `$_SERVER['MAX_REQUESTS']` and breaks out after that many iterations, causing the script to end and FrankenPHP to respawn it fresh. This is similar to PHP-FPM’s `pm.max_requests`.
  - **Persistent State:** Avoid using global or static variables to store request-specific information without resetting them. For example, if you have a static array that you append to on each request, it will keep growing unless you clear it. Each request should ideally start from a clean state. Since superglobals are reset by FrankenPHP, they are fine. But any truly global variables you create in the worker script will retain their values between requests. You can manually reset or reinitialize such variables at the start or end of your handler if needed.
  - **Connections:** If you keep a DB connection open in worker mode, handle errors that can occur if the connection times out between requests. You might need to reconnect if a connection has gone stale. This is similar to persistent connections in any long-running PHP process.

- **Signal Handling:** If you run FrankenPHP as a standalone binary or via Docker, it intercepts signals to manage graceful shutdown of workers. Your PHP code typically wouldn’t deal with this, but just be aware that `ctrl+c` on a dev server or container stop will signal the process to stop accepting new requests and shut down workers.

- **$_ENV vs $_SERVER:** As noted, environment variables are placed in `$_SERVER`. If you rely on `getenv()` or `$_ENV`, be cautious. In some reported cases, `$_ENV` might not have what you expect inside the request handler because FrankenPHP might override it per request with certain values (like Caddy’s internal env variables). The workaround is to merge your environment into `$_ENV` at startup (as shown in that GitHub issue snippet) or simply use `$_SERVER` which will have them. For example, use `$_SERVER['MY_CUSTOM_ENV']` instead of `$_ENV['MY_CUSTOM_ENV']` inside your code if you set `MY_CUSTOM_ENV` in the Docker or system environment.

- **Built-in Functions:** All core PHP functions work. The presence of Caddy doesn’t remove anything like `mail()` (though if using `mail()`, ensure the container has an SMTP server or use a library). Functions like `sys_get_temp_dir()` will return the temp directory (likely `/tmp`). `phpinfo()` will show FrankenPHP as the Server API (SAPI) in use, which can be useful to confirm your setup.

- **Configuration:** FrankenPHP uses php.ini as usual. If using the Docker image, you can add custom `.ini` files by mounting them and using `PHP_INI_SCAN_DIR` environment variable. You can also set specific settings via the `frankenphp { php_ini ... }` Caddyfile directive. This is helpful if you need to tweak `memory_limit`, `max_execution_time`, etc., without rebuilding the Docker image.

- **Real-time and SSE:** FrankenPHP has advanced features like a built-in Mercure hub for real-time communications (Server-Sent Events). If you plan to use those, it’s beyond the scope of this basic guide, but know that you can push updates to connected clients using Mercure. This might require using specific headers and endpoints that Mercure (within Caddy) provides. For building typical AJAX/REST or standard websites, you don’t need to use these unless you want live push.

- **Debugging:** You can use Xdebug with FrankenPHP (the Docker image includes it, you just have to enable it via environment variable). Because FrankenPHP is always running, Xdebug will work a lot like with FPM – you set up an IDE key and trigger a connection. Just ensure the container or process has Xdebug enabled and configured to connect back to your IDE host.

- **Graceful Reloads:** If you change PHP code while the FrankenPHP worker is running, the changes won’t take effect until the worker is restarted (since the code is loaded in memory). In dev mode, FrankenPHP supports watching files and auto-restarting the worker on changes. You can enable this by running the server with `--watch` or in the Caddy config. In classic mode, code changes are picked up on next request (because each request starts fresh). But in worker mode, use the watch feature or manually restart the server when you make changes to see them reflected.

## Best Practices for Building APIs and Dynamic Pages with FrankenPHP

Finally, let’s summarize some best practices to follow when developing without a framework on FrankenPHP:

- **Choose the Right Mode:** If you're building a small site or prototype, classic mode is simplest – it behaves just like PHP on Apache. If you're building an API or larger app where performance matters, consider worker mode for persistent app state. Remember that *any* PHP app (even WordPress or a custom one) can run in classic mode on FrankenPHP ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=The%20default%20mode%20with%20FrankenPHP,PHP%20in%20a%20sense%2C%20where)), so you have flexibility. You can even start in classic mode and optimize later by switching to a worker script once your code is stable and you want the extra speed.

- **Structure for Clarity:** Use clear separation of concerns. Even without a framework, you can separate your HTML presentation (in include files or a simple templating system) from your logic (in your PHP scripts). This makes it easier to maintain. For APIs, keep your output separate (as data structures that you json_encode).

- **Reuse Initialization in Worker Mode:** If in worker mode, load things once. For example, if you have configuration files or large data sets, load them before the request loop and store them in a variable that your handler can access. The same goes for including function or class definitions – include them once at the top (they will stay loaded for subsequent requests). This is a major advantage of worker mode. Just avoid carrying over anything request-specific across iterations.

- **Avoid Global State Issues:** Each request should be independent. Do not assume a variable has a default value just because you set it in a previous request. Always initialize variables at the start of handling a request. If using a static or global variable intentionally (say a cache), consider resetting it when appropriate or after X requests.

- **Leverage Built-in Server Features:** FrankenPHP (Caddy) gives you HTTP/2, HTTP/3, TLS, and more automatically. You can also configure things like URL rewrites, security headers, and Gzip/Brotli compression at the server level (in Caddy) if you want. Often, though, just enabling `encode gzip zstd` in the Caddy config will compress your responses without any PHP code change ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/#:~:text=localhost%20%7B%20,and%20serve%20assets%20php_server)) ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/#:~:text=encode%20zstd%20br%20gzip%20,and%20serve%20assets%20php_server)). Likewise, to set up a basic authentication or other rules, you could use Caddyfile directives. This can offload work from PHP. However, doing it in PHP (checking a login session and showing/hiding content) is perfectly fine too.

- **Test in Both Modes:** If you develop in classic mode, test your code in worker mode if you plan to switch, and vice versa. Sometimes, code that works in classic (process-per-request) might have subtle issues in worker (like not resetting a static). Catch and fix those early. FrankenPHP’s compatibility is high, but the execution model differences can reveal logic issues (not necessarily FrankenPHP issues).

- **Use Composer Packages:** Even without a full framework, you can pull in libraries for common tasks – e.g., routing (FastRoute), templates (Twig or Plates), HTTP clients, etc. They will work on FrankenPHP. This can save you time and avoid reinventing wheels. For example, if building an API, you might use a library to handle routing paths to callback functions; or use an ORM like Doctrine or an HTTP utility library. FrankenPHP doesn’t restrict this; it runs normal PHP code.

- **Profiling and Optimization:** If performance is a goal, profile your application. FrankenPHP is fast, but your PHP code still needs to be written efficiently. Use tools or simple timing calls to identify slow spots. Because worker mode keeps things in memory, you might cache certain data in variables between requests – but be cautious to invalidation. Also, watch memory usage if caching a lot.

- **Error Handling:** Error handling is crucial in a long-running worker. A fatal error in classic mode only affects that one request; in worker mode, it could bring down the worker. FrankenPHP will attempt to restart a worker that crashes, but it’s best to handle exceptions and errors gracefully. Use try/catch in critical sections, and consider using error_reporting to your advantage. In production, disable display_errors and log them instead, so a user request doesn’t get a broken HTML with an error – you can catch it and maybe return a 500 JSON or friendly page.

- **Building APIs:** When building APIs (REST or otherwise):
  - Use clear URL structures (e.g., `/api/users` for a list, `/api/users/123` for details).
  - Handle different methods (GET, POST, PUT, DELETE) via `$_SERVER['REQUEST_METHOD']`.
  - Validate input thoroughly (since you might not have a framework doing it).
  - Respond with proper status codes (200 for success, 201 for creation, 400/422 for bad input, 500 for server errors, etc.).
  - Set `Content-Type` appropriately (JSON or XML).
  - Consider implementing authentication if needed (even a simple API token check via a header).

- **SEO and Dynamic Pages:** For dynamic web pages (not APIs), you might generate HTML. FrankenPHP doesn’t change how you do this. You can echo HTML or include PHP templates. Just ensure to output proper headers (like content-type text/html; charset=UTF-8) if you do manual headers, or simply rely on the default (PHP defaults to `Content-Type: text/html; charset=UTF-8`). Also consider enabling output buffering (`ob_start()`) if you want to capture output and modify or compress it before sending (though Caddy can compress output for you at the HTTP layer).

By following these practices, you can build robust, full-featured PHP applications on FrankenPHP without a framework. You’ll benefit from FrankenPHP’s speed and modern features while still writing plain PHP – using the same superglobals and techniques you’re already comfortable with. FrankenPHP essentially gives you the best of both worlds: traditional PHP simplicity and next-generation performance and integration.

## Conclusion

FrankenPHP might sound new and a bit "magical," but as we’ve seen, it doesn’t require you to learn a new way of coding. Most of the changes are under the hood – using Go and Caddy to run PHP more efficiently – while your day-to-day PHP coding stays familiar. 

To recap:
- **Requests/Responses:** Handled similarly to classic PHP, with an option for a persistent worker to speed things up. Use `frankenphp_handle_request()` in worker mode to loop through requests.
- **Superglobals:** Still your main interface – `$_GET`, `$_POST`, `$_SERVER` – with FrankenPHP populating them per request (even in worker mode) ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=request%20directly%20to%20the%20superglobals,7%20compatible)). Just remember the quirk about initial `$_SERVER` in worker scripts.
- **GET/POST/Files/AJAX:** All work as usual. Parse input from superglobals or streams, output responses with echo or header functions. File uploads go to `$_FILES` and need `move_uploaded_file` as always.
- **Code Organization:** You can organize code with multiple files or a single entry point. FrankenPHP is flexible; tailor it to your app’s needs. Using a `public` directory as document root is recommended.
- **Special Considerations:** Watch out for thread-unsafe extensions, manage long-running state in worker mode, and configure your environment via `php.ini` or Caddyfile as needed.
- **Best Practices:** Write clean, independent request handling code. Leverage FrankenPHP’s strengths (persistent resources, built-in server features) and mitigate its challenges (shared environment, long-running process issues) as described.

With this understanding, you should be able to confidently build applications on FrankenPHP. You get modern performance and deployment ease (thanks to Caddy’s one-command setup and features) without giving up the simplicity of raw PHP. Happy coding with FrankenPHP! 

**Sources:** The information and recommendations above are drawn from the official FrankenPHP documentation and community discussions for accuracy ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=request%20directly%20to%20the%20superglobals,7%20compatible)). FrankenPHP’s creator emphasizes its compatibility with “plain old superglobals” to ensure any PHP app can run on it ([FrankenPHP with Laravel can do a magical thing : r/laravel](https://www.reddit.com/r/laravel/comments/18m2hax/frankenphp_with_laravel_can_do_a_magical_thing/#:~:text=request%20directly%20to%20the%20superglobals,7%20compatible)), and real-world usage has shown it to be an effective drop-in replacement for traditional setups with significant performance benefits ([FrankenPHP: the modern PHP app server](https://frankenphp.dev/#:~:text=)).