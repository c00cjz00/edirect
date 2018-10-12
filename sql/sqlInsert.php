<?php
$DB="slideFile";
$owner="summerhill001@gmail.com"; $permission=0;
require_once("DB_config.php");
require_once("DB_class.php");
$fArr=array();
$db = new DB();
$db->connect_db($_DB['host'], $_DB['username'], $_DB['password'], $_DB['dbname']);
$sql = "SELECT filename FROM `$DB`";
$db->query($sql);
while($row = $db->fetch_array()){
 $f=trim($row['filename']); array_push($fArr,$f); $$f=1;
}
//$cmd="find * |grep dzi\$";
$cmd="find *  -not \( -path *_files/ -prune \) -name \*.dzi";

$result=shell_exec($cmd);

$fp = fopen("dzi.list", "w");
fwrite($fp, trim($result));
fclose($fp);


$rArr=explode("\n",trim($result));
for($i=0;$i<count($rArr);$i++){
 $f="/slide/slide_Result/".trim($rArr[$i]);
 if (substr($f,-3,3)=="dzi"){
  if (!isset($$f)){
   $sql="INSERT INTO $DB (filename,owner,permission) VALUES ('$f','$owner','$permission')";
   echo $sql."\n"; $db->query($sql);   
  }
 }
}
?>
