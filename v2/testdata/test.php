<?php
		echo "This is a test PHP file";
		
		// Display any path parameters that might be set
		if (isset($_PATH) && count($_PATH) > 0) {
			echo "\nPath parameters:\n";
			foreach ($_PATH as $key => $value) {
				echo "$key: $value\n";
			}
		}
	?>