./esearch -db nuccore -query "Influenza A virus [TITL] NOT (human) NOT (homo) AND (TAIWAN) AND (AVIAN)"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/AIV.txt
sleep 2
./esearch -db nuccore -query "Classical swine fever [TITL] NOT (human) NOT (homo) AND (TAIWAN) AND (swine )"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/CSFV.txt
sleep 2
./esearch -db nuccore -query "Duck hepatitis [TITL] NOT (human) NOT (homo) AND (TAIWAN) AND (duck)"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/DHV.txt
sleep 2
./esearch -db nuccore -query "Foot-and-mouth [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/FMDV.txt
sleep 2
./esearch -db nuccore -query "porcine circovirus [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/PCV.txt
sleep 2
./esearch -db nuccore -query "Porcine Reproductive and Respiratory Syndrome [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/PRRSV.txt
sleep 2
./esearch -db nuccore -query "Rabies virus [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -pattern INSDSeq -element INSDSeq_accession-version INSDSeq_update-date INSDSeq_organism INSDSeq_sequence INSDSeq_definition INSDQualifier_value > geneDB/RabiesV.txt
sleep 2
./esearch -db nuccore -query "Influenza A virus [TITL] NOT (human) NOT human NOT (homo) AND (TAIWAN) AND (AVIAN)"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/AIV2.txt
sleep 2
./esearch -db nuccore -query "Classical swine fever [TITL] NOT (human) NOT (homo) AND (TAIWAN) AND (swine )"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/CSFV2.txt
sleep 2
./esearch -db nuccore -query "Duck hepatitis [TITL] NOT (human) NOT (homo) AND (TAIWAN) AND (duck)"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/DHV2.txt
sleep 2
./esearch -db nuccore -query "Foot-and-mouth [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/FMDV2.txt
sleep 2
./esearch -db nuccore -query "porcine circovirus [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/PCV2.txt
sleep 2
./esearch -db nuccore -query "Porcine Reproductive and Respiratory Syndrome [TITL] NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/PRRSV2.txt
sleep 2

./esearch -db nuccore -query "Rabies virus [TITL]NOT (human) NOT (homo) AND (TAIWAN)"  | ./efetch -format gpc | ./xtract -insd source organism host country collection_date > geneDB/RabiesV2.txt
sleep 2

#php parser.php

