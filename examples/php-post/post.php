
<?php 
    if (empty($_POST)) {
        echo "No POST data received.";
    }
?>


<?php foreach ($_POST as $key => $value): ?>
    <p><?php echo $key; ?>: <?php echo $value; ?></p>
<?php endforeach; ?>

