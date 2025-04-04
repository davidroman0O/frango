<?php
require_once '/var/folders/h2/lww7d7p5049dx4qzhxgk33640000gn/T/frango-fcfbf14f/vfs-131770ab/_frango_path_globals.php';

		echo "This is a test PHP file";
		
		// Display any path parameters that might be set
		if (isset($_PATH) && count($_PATH) > 0) {
			echo "\nPath parameters:\n";
			foreach ($_PATH as $key => $value) {
				echo "$key: $value\n";
			}
		}
	?>