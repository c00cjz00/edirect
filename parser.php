<?php
require_once("sql/DB_config.php");
require_once("sql/DB_class.php");
$db = new DB();
$db->connect_db($_DB['host'], $_DB['username'], $_DB['password'], $_DB['dbname']);

$fileArr=array("geneDB/AIV","geneDB/CSFV","geneDB/DHV","geneDB/FMDV","geneDB/PCV","geneDB/PRRSV","geneDB/RabiesV","geneDB/PEDV");
$tbArr=array("A001_ncbi","B001_ncbi","C001_ncbi","D001_ncbi","E001_ncbi","F001_ncbi","G001_ncbi","H001_ncbi",);
$tb2Arr=array("A001_AI禽流感","B001_CSF豬瘟病毒","C001_DHV鴨肝炎病毒","D001_FMDV口蹄疫病毒","E001_PCV2豬環狀病毒","F001_PRRSV豬繁殖障礙病毒","G001_RABIES狂犬病","H001_豬流行性下痢");
//$tbArr=array("A001_AI禽流感_guest","B001_CSF豬瘟病毒_guest","C001_DHV鴨肝炎病毒_guest","D001_FMDV口蹄疫病毒_guest","E001_PCV2豬環狀病毒_guest","F001_PRRSV豬繁殖障礙病毒_guest","G001_RABIES狂犬病_guest");

for($j=0;$j<count($tbArr);$j++){
 $tb=$tbArr[$j]; $file=$fileArr[$j];
 $tb2=$tb2Arr[$j];
 $sql="truncate table ".$tb; $db->query($sql);
 $sql="truncate table ".$tb2; $db->query($sql);
 $record=parser($file);
 $recordArr=explode("\n",trim($record));
 for($i=0;$i<count($recordArr);$i++){
  $tmp=trim($recordArr[$i]);
  if ($tmp!="") {
   $tmp=ereg_replace("'","\'",$tmp); 
   $r="'".ereg_replace("\t","','",$tmp)."'";
   $sql="INSERT INTO ".$tb." VALUES (".$r.")";    echo $sql."\n";   $db->query($sql);     
   $sql="INSERT INTO ".$tb2." VALUES (".$r.")";    echo $sql."\n";   $db->query($sql);
      
  }
 }
}
/* $result = $db->sql_query("INSERT INTO goodsOrdering VALUES (NULL, '$CAS', '$guestFirstName', '$guestLastName', '$guestCompany', '$guestEmail', '$guestPhone', '$guestFax', '$guestAddress', '$ip1', '$ip2', '$confirmData')");


$rArr=explode("\n",trim($result));
for($i=0;$i<count($rArr);$i++){
 $f="/slide/slide_Result/".trim($rArr[$i]);
  if (substr($f,-3,3)=="dzi"){
    if (!isset($$f)){
       $sql="INSERT INTO $DB (filename,owner,permission) VALUES ('$f','$owner','$permission')";
          echo $sql."\n"; $db->query($sql);
            }
  */          
 
function parser($file){
 $f1=$file.".txt";; $f2=$file."2.txt";
 $tmp1Arr=file($f1);
 $tmp2Arr=file($f2);


 $record=""; $p=0;
 for($i=0;$i<count($tmp2Arr);$i++){
 $tmp1=trim($tmp1Arr[$i]); $tmp2=trim($tmp2Arr[$i]);
 $smp1Arr=explode("\t",$tmp1); $smp2Arr=explode("\t",$tmp2);

 $date=trim($smp2Arr[4]);  $year_collect="";
 if (is_Date($date)){   
  $date_collect=date_create($date);
  $year_collect=date_format($date_collect,"Y");
  if  (is_numeric($date) && (strlen($date)==4)) $year_collect=$date;
 } 
 
 
 if (ereg("Influenza",$tmp2)){
  if (ereg("tw",$tmp2)|| eregi("taiwan",$tmp2)){
   $p++;
   $s=explode("/",$smp1Arr[2]); $host=$s[1];  
   $country=($smp2Arr[3]);
   if ($smp2Arr[3]=="-") $country=$s[2];
   $y=explode("-",$smp1Arr[1]); $year2=$y[count($y)-1];   
   $s=explode("(",$smp1Arr[2]); 
   if (count($s)==3){    
    $serotype=substr($s[2],0,-2);
    $h1=substr($serotype,0,2); $n1=substr($serotype,2,2);
    $s=explode("/",$s[1]); $year=$s[count($s)-1];
    $year=trim($year);
    //echo $year."\n";
    if (strlen($year)==2){
     if ((substr($year,0,1)=="0") || (substr($year,0,1)=="1")) {
      $year="20".$year;
     }else{
      $year="19".$year;
     }
    }
    if ($year=="Taiwan") $year=$year2;   
    if ($year_collect!="") $year=$year_collect;
    $a=trim($smp2Arr[1]);

    $b=trim($smp1Arr[4]);
    $memo=trim(str_replace($a,"",$b));
    $pArr=array("PB2","PB1","PA","HA","NP","NA","MP","NS");
	if (ereg("segment 1",$memo)){
	$seg="1 (".$pArr[0].")";
	}elseif (ereg("segment 2",$memo)){
	$seg="2 (".$pArr[1].")";
	}elseif (ereg("segment 3",$memo)){
	$seg="3 (".$pArr[2].")";
	}elseif (ereg("segment 4",$memo)){
	$seg="4 (".$pArr[3].")";
	}elseif (ereg("segment 5",$memo)){
	$seg="5 (".$pArr[4].")";
	}elseif (ereg("segment 6",$memo)){
	$seg="6 (".$pArr[5].")";
	}elseif (ereg("segment 7",$memo)){
	$seg="7 (".$pArr[6].")";
	}elseif (ereg("segment 8",$memo)){
	$seg="8 (".$pArr[7].")";
	}else{
	$seg="";
	}    

    $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tInfluenza A virus\t".$smp2Arr[1]."\t".$seg."\t".$h1."\t".$n1."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";
   }
  }
 }elseif (ereg("Classical swine fever",$tmp2)){
  $p++;
  $host="swine"; $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1]; 
  if ($year_collect!="") $year=$year_collect;  
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  $serotype="CSF-1";
  //echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";
  
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));            
                
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tClassical swine fever virus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";  
 }elseif (ereg("Duck hepatitis",$tmp2)){
  $p++;
  
  $host="duck"; $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1]; 
  if ($year_collect!="") $year=$year_collect;    
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  
  if (ereg("virus 1",$smp1Arr[2])) {
  $serotype="DHV-1";
  }elseif (ereg("virus 2",$smp1Arr[2])) {
  $serotype="DHV-2";
  }else{
  $serotype="-";
  }
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));
  
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tDuck hepatitis virus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";
  //  echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";     
 }elseif (ereg("Foot-and-mouth",$tmp2)){
  $p++;
//echo "---".implode(" ",$smp1Arr)."------------\n\n";
  $host="swine"; $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1];
  if ($year_collect!="") $year=$year_collect;    
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  
  if ($smp2Arr[2]!="-") $host=$smp2Arr[2];
  if (ereg("type O",$smp1Arr[2])) {
  $serotype="FMDV-O";
  }else{
  $serotype="-";
  }
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));
    
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tFoot-and-mouth virus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";
//  echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";         
 }elseif (ereg("Porcine circovirus 2",$tmp2)){
  $p++;
  
  $host="pig"; $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1]; 
  if ($year_collect!="") $year=$year_collect;    
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  $serotype="PVC2";
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));
  
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tPorcine circovirus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n"; 
//  echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";    
 }elseif (ereg("Porcine reproductive",$tmp2)){
  $p++;
  
  $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1];
  if ($year_collect!="") $year=$year_collect;    
  $host=($smp2Arr[2]); if ($host=="-") $host="-";  
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  $serotype="VR-2332";
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));
  
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tPorcine reproductive and respiratory syndrome virus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";
//  echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";         
 }elseif (ereg("Rabies",$tmp2)){
  $p++;
  
  $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1];
  if ($year_collect!="") $year=$year_collect;    
  $host=($smp2Arr[2]); if ($host=="-") $host="-";
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  $serotype="Rabies-1";
  //echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));
    
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tRabies lyssavirus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";
 }elseif (ereg("Porcine epidemic diarrhea",$tmp2)){
  $p++;
  
  $y=explode("-",$smp1Arr[1]); $year=$y[count($y)-1];
  if ($year_collect!="") $year=$year_collect;    
  $host=($smp2Arr[2]); if ($host=="-") $host="-";
  $country=($smp2Arr[3]); if ($country=="-") $country="Taiwan";
  $serotype="S protein";
  //echo $smp2Arr[0]."\t".$year."\t".$host."\t".$country."\t".$serotype."\t".$smp1Arr[2]."\t".$smp1Arr[3]."\n";
  $a=trim($smp2Arr[1]); $b=trim($smp1Arr[4]); $memo=trim(str_replace($a,"",$b));
    
  $record.=$p."\t".$smp2Arr[0]."\t".$host."\t\tPorcine epidemic diarrhea virus\t".$smp2Arr[1]."\t\t".$serotype."\t".$year."\t".$country."\t\t".$memo."\t".$smp1Arr[3]."\n";
 }
 }
 return $record;
}
function is_Date($str){ 
    $str = str_replace('/', '-', $str);     
    $stamp = strtotime($str);
    if (is_numeric($stamp)){  
       $month = date( 'm', $stamp ); 
       $day   = date( 'd', $stamp ); 
       $year  = date( 'Y', $stamp ); 
       return checkdate($month, $day, $year); 
    }  
    return false; 
}
?>
