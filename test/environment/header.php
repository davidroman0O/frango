<?php
// Header file for include test

// Define a variable that will be accessible in the main file
$headerVar = "Variable from header.php";

// Output the header HTML
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PHP Include Test</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        header { background: #f0f0f0; padding: 10px; margin-bottom: 20px; }
        footer { background: #f0f0f0; padding: 10px; margin-top: 20px; text-align: center; }
        .result { background: #e9f7e9; border: 1px solid #ccc; padding: 15px; margin: 15px 0; }
    </style>
</head>
<body>
    <header>
        <h1>PHP Include/Require Test</h1>
        <p>Testing the ability to include and require other PHP files.</p>
    </header> 