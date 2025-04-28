
Expectation of the API


- `$_GET` must contain query parameters (`?foo=bar`)
	```php
	echo $_GET['foo']; // bar
	```
	- if query is `?id[]=1&id[]=2`, `$_GET['id']` must be an array
		```php
		// $_GET['id'] is ['1', '2']
		foreach ($_GET['id'] as $val) {
			echo $val;
		}
		```

- `$_POST` must contain form-encoded fields
	```php
	echo $_POST['email'];
	```

- `$_FILES` must support uploads with correct structure
	```php
	echo $_FILES['file']['name'];
	move_uploaded_file($_FILES['file']['tmp_name'], '/tmp/' . $_FILES['file']['name']);
	```

- `$_COOKIE` must be populated from request headers
	```php
	echo $_COOKIE['session_id'];
	```

- `$_REQUEST` should merge `$_GET`, `$_POST`, and `$_COOKIE`
	```php
	echo $_REQUEST['email'];
	```

- `$_SERVER` should include:
	- `'REQUEST_METHOD'`
		```php
		if ($_SERVER['REQUEST_METHOD'] === 'DELETE') { ... }
		```
	- `'REQUEST_URI'`
		```php
		echo $_SERVER['REQUEST_URI']; // /nested/12?foo=bar
		```
	- `'QUERY_STRING'`
		```php
		echo $_SERVER['QUERY_STRING']; // foo=bar
		```
	- `'SCRIPT_NAME'` and `'SCRIPT_FILENAME'`
		```php
		echo $_SERVER['SCRIPT_NAME'];      // /index.php
		echo $_SERVER['SCRIPT_FILENAME'];  // /var/www/html/index.php
		```
	- `'PATH_INFO'` if applicable
		```php
		echo $_SERVER['PATH_INFO']; // /12
		```
	- `'HTTP_*'` for all headers
		```php
		echo $_SERVER['HTTP_AUTHORIZATION'];
		```

- `php://input` should contain raw body for PUT/PATCH/DELETE
	```php
	$body = file_get_contents("php://input");
	$data = json_decode($body, true);
	echo $data['foo'];
	```

- developers will use `file_get_contents("php://input")` and parse manually

- route parameters like `/nested/{id}` should be injected into:
	- `$_GET['id']`
		```php
		echo $_GET['id'];
		```
	- or `$routeParams = ['id' => '123']`
		```php
		global $routeParams;
		echo $routeParams['id'];
		```
	- or `$_FRANGO['params']`
		```php
		echo $_FRANGO['params']['id'];
		```

- path segments can optionally be injected as:
	- `$_FRANGO['path_segments'] = ['nested', '123']`
		```php
		echo $_FRANGO['path_segments'][1]; // 123
		```

- file uploads should populate `$_FILES['upload']` with:
	- `'name'`, `'tmp_name'`, `'type'`, `'size'`, `'error'`

- `$_SERVER['HTTP_AUTHORIZATION']` and similar must be set

- `$_SESSION` must work if sessions are started in PHP
	```php
	session_start();
	$_SESSION['user'] = 'david';
	```

- `__FILE__`, `__DIR__`, `PHP_SELF` should resolve as expected
	```php
	echo __FILE__;
	echo __DIR__;
	echo $_SERVER['PHP_SELF'];
	```

- `$_ENV` and `getenv()` must work
	```php
	echo $_ENV['APP_ENV'];
	echo getenv('APP_ENV');
	```

- `$_PUT`, `$_PATCH`, `$_DELETE`, `$_FORM`, `$_PATH`, `$_PATH_SEGMENTS` do not exist in native PHP and should not be expected

- optional helpers like `$frango['json_body'] = frango_json()` can improve DX
	```php
	$body = $frango['json_body'];
	echo $body['foo'];
	```

- avoid breaking expectations of native superglobals or behavior

- relative paths should resolve from the executing script’s directory
	```php
	include 'lib/utils.php';
	```

- absolute paths should also work as expected
	```php
	include __DIR__ . '/lib/utils.php';
	```

- `include_path` in `php.ini` can affect lookup behavior, but most devs just use `__DIR__` to be explicit

- autoloaders should work, especially for Composer
	```php
	require __DIR__ . '/vendor/autoload.php';
	use App\Service\Thing;
	```

- Composer must work if `vendor` and `autoload.php` are available — if your Go layer isolates execution too much, devs will expect to be able to:
	```bash
	composer install
	```
	and have:
	```php
	require_once __DIR__ . '/vendor/autoload.php';
	```

- if you're using sandboxed environments or changing the working directory in Go before launching the PHP process, you **must preserve the expected `__DIR__`, `getcwd()`, and `include` behavior**

If you're injecting code, routing to other files manually, or mounting paths with Go logic, make sure:
- `$_SERVER['SCRIPT_FILENAME']` points to the actual script file
- `chdir()` or base dir isn't messing with `include` resolution
- devs can structure code like they would in Laravel or legacy apps with `lib/`, `helpers/`, or `src/`

Concerning VFS

- That files are present on disk, or at least accessible by path

- That __DIR__, __FILE__, realpath() behave as expected

- That relative includes work (include 'lib.php')

- That Composer autoloaders and spl_autoload_register work (they rely on file paths)



Perfect, since you're mounting everything into a **temporary real filesystem**, that’s exactly what PHP expects — so you’re already 90% of the way there.

Now let’s go deep on those three expectations:

---

### `__DIR__`, `__FILE__`, `realpath()` — expected behavior in PHP

These functions and constants rely **entirely on the path used to execute the file** and the underlying OS FS.

- `__FILE__` gives the full absolute path of the file being parsed
	```php
	echo __FILE__; // /tmp/frango/prod/routes/index.php
	```

- `__DIR__` is derived from `__FILE__` at compile-time
	```php
	echo __DIR__; // /tmp/frango/prod/routes
	```

- `realpath()` resolves symbolic links, `..`, etc.
	```php
	echo realpath('./lib/init.php'); // /tmp/frango/prod/routes/lib/init.php
	```

- When executing a PHP file (`index.php`, etc.), pass the **real, resolved path** as `SCRIPT_FILENAME`
- Ensure `chdir()` is not messing with `getcwd()` unexpectedly, unless you intend to sandbox
- VFS links or overlays must resolve into **real files on disk**, not Go-backed stubs
- If using symlinks inside the temp folder, ensure Go writes them correctly (`os.Symlink()`) and the temp FS respects them (Linux/macOS: yes, Windows: depends)

- The **temp dir** must contain the **entire vendor structure**
- `include_path` in PHP config must not interfere
- If your VFS has read-only snapshots, `vendor/` should be in a shared, read-only branch
- Composer classmap and PSR-4/PSR-0 resolution logic **must resolve to real files**, so symbolic or hard links must resolve correctly

- ✅ `include`, `require` will work out of the box
- ✅ relative paths are resolved based on current file path
- ✅ autoloading logic won’t break
- ✅ `realpath()`, `file_exists()`, `is_readable()`, etc., will behave properly

To discuss:

- always pass `SCRIPT_FILENAME` as the **real resolved file path** (not a VFS or internal reference)
- always pass `SCRIPT_NAME` and `PHP_SELF` relative to the web root (like `/routes/index.php`)
- always `chdir()` to the script dir if simulating isolation
- allow mounting shared `vendor/` folder into each VFS branch or inject only once for all
- if you sandbox with a `chroot`-like pattern, make sure PHP still sees real paths to `vendor/autoload.php`



