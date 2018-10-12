<?php
$table="TB_Genome";
$owner="summerhill001@gmail.com"; $permission=0;
require_once("DB_config.php");
require_once("DB_class.php");
$db = new DB();
$db->connect_db($_DB['host'], $_DB['username'], $_DB['password'], $_DB['dbname']);
$file="geno001.txt";
$tmpArr=file($file);
for($i=0;$i<count($tmpArr);$i++){
 $smpArr=explode(";",trim($tmpArr[$i]));	
 $strainName=$smpArr[7];
 $haSubtypes=$smpArr[0];
 $naSubtypes=$smpArr[1];
 $genoType="H".$haSubtypes."N".$haSubtypes;
 //$genoType="";
 $year=$smpArr[2];
 $countryRegion=$smpArr[3];
 $hostSpecies=$smpArr[4];
 $samplingSource=$smpArr[6];
 $wildDomestic=$smpArr[5];
 $laboratory="淡水家衛所";
 $sequencePB2=rand(0,1); if ($sequencePB2==1) $sequencePB2="[PB2]";
 $sequencePB1=rand(0,1); if ($sequencePB1==1) $sequencePB1="[PB1]";
 $sequencePA=rand(0,1); if ($sequencePA==1) $sequencePA="[PA]";
 $sequenceHA=rand(0,1); if ($sequenceHA==1) $sequenceHA="[HA]";
 $sequenceNP=rand(0,1); if ($sequenceNP==1) $sequenceNP="[NP]";
 $sequenceNA=rand(0,1); if ($sequenceNA==1) $sequenceNA="[NA]";
 $sequenceM=rand(0,1); if ($sequenceM==1) $sequenceM="[M]";
 $sequenceNS=rand(0,1); if ($sequenceNS==1) $sequenceNS="[NS]";
 
 $remark="";
 $sql="INSERT INTO $table (strainName,haSubtypes,naSubtypes,genoType,year,countryRegion,hostSpecies,samplingSource,wildDomestic,laboratory,sequencePB2,sequencePB1,sequencePA,sequenceHA,sequenceNP,sequenceNA,sequenceM,sequenceNS,remark) VALUES ('$strainName','$haSubtypes','$naSubtypes','$genoType','$year','$countryRegion','$hostSpecies','$samplingSource','$wildDomestic','$laboratory','$sequencePB2','$sequencePB1','$sequencePA','$sequenceHA','$sequenceNP','$sequenceNA','$sequenceM','$sequenceNS','$remark')";
 echo $sql."\n"; $db->query($sql);   
}
?>
