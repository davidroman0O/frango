<?php
require_once __DIR__ . "/../lib/render_util.php";
$title = $_SERVER["FRANGO_VAR_title"] ?? "Default Title";
echo "<h1>" . json_decode($title) . "</h1>";
echo "<p>" . get_extra_message() . "</p>";
?>
