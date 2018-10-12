<?php
require_once("DB_config.php");
require_once("DB_class.php");
$id = $_GET['id'];  $_DB['dbname']=$_GET['db']; $tbName = $_GET['tb']; $keyValue = $_GET['keyValue'];
//$id=1; $_DB['dbname']="nvri_bio20171009013840"; $tbName="A001_AI禽流感"; 

if (($id !="") && ($_DB['dbname']!="") && ($tbName!="")) {
 $blast="blastn";
 $db = new DB();
 $db->connect_db($_DB['host'], $_DB['username'], $_DB['password'], $_DB['dbname']);
 $sql = "SELECT a01,a04,s01,a06 FROM `".$tbName."` WHERE id = '".$id."'";

 $db->query($sql);
 while($row = $db->fetch_array()){
  $seq=trim($row['s01']);
  $seg=trim($row['a06']);
  $seqid=trim($row['a01']);

  if (substr($seq,0,1)==">"){
   $sequence=$seq;
  }else{
   $sequence=">".$seqid." ".trim($row['a04'])."\n".$seq;
  }
 }
  
 if (isset($sequence)){
  if ($keyValue=="blast"){  
   $encoded_query = ''; $tmpArr=explode("\n",trim($sequence));
   for($i=0;$i<count($tmpArr);$i++){ 
    $seq=trim($tmpArr[$i]); $encoded_query .= urlencode($seq)."%0A"; 
   } 
   $link="https://blast.ncbi.nlm.nih.gov/Blast.cgi?PROGRAM=".$blast."&PAGE_TYPE=BlastSearch&LINK_LOC=blasttab&LAST_PAGE=".$blast."&QUERY=".$encoded_query;

   print '<a href=".$link.">click this link</a>';
   header("Refresh:1; URL=".$link);
   exit;
  }elseif($keyValue=="browse"){
   date_default_timezone_set("Asia/Taipei");
   $nw=date("Ymdhis");
   $prgfile_hx = tempnam("/tmp", "nvri_"); $fp = fopen($prgfile_hx, "w"); fwrite($fp, $sequence); fclose($fp);
   $cmd="/srv/www/htdocs/nvriGenome/tools/JBrowse/bin/prepare-refseqs.pl --fasta ".$prgfile_hx." --out /srv/www/htdocs/nvriGenome/JBrowseData/".$nw;
   exec($cmd);
   if($seg=="4 (HA)"){
   $link="../tools/JBrowse/?data=../../JBrowseData%2F".$nw."&loc=".$seqid.":1000..1100";
   }else{
   $link="../tools/JBrowse/?data=../../JBrowseData%2F".$nw;
   }
   print '<a href="'.$link.'">click this link</a>';
   header("Refresh:1; URL=".$link);
   exit;   
  }
 }
}

/*
$sequence="AGCAAAAGCAGGGGATCAATCAAGAAATCAAAATGAAGAAAGTCCTGCTTTTTGCAGCAATCATCATCTG
CATTCAGGCAGACGAAATCTGCATTGGGTACCTGAGCAACAATTCAACAGAGAAAGTGGACACAATAATT
GAGAGTAATGTCACGGTTACTAGCTCAGTTGAACTGGTTGAAAATGAGCACACTGGATCTTTCTGCTCAA
TCGACGGGAAAGTGCCGATAAGTCTTGGTGATTGCTCCTTTGCTGGATGGATCCTTGGGAACCCGATGTG
TGATGATTTGATTGGAAAAACATCATGGTCTTACATAGTAGAGAAACCGAACCCCGTTAATGGAATATGC
TATCCGGGTACTCTAGAAAATGAGGAAGAATTGAGACTGAAGTTTAGTGGGGTTCTCGAATTCAGCAAAT
TTGAAGCCTTCACCTCAAACGGATGGGGAGCAGTGAATTCCGGTGCTGGTGTTACCGCAGCCTGCAAATT
TGGAAGCAGTAACTCTTTTTTCAGAAACATGGTATGGTTGATACATCAATCAGGGACATACCCTGTCATA
CGGAGGACATTCAACAACACCAAAGGAGGAGATGTATTAATGGTATGGGGAGTTCACCATCCTGCAACTC
TAAAGGAACACCAAGATTTGTACAAAAAAGACAGTTCTTATGTAGCAGTGGGTTCAGAGAGTTATAACAG
GAGGTTCACCCCTGAGATTAGCACAAGACCTAGAGTAAATGGTCAGGCTGGGAGAATGACCTTCTACTGG
ACCATAGTGAAACCTGGAGAGGCAATAACATTTGAGTCAAATGGTGCATTTCTCGCCCCTCGATATGCTT
TTGAATTAGTGTCTTTGGGAAATGGGAAATTGTTCAGAAGTGACTTGAATATTGAATCCTGCTCAACTAA
ATGCCAGTCTGAAATTGGAGGGATCAACACCAATAGAAGCTTCCATAGTGTCCATAGAAACACAATAGGA
AACTGCCCCAAATATGTGAATGTTAAATCATTAAGGCTTGCTACCGGGCTCAGAAATGTCCCTGCGATTG
CTGCAAGAGGCCTGTTTGGTGCAATAGCTGGTTTCATAGAAGGTGGTTGGCCAGGTTTAATCAATGGTTG
G";
$encoded_query = ''; 
$tmpArr=explode("\n",trim($sequence));
for($i=0;$i<count($tmpArr);$i++){ 
 $seq=trim($tmpArr[$i]); $encoded_query .= urlencode($seq)."%0A"; 
}
$blast="blastn";
$link="https://blast.ncbi.nlm.nih.gov/Blast.cgi?PROGRAM=".$blast."&PAGE_TYPE=BlastSearch&LINK_LOC=blasttab&LAST_PAGE=".$blast."&QUERY=".$encoded_query;
print '<a href=".$link.">click this link</a>';
header("Refresh:1; URL=".$link);
exit;
*/

?>
