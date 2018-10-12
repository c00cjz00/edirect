<?php
    session_start();
    $guid = $_COOKIE['session_id'];
    setcookie("session_id", "",0);
    session_destroy();
   header("location:test.php");
?>
