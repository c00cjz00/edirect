<?php
require('DB_config.php');

$id = $_GET['id'];
$target = $_GET['target'];
$db = $_GET['db'];
//$id=1;$target="PB2";
if (($id !="") && ($db!="")) {
 $con = mysql_connect($db_host, $db_user, $db_pass);
 mysql_select_db($db_name, $con);
 $query = "SELECT strainName,".$target." FROM ".$db." WHERE id = '$id'";
 $result = mysql_query($query, $con) or die ('Error in query');    
 if (mysql_num_rows($result))  {
  $row = mysql_fetch_row($result);
  $strainName=$row[0];
  $sequence=$row[1];
  $a=1; $t=1; $c=1; $g=1; $u=1; $blast="blastn";

  $encoded_query = ''; $tmpArr=explode("\n",trim($sequence));
  for($i=0;$i<count($tmpArr);$i++){ 
   $seq=trim($tmpArr[$i]); $encoded_query .= urlencode($seq)."%0A"; 
   if ((substr($seq,0,3)==">") || (substr($seq,0,1)==">") ||  (substr($seq,0,4)=="&gt;")) {
   }else{
    $len=strlen($seq);
	for($j=0;$j<$len;$j++){
 	 $w=strtolower(substr($seq,$j,1));  
	 if (!isset($$w)) { 
	  $blast="blastp"; break; 
	 }
	}
   }
  }  
//  $link="https://blast.ncbi.nlm.nih.gov/Blast.cgi?PROGRAM=".$blast."&PAGE_TYPE=BlastSearch&BLAST_SPEC=Influenzavirus&LINK_LOC=blasttab&LAST_PAGE=".$blast."&QUERY=".$encoded_query;
  $link="https://blast.ncbi.nlm.nih.gov/Blast.cgi?PROGRAM=".$blast."&PAGE_TYPE=BlastSearch&LINK_LOC=blasttab&LAST_PAGE=".$blast."&QUERY=".$encoded_query;

  print '<a href=".$link.">click this link</a>';
  header("Refresh:1; URL=".$link);
	exit;
 }
}




?>
