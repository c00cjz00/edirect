<?php
require_once("DB_config.php");
require_once("DB_class.php");
$_DB['dbname']="atri001_試驗資料庫總資料表";
$db = new DB(); $db->connect_db($_DB['host'], $_DB['username'], $_DB['password'], $_DB['dbname']);
$sql="show tables";
$result=$db->query($sql);
while ($row = mysql_fetch_row($result)) {
$table1Arr[]=$row[0];
} 
$_DB['dbname']="atri_bio20171009013840";
$db = new DB(); $db->connect_db($_DB['host'], $_DB['username'], $_DB['password'], $_DB['dbname']);
$sql="show tables";
$result=$db->query($sql);
while ($row = mysql_fetch_row($result)) {
$table2Arr[]=$row[0];
} 
for($i=0;$i<count($table1Arr);$i++){
 $table=$table1Arr[$i]; $table=addslashes($table);
 
 echo $table."\n";
 $sql = "SELECT * FROM `".$table."`"; $result=$db->query($sql);
 while ($property = mysql_fetch_field($result)){
 
 $column=$property->name; 
 echo $column;
 }
 echo "\n\n";
}


?>
