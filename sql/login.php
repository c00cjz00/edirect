<html><head><title>Login</title>
</head>
<body>

<form action="actionlogin.php" method="POST">
Email:<br />
<input type="text" name="email">
<br />
Password:<br />
<input type="password" name="password">
<br />
<input type="submit" name="submit" value="Login">
<input type="hidden" name="refer" value="<?php echo (isset($_GET['refer'])) ? $_GET['refer'] : 'index.php'; ?>">
</form>
</body>
</html>
