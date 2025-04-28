<?php
// Binary response test (using a simple hard-coded PNG)
header('Content-Type: image/png');
header('Content-Disposition: inline; filename="frango_test.png"');

// This is a minimal valid PNG file (1x1 transparent pixel)
$png_data = base64_decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+P+/HgAFDQJCmBFW2AAAAABJRU5ErkJggg==');
echo $png_data;