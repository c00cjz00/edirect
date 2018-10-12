// ===========================================================================
//
//                            PUBLIC DOMAIN NOTICE
//            National Center for Biotechnology Information (NCBI)
//
//  This software/database is a "United States Government Work" under the
//  terms of the United States Copyright Act. It was written as part of
//  the author's official duties as a United States Government employee and
//  thus cannot be copyrighted. This software/database is freely available
//  to the public for use. The National Library of Medicine and the U.S.
//  Government do not place any restriction on its use or reproduction.
//  We would, however, appreciate having the NCBI and the author cited in
//  any work or product based on this material.
//
//  Although all reasonable efforts have been taken to ensure the accuracy
//  and reliability of the software and data, the NLM and the U.S.
//  Government do not and cannot warrant the performance or results that
//  may be obtained by using this software or data. The NLM and the U.S.
//  Government disclaim all warranties, express or implied, including
//  warranties of performance, merchantability or fitness for any particular
//  purpose.
//
// ===========================================================================
//
// File Name:  xtract.go
//
// Author:  Jonathan Kans
//
// ==========================================================================

/*
  Download external GO libraries by running:

  cd "$GOPATH"
  go get -u golang.org/x/text/runes
  go get -u golang.org/x/text/transform
  go get -u golang.org/x/text/unicode/norm

  Test for presence of go compiler, cross-compile xtract executables, and pack into archive, by running:

  if hash go 2>/dev/null
  then
    env GOOS=darwin GOARCH=amd64 go build -o xtract.Darwin -v xtract.go
    env GOOS=linux GOARCH=amd64 go build -o xtract.Linux -v xtract.go
    env GOOS=windows GOARCH=386 go build -o xtract.CYGWIN_NT -v xtract.go
    tar -czf archive.tar.gz xtract.[A-Z]*
    rm xtract.[A-Z]*
  fi
*/

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"container/heap"
	"fmt"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"hash/crc32"
	"html"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/user"
	"path"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// VERSION AND HELP MESSAGE TEXT

const xtractVersion = "7.40"

const xtractHelp = `
Overview

  Xtract uses command-line arguments to convert XML data into a tab-delimited table.

  -pattern places the data from individual records into separate rows.

  -element extracts values from specified fields into separate columns.

  -group, -block, and -subset limit element exploration to selected XML subregions.

Processing Flags

  -strict          Remove HTML highlight tags
  -mixed           Allow PubMed mixed content

  -accent          Delete Unicode accents
  -ascii           Unicode to numeric character references
  -compress        Compress runs of spaces
  -spaces          Fix non-ASCII spaces

Data Source

  -input           Read XML from file instead of stdin

Exploration Argument Hierarchy

  -pattern         Name of record within set
  -group             Use of different argument
  -block               names allows command-line
  -subset                control of nested looping

Exploration Constructs

  Object           DateCreated
  Parent/Child     Book/AuthorList
  Heterogeneous    "PubmedArticleSet/*"
  Nested           "*/Taxon"
  Recursive        "**/Gene-commentary"

Conditional Execution

  -if              Element [@attribute] required
  -unless          Skip if element matches
  -and             All tests must pass
  -or              Any passing test suffices
  -else            Execute if conditional test failed
  -position        Must be at [first|last] location in list

String Constraints

  -equals          String must match exactly
  -contains        Substring must be present
  -starts-with     Substring must be at beginning
  -ends-with       Substring must be at end
  -is-not          String must not match

Numeric Constraints

  -gt              Greater than
  -ge              Greater than or equal to
  -lt              Less than
  -le              Less than or equal to
  -eq              Equal to
  -ne              Not equal to

Format Customization

  -ret             Override line break between patterns
  -tab             Replace tab character between fields
  -sep             Separator between group members
  -pfx             Prefix to print before group
  -sfx             Suffix to print after group
  -clr             Clear queued tab separator
  -pfc             Preface combines -clr and -pfx
  -rst             Reset -sep, -pfx, and -sfx
  -def             Default placeholder for missing fields
  -lbl             Insert arbitrary text

Element Selection

  -element         Print all items that match tag name
  -first           Only print value of first item
  -last            Only print value of last item
  -NAME            Record value in named variable

-element Constructs

  Tag              Caption
  Group            Initials,LastName
  Parent/Child     MedlineCitation/PMID
  Attribute        DescriptorName@MajorTopicYN
  Recursive        "**/Gene-commentary_accession"
  Object Count     "#Author"
  Item Length      "%Title"
  Element Depth    "^PMID"
  Variable         "&NAME"

Special -element Operations

  Parent Index     "+"
  XML Subtree      "*"
  Children         "$"
  Attributes       "@"

Numeric Processing

  -num             Count
  -len             Length
  -sum             Sum
  -min             Minimum
  -max             Maximum
  -inc             Increment
  -dec             Decrement
  -sub             Difference
  -avg             Average
  -dev             Deviation

String Processing

  -encode          URL-encode <, >, &, ", and ' characters
  -upper           Convert text to upper-case
  -lower           Convert text to lower-case
  -title           Capitalize initial letters of words

Phrase Processing

  -terms           Partition phrase at spaces
  -words           Split at punctuation marks
  -pairs           Adjacent informative words
  -letters         Separate individual letters
  -indices         Experimental index generation

Phrase Filtering

  -phrase          Keep records that contain given phrase

Sequence Coordinates

  -0-based         Zero-Based
  -1-based         One-Based
  -ucsc-based      Half-Open

Command Generator

  -insd            Generate INSDSeq extraction commands

-insd Argument Order

  Descriptors      INSDSeq_sequence INSDSeq_definition INSDSeq_division
  Flags            [complete|partial]
  Feature(s)       CDS,mRNA
  Qualifiers       INSDFeature_key "#INSDInterval" gene product

Miscellaneous

  -head            Print before everything else
  -tail            Print after everything else
  -hd              Print before each record
  -tl              Print after each record

Reformatting

  -format          [copy|compact|flush|indent|expand]

Modification

  -filter          Object
                     [retain|remove|encode|decode|shrink|expand|accent]
                       [content|cdata|comment|object|attributes|container]

Validation

  -verify          Report XML data integrity problems

Summary

  -outline         Display outline of XML structure
  -synopsis        Display count of unique XML paths

Documentation

  -help            Print this document
  -examples        Examples of EDirect and xtract usage
  -extras          Batch and local processing examples
  -version         Print version number

Notes

  String constraints use case-insensitive comparisons.

  Numeric constraints and selection arguments use integer values.

  -num and -len selections are synonyms for Object Count (#) and Item Length (%).

  -words, -pairs, and -indices convert to lower case.

Examples

  -pattern DocumentSummary -element Id -first Name Title

  -pattern "PubmedArticleSet/*" -block Author -sep " " -element Initials,LastName

  -pattern PubmedArticle -block MeshHeading -if "@MajorTopicYN" -equals Y -sep " / " -element DescriptorName,QualifierName

  -pattern GenomicInfoType -element ChrAccVer ChrStart ChrStop

  -pattern Taxon -block "*/Taxon" -unless Rank -equals "no rank" -tab "\n" -element Rank,ScientificName

  -pattern Entrezgene -block "**/Gene-commentary"

  -block INSDReference -position 2

  -if Author -and Title

  -if "#Author" -lt 6 -and "%Title" -le 70

  -if DateCreated/Year -gt 2005

  -if ChrStop -lt ChrStart

  -if CommonName -contains mouse

  -if "&ABST" -starts-with "Transposable elements"

  -if MapLocation -element MapLocation -else -lbl "\-"

  -min ChrStart,ChrStop

  -max ExonCount

  -inc @aaPosition -element @residue

  -1-based ChrStart

  -insd CDS gene product protein_id translation

  -insd complete mat_peptide "%peptide" product peptide

  -insd CDS INSDInterval_iscomp@value INSDInterval_from INSDInterval_to

  -filter ExpXml decode content

  -filter LocationHist remove object
`

const xtractExtras = `
Local Record Indexing

  -archive    Base path for individual XML files
  -index      Name of element to use for identifier

  -flag       [strict|mixed|none]
  -gzip       Use compression for local XML files
  -hash       Print UIDs and checksum values to stdout
  -skip       File of UIDs to skip

Sample File Download

  ftp-cp ftp.ncbi.nlm.nih.gov /entrez/entrezdirect/samples carotene.xml.zip
  unzip carotene.xml.zip
  rm carotene.xml.zip

Mammalian Sequence Download

  download-sequence gbmam gbpri gbrod

Human Subset Extraction

  #!/bin/sh

  for fl in gbpri?.aso.gz gbpri??.aso.gz
  do
    run-ncbi-converter asn2all -i "$fl" -a t -b -c -O 9606 -f s > ${fl%.aso.gz}.xml
  done

Deleted PMID File Download

  ftp-cp ftp.ncbi.nlm.nih.gov /pubmed deleted.pmids.gz
  gunzip deleted.pmids.gz

PubMed Download

  download-pubmed baseline updatefiles

PubMed Archive Creation

  stash-pubmed mixed /Volumes/myssd/Pubmed

PubMed Archive Maintenance

  cat deleted.pmids |
  erase-pubmed /Volumes/myssd/Pubmed

PubMed Archive Retrieval

  cat lycopene.uid |
  fetch-pubmed mixed /Volumes/myssd/Pubmed > lycopene.xml
`

const xtractAdvanced = `
Processing Commands

  -prepare    [release|report] Compare daily update to archive
  -ignore     Ignore contents of object in -prepare comparisons
  -missing    Print list of missing identifiers

Update Candidate Report

  gzcat medline*.xml.gz | xtract -strict -compress -format flush |
  xtract -prepare report -ignore DateRevised -archive /Volumes/myssd/Pubmed \
    -index MedlineCitation/PMID -pattern PubmedArticle

Unnecessary Update Removal

  gzcat medline*.xml.gz | xtract -strict -compress -format flush |
  xtract -prepare release -ignore DateRevised -archive /Volumes/myssd/Pubmed -index MedlineCitation/PMID \
    -head "<PubmedArticleSet>" -tail "</PubmedArticleSet>" -pattern PubmedArticle |
  xtract -format indent -xml '<?xml version="1.0" encoding="utf-8"?>' \
    -doctype '<!DOCTYPE PubmedArticleSet PUBLIC "-//NLM//DTD PubMedArticle, 1st January 2017//EN" "https://dtd.nlm.nih.gov/ncbi/pubmed/out/pubmed_170101.dtd">' |
  gzip -9 > updatesubset.xml.gz

Reconstruct Release Files

  split -a 3 -l 30000 release.uid uids-
  n=1
  for x in uids-???
  do
    xmlfile=$(printf "medline17n%04d.xml.gz" "$n")
    n=$((n+1))
    echo "$xmlfile"
    cat "$x" | xtract -archive /Volumes/myssd/Pubmed -head "<PubmedArticleSet>" -tail "</PubmedArticleSet>" |
    xtract -format indent -xml '<?xml version="1.0" encoding="utf-8"?>' \
      -doctype '<!DOCTYPE PubmedArticleSet PUBLIC "-//NLM//DTD PubMedArticle, 1st January 2017//EN" "https://dtd.nlm.nih.gov/ncbi/pubmed/out/pubmed_170101.dtd">' |
    gzip -9 > "$xmlfile"
  done
  rm -rf uids-???

Experimental Postings File Creation

  efetch -db pubmed -id 12857958,2981625 -format xml |
  xtract -e2index PubmedArticle MedlineCitation/PMID ArticleTitle,AbstractText,Keyword |
  xtract -pattern IdxDocument -UID IdxUid \
    -block NORM -pfc "\n" -element "&UID",NORM |
  LC_ALL='C' sort -k 2f -k 1n |
  xtract -posting "/Volumes/myssd/Postings/NORM"

DISABLE ANTI-VIRUS FILE SCANNING FOR LOCAL ARCHIVES OR MOVE TO TRUSTED FILES

DISABLE SPOTLIGHT INDEXING FOR EXTERNAL DISKS CONTAINING LOCAL ARCHIVES
`

const xtractInternal = `
ReadBlocks -> SplitPattern => StreamTokens => ParseXML => ProcessQuery -> MergeResults

Performance Default Overrides

  -proc     Number of CPU processors used
  -cons     Ratio of parsers to processors
  -serv     Concurrent parser instances
  -chan     Communication channel depth
  -heap     Order restoration heap size
  -farm     Node allocation buffer length
  -gogc     Garbage collection tuning knob

Debugging

  -debug    Display run-time parameter summary
  -empty    Flag records with no output
  -ident    Print record index numbers
  -stats    Show processing time for each record
  -timer    Report processing duration and rate
  -trial    Optimize -proc value, requires -input

Documentation

  -keys     Keyboard navigation shortcuts
  -unix     Common Unix commands

Performance Tuning Script

  XtractTrials() {
    echo -e "<Trials>"
    for tries in {1..5}
    do
      xtract -debug -input "$1" -proc "$2" -pattern PubmedArticle -element LastName
    done
    echo -e "</Trials>"
  }

  for proc in {1..8}
  do
    XtractTrials "carotene.xml" "$proc" |
    xtract -pattern Trials -lbl "$proc" -avg Rate -dev Rate
  done

Processor Titration Results

  1    27622    31
  2    51799    312
  3    74853    593
  4    95867    1337
  5    97171    4019
  6    93460    2458
  7    87467    1030
  8    82448    2651

Execution Profiling

  xtract -profile -input carotene.xml -pattern PubmedArticle -element LastName
  go tool pprof --pdf ./xtract ./cpu.pprof > ./callgraph.pdf
`

const xtractExamples = `
Author Frequency

  esearch -db pubmed -query "rattlesnake phospholipase" |
  efetch -format docsum |
  xtract -pattern DocumentSummary -sep "\n" -element Name |
  sort-uniq-count-rank

  39    Marangoni S
  31    Toyama MH
  26    Soares AM
  25    Bon C
  ...

Publications

  efetch -db pubmed -id 6271474,5685784,4882854,6243420 -format xml |
  xtract -pattern PubmedArticle -element MedlineCitation/PMID "#Author" \
    -block Author -position first -sep " " -element Initials,LastName \
    -block Article -element ArticleTitle

  6271474    5    MJ Casadaban     Tn3: transposition and control.
  5685784    2    RK Mortimer      Suppressors and suppressible mutations in yeast.
  4882854    2    ED Garber        Proteins and enzymes as taxonomic tools.
  6243420    1    NR Cozzarelli    DNA gyrase and the supercoiling of DNA.

Formatted Authors

  efetch -db pubmed -id 1413997,6301692,781293 -format xml |
  xtract -pattern PubmedArticle -element MedlineCitation/PMID \
    -block DateCreated -sep "-" -element Year,Month,Day \
    -block Author -sep " " -tab "" \
      -element "&COM" Initials,LastName -COM "(, )"

  1413997    1992-11-25    RK Mortimer, CR Contopoulou, JS King
  6301692    1983-06-17    MA Krasnow, NR Cozzarelli
  781293     1976-10-02    MJ Casadaban

Medical Subject Headings

  efetch -db pubmed -id 6092233,2539356,1937004 -format xml |
  xtract -pattern PubmedArticle -element MedlineCitation/PMID \
    -block MeshHeading \
      -subset DescriptorName -pfc "\n" -sep "|" -element @MajorTopicYN,DescriptorName \
      -subset QualifierName -pfc " / " -sep "|" -element @MajorTopicYN,QualifierName |
  sed -e 's/N|//g' -e 's/Y|/*/g'

  6092233
  Base Sequence
  DNA Restriction Enzymes
  DNA, Fungal / genetics / *isolation & purification
  *Genes, Fungal
  ...

Peptide Sequences

  esearch -db protein -query "conotoxin AND mat_peptide [FKEY]" |
  efetch -format gpc |
  xtract -insd complete mat_peptide "%peptide" product peptide |
  grep -i conotoxin | sort -t $'\t' -u -k 2,2n | head -n 8

  ADB43131.1    15    conotoxin Cal 1b      LCCKRHHGCHPCGRT
  ADB43128.1    16    conotoxin Cal 5.1     DPAPCCQHPIETCCRR
  AIC77105.1    17    conotoxin Lt1.4       GCCSHPACDVNNPDICG
  ADB43129.1    18    conotoxin Cal 5.2     MIQRSQCCAVKKNCCHVG
  ADD97803.1    20    conotoxin Cal 1.2     AGCCPTIMYKTGACRTNRCR
  AIC77085.1    21    conotoxin Bt14.8      NECDNCMRSFCSMIYEKCRLK
  ADB43125.1    22    conotoxin Cal 14.2    GCPADCPNTCDSSNKCSPGFPG
  AIC77154.1    23    conotoxin Bt14.19     VREKDCPPHPVPGMHKCVCLKTC

Chromosome Locations

  esearch -db gene -query "calmodulin [PFN] AND mammalia [ORGN]" |
  efetch -format docsum |
  xtract -pattern DocumentSummary \
    -def "-" -element Id Name MapLocation ScientificName

  801       CALM1    14q32.11     Homo sapiens
  808       CALM3    19q13.32     Homo sapiens
  805       CALM2    2p21         Homo sapiens
  24242     Calm1    6q32         Rattus norvegicus
  12313     Calm1    12 E         Mus musculus
  326597    CALM     -            Bos taurus
  50663     Calm2    6q12         Rattus norvegicus
  24244     Calm3    1q21         Rattus norvegicus
  12315     Calm3    7 9.15 cM    Mus musculus
  12314     Calm2    17 E4        Mus musculus
  617095    CALM1    -            Bos taurus
  396838    CALM3    6            Sus scrofa
  ...

Gene Regions

  esearch -db gene -query "DDT [GENE] AND mouse [ORGN]" |
  efetch -format docsum |
  xtract -pattern GenomicInfoType -element ChrAccVer ChrStart ChrStop |
  xargs -n 3 sh -c 'efetch -db nuccore -format gb \
    -id "$0" -chr_start "$1" -chr_stop "$2"'

  LOCUS       NC_000076               2142 bp    DNA     linear   CON 09-FEB-2015
  DEFINITION  Mus musculus strain C57BL/6J chromosome 10, GRCm38.p3 C57BL/6J.
  ACCESSION   NC_000076 REGION: complement(75771233..75773374) GPC_000000783
  VERSION     NC_000076.6
  ...
  FEATURES             Location/Qualifiers
       source          1..2142
                       /organism="Mus musculus"
                       /mol_type="genomic DNA"
                       /strain="C57BL/6J"
                       /db_xref="taxon:10090"
                       /chromosome="10"
       gene            1..2142
                       /gene="Ddt"
       mRNA            join(1..159,462..637,1869..2142)
                       /gene="Ddt"
                       /product="D-dopachrome tautomerase"
                       /transcript_id="NM_010027.1"
       CDS             join(52..159,462..637,1869..1941)
                       /gene="Ddt"
                       /codon_start=1
                       /product="D-dopachrome decarboxylase"
                       /protein_id="NP_034157.1"
                       /translation="MPFVELETNLPASRIPAGLENRLCAATATILDKPEDRVSVTIRP
                       GMTLLMNKSTEPCAHLLVSSIGVVGTAEQNRTHSASFFKFLTEELSLDQDRIVIRFFP
                       ...

Taxonomic Names

  esearch -db taxonomy -query "txid10090 [SBTR] OR camel [COMN]" |
  efetch -format docsum |
  xtract -pattern DocumentSummary -if CommonName \
    -element Id ScientificName CommonName

  57486    Mus musculus molossinus    Japanese wild mouse
  39442    Mus musculus musculus      eastern European house mouse
  35531    Mus musculus bactrianus    southwestern Asian house mouse
  10092    Mus musculus domesticus    western European house mouse
  10091    Mus musculus castaneus     southeastern Asian house mouse
  10090    Mus musculus               house mouse
  9838     Camelus dromedarius        Arabian camel
  9837     Camelus bactrianus         Bactrian camel

Structural Similarity

  esearch -db structure -query "crotalus [ORGN] AND phospholipase A2" |
  elink -related |
  efilter -query "archaea [ORGN]" |
  efetch -format docsum |
  xtract -pattern DocumentSummary \
    -if PdbClass -equals Hydrolase \
      -element PdbAcc PdbDescr

  3VV2    Crystal Structure Of Complex Form Between S324a-subtilisin And Mutant Tkpro
  3VHQ    Crystal Structure Of The Ca6 Site Mutant Of Pro-Sa-Subtilisin
  2ZWP    Crystal Structure Of Ca3 Site Mutant Of Pro-S324a
  2ZWO    Crystal Structure Of Ca2 Site Mutant Of Pro-S324a
  ...

Multiple Links

  esearch -db pubmed -query "conotoxin AND dopamine [MAJR]" |
  elink -target protein -cmd neighbor |
  xtract -pattern LinkSet -if Link/Id -element IdList/Id Link/Id

  23624852    17105332
  14657161    27532980    27532978
  12944511    31542395
  11222635    144922602

Gene Comments

  esearch -db gene -query "rbcL [GENE] AND maize [ORGN]" |
  efetch -format xml |
  xtract -pattern Entrezgene -block "**/Gene-commentary" \
    -if Gene-commentary_type@value -equals genomic \
      -tab "\n" -element Gene-commentary_accession |
  sort | uniq

  NC_001666
  X86563
  Z11973

Vitamin Biosynthesis

  esearch -db pubmed -query "tomato lycopene cyclase" |
  elink -related |
  elink -target protein |
  efilter -organism mammals |
  efetch -format gpc |
  xtract -pattern INSDSeq -if INSDSeq_definition -contains carotene \
    -element INSDSeq_accession-version INSDSeq_definition

  NP_573480.1       beta,beta-carotene 9',10'-oxygenase [Mus musculus]
  NP_001156500.1    beta,beta-carotene 15,15'-dioxygenase isoform 2 [Mus musculus]
  NP_067461.2       beta,beta-carotene 15,15'-dioxygenase isoform 1 [Mus musculus]
  NP_001297121.1    beta-carotene oxygenase 2 [Mustela putorius furo]
  AAS20392.1        carotene-9',10'-monooxygenase [Mustela putorius furo]

Indexed Fields

  einfo -db pubmed |
  xtract -pattern Field \
    -if IsDate -equals Y -and IsHidden -equals N \
      -pfx "[" -sep "]\t" -element Name,FullName |
  sort -t $'\t' -k 2f

  [CDAT]    Date - Completion
  [CRDT]    Date - Create
  [EDAT]    Date - Entrez
  [MHDA]    Date - MeSH
  [MDAT]    Date - Modification
  [PDAT]    Date - Publication

Author Numbers

  esearch -db pubmed -query "conotoxin" |
  efetch -format xml |
  xtract -pattern PubmedArticle -num Author |
  sort-uniq-count -n |
  reorder-columns 2 1 |
  head -n 15 |
  xy-plot auth.png

  0     11
  1     193
  2     854
  3     844
  4     699
  5     588
  6     439
  7     291
  8     187
  9     124
  10    122
  11    58
  12    33
  13    18

  900 +
      |           ********
  800 +           *       **
      |          *          *
  700 +          *          ***
      |          *             **
  600 +         *                *
      |         *                ***
  500 +         *                   **
      |        *                      ***
  400 +       *                          **
      |       *                            *
  300 +       *                            ***
      |      *                                *
  200 +      *                                 ******
      |     *                                        *********
  100 +   **                                                  *
      |  *                                                     **********
    0 + *                                                                ******
        +---------+---------+---------+---------+---------+---------+---------+
        0         2         4         6         8        10        12        14

Record Counts

  echo "diphtheria measles pertussis polio tuberculosis" |
  xargs -n 1 sh -c 'esearch -db pubmed -query "$0 [MESH]" |
  efilter -days 365 -datetype PDAT |
  xtract -pattern ENTREZ_DIRECT -lbl "$0" -element Count'

  diphtheria      18
  measles         166
  pertussis       98
  polio           75
  tuberculosis    1386

Gene Products

  for sym in HBB DMD TTN ATP7B HFE BRCA2 CFTR PAH PRNP RAG1
  do
    esearch -db gene -query "$sym [GENE] AND human [ORGN]" |
    efilter -query "alive [PROP]" | efetch -format docsum |
    xtract -pattern GenomicInfoType \
      -element ChrAccVer ChrStart ChrStop |
    while read acc str stp
    do
      efetch -db nuccore -format gbc \
        -id "$acc" -chr_start "$str" -chr_stop "$stp" |
      xtract -insd CDS,mRNA INSDFeature_key "#INSDInterval" \
        gene "%transcription" "%translation" \
        product transcription translation |
      grep -i $'\t'"$sym"$'\t'
    done
  done

  NC_000011.10    mRNA    3     HBB    626      hemoglobin, beta                     ACATTTGCTT...
  NC_000011.10    CDS     3     HBB    147      hemoglobin subunit beta              MVHLTPEEKS...
  NC_000023.11    mRNA    78    DMD    13805    dystrophin, transcript variant X2    AGGAAGATGA...
  NC_000023.11    mRNA    77    DMD    13794    dystrophin, transcript variant X6    ACTTTCCCCC...
  NC_000023.11    mRNA    77    DMD    13800    dystrophin, transcript variant X5    ACTTTCCCCC...
  NC_000023.11    mRNA    77    DMD    13785    dystrophin, transcript variant X7    ACTTTCCCCC...
  NC_000023.11    mRNA    74    DMD    13593    dystrophin, transcript variant X8    ACTTTCCCCC...
  NC_000023.11    mRNA    75    DMD    13625    dystrophin, transcript variant X9    ACTTTCCCCC...
  ...

Genome Range

  esearch -db gene -query "Homo sapiens [ORGN] AND Y [CHR]" |
  efilter -status alive | efetch -format docsum |
  xtract -pattern DocumentSummary -NAME Name -DESC Description \
    -block GenomicInfoType -if ChrLoc -equals Y \
      -min ChrStart,ChrStop -element "&NAME" "&DESC" |
  sort -k 1,1n | cut -f 2- | grep -v uncharacterized |
  between-two-genes ASMT IL3RA

  IL3RA        interleukin 3 receptor subunit alpha
  SLC25A6      solute carrier family 25 member 6
  LINC00106    long intergenic non-protein coding RNA 106
  ASMTL-AS1    ASMTL antisense RNA 1
  ASMTL        acetylserotonin O-methyltransferase-like
  P2RY8        purinergic receptor P2Y8
  AKAP17A      A-kinase anchoring protein 17A
  ASMT         acetylserotonin O-methyltransferase

3'UTR Sequences

  ThreePrimeUTRs() {
    xtract -pattern INSDSeq -ACC INSDSeq_accession-version -SEQ INSDSeq_sequence \
      -block INSDFeature -if INSDFeature_key -equals CDS \
        -pfc "\n" -element "&ACC" -rst -last INSDInterval_to -element "&SEQ" |
    while read acc pos seq
    do
      if [ $pos -lt ${#seq} ]
      then
        echo -e ">$acc 3'UTR: $((pos+1))..${#seq}"
        echo "${seq:$pos}" | fold -w 50
      elif [ $pos -ge ${#seq} ]
      then
        echo -e ">$acc NO 3'UTR"
      fi
    done
  }

  esearch -db nuccore -query "5.5.1.19 [ECNO]" |
  efilter -molecule mrna -source refseq |
  efetch -format gbc | ThreePrimeUTRs

  >NM_001328461.1 3'UTR: 1737..1871
  gatgaatatagagttactgtgttgtaagctaatcatcatactgatgcaag
  tgcattatcacatttacttctgctgatgattgttcataagattatgagtt
  agccatttatcaaaaaaaaaaaaaaaaaaaaaaaa
  >NM_001316759.1 3'UTR: 1628..1690
  atccgagtaattcggaatcttgtccaattttatatagcctatattaatac
  ...

Amino Acid Substitutions

  ApplySNPs() {
    seq=""
    last=""

    while read rsid accn pos res
    do
      if [ "$accn" != "$last" ]
      then
        insd=$(efetch -db protein -id "$accn" -format gbc < /dev/null)
        seq=$(echo $insd | xtract -pattern INSDSeq -element INSDSeq_sequence)
        last=$accn
      fi

      pos=$((pos+1))
      pfx=""
      sfx=""
      echo ">rs$rsid [$accn $res@$pos]"
      if [ $pos -gt 1 ]
      then
        pfx=$(echo ${seq:0:$pos-1})
      fi
      if [ $pos -lt ${#seq} ]
      then
        sfx=$(echo ${seq:$pos})
      fi
      echo "$pfx$res$sfx" | fold -w 50
    done
  }

  esearch -db gene -query "OPN1MW [GENE] AND human [ORGN]" |
  elink -target snp | efetch -format xml |
  xtract -pattern Rs -RSID Rs@rsId \
    -block FxnSet -if @fxnClass -equals missense \
      -sep "." -element "&RSID" @protAcc,@protVer @aaPosition \
      -tab "\n" -element @residue |
  sort -t $'\t' -k 2,2 -k 3,3n -k 4,4 | uniq |
  ApplySNPs

  >rs104894915 [NP_000504.1 K@94]
  maqqwslqrlagrhpqdsyedstqssiftytnsnstrgpfegpnyhiapr
  wvyhltsvwmifvviasvftnglvlaatmkfkklrhplnwilvKlavadl
  aetviastisvvnqvygyfvlghpmcvlegytvslcgitglwslaiiswe
  ...

Amino Acid Composition

  #!/bin/bash -norc

  abbrev=( Ala Asx Cys Asp Glu Phe Gly His Ile \
           Xle Lys Leu Met Asn Pyl Pro Gln Arg \
           Ser Thr Sec Val Trp Xxx Tyr Glx )

  AminoAcidComp() {
    local count
    while read num lttr
    do
      idx=$(printf %i "'$lttr'")
      ofs=$((idx-97))
      count[$ofs]="$num"
    done <<< "$1"
    for i in {0..25}
    do
      echo -e "${abbrev[$i]}\t${count[$i]-0}"
    done |
    sort
  }

  AminoAcidJoin() {
    result=""
    while read acc seq gene
    do
      comp="$(echo "$seq" | tr A-Z a-z | sed 's/[^a-z]//g' | fold -w 1 | sort-uniq-count)"
      current=$(AminoAcidComp "$comp")
      current=$(echo -e "GENE\t$gene\n$current")
      if [ -n "$result" ]
      then
        result=$(join -t $'\t' <(echo "$result") <(echo "$current"))
      else
        result=$current
      fi
    done
    echo "$result" |
    grep -e "GENE" -e "[1-9]"
  }

  ids="NP_001172026,NP_000509,NP_004001,NP_001243779"
  efetch -db protein -id "$ids" -format gpc |
  xtract -insd INSDSeq_sequence CDS gene |
  AminoAcidJoin

  GENE    INS    HBB    DMD    TTN
  Ala     10     15     210    2084
  Arg     5      3      193    1640
  Asn     3      6      153    1111
  Asp     2      7      185    1720
  Cys     6      2      35     513
  Gln     7      3      301    942
  Glu     8      8      379    3193
  Gly     12     13     104    2066
  His     2      9      84     478
  Ile     2      0      165    2062
  Leu     20     18     438    2117
  Lys     2      11     282    2943
  Met     2      2      79     398
  Phe     3      8      77     908
  Pro     6      7      130    2517
  Ser     5      5      239    2463
  Thr     3      7      194    2546
  Trp     2      2      67     466
  Tyr     4      3      61     999
  Val     6      18     186    3184

Processing in Groups

  ...
  efetch -format acc |
  join-into-groups-of 200 |
  xargs -n 1 sh -c 'epost -db nuccore -format acc -id "$0" |
  efetch -format gb'

Phrase Indexing

  efetch -db pubmed -id 12857958,2981625 -format xml |
  xtract -head "<IdxDocumentSet>" -tail "</IdxDocumentSet>" \
    -hd "  <IdxDocument>\n" -tl "  </IdxDocument>" \
    -pattern PubmedArticle \
      -pfx "    <IdxUid>" -sfx "</IdxUid>\n" \
      -element MedlineCitation/PMID \
      -clr -rst -tab "" \
      -lbl "    <IdxSearchFields>\n" \
      -indices ArticleTitle,AbstractText,Keyword \
      -clr -lbl "    </IdxSearchFields>\n" |
  xtract -pattern IdxDocument -UID IdxUid \
    -block NORM -pfc "\n" -element "&UID",NORM \
    -block PAIR -pfc "\n" -element "&UID",PAIR

  12857958    allow
  12857958    assays
  12857958    binding
  12857958    braid
  12857958    braiding
  12857958    carlo
  12857958    catenane
  12857958    chiral
  12857958    chirality
  ...
  12857958    type
  12857958    underlying
  12857958    writhe
  12857958    allow topo
  12857958    binding assays
  12857958    braid relaxation
  12857958    braid supercoil
  12857958    braiding system
  12857958    carlo simulations
  ...

Phrase Searching

  PhraseSearch() {
    entrez-phrase-search -db pubmed -field WORD "$@" |
    efetch -format xml |
    xtract -phrase "$*" \
      -head "<PubmedArticleSet>" -tail "</PubmedArticleSet>" -pattern PubmedArticle
  }

  PhraseSearch selective serotonin reuptake inhibitor + monoamine oxidase inhibitor |
  xtract -pattern PubmedArticle -element MedlineCitation/PMID \
    -block Keyword -pfc "\n  " -element Keyword

  24657329
    Antidepressant
    Organic cation transporter 2
    Piperine
    Uptake 2
  24280122
    5-HIAA
    5-HT
    5-HTP
    5-hydroxyindoleacetic acid
    5-hydroxytryptophan
    ...
`

const pubMedArtSample = `
<PubmedArticle>
<MedlineCitation Status="MEDLINE" Owner="NLM">
<PMID Version="1">6301692</PMID>
<DateCreated>
<Year>1983</Year>
<Month>06</Month>
<Day>17</Day>
</DateCreated>
<DateCompleted>
<Year>1983</Year>
<Month>06</Month>
<Day>17</Day>
</DateCompleted>
<DateRevised>
<Year>2007</Year>
<Month>11</Month>
<Day>14</Day>
</DateRevised>
<Article PubModel="Print">
<Journal>
<ISSN IssnType="Print">0092-8674</ISSN>
<JournalIssue CitedMedium="Print">
<Volume>32</Volume>
<Issue>4</Issue>
<PubDate>
<Year>1983</Year>
<Month>Apr</Month>
</PubDate>
</JournalIssue>
<Title>Cell</Title>
<ISOAbbreviation>Cell</ISOAbbreviation>
</Journal>
<ArticleTitle>Site-specific relaxation and recombination by the Tn3 resolvase: recognition of the DNA path between oriented res sites.</ArticleTitle>
<Pagination>
<MedlinePgn>1313-24</MedlinePgn>
</Pagination>
<Abstract>
<AbstractText Label="RESULTS>We studied the dynamics of site-specific recombination by the resolvase encoded by the Escherichia coli transposon Tn3.
The pure enzyme recombined supercoiled plasmids containing two directly repeated recombination sites, called res sites.
Resolvase is the first strictly site-specific topoisomerase.
It relaxed only plasmids containing directly repeated res sites; substrates with zero, one or two inverted sites were inert.
Even when the proximity of res sites was ensured by catenation of plasmids with a single site, neither relaxation nor recombination occurred.
The two circular products of recombination were catenanes interlinked only once.
These properties of resolvase require that the path of the DNA between res sites be clearly defined and that strand exchange occur with a unique geometry.</AbstractText>
<AbstractText Label="SUMMARY">A model in which one subunit of a dimeric resolvase is bound at one res site,
while the other searches along adjacent DNA until it encounters the second site,
would account for the ability of resolvase to distinguish intramolecular from intermolecular sites,
to sense the relative orientation of sites and to produce singly interlinked catenanes.
Because resolvase is a type 1 topoisomerase, we infer that it makes the required duplex bDNA breaks of recombination one strand at a time.</AbstractText>
</Abstract>
<AuthorList CompleteYN="Y">
<Author ValidYN="Y">
<LastName>Krasnow</LastName>
<ForeName>Mark A</ForeName>
<Initials>MA</Initials>
</Author>
<Author ValidYN="Y">
<LastName>Cozzarelli</LastName>
<ForeName>Nicholas R</ForeName>
<Initials>NR</Initials>
</Author>
</AuthorList>
<Language>eng</Language>
<GrantList CompleteYN="Y">
<Grant>
<GrantID>GM-07281</GrantID>
<Acronym>GM</Acronym>
<Agency>NIGMS NIH HHS</Agency>
<Country>United States</Country>
</Grant>
</GrantList>
<PublicationTypeList>
<PublicationType UI="D016428">Journal Article</PublicationType>
<PublicationType UI="D013487">Research Support, U.S. Gov't, P.H.S.</PublicationType>
</PublicationTypeList>
</Article>
<MedlineJournalInfo>
<Country>United States</Country>
<MedlineTA>Cell</MedlineTA>
<NlmUniqueID>0413066</NlmUniqueID>
<ISSNLinking>0092-8674</ISSNLinking>
</MedlineJournalInfo>
<ChemicalList>
<Chemical>
<RegistryNumber>0</RegistryNumber>
<NameOfSubstance UI="D004269">DNA, Bacterial</NameOfSubstance>
</Chemical>
<Chemical>
<RegistryNumber>0</RegistryNumber>
<NameOfSubstance UI="D004278">DNA, Superhelical</NameOfSubstance>
</Chemical>
<Chemical>
<RegistryNumber>0</RegistryNumber>
<NameOfSubstance UI="D004279">DNA, Viral</NameOfSubstance>
</Chemical>
<Chemical>
<RegistryNumber>EC 2.7.7.-</RegistryNumber>
<NameOfSubstance UI="D009713">Nucleotidyltransferases</NameOfSubstance>
</Chemical>
<Chemical>
<RegistryNumber>EC 2.7.7.-</RegistryNumber>
<NameOfSubstance UI="D019895">Transposases</NameOfSubstance>
</Chemical>
<Chemical>
<RegistryNumber>EC 5.99.1.2</RegistryNumber>
<NameOfSubstance UI="D004264">DNA Topoisomerases, Type I</NameOfSubstance>
</Chemical>
</ChemicalList>
<CitationSubset>IM</CitationSubset>
<MeshHeadingList>
<MeshHeading>
<DescriptorName UI="D004264" MajorTopicYN="N">DNA Topoisomerases, Type I</DescriptorName>
<QualifierName UI="Q000378" MajorTopicYN="N">metabolism</QualifierName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D004269" MajorTopicYN="N">DNA, Bacterial</DescriptorName>
<QualifierName UI="Q000378" MajorTopicYN="Y">metabolism</QualifierName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D004278" MajorTopicYN="N">DNA, Superhelical</DescriptorName>
<QualifierName UI="Q000378" MajorTopicYN="N">metabolism</QualifierName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D004279" MajorTopicYN="N">DNA, Viral</DescriptorName>
<QualifierName UI="Q000378" MajorTopicYN="Y">metabolism</QualifierName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D008957" MajorTopicYN="N">Models, Genetic</DescriptorName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D009690" MajorTopicYN="Y">Nucleic Acid Conformation</DescriptorName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D009713" MajorTopicYN="N">Nucleotidyltransferases</DescriptorName>
<QualifierName UI="Q000302" MajorTopicYN="N">isolation &amp; purification</QualifierName>
<QualifierName UI="Q000378" MajorTopicYN="Y">metabolism</QualifierName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D010957" MajorTopicYN="N">Plasmids</DescriptorName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D011995" MajorTopicYN="Y">Recombination, Genetic</DescriptorName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D012091" MajorTopicYN="N">Repetitive Sequences, Nucleic Acid</DescriptorName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D013539" MajorTopicYN="N">Simian virus 40</DescriptorName>
</MeshHeading>
<MeshHeading>
<DescriptorName UI="D019895" MajorTopicYN="N">Transposases</DescriptorName>
</MeshHeading>
</MeshHeadingList>
</MedlineCitation>
<PubmedData>
<History>
<PubMedPubDate PubStatus="pubmed">
<Year>1983</Year>
<Month>4</Month>
<Day>1</Day>
</PubMedPubDate>
<PubMedPubDate PubStatus="medline">
<Year>1983</Year>
<Month>4</Month>
<Day>1</Day>
<Hour>0</Hour>
<Minute>1</Minute>
</PubMedPubDate>
<PubMedPubDate PubStatus="entrez">
<Year>1983</Year>
<Month>4</Month>
<Day>1</Day>
<Hour>0</Hour>
<Minute>0</Minute>
</PubMedPubDate>
</History>
<PublicationStatus>ppublish</PublicationStatus>
<ArticleIdList>
<ArticleId IdType="pubmed">6301692</ArticleId>
<ArticleId IdType="pii">0092-8674(83)90312-4</ArticleId>
</ArticleIdList>
</PubmedData>
</PubmedArticle>
`

const insdSeqSample = `
<INSDSeq>
<INSDSeq_locus>AF480315_1</INSDSeq_locus>
<INSDSeq_length>67</INSDSeq_length>
<INSDSeq_moltype>AA</INSDSeq_moltype>
<INSDSeq_topology>linear</INSDSeq_topology>
<INSDSeq_division>INV</INSDSeq_division>
<INSDSeq_update-date>25-JUL-2016</INSDSeq_update-date>
<INSDSeq_create-date>31-DEC-2003</INSDSeq_create-date>
<INSDSeq_definition>four-loop conotoxin preproprotein, partial [Conus purpurascens]</INSDSeq_definition>
<INSDSeq_primary-accession>AAQ05867</INSDSeq_primary-accession>
<INSDSeq_accession-version>AAQ05867.1</INSDSeq_accession-version>
<INSDSeq_other-seqids>
<INSDSeqid>gb|AAQ05867.1|AF480315_1</INSDSeqid>
<INSDSeqid>gi|33320307</INSDSeqid>
</INSDSeq_other-seqids>
<INSDSeq_source>Conus purpurascens</INSDSeq_source>
<INSDSeq_organism>Conus purpurascens</INSDSeq_organism>
<INSDSeq_taxonomy>Eukaryota; Metazoa; Lophotrochozoa; Mollusca; Gastropoda; Caenogastropoda; Hypsogastropoda; Neogastropoda; Conoidea; Conidae; Conus</INSDSeq_taxonomy>
<INSDSeq_references>
<INSDReference>
<INSDReference_reference>1</INSDReference_reference>
<INSDReference_position>1..67</INSDReference_position>
<INSDReference_authors>
<INSDAuthor>Duda,T.F. Jr.</INSDAuthor>
<INSDAuthor>Palumbi,S.R.</INSDAuthor>
</INSDReference_authors>
<INSDReference_title>Convergent evolution of venoms and feeding ecologies among polyphyletic piscivorous Conus species</INSDReference_title>
<INSDReference_journal>Unpublished</INSDReference_journal>
</INSDReference>
<INSDReference>
<INSDReference_reference>2</INSDReference_reference>
<INSDReference_position>1..67</INSDReference_position>
<INSDReference_authors>
<INSDAuthor>Duda,T.F. Jr.</INSDAuthor>
<INSDAuthor>Palumbi,S.R.</INSDAuthor>
</INSDReference_authors>
<INSDReference_title>Direct Submission</INSDReference_title>
<INSDReference_journal>Submitted (04-FEB-2002) Naos Marine Lab, Smithsonian Tropical Research Institute, Apartado 2072, Balboa, Ancon, Panama, Republic of Panama</INSDReference_journal>
</INSDReference>
</INSDSeq_references>
<INSDSeq_comment>Method: conceptual translation supplied by author.</INSDSeq_comment>
<INSDSeq_source-db>accession AF480315.1</INSDSeq_source-db>
<INSDSeq_feature-table>
<INSDFeature>
<INSDFeature_key>source</INSDFeature_key>
<INSDFeature_location>1..67</INSDFeature_location>
<INSDFeature_intervals>
<INSDInterval>
<INSDInterval_from>1</INSDInterval_from>
<INSDInterval_to>67</INSDInterval_to>
<INSDInterval_accession>AAQ05867.1</INSDInterval_accession>
</INSDInterval>
</INSDFeature_intervals>
<INSDFeature_quals>
<INSDQualifier>
<INSDQualifier_name>organism</INSDQualifier_name>
<INSDQualifier_value>Conus purpurascens</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>isolate</INSDQualifier_name>
<INSDQualifier_value>purpurascens-2c</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>db_xref</INSDQualifier_name>
<INSDQualifier_value>taxon:41690</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>clone_lib</INSDQualifier_name>
<INSDQualifier_value>venom duct cDNA library</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>country</INSDQualifier_name>
<INSDQualifier_value>Panama</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>note</INSDQualifier_name>
<INSDQualifier_value>isolated from the Bay of Panama</INSDQualifier_value>
</INSDQualifier>
</INSDFeature_quals>
</INSDFeature>
<INSDFeature>
<INSDFeature_key>Protein</INSDFeature_key>
<INSDFeature_location>&lt;1..67</INSDFeature_location>
<INSDFeature_intervals>
<INSDInterval>
<INSDInterval_from>1</INSDInterval_from>
<INSDInterval_to>67</INSDInterval_to>
<INSDInterval_accession>AAQ05867.1</INSDInterval_accession>
</INSDInterval>
</INSDFeature_intervals>
<INSDFeature_partial5 value="true"/>
<INSDFeature_quals>
<INSDQualifier>
<INSDQualifier_name>product</INSDQualifier_name>
<INSDQualifier_value>four-loop conotoxin preproprotein</INSDQualifier_value>
</INSDQualifier>
</INSDFeature_quals>
</INSDFeature>
<INSDFeature>
<INSDFeature_key>mat_peptide</INSDFeature_key>
<INSDFeature_location>41..67</INSDFeature_location>
<INSDFeature_intervals>
<INSDInterval>
<INSDInterval_from>41</INSDInterval_from>
<INSDInterval_to>67</INSDInterval_to>
<INSDInterval_accession>AAQ05867.1</INSDInterval_accession>
</INSDInterval>
</INSDFeature_intervals>
<INSDFeature_quals>
<INSDQualifier>
<INSDQualifier_name>product</INSDQualifier_name>
<INSDQualifier_value>four-loop conotoxin</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>calculated_mol_wt</INSDQualifier_name>
<INSDQualifier_value>3008</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>peptide</INSDQualifier_name>
<INSDQualifier_value>PCKKTGRKCFPHQKDCCGRACIITICP</INSDQualifier_value>
</INSDQualifier>
</INSDFeature_quals>
</INSDFeature>
<INSDFeature>
<INSDFeature_key>CDS</INSDFeature_key>
<INSDFeature_location>1..67</INSDFeature_location>
<INSDFeature_intervals>
<INSDInterval>
<INSDInterval_from>1</INSDInterval_from>
<INSDInterval_to>67</INSDInterval_to>
<INSDInterval_accession>AAQ05867.1</INSDInterval_accession>
</INSDInterval>
</INSDFeature_intervals>
<INSDFeature_partial5 value="true"/>
<INSDFeature_quals>
<INSDQualifier>
<INSDQualifier_name>coded_by</INSDQualifier_name>
<INSDQualifier_value>AF480315.1:&lt;1..205</INSDQualifier_value>
</INSDQualifier>
<INSDQualifier>
<INSDQualifier_name>codon_start</INSDQualifier_name>
<INSDQualifier_value>2</INSDQualifier_value>
</INSDQualifier>
</INSDFeature_quals>
</INSDFeature>
</INSDSeq_feature-table>
<INSDSeq_sequence>vvivavlfltacqlitaddsrrtqkhralrsttkratsnrpckktgrkcfphqkdccgraciiticp</INSDSeq_sequence>
</INSDSeq>
`

const geneDocSumSample = `
<DocumentSummary>
<Id>3581</Id>
<Name>IL9R</Name>
<Description>interleukin 9 receptor</Description>
<Status>0</Status>
<CurrentID>0</CurrentID>
<Chromosome>X, Y</Chromosome>
<GeneticSource>genomic</GeneticSource>
<MapLocation>Xq28 and Yq12</MapLocation>
<OtherAliases>CD129, IL-9R</OtherAliases>
<OtherDesignations>interleukin-9 receptor|IL-9 receptor</OtherDesignations>
<NomenclatureSymbol>IL9R</NomenclatureSymbol>
<NomenclatureName>interleukin 9 receptor</NomenclatureName>
<NomenclatureStatus>Official</NomenclatureStatus>
<Mim>
<int>300007</int>
</Mim>
<GenomicInfo>
<GenomicInfoType>
<ChrLoc>X</ChrLoc>
<ChrAccVer>NC_000023.11</ChrAccVer>
<ChrStart>155997580</ChrStart>
<ChrStop>156013016</ChrStop>
<ExonCount>14</ExonCount>
</GenomicInfoType>
<GenomicInfoType>
<ChrLoc>Y</ChrLoc>
<ChrAccVer>NC_000024.10</ChrAccVer>
<ChrStart>57184100</ChrStart>
<ChrStop>57199536</ChrStop>
<ExonCount>14</ExonCount>
</GenomicInfoType>
</GenomicInfo>
<GeneWeight>5425</GeneWeight>
<Summary>The protein encoded by this gene is a cytokine receptor that specifically mediates the biological effects of interleukin 9 (IL9).
The functional IL9 receptor complex requires this protein as well as the interleukin 2 receptor, gamma (IL2RG), a common gamma subunit shared by the receptors of many different cytokines.
The ligand binding of this receptor leads to the activation of various JAK kinases and STAT proteins, which connect to different biologic responses.
This gene is located at the pseudoautosomal regions of X and Y chromosomes.
Genetic studies suggested an association of this gene with the development of asthma.
Multiple pseudogenes on chromosome 9, 10, 16, and 18 have been described.
Alternatively spliced transcript variants have been found for this gene.</Summary>
<ChrSort>X</ChrSort>
<ChrStart>155997580</ChrStart>
<Organism>
<ScientificName>Homo sapiens</ScientificName>
<CommonName>human</CommonName>
<TaxID>9606</TaxID>
</Organism>
</DocumentSummary>
`

const keyboardShortcuts = `
Command History

  Ctrl-n     Next command
  Ctrl-p     Previous command

Move Cursor Forward

  Ctrl-e     To end of line
  Ctrl-f     By one character
  Esc-f      By one argument

Move Cursor Backward

  Ctrl-a     To beginning of line
  Ctrl-b     By one character
  Esc-b      By one argument

Delete

  Del        Previous character
  Ctrl-d     Next character
  Ctrl-k     To end of line
  Ctrl-u     Entire line
  Ctrl-w     Previous word
  Esc-Del    Previous argument
  Esc-d      Next argument

Autocomplete

  Tab        Completes directory or file names

Program Control

  Ctrl-c     Quit running program
  ^x^y       Run last command replacing x with y
`

const unixCommands = `
Process by Contents

 sort      Sorts lines of text

  -f       Ignore case
  -n       Numeric comparison
  -r       Reverse result order

  -k       Field key (start,stop or first)
  -u       Unique lines with identical keys

  -b       Ignore leading blanks
  -s       Stable sort
  -t       Specify field separator

 uniq      Removes repeated lines

  -c       Count occurrences
  -i       Ignore case

  -f       Ignore first n fields
  -s       Ignore first n characters

  -d       Only output repeated lines
  -u       Only output non-repeated lines

 grep      Matches patterns using regular expressions

  -i       Ignore case
  -v       Invert search
  -w       Search expression as a word
  -x       Search expression as whole line

  -e       Specify individual pattern

  -c       Only count number of matches
  -n       Print line numbers

Regular Expressions

 Characters

  .        Any single character (except newline)
  \w       Alphabetic [A-Za-z], numeric [0-9], or underscore (_)
  \s       Whitespace (space or tab)
  \        Escapes special characters
  []       Matches any enclosed characters

 Positions

  ^        Beginning of line
  $        End of line
  \b       Word boundary

 Repeat Matches

  ?        0 or 1
  *        0 or more
  +        1 or more
  {n}      Exactly n

 Escape Sequences

  \n       Line break
  \t       Tab character

Modify Contents

 sed       Replaces text strings

  -e       Specify individual expression

 tr        Translates characters

  -d       Delete character

 rev       Reverses characters on line

Format Contents

 column    Aligns columns by content width

  -s       Specify field separator
  -t       Create table

 expand    Aligns columns to specified positions

  -t       Tab positions

 fold      Wraps lines at a specific width

  -w       Line width

Filter by Position

 cut       Removes parts of lines

  -c       Characters to keep
  -f       Fields to keep
  -d       Specify field separator
  -s       Suppress lines with no delimiters

 head      Prints first lines

  -n       Number of lines

 tail      Prints last lines

  -n       Number of lines

Miscellaneous

 wc        Counts words, lines, or characters

  -c       Characters
  -l       Lines
  -w       Words

 xargs     Constructs arguments

  -n       Number of words per batch

File Compression

 tar       Archive files

  -c       Create archive
  -f       Name of output file
  -z       Compress archive with gzip

 gzip      Compress file

  -k       Keep original file
  -9       Best compression

 unzip     Decompress .zip archive

  -p       Pipe to stdout

 gzcat     Decompress .gz archive and pipe to stdout

Directory and File Navigation

 cd        Changes directory

  /        Root
  ~        Home
  .        Current
  ..       Parent
  -        Previous

 ls        Lists file names

  -1       One entry per line
  -a       Show files beginning with dot (.)
  -l       List in long format
  -R       Recursively explore subdirectories
  -S       Sort files by size
  -t       Sort by most recently modified

 pwd       Prints working directory path
 
File Redirection

  <        Read stdin from file
  >        Redirect stdout to file
  >>       Append to file
  2>       Redirect stderr
  2>&1     Merge stderr into stdout
  |        Pipe between programs
  <(cmd)   Execute command, read results as file
 
Shell Script Variables

  $0       Name of script
  $n       Nth argument
  $#       Number of arguments
  "$*"     Argument list as one argument
  "$@"     Argument list as separate arguments
  $?       Exit status of previous command
 
Shell Script Tests

  -d       Directory exists
  -f       File exists
  -s       File is not empty
  -n       Length of string is non-zero
  -z       Variable is empty or not set
`

// TYPED CONSTANTS

type LevelType int

const (
	_ LevelType = iota
	UNIT
	SUBSET
	SECTION
	BLOCK
	BRANCH
	GROUP
	DIVISION
	PATTERN
)

type IndentType int

const (
	SINGULARITY IndentType = iota
	COMPACT
	FLUSH
	INDENT
	SUBTREE
	WRAPPED
)

type SideType int

const (
	_ SideType = iota
	LEFT
	RIGHT
)

type TagType int

const (
	NOTAG TagType = iota
	STARTTAG
	SELFTAG
	STOPTAG
	ATTRIBTAG
	CONTENTTAG
	CDATATAG
	COMMENTTAG
	DOCTYPETAG
	OBJECTTAG
	CONTAINERTAG
	ISCLOSED
)

type OpType int

const (
	UNSET OpType = iota
	ELEMENT
	FIRST
	LAST
	ENCODE
	UPPER
	LOWER
	TITLE
	TERMS
	WORDS
	PAIRS
	LETTERS
	INDICES
	PFX
	SFX
	SEP
	TAB
	RET
	LBL
	CLR
	PFC
	RST
	DEF
	POSITION
	IF
	UNLESS
	MATCH
	AVOID
	AND
	OR
	EQUALS
	CONTAINS
	STARTSWITH
	ENDSWITH
	ISNOT
	GT
	GE
	LT
	LE
	EQ
	NE
	NUM
	LEN
	SUM
	MIN
	MAX
	INC
	DEC
	SUB
	AVG
	DEV
	ZEROBASED
	ONEBASED
	UCSCBASED
	ELSE
	VARIABLE
	VALUE
	STAR
	DOLLAR
	ATSIGN
	COUNT
	LENGTH
	DEPTH
	INDEX
	UNRECOGNIZED
)

type ArgumentType int

const (
	_ ArgumentType = iota
	EXPLORATION
	CONDITIONAL
	EXTRACTION
	CUSTOMIZATION
)

type SpecialType int

const (
	NOPROCESS SpecialType = iota
	DOFORMAT
	DOOUTLINE
	DOSYNOPSIS
	DOVERIFY
	DOFILTER
	DOQUERY
	DOINDEX
)

type SeqEndType int

const (
	_ SeqEndType = iota
	ISSTART
	ISSTOP
	ISPOS
)

type SequenceType struct {
	Based int
	Which SeqEndType
}

// ARGUMENT MAPS

var markupRunes = map[rune]rune{
	'\u00B2': '2',
	'\u00B3': '3',
	'\u00B9': '1',
	'\u2070': '0',
	'\u2071': '1',
	'\u2074': '4',
	'\u2075': '5',
	'\u2076': '6',
	'\u2077': '7',
	'\u2078': '8',
	'\u2079': '9',
	'\u207A': '+',
	'\u207B': '-',
	'\u207C': '=',
	'\u207D': '(',
	'\u207E': ')',
	'\u207F': 'n',
	'\u2080': '0',
	'\u2081': '1',
	'\u2082': '2',
	'\u2083': '3',
	'\u2084': '4',
	'\u2085': '5',
	'\u2086': '6',
	'\u2087': '7',
	'\u2088': '8',
	'\u2089': '9',
	'\u208A': '+',
	'\u208B': '-',
	'\u208C': '=',
	'\u208D': '(',
	'\u208E': ')',
}

var accentRunes = map[rune]rune{
	'\u00D8': 'O',
	'\u00F0': 'd',
	'\u00F8': 'o',
	'\u0111': 'd',
	'\u0131': 'i',
	'\u0141': 'L',
	'\u0142': 'l',
	'\u02BC': '\'',
}

var ligatureRunes = map[rune]string{
	'\u00DF': "ss",
	'\u00E6': "ae",
	'\uFB00': "ff",
	'\uFB01': "fi",
	'\uFB02': "fl",
	'\uFB03': "ffi",
	'\uFB04': "ffl",
	'\uFB05': "ft",
	'\uFB06': "st",
}

var argTypeIs = map[string]ArgumentType{
	"-unit":        EXPLORATION,
	"-Unit":        EXPLORATION,
	"-subset":      EXPLORATION,
	"-Subset":      EXPLORATION,
	"-section":     EXPLORATION,
	"-Section":     EXPLORATION,
	"-block":       EXPLORATION,
	"-Block":       EXPLORATION,
	"-branch":      EXPLORATION,
	"-Branch":      EXPLORATION,
	"-group":       EXPLORATION,
	"-Group":       EXPLORATION,
	"-division":    EXPLORATION,
	"-Division":    EXPLORATION,
	"-pattern":     EXPLORATION,
	"-Pattern":     EXPLORATION,
	"-position":    CONDITIONAL,
	"-if":          CONDITIONAL,
	"-unless":      CONDITIONAL,
	"-match":       CONDITIONAL,
	"-avoid":       CONDITIONAL,
	"-and":         CONDITIONAL,
	"-or":          CONDITIONAL,
	"-equals":      CONDITIONAL,
	"-contains":    CONDITIONAL,
	"-starts-with": CONDITIONAL,
	"-ends-with":   CONDITIONAL,
	"-is-not":      CONDITIONAL,
	"-gt":          CONDITIONAL,
	"-ge":          CONDITIONAL,
	"-lt":          CONDITIONAL,
	"-le":          CONDITIONAL,
	"-eq":          CONDITIONAL,
	"-ne":          CONDITIONAL,
	"-element":     EXTRACTION,
	"-first":       EXTRACTION,
	"-last":        EXTRACTION,
	"-encode":      EXTRACTION,
	"-upper":       EXTRACTION,
	"-lower":       EXTRACTION,
	"-title":       EXTRACTION,
	"-terms":       EXTRACTION,
	"-words":       EXTRACTION,
	"-pairs":       EXTRACTION,
	"-letters":     EXTRACTION,
	"-indices":     EXTRACTION,
	"-num":         EXTRACTION,
	"-len":         EXTRACTION,
	"-sum":         EXTRACTION,
	"-min":         EXTRACTION,
	"-max":         EXTRACTION,
	"-inc":         EXTRACTION,
	"-dec":         EXTRACTION,
	"-sub":         EXTRACTION,
	"-avg":         EXTRACTION,
	"-dev":         EXTRACTION,
	"-0-based":     EXTRACTION,
	"-zero-based":  EXTRACTION,
	"-1-based":     EXTRACTION,
	"-one-based":   EXTRACTION,
	"-ucsc":        EXTRACTION,
	"-ucsc-based":  EXTRACTION,
	"-ucsc-coords": EXTRACTION,
	"-bed-based":   EXTRACTION,
	"-bed-coords":  EXTRACTION,
	"-else":        EXTRACTION,
	"-pfx":         CUSTOMIZATION,
	"-sfx":         CUSTOMIZATION,
	"-sep":         CUSTOMIZATION,
	"-tab":         CUSTOMIZATION,
	"-ret":         CUSTOMIZATION,
	"-lbl":         CUSTOMIZATION,
	"-clr":         CUSTOMIZATION,
	"-pfc":         CUSTOMIZATION,
	"-rst":         CUSTOMIZATION,
	"-def":         CUSTOMIZATION,
}

var opTypeIs = map[string]OpType{
	"-element":     ELEMENT,
	"-first":       FIRST,
	"-last":        LAST,
	"-encode":      ENCODE,
	"-upper":       UPPER,
	"-lower":       LOWER,
	"-title":       TITLE,
	"-terms":       TERMS,
	"-words":       WORDS,
	"-pairs":       PAIRS,
	"-letters":     LETTERS,
	"-indices":     INDICES,
	"-pfx":         PFX,
	"-sfx":         SFX,
	"-sep":         SEP,
	"-tab":         TAB,
	"-ret":         RET,
	"-lbl":         LBL,
	"-clr":         CLR,
	"-pfc":         PFC,
	"-rst":         RST,
	"-def":         DEF,
	"-position":    POSITION,
	"-if":          IF,
	"-unless":      UNLESS,
	"-match":       MATCH,
	"-avoid":       AVOID,
	"-and":         AND,
	"-or":          OR,
	"-equals":      EQUALS,
	"-contains":    CONTAINS,
	"-starts-with": STARTSWITH,
	"-ends-with":   ENDSWITH,
	"-is-not":      ISNOT,
	"-gt":          GT,
	"-ge":          GE,
	"-lt":          LT,
	"-le":          LE,
	"-eq":          EQ,
	"-ne":          NE,
	"-num":         NUM,
	"-len":         LEN,
	"-sum":         SUM,
	"-min":         MIN,
	"-max":         MAX,
	"-inc":         INC,
	"-dec":         DEC,
	"-sub":         SUB,
	"-avg":         AVG,
	"-dev":         DEV,
	"-0-based":     ZEROBASED,
	"-zero-based":  ZEROBASED,
	"-1-based":     ONEBASED,
	"-one-based":   ONEBASED,
	"-ucsc":        UCSCBASED,
	"-ucsc-based":  UCSCBASED,
	"-ucsc-coords": UCSCBASED,
	"-bed-based":   UCSCBASED,
	"-bed-coords":  UCSCBASED,
	"-else":        ELSE,
}

var levelTypeIs = map[string]LevelType{
	"-unit":     UNIT,
	"-Unit":     UNIT,
	"-subset":   SUBSET,
	"-Subset":   SUBSET,
	"-section":  SECTION,
	"-Section":  SECTION,
	"-block":    BLOCK,
	"-Block":    BLOCK,
	"-branch":   BRANCH,
	"-Branch":   BRANCH,
	"-group":    GROUP,
	"-Group":    GROUP,
	"-division": DIVISION,
	"-Division": DIVISION,
	"-pattern":  PATTERN,
	"-Pattern":  PATTERN,
}

var slock sync.RWMutex

var sequenceTypeIs = map[string]SequenceType{
	"INSDSeq:INSDInterval_from":       {1, ISSTART},
	"INSDSeq:INSDInterval_to":         {1, ISSTOP},
	"DocumentSummary:ChrStart":        {0, ISSTART},
	"DocumentSummary:ChrStop":         {0, ISSTOP},
	"DocumentSummary:Chr_start":       {1, ISSTART},
	"DocumentSummary:Chr_end":         {1, ISSTOP},
	"DocumentSummary:Chr_inner_start": {1, ISSTART},
	"DocumentSummary:Chr_inner_end":   {1, ISSTOP},
	"DocumentSummary:Chr_outer_start": {1, ISSTART},
	"DocumentSummary:Chr_outer_end":   {1, ISSTOP},
	"DocumentSummary:start":           {1, ISSTART},
	"DocumentSummary:stop":            {1, ISSTOP},
	"DocumentSummary:display_start":   {1, ISSTART},
	"DocumentSummary:display_stop":    {1, ISSTOP},
	"Entrezgene:Seq-interval_from":    {0, ISSTART},
	"Entrezgene:Seq-interval_to":      {0, ISSTOP},
	"GenomicInfoType:ChrStart":        {0, ISSTART},
	"GenomicInfoType:ChrStop":         {0, ISSTOP},
	"Rs:@aaPosition":                  {0, ISPOS},
	"Rs:@asnFrom":                     {0, ISSTART},
	"Rs:@asnTo":                       {0, ISSTOP},
	"Rs:@end":                         {0, ISSTOP},
	"Rs:@leftContigNeighborPos":       {0, ISSTART},
	"Rs:@physMapInt":                  {0, ISPOS},
	"Rs:@protLoc":                     {0, ISPOS},
	"Rs:@rightContigNeighborPos":      {0, ISSTOP},
	"Rs:@start":                       {0, ISSTART},
	"Rs:@structLoc":                   {0, ISPOS},
}

var plock sync.RWMutex

var isStopWord = map[string]bool{
	"!":             true,
	"\"":            true,
	"#":             true,
	"$":             true,
	"%":             true,
	"&":             true,
	"'":             true,
	"(":             true,
	")":             true,
	"*":             true,
	"+":             true,
	",":             true,
	"-":             true,
	".":             true,
	"/":             true,
	":":             true,
	";":             true,
	"<":             true,
	"=":             true,
	">":             true,
	"?":             true,
	"@":             true,
	"[":             true,
	"\\":            true,
	"]":             true,
	"^":             true,
	"_":             true,
	"`":             true,
	"{":             true,
	"|":             true,
	"}":             true,
	"~":             true,
	"a":             true,
	"about":         true,
	"again":         true,
	"all":           true,
	"almost":        true,
	"also":          true,
	"although":      true,
	"always":        true,
	"among":         true,
	"an":            true,
	"and":           true,
	"another":       true,
	"any":           true,
	"are":           true,
	"as":            true,
	"at":            true,
	"be":            true,
	"because":       true,
	"been":          true,
	"before":        true,
	"being":         true,
	"between":       true,
	"both":          true,
	"but":           true,
	"by":            true,
	"can":           true,
	"could":         true,
	"did":           true,
	"do":            true,
	"does":          true,
	"done":          true,
	"due":           true,
	"during":        true,
	"each":          true,
	"either":        true,
	"enough":        true,
	"especially":    true,
	"etc":           true,
	"for":           true,
	"found":         true,
	"from":          true,
	"further":       true,
	"had":           true,
	"has":           true,
	"have":          true,
	"having":        true,
	"here":          true,
	"how":           true,
	"however":       true,
	"i":             true,
	"if":            true,
	"in":            true,
	"into":          true,
	"is":            true,
	"it":            true,
	"its":           true,
	"itself":        true,
	"just":          true,
	"kg":            true,
	"km":            true,
	"made":          true,
	"mainly":        true,
	"make":          true,
	"may":           true,
	"mg":            true,
	"might":         true,
	"ml":            true,
	"mm":            true,
	"most":          true,
	"mostly":        true,
	"must":          true,
	"nearly":        true,
	"neither":       true,
	"no":            true,
	"nor":           true,
	"obtained":      true,
	"of":            true,
	"often":         true,
	"on":            true,
	"our":           true,
	"overall":       true,
	"perhaps":       true,
	"pmid":          true,
	"quite":         true,
	"rather":        true,
	"really":        true,
	"regarding":     true,
	"seem":          true,
	"seen":          true,
	"several":       true,
	"should":        true,
	"show":          true,
	"showed":        true,
	"shown":         true,
	"shows":         true,
	"significantly": true,
	"since":         true,
	"so":            true,
	"some":          true,
	"such":          true,
	"than":          true,
	"that":          true,
	"the":           true,
	"their":         true,
	"theirs":        true,
	"them":          true,
	"then":          true,
	"there":         true,
	"therefore":     true,
	"these":         true,
	"they":          true,
	"this":          true,
	"those":         true,
	"through":       true,
	"thus":          true,
	"to":            true,
	"upon":          true,
	"use":           true,
	"used":          true,
	"using":         true,
	"various":       true,
	"very":          true,
	"was":           true,
	"we":            true,
	"were":          true,
	"what":          true,
	"when":          true,
	"which":         true,
	"while":         true,
	"with":          true,
	"within":        true,
	"without":       true,
	"would":         true,
}

// DATA OBJECTS

type Tables struct {
	InBlank   [256]bool
	AltBlank  [256]bool
	InFirst   [256]bool
	InElement [256]bool
	ChanDepth int
	FarmSize  int
	HeapSize  int
	NumServe  int
	Index     string
	Parent    string
	Match     string
	Attrib    string
	Stash     string
	Posting   string
	Zipp      bool
	Hash      bool
	Hd        string
	Tl        string
	DoStrict  bool
	DoMixed   bool
	DeAccent  bool
	DoASCII   bool
}

type Node struct {
	Name       string
	Parent     string
	Contents   string
	Attributes string
	Attribs    []string
	Children   *Node
	Next       *Node
}

type Step struct {
	Type   OpType
	Value  string
	Parent string
	Match  string
	Attrib string
	Wild   bool
}

type Operation struct {
	Type   OpType
	Value  string
	Stages []*Step
}

type Block struct {
	Visit      string
	Parent     string
	Match      string
	Working    []string
	Parsed     []string
	Position   string
	Conditions []*Operation
	Commands   []*Operation
	Failure    []*Operation
	Subtasks   []*Block
}

// UTILITIES

func IsNotJustWhitespace(str string) bool {

	for _, ch := range str {
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' && ch != '\f' {
			return true
		}
	}

	return false
}

func IsNotASCII(str string) bool {

	for _, ch := range str {
		if ch > 127 {
			return true
		}
	}

	return false
}

func HasAmpOrNotASCII(str string) bool {

	for _, ch := range str {
		if ch == '&' || ch > 127 {
			return true
		}
	}

	return false
}

func IsAllCapsOrDigits(str string) bool {

	for _, ch := range str {
		if !unicode.IsUpper(ch) && !unicode.IsDigit(ch) {
			return false
		}
	}

	return true
}

func IsAllNumeric(str string) bool {

	for _, ch := range str {
		if !unicode.IsDigit(ch) &&
			ch != '.' &&
			ch != '+' &&
			ch != '-' &&
			ch != '*' &&
			ch != '/' &&
			ch != ',' &&
			ch != '$' &&
			ch != '#' &&
			ch != '%' &&
			ch != '(' &&
			ch != ')' {
			return false
		}
	}

	return true
}

func HasAngleBracket(str string) bool {

	hasAmp := false
	hasSemi := false

	for _, ch := range str {
		if ch == '<' || ch == '>' {
			return true
		} else if ch == '&' {
			hasAmp = true
		} else if ch == ';' {
			hasSemi = true
		}
	}

	if hasAmp && hasSemi {
		if strings.Contains(str, "&lt;") || strings.Contains(str, "&gt;") || strings.Contains(str, "&amp;") {
			return true
		}
	}

	return false
}

func CompressRunsOfSpaces(str string) string {

	whiteSpace := false
	var buffer bytes.Buffer

	for _, ch := range str {
		if unicode.IsSpace(ch) {
			if !whiteSpace {
				buffer.WriteRune(' ')
			}
			whiteSpace = true
		} else {
			buffer.WriteRune(ch)
			whiteSpace = false
		}
	}

	return buffer.String()
}

func HasFlankingSpace(str string) bool {

	if str == "" {
		return false
	}

	ch := str[0]
	if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' {
		return true
	}

	strlen := len(str)
	ch = str[strlen-1]
	if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' {
		return true
	}

	return false
}

func HasBadSpace(str string) bool {

	for _, ch := range str {
		if unicode.IsSpace(ch) && ch != ' ' {
			return true
		}
	}

	return false
}

func CleanupBadSpaces(str string) string {

	var buffer bytes.Buffer

	for _, ch := range str {
		if unicode.IsSpace(ch) {
			buffer.WriteRune(' ')
		} else {
			buffer.WriteRune(ch)
		}
	}

	return buffer.String()
}

func TrimPunctuation(str string) string {

	max := len(str)

	doOneTrim := func() {

		if max > 0 {
			ch := str[0]
			if ch == '.' ||
				ch == ',' ||
				ch == ':' ||
				ch == ';' ||
				ch == '=' ||
				ch == '\'' ||
				ch == '"' ||
				ch == ')' ||
				ch == ']' {
				// trim leading punctuation
				str = str[1:]
				max--
			}
		}

		if max > 0 {
			ch := str[max-1]
			if ch == '.' ||
				ch == ',' ||
				ch == ':' ||
				ch == ';' ||
				ch == '=' ||
				ch == '\'' ||
				ch == '"' ||
				ch == '(' ||
				ch == '[' {
				// trim trailing punctuation
				str = str[:max-1]
				max--
			}
		}

		if max > 1 && str[0] == '(' && str[max-1] == ')' {
			// trim flanking parentheses
			str = str[1 : max-1]
			max -= 2
		}

		if max > 1 && str[0] == '[' && str[max-1] == ']' {
			// trim flanking brackets
			str = str[1 : max-1]
			max -= 2
		}

		hasLeftP := strings.Contains(str, "(")
		hasRightP := strings.Contains(str, ")")

		if max > 1 && str[0] == '(' && str[1] == '(' && !hasRightP {
			// trim leading double parentheses
			str = str[2:]
			max -= 2
		}

		if max > 1 && str[max-1] == ')' && str[max-2] == ')' && !hasLeftP {
			// trim trailing double parentheses
			str = str[:max-2]
			max -= 2
		}

		if max > 0 && str[0] == '(' && !hasRightP {
			// trim isolated left parentheses
			str = str[1:]
			max--
		}

		if max > 1 && str[max-1] == ')' && !hasLeftP {
			// trim isolated right parentheses
			str = str[:max-1]
			max--
		}

		hasLeftB := strings.Contains(str, "[")
		hasRightB := strings.Contains(str, "]")

		if max > 0 && str[0] == '[' && !hasRightB {
			// trim isolated left bracket
			str = str[1:]
			max--
		}

		if max > 1 && str[max-1] == ']' && !hasLeftB {
			// trim isolated right bracket
			str = str[:max-1]
			max--
		}
	}

	last := max + 1

	// loop until no changes
	for last > max && max > 0 {
		last = max
		doOneTrim()
	}

	return str
}

func HTMLAhead(text string, pos int) int {

	max := len(text) - pos

	if max > 2 && text[pos+2] == '>' {
		ch := text[pos+1]
		if ch == 'i' || ch == 'b' || ch == 'u' {
			return 3
		}
	} else if max > 3 && text[pos+3] == '>' {
		if text[pos+1] == '/' {
			ch := text[pos+2]
			if ch == 'i' || ch == 'b' || ch == 'u' {
				return 4
			}
		}
		if text[pos+2] == '/' {
			ch := text[pos+1]
			if ch == 'i' || ch == 'b' || ch == 'u' {
				return 4
			}
		}
	} else if max > 4 && text[pos+4] == '>' {
		if text[pos+1] == 's' && text[pos+2] == 'u' {
			ch := text[pos+3]
			if ch == 'p' || ch == 'b' {
				return 5
			}
		}
		/*
			if text[pos+3] == '/' && text[pos+2] == ' ' {
				ch := text[pos+1]
				if ch == 'i' || ch == 'b' || ch == 'u' {
					return 5
				}
			}
		*/
	} else if max > 5 && text[pos+5] == '>' {
		if text[pos+1] == '/' && text[pos+2] == 's' && text[pos+3] == 'u' {
			ch := text[pos+4]
			if ch == 'p' || ch == 'b' {
				return 6
			}
		}
		if text[pos+4] == '/' && text[pos+1] == 's' && text[pos+2] == 'u' {
			ch := text[pos+3]
			if ch == 'p' || ch == 'b' {
				return 6
			}
		}
		/*
			} else if max > 6 && text[pos+6] == '>' {
				if text[pos+5] == '/' && text[pos+4] == ' ' && text[pos+1] == 's' && text[pos+2] == 'u' {
					ch := text[pos+3]
					if ch == 'p' || ch == 'b' {
						return 7
					}
				}
		*/
	}

	return 0
}

func HTMLBehind(bufr []byte, pos int) bool {

	if pos > 1 && bufr[pos-2] == '<' {
		ch := bufr[pos-1]
		if ch == 'i' || ch == 'b' || ch == 'u' {
			return true
		}
	} else if pos > 2 && bufr[pos-3] == '<' {
		if bufr[pos-2] == '/' {
			ch := bufr[pos-1]
			if ch == 'i' || ch == 'b' || ch == 'u' {
				return true
			}
		}
		if bufr[pos-1] == '/' {
			ch := bufr[pos-2]
			if ch == 'i' || ch == 'b' || ch == 'u' {
				return true
			}
		}
	} else if pos > 3 && bufr[pos-4] == '<' {
		if bufr[pos-3] == 's' && bufr[pos-2] == 'u' {
			ch := bufr[pos-1]
			if ch == 'p' || ch == 'b' {
				return true
			}
		}
		/*
			if bufr[pos-1] == '/' && bufr[pos-2] == ' ' {
				ch := bufr[pos-3]
				if ch == 'i' || ch == 'b' || ch == 'u' {
					return true
				}
			}
		*/
	} else if pos > 4 && bufr[pos-5] == '<' {
		if bufr[pos-4] == '/' && bufr[pos-3] == 's' && bufr[pos-2] == 'u' {
			ch := bufr[pos-1]
			if ch == 'p' || ch == 'b' {
				return true
			}
		}
		if bufr[pos-1] == '/' && bufr[pos-4] == 's' && bufr[pos-3] == 'u' {
			ch := bufr[pos-2]
			if ch == 'p' || ch == 'b' {
				return true
			}
		}
		/*
			} else if pos > 5 && bufr[pos-6] == '<' {
				if bufr[pos-1] == '/' && bufr[pos-2] == ' ' && bufr[pos-5] == 's' && bufr[pos-4] == 'u' {
					ch := bufr[pos-3]
					if ch == 'p' || ch == 'b' {
						return true
					}
				}
		*/
	}

	return false
}

func HasMarkup(str string) bool {

	for _, ch := range str {
		if ch <= 127 {
			continue
		}
		// quick min-to-max check for Unicode superscript or subscript characters
		if (ch >= '\u00B2' && ch <= '\u00B9') || (ch >= '\u2070' && ch <= '\u208E') {
			return true
		}
	}

	return false
}

func RemoveUnicodeMarkup(str string) string {

	var buffer bytes.Buffer

	for _, ch := range str {
		if ch > 127 {
			if (ch >= '\u00B2' && ch <= '\u00B9') || (ch >= '\u2070' && ch <= '\u208E') {
				rn, ok := markupRunes[ch]
				if ok {
					ch = rn
				}
			}
		}
		buffer.WriteRune(ch)
	}

	return buffer.String()
}

func SimulateUnicodeMarkup(str string) string {

	var buffer bytes.Buffer

	for _, ch := range str {
		if ch > 127 {
			if (ch >= '\u00B2' && ch <= '\u00B9') || (ch >= '\u2070' && ch <= '\u207F') {
				rn, ok := markupRunes[ch]
				if ok {
					buffer.WriteString("<sup>")
					buffer.WriteRune(rn)
					buffer.WriteString("</sup>")
					continue
				}
			} else if ch >= '\u2080' && ch <= '\u208E' {
				rn, ok := markupRunes[ch]
				if ok {
					buffer.WriteString("<sub>")
					buffer.WriteRune(rn)
					buffer.WriteString("</sub>")
					continue
				}
			}
		}
		buffer.WriteRune(ch)
	}

	return buffer.String()
}

func SplitInTwoAt(str, chr string, side SideType) (string, string) {

	slash := strings.SplitN(str, chr, 2)
	if len(slash) > 1 {
		return slash[0], slash[1]
	}

	if side == LEFT {
		return str, ""
	}

	return "", str
}

func ConvertSlash(str string) string {

	if str == "" {
		return str
	}

	length := len(str)
	res := make([]byte, length+1, length+1)

	isSlash := false
	idx := 0
	for _, ch := range str {
		if isSlash {
			switch ch {
			case 'n':
				// line feed
				res[idx] = '\n'
			case 'r':
				// carriage return
				res[idx] = '\r'
			case 't':
				// horizontal tab
				res[idx] = '\t'
			case 'f':
				// form feed
				res[idx] = '\f'
			case 'a':
				// audible bell from terminal (undocumented)
				res[idx] = '\x07'
			default:
				res[idx] = byte(ch)
			}
			idx++
			isSlash = false
		} else if ch == '\\' {
			isSlash = true
		} else {
			res[idx] = byte(ch)
			idx++
		}
	}

	res = res[0:idx]

	return string(res)
}

func ParseFlag(str string) OpType {

	op, ok := opTypeIs[str]
	if ok {
		return op
	}

	if len(str) > 1 && str[0] == '-' && IsAllCapsOrDigits(str[1:]) {
		return VARIABLE
	}

	if len(str) > 0 && str[0] == '-' {
		return UNRECOGNIZED
	}

	return UNSET
}

var (
	rlock sync.Mutex
	replr *strings.Replacer
	rpair *strings.Replacer
)

func DoHTMLReplace(str string) string {

	// replacer/repairer not reentrant, protected by mutex
	rlock.Lock()

	if replr == nil {
		// handles mixed-content tags, with zero, one, or two levels of encoding
		replr = strings.NewReplacer(
			"<i>", "",
			"</i>", "",
			"<i/>", "",
			"<i />", "",
			"<b>", "",
			"</b>", "",
			"<b/>", "",
			"<b />", "",
			"<u>", "",
			"</u>", "",
			"<u/>", "",
			"<u />", "",
			"<sub>", "",
			"</sub>", "",
			"<sub/>", "",
			"<sub />", "",
			"<sup>", "",
			"</sup>", "",
			"<sup/>", "",
			"<sup />", "",
			"&lt;i&gt;", "",
			"&lt;/i&gt;", "",
			"&lt;i/&gt;", "",
			"&lt;i /&gt;", "",
			"&lt;b&gt;", "",
			"&lt;/b&gt;", "",
			"&lt;b/&gt;", "",
			"&lt;b /&gt;", "",
			"&lt;u&gt;", "",
			"&lt;/u&gt;", "",
			"&lt;u/&gt;", "",
			"&lt;u /&gt;", "",
			"&lt;sub&gt;", "",
			"&lt;/sub&gt;", "",
			"&lt;sub/&gt;", "",
			"&lt;sub /&gt;", "",
			"&lt;sup&gt;", "",
			"&lt;/sup&gt;", "",
			"&lt;sup/&gt;", "",
			"&lt;sup /&gt;", "",
			"&amp;lt;i&amp;gt;", "",
			"&amp;lt;/i&amp;gt;", "",
			"&amp;lt;i/&amp;gt;", "",
			"&amp;lt;i /&amp;gt;", "",
			"&amp;lt;b&amp;gt;", "",
			"&amp;lt;/b&amp;gt;", "",
			"&amp;lt;b/&amp;gt;", "",
			"&amp;lt;b /&amp;gt;", "",
			"&amp;lt;u&amp;gt;", "",
			"&amp;lt;/u&amp;gt;", "",
			"&amp;lt;u/&amp;gt;", "",
			"&amp;lt;u /&amp;gt;", "",
			"&amp;lt;sub&amp;gt;", "",
			"&amp;lt;/sub&amp;gt;", "",
			"&amp;lt;sub/&amp;gt;", "",
			"&amp;lt;sub /&amp;gt;", "",
			"&amp;lt;sup&amp;gt;", "",
			"&amp;lt;/sup&amp;gt;", "",
			"&amp;lt;sup/&amp;gt;", "",
			"&amp;lt;sup /&amp;gt;", "",
			"&amp;amp;", "&amp;",
		)
	}

	if replr != nil {
		str = replr.Replace(str)
	}

	rlock.Unlock()

	return str
}

func DoHTMLRepair(str string) string {

	// replacer/repairer not reentrant, protected by mutex
	rlock.Lock()

	if rpair == nil {
		// handles mixed-content tags, with zero, one, or two levels of encoding
		rpair = strings.NewReplacer(
			"&lt;i&gt;", "<i>",
			"&lt;/i&gt;", "</i>",
			"&lt;i/&gt;", "<i/>",
			"&lt;i /&gt;", "<i/>",
			"&lt;b&gt;", "<b>",
			"&lt;/b&gt;", "</b>",
			"&lt;b/&gt;", "<b/>",
			"&lt;b /&gt;", "<b/>",
			"&lt;u&gt;", "<u>",
			"&lt;/u&gt;", "</u>",
			"&lt;u/&gt;", "<u/>",
			"&lt;u /&gt;", "<u/>",
			"&lt;sub&gt;", "<sub>",
			"&lt;/sub&gt;", "</sub>",
			"&lt;sub/&gt;", "<sub/>",
			"&lt;sub /&gt;", "<sub/>",
			"&lt;sup&gt;", "<sup>",
			"&lt;/sup&gt;", "</sup>",
			"&lt;sup/&gt;", "<sup/>",
			"&lt;sup /&gt;", "<sup/>",
			"&amp;lt;i&amp;gt;", "<i>",
			"&amp;lt;/i&amp;gt;", "</i>",
			"&amp;lt;i/&amp;gt;", "<i/>",
			"&amp;lt;i /&amp;gt;", "<i/>",
			"&amp;lt;b&amp;gt;", "<b>",
			"&amp;lt;/b&amp;gt;", "</b>",
			"&amp;lt;b/&amp;gt;", "<b/>",
			"&amp;lt;b /&amp;gt;", "<b/>",
			"&amp;lt;u&amp;gt;", "<u>",
			"&amp;lt;/u&amp;gt;", "</u>",
			"&amp;lt;u/&amp;gt;", "<u/>",
			"&amp;lt;u /&amp;gt;", "<u/>",
			"&amp;lt;sub&amp;gt;", "<sub>",
			"&amp;lt;/sub&amp;gt;", "</sub>",
			"&amp;lt;sub/&amp;gt;", "<sub/>",
			"&amp;lt;sub /&amp;gt;", "<sub/>",
			"&amp;lt;sup&amp;gt;", "<sup>",
			"&amp;lt;/sup&amp;gt;", "</sup>",
			"&amp;lt;sup/&amp;gt;", "<sup/>",
			"&amp;lt;sup /&amp;gt;", "<sup/>",
			"&amp;amp;", "&amp;",
		)
	}

	if rpair != nil {
		str = rpair.Replace(str)
	}

	rlock.Unlock()

	return str
}

func DoTrimFlankingHTML(str string) string {

	badPrefix := [10]string{
		"<i></i>",
		"<b></b>",
		"<u></u>",
		"<sup></sup>",
		"<sub></sub>",
		"</i>",
		"</b>",
		"</u>",
		"</sup>",
		"</sub>",
	}

	badSuffix := [10]string{
		"<i></i>",
		"<b></b>",
		"<u></u>",
		"<sup></sup>",
		"<sub></sub>",
		"<i>",
		"<b>",
		"<u>",
		"<sup>",
		"<sub>",
	}

	if strings.Contains(str, "<") {
		goOn := true
		for goOn {
			goOn = false
			for _, tag := range badPrefix {
				if strings.HasPrefix(str, tag) {
					str = str[len(tag):]
					goOn = true
				}
			}
			for _, tag := range badSuffix {
				if strings.HasSuffix(str, tag) {
					str = str[:len(str)-len(tag)]
					goOn = true
				}
			}
		}
	}

	return str
}

func HasBadAccent(str string) bool {

	for _, ch := range str {
		if ch <= 127 {
			continue
		}
		// quick min-to-max check for additional characters to treat as accents
		if ch >= '\u00D8' && ch <= '\u02BC' {
			return true
		} else if ch >= '\uFB00' && ch <= '\uFB06' {
			return true
		}
	}

	return false
}

func FixBadAccent(str string) string {

	var buffer bytes.Buffer

	for _, ch := range str {
		if ch > 127 {
			if ch >= '\u00D8' && ch <= '\u02BC' {
				rn, ok := accentRunes[ch]
				if ok {
					buffer.WriteRune(rn)
					continue
				}
				st, ok := ligatureRunes[ch]
				if ok {
					buffer.WriteString(st)
					continue
				}
			}
			if ch >= '\uFB00' && ch <= '\uFB06' {
				st, ok := ligatureRunes[ch]
				if ok {
					buffer.WriteString(st)
					continue
				}
			}
		}
		buffer.WriteRune(ch)
	}

	return buffer.String()
}

var (
	tlock sync.Mutex
	tform transform.Transformer
)

func DoAccentTransform(str string) string {

	// transformer not reentrant, protected by mutex
	tlock.Lock()

	if tform == nil {
		tform = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	}

	if tform != nil {
		str, _, _ = transform.String(tform, str)
	}

	tlock.Unlock()

	// look for characters not in current external runes conversion table
	if HasBadAccent(str) {
		str = FixBadAccent(str)
	}

	return str
}

func UnicodeToASCII(str string) string {

	var buffer bytes.Buffer

	for _, ch := range str {
		if ch > 127 {
			s := strconv.QuoteToASCII(string(ch))
			s = strings.ToUpper(s[3:7])
			for {
				if !strings.HasPrefix(s, "0") {
					break
				}
				s = s[1:]
			}
			buffer.WriteString("&#x")
			buffer.WriteString(s)
			buffer.WriteRune(';')
			continue
		}
		buffer.WriteRune(ch)
	}

	return buffer.String()
}

// CREATE COMMON DRIVER TABLES

// InitTables creates lookup tables to simplify the tokenizer
func InitTables() *Tables {

	tbls := &Tables{}

	for i := range tbls.InBlank {
		tbls.InBlank[i] = false
	}
	tbls.InBlank[' '] = true
	tbls.InBlank['\t'] = true
	tbls.InBlank['\n'] = true
	tbls.InBlank['\r'] = true
	tbls.InBlank['\f'] = true

	// alternative version of InBlank allows newlines to be counted
	for i := range tbls.AltBlank {
		tbls.AltBlank[i] = false
	}
	tbls.AltBlank[' '] = true
	tbls.AltBlank['\t'] = true
	tbls.AltBlank['\r'] = true
	tbls.AltBlank['\f'] = true

	// first character of element cannot be a digit, dash, or period
	for i := range tbls.InFirst {
		tbls.InFirst[i] = false
	}
	for ch := 'A'; ch <= 'Z'; ch++ {
		tbls.InFirst[ch] = true
	}
	for ch := 'a'; ch <= 'z'; ch++ {
		tbls.InFirst[ch] = true
	}
	tbls.InFirst['_'] = true

	// remaining characters also includes colon for namespace
	for i := range tbls.InElement {
		tbls.InElement[i] = false
	}
	for ch := 'A'; ch <= 'Z'; ch++ {
		tbls.InElement[ch] = true
	}
	for ch := 'a'; ch <= 'z'; ch++ {
		tbls.InElement[ch] = true
	}
	for ch := '0'; ch <= '9'; ch++ {
		tbls.InElement[ch] = true
	}
	tbls.InElement['_'] = true
	tbls.InElement['-'] = true
	tbls.InElement['.'] = true
	tbls.InElement[':'] = true

	return tbls
}

// PARSE COMMAND-LINE ARGUMENTS

// ParseArguments parses nested exploration instruction from command-line arguments
func ParseArguments(args []string, pttrn string) *Block {

	// different names of exploration control arguments allow multiple levels of nested "for" loops in a linear command line
	// (capitalized versions for backward-compatibility with original Perl implementation handling of recursive definitions)
	var (
		lcname = []string{
			"",
			"-unit",
			"-subset",
			"-section",
			"-block",
			"-branch",
			"-group",
			"-division",
			"-pattern",
		}

		ucname = []string{
			"",
			"-Unit",
			"-Subset",
			"-Section",
			"-Block",
			"-Branch",
			"-Group",
			"-Division",
			"-Pattern",
		}
	)

	// parseCommands recursive definition
	var parseCommands func(parent *Block, startLevel LevelType)

	// parseCommands does initial parsing of exploration command structure
	parseCommands = func(parent *Block, startLevel LevelType) {

		// find next highest level exploration argument
		findNextLevel := func(args []string, level LevelType) (LevelType, string, string) {

			if len(args) > 1 {

				for {

					if level < UNIT {
						break
					}

					lctag := lcname[level]
					uctag := ucname[level]

					for _, txt := range args {
						if txt == lctag || txt == uctag {
							return level, lctag, uctag
						}
					}

					level--
				}
			}

			return 0, "", ""
		}

		arguments := parent.Working

		level, lctag, uctag := findNextLevel(arguments, startLevel)

		if level < UNIT {

			// break recursion
			return
		}

		// group arguments at a given exploration level
		subsetCommands := func(args []string) *Block {

			max := len(args)

			visit := ""

			// extract name of object to visit
			if max > 1 {
				visit = args[1]
				args = args[2:]
				max -= 2
			}

			partition := 0
			for cur, str := range args {

				// record point of next exploration command
				partition = cur + 1

				// skip if not a command
				if len(str) < 1 || str[0] != '-' {
					continue
				}

				if argTypeIs[str] == EXPLORATION {
					partition = cur
					break
				}
			}

			// parse parent/child construct
			// colon indicates a namespace prefix in any or all of the components
			prnt, match := SplitInTwoAt(visit, "/", RIGHT)

			// promote arguments parsed at this level
			return &Block{Visit: visit, Parent: prnt, Match: match, Parsed: args[0:partition], Working: args[partition:]}
		}

		cur := 0

		// search for positions of current exploration command

		for idx, txt := range arguments {
			if txt == lctag || txt == uctag {
				if idx == 0 {
					continue
				}

				blk := subsetCommands(arguments[cur:idx])
				parseCommands(blk, level-1)
				parent.Subtasks = append(parent.Subtasks, blk)

				cur = idx
			}
		}

		if cur < len(arguments) {
			blk := subsetCommands(arguments[cur:])
			parseCommands(blk, level-1)
			parent.Subtasks = append(parent.Subtasks, blk)
		}

		// clear execution arguments from parent after subsetting
		parent.Working = nil
	}

	parseConditionals := func(cmds *Block, arguments []string) []*Operation {

		max := len(arguments)
		if max < 1 {
			return nil
		}

		// check for missing condition command
		txt := arguments[0]
		if txt != "-if" && txt != "-unless" && txt != "-match" && txt != "-avoid" && txt != "-position" {
			fmt.Fprintf(os.Stderr, "\nERROR: Missing -if command before '%s'\n", txt)
			os.Exit(1)
		}
		if txt == "-position" && max > 2 {
			fmt.Fprintf(os.Stderr, "\nERROR: Cannot combine -position with -if or -unless commands\n")
			os.Exit(1)
		}
		// check for missing argument after last condition
		txt = arguments[max-1]
		if len(txt) > 0 && txt[0] == '-' {
			fmt.Fprintf(os.Stderr, "\nERROR: Item missing after %s command\n", txt)
			os.Exit(1)
		}

		cond := make([]*Operation, 0, max)

		status := UNSET

		// parse conditional clause into execution step
		parseStep := func(op *Operation, elementColonValue bool) {

			if op == nil {
				return
			}

			str := op.Value

			status := ELEMENT

			// check for pound, percent, or caret character at beginning of name
			if len(str) > 1 {
				switch str[0] {
				case '&':
					if IsAllCapsOrDigits(str[1:]) {
						status = VARIABLE
						str = str[1:]
					} else if strings.Contains(str, ":") {
						fmt.Fprintf(os.Stderr, "\nERROR: Unsupported construct '%s', use -if &VARIABLE -equals VALUE instead\n", str)
						os.Exit(1)
					} else {
						fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized variable '%s'\n", str)
						os.Exit(1)
					}
				case '#':
					status = COUNT
					str = str[1:]
				case '%':
					status = LENGTH
					str = str[1:]
				case '^':
					status = DEPTH
					str = str[1:]
				default:
				}
			} else if str == "+" {
				status = INDEX
			}

			// parse parent/element@attribute construct
			// colon indicates a namespace prefix in any or all of the components
			prnt, match := SplitInTwoAt(str, "/", RIGHT)
			match, attrib := SplitInTwoAt(match, "@", LEFT)
			val := ""

			// leading colon indicates namespace prefix wildcard
			wildcard := false
			if strings.HasPrefix(prnt, ":") || strings.HasPrefix(match, ":") || strings.HasPrefix(attrib, ":") {
				wildcard = true
			}

			if elementColonValue {

				// allow parent/element@attribute:value construct for deprecated -match and -avoid, and for subsequent -and and -or commands
				match, val = SplitInTwoAt(str, ":", LEFT)
				prnt, match = SplitInTwoAt(match, "/", RIGHT)
				match, attrib = SplitInTwoAt(match, "@", LEFT)
			}

			tsk := &Step{Type: status, Value: str, Parent: prnt, Match: match, Attrib: attrib, Wild: wildcard}

			op.Stages = append(op.Stages, tsk)

			// transform old -match "element:value" to -match element -equals value
			if val != "" {
				tsk := &Step{Type: EQUALS, Value: val}
				op.Stages = append(op.Stages, tsk)
			}
		}

		idx := 0

		// conditionals should alternate between command and object/value
		expectDash := true
		last := ""

		var op *Operation

		// flag to allow element-colon-value for deprecated -match and -avoid commands, otherwise colon is for namespace prefixes
		elementColonValue := false

		// parse command strings into operation structure
		for idx < max {
			str := arguments[idx]
			idx++

			// conditionals should alternate between command and object/value
			if expectDash {
				if len(str) < 1 || str[0] != '-' {
					fmt.Fprintf(os.Stderr, "\nERROR: Unexpected '%s' argument after '%s'\n", str, last)
					os.Exit(1)
				}
				expectDash = false
			} else {
				if len(str) > 0 && str[0] == '-' {
					fmt.Fprintf(os.Stderr, "\nERROR: Unexpected '%s' command after '%s'\n", str, last)
					os.Exit(1)
				}
				expectDash = true
			}
			last = str

			switch status {
			case UNSET:
				status = ParseFlag(str)
			case POSITION:
				cmds.Position = str
				status = UNSET
			case MATCH, AVOID:
				elementColonValue = true
				fallthrough
			case IF, UNLESS, AND, OR:
				op = &Operation{Type: status, Value: str}
				cond = append(cond, op)
				parseStep(op, elementColonValue)
				status = UNSET
			case EQUALS, CONTAINS, STARTSWITH, ENDSWITH, ISNOT:
				if op != nil {
					if len(str) > 1 && str[0] == '\\' {
						// first character may be backslash protecting dash (undocumented)
						str = str[1:]
					}
					tsk := &Step{Type: status, Value: str}
					op.Stages = append(op.Stages, tsk)
					op = nil
				} else {
					fmt.Fprintf(os.Stderr, "\nERROR: Unexpected adjacent string match constraints\n")
					os.Exit(1)
				}
				status = UNSET
			case GT, GE, LT, LE, EQ, NE:
				if op != nil {
					if len(str) > 1 && str[0] == '\\' {
						// first character may be backslash protecting minus sign (undocumented)
						str = str[1:]
					}
					if len(str) < 1 {
						fmt.Fprintf(os.Stderr, "\nERROR: Empty numeric match constraints\n")
						os.Exit(1)
					}
					ch := str[0]
					if (ch >= '0' && ch <= '9') || ch == '-' || ch == '+' {
						// literal numeric constant
						tsk := &Step{Type: status, Value: str}
						op.Stages = append(op.Stages, tsk)
					} else {
						// numeric test allows element as second argument
						orig := str
						if ch == '#' || ch == '%' || ch == '^' {
							// check for pound, percent, or caret character at beginning of element (undocumented)
							str = str[1:]
							if len(str) < 1 {
								fmt.Fprintf(os.Stderr, "\nERROR: Unexpected numeric match constraints\n")
								os.Exit(1)
							}
							ch = str[0]
						}
						if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
							prnt, match := SplitInTwoAt(str, "/", RIGHT)
							match, attrib := SplitInTwoAt(match, "@", LEFT)
							wildcard := false
							if strings.HasPrefix(prnt, ":") || strings.HasPrefix(match, ":") || strings.HasPrefix(attrib, ":") {
								wildcard = true
							}
							tsk := &Step{Type: status, Value: orig, Parent: prnt, Match: match, Attrib: attrib, Wild: wildcard}
							op.Stages = append(op.Stages, tsk)
						} else {
							fmt.Fprintf(os.Stderr, "\nERROR: Unexpected numeric match constraints\n")
							os.Exit(1)
						}
					}
					op = nil
				} else {
					fmt.Fprintf(os.Stderr, "\nERROR: Unexpected adjacent numeric match constraints\n")
					os.Exit(1)
				}
				status = UNSET
			case UNRECOGNIZED:
				fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized argument '%s'\n", str)
				os.Exit(1)
			default:
				fmt.Fprintf(os.Stderr, "\nERROR: Unexpected argument '%s'\n", str)
				os.Exit(1)
			}
		}

		return cond
	}

	parseExtractions := func(cmds *Block, arguments []string) []*Operation {

		max := len(arguments)
		if max < 1 {
			return nil
		}

		// check for missing -element (or -first, etc.) command
		txt := arguments[0]
		if len(txt) < 1 || txt[0] != '-' {
			fmt.Fprintf(os.Stderr, "\nERROR: Missing -element command before '%s'\n", txt)
			os.Exit(1)
		}
		// check for missing argument after last -element (or -first, etc.) command
		txt = arguments[max-1]
		if len(txt) > 0 && txt[0] == '-' {
			if txt == "-rst" {
				fmt.Fprintf(os.Stderr, "\nERROR: Unexpected position for %s command\n", txt)
				os.Exit(1)
			} else if txt == "-clr" {
			} else if max < 2 || arguments[max-2] != "-lbl" {
				fmt.Fprintf(os.Stderr, "\nERROR: Item missing after %s command\n", txt)
				os.Exit(1)
			}
		}

		comm := make([]*Operation, 0, max)

		status := UNSET

		// parse next argument
		nextStatus := func(str string) OpType {

			status = ParseFlag(str)

			switch status {
			case VARIABLE:
				op := &Operation{Type: status, Value: str[1:]}
				comm = append(comm, op)
				status = VALUE
			case CLR, RST:
				op := &Operation{Type: status, Value: ""}
				comm = append(comm, op)
				status = UNSET
			case ELEMENT, FIRST, LAST, ENCODE, UPPER, LOWER, TITLE, TERMS, WORDS, PAIRS, LETTERS, INDICES:
			case NUM, LEN, SUM, MIN, MAX, INC, DEC, SUB, AVG, DEV, ZEROBASED, ONEBASED, UCSCBASED:
			case TAB, RET, PFX, SFX, SEP, LBL, PFC, DEF:
			case UNSET:
				fmt.Fprintf(os.Stderr, "\nERROR: No -element before '%s'\n", str)
				os.Exit(1)
			case UNRECOGNIZED:
				fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized argument '%s'\n", str)
				os.Exit(1)
			default:
				fmt.Fprintf(os.Stderr, "\nERROR: Misplaced %s command\n", str)
				os.Exit(1)
			}

			return status
		}

		// parse extraction clause into individual steps
		parseSteps := func(op *Operation, pttrn string) {

			if op == nil {
				return
			}

			stat := op.Type
			str := op.Value

			// element names combined with commas are treated as a prefix-separator-suffix group
			comma := strings.Split(str, ",")

			for _, item := range comma {
				status := stat

				// check for special character at beginning of name
				if len(item) > 1 {
					switch item[0] {
					case '&':
						if IsAllCapsOrDigits(item[1:]) {
							status = VARIABLE
							item = item[1:]
						} else {
							fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized variable '%s'\n", item)
							os.Exit(1)
						}
					case '#':
						status = COUNT
						item = item[1:]
					case '%':
						status = LENGTH
						item = item[1:]
					case '^':
						status = DEPTH
						item = item[1:]
					case '*':
						for _, ch := range item {
							if ch != '*' {
								break
							}
						}
						status = STAR
					default:
					}
				} else {
					switch item {
					case "*":
						status = STAR
					case "+":
						status = INDEX
					case "$":
						status = DOLLAR
					case "@":
						status = ATSIGN
					default:
					}
				}

				// parse parent/element@attribute construct
				// colon indicates a namespace prefix in any or all of the components
				prnt, match := SplitInTwoAt(item, "/", RIGHT)
				match, attrib := SplitInTwoAt(match, "@", LEFT)

				// leading colon indicates namespace prefix wildcard
				wildcard := false
				if strings.HasPrefix(prnt, ":") || strings.HasPrefix(match, ":") || strings.HasPrefix(attrib, ":") {
					wildcard = true
				}

				// sequence coordinate adjustments
				switch status {
				case ZEROBASED, ONEBASED, UCSCBASED:
					seq := pttrn + ":"
					if attrib != "" {
						seq += "@"
						seq += attrib
					} else if match != "" {
						seq += match
					}
					// confirm -0-based or -1-based arguments are known sequence position elements or attributes
					slock.RLock()
					seqtype, ok := sequenceTypeIs[seq]
					slock.RUnlock()
					if !ok {
						fmt.Fprintf(os.Stderr, "\nERROR: Element '%s' is not suitable for sequence coordinate conversion\n", item)
						os.Exit(1)
					}
					switch status {
					case ZEROBASED:
						status = ELEMENT
						// if 1-based coordinates, decrement to get 0-based value
						if seqtype.Based == 1 {
							status = DEC
						}
					case ONEBASED:
						status = ELEMENT
						// if 0-based coordinates, increment to get 1-based value
						if seqtype.Based == 0 {
							status = INC
						}
					case UCSCBASED:
						status = ELEMENT
						// half-open intervals, start is 0-based, stop is 1-based
						if seqtype.Based == 0 && seqtype.Which == ISSTOP {
							status = INC
						} else if seqtype.Based == 1 && seqtype.Which == ISSTART {
							status = DEC
						}
					default:
						status = ELEMENT
					}
				default:
				}

				tsk := &Step{Type: status, Value: item, Parent: prnt, Match: match, Attrib: attrib, Wild: wildcard}

				op.Stages = append(op.Stages, tsk)
			}
		}

		idx := 0

		// parse command strings into operation structure
		for idx < max {
			str := arguments[idx]
			idx++

			if argTypeIs[str] == CONDITIONAL {
				fmt.Fprintf(os.Stderr, "\nERROR: Misplaced %s command\n", str)
				os.Exit(1)
			}

			switch status {
			case UNSET:
				status = nextStatus(str)
			case ELEMENT, FIRST, LAST, ENCODE, UPPER, LOWER, TITLE, TERMS, WORDS, PAIRS, LETTERS, INDICES,
				NUM, LEN, SUM, MIN, MAX, INC, DEC, SUB, AVG, DEV, ZEROBASED, ONEBASED, UCSCBASED:
				for !strings.HasPrefix(str, "-") {
					// create one operation per argument, even if under a single -element statement
					op := &Operation{Type: status, Value: str}
					comm = append(comm, op)
					parseSteps(op, pttrn)
					if idx >= max {
						break
					}
					str = arguments[idx]
					idx++
				}
				status = UNSET
				if idx < max {
					status = nextStatus(str)
				}
			case TAB, RET, PFX, SFX, SEP, LBL, PFC, DEF:
				op := &Operation{Type: status, Value: ConvertSlash(str)}
				comm = append(comm, op)
				status = UNSET
			case VARIABLE:
				op := &Operation{Type: status, Value: str[1:]}
				comm = append(comm, op)
				status = VALUE
			case VALUE:
				op := &Operation{Type: status, Value: str}
				comm = append(comm, op)
				parseSteps(op, pttrn)
				status = UNSET
			case UNRECOGNIZED:
				fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized argument '%s'\n", str)
				os.Exit(1)
			default:
			}
		}

		return comm
	}

	// parseOperations recursive definition
	var parseOperations func(parent *Block)

	// parseOperations converts parsed arguments to operations lists
	parseOperations = func(parent *Block) {

		args := parent.Parsed

		partition := 0
		for cur, str := range args {

			// record junction between conditional and extraction commands
			partition = cur + 1

			// skip if not a command
			if len(str) < 1 || str[0] != '-' {
				continue
			}

			if argTypeIs[str] != CONDITIONAL {
				partition = cur
				break
			}
		}

		// split arguments into conditional tests and extraction or customization commands
		conditionals := args[0:partition]
		args = args[partition:]

		partition = 0
		foundElse := false
		for cur, str := range args {

			// record junction at -else command
			partition = cur + 1

			// skip if not a command
			if len(str) < 1 || str[0] != '-' {
				continue
			}

			if str == "-else" {
				partition = cur
				foundElse = true
				break
			}
		}

		extractions := args[0:partition]
		alternative := args[partition:]

		if len(alternative) > 0 && alternative[0] == "-else" {
			alternative = alternative[1:]
		}

		// validate argument structure and convert to operations lists
		parent.Conditions = parseConditionals(parent, conditionals)
		parent.Commands = parseExtractions(parent, extractions)
		parent.Failure = parseExtractions(parent, alternative)

		// reality checks on placement of -else command
		if foundElse {
			if len(conditionals) < 1 {
				fmt.Fprintf(os.Stderr, "\nERROR: Misplaced -else command\n")
				os.Exit(1)
			}
			if len(alternative) < 1 {
				fmt.Fprintf(os.Stderr, "\nERROR: Misplaced -else command\n")
				os.Exit(1)
			}
			if len(parent.Subtasks) > 0 {
				fmt.Fprintf(os.Stderr, "\nERROR: Misplaced -else command\n")
				os.Exit(1)
			}
		}

		for _, sub := range parent.Subtasks {
			parseOperations(sub)
		}
	}

	// ParseArguments

	head := &Block{}

	for _, txt := range args {
		head.Working = append(head.Working, txt)
	}

	// initial parsing of exploration command structure
	parseCommands(head, PATTERN)

	if len(head.Subtasks) != 1 {
		return nil
	}

	// skip past empty placeholder
	head = head.Subtasks[0]

	// convert command strings to array of operations for faster processing
	parseOperations(head)

	// check for no -element or multiple -pattern commands
	noElement := true
	numPatterns := 0
	for _, txt := range args {
		if argTypeIs[txt] == EXTRACTION {
			noElement = false
		}
		if txt == "-pattern" || txt == "-Pattern" {
			numPatterns++
		}
	}

	if numPatterns < 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: No -pattern in command-line arguments\n")
		os.Exit(1)
	}

	if numPatterns > 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Only one -pattern command is permitted\n")
		os.Exit(1)
	}

	if noElement {
		fmt.Fprintf(os.Stderr, "\nERROR: No -element statement in argument list\n")
		os.Exit(1)
	}

	return head
}

// READ XML INPUT FILE INTO SET OF BLOCKS

type XMLReader struct {
	Reader     io.Reader
	Buffer     []byte
	Remainder  string
	Position   int64
	Delta      int
	Closed     bool
	Docompress bool
	Docleanup  bool
	LeaveHTML  bool
}

func NewXMLReader(in io.Reader, doCompress, doCleanup, leaveHTML bool) *XMLReader {

	if in == nil {
		return nil
	}

	rdr := &XMLReader{Reader: in, Docompress: doCompress, Docleanup: doCleanup, LeaveHTML: leaveHTML}

	// 65536 appears to be the maximum number of characters presented to io.Reader when input is piped from stdin
	// increasing size of buffer when input is from a file does not improve program performance
	// additional 16384 bytes are reserved for copying previous remainder to start of buffer before next read
	const XMLBUFSIZE = 65536 + 16384

	rdr.Buffer = make([]byte, XMLBUFSIZE)

	return rdr
}

// NextBlock reads buffer, concatenates if necessary to place long element content into a single string
// all result strings end in > character that is used as a sentinel in subsequent code
func (rdr *XMLReader) NextBlock() string {

	if rdr == nil {
		return ""
	}

	// read one buffer, trim at last > and retain remainder for next call, signal if no > character
	nextBuffer := func() (string, bool, bool) {

		if rdr.Closed {
			return "", false, true
		}

		// prepend previous remainder to beginning of buffer
		m := copy(rdr.Buffer, rdr.Remainder)
		rdr.Remainder = ""
		if m > 16384 {
			// previous remainder is larger than reserved section, write and signal need to continue reading
			return string(rdr.Buffer[:m]), true, false
		}

		// read next block, append behind copied remainder from previous read
		n, err := rdr.Reader.Read(rdr.Buffer[m:])
		// with data piped through stdin, read function may not always return the same number of bytes each time
		if err != nil {
			if err != io.EOF {
				// real error
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				rdr.Closed = true
				return "", false, true
			}
			// end of file
			rdr.Closed = true
			if n == 0 {
				// if EOF and no more data, do not send final remainder (not terminated by right angle bracket that is used as a sentinel)
				return "", false, true
			}
		}

		// keep track of file offset
		rdr.Position += int64(rdr.Delta)
		rdr.Delta = n

		// slice of actual characters read
		bufr := rdr.Buffer[:n+m]

		// look for last > character
		// safe to back up on UTF-8 rune array when looking for 7-bit ASCII character
		pos := -1
		for pos = len(bufr) - 1; pos >= 0; pos-- {
			if bufr[pos] == '>' {
				if rdr.LeaveHTML {
					// optionally skip backwards past embedded i, b, u, sub, and sup HTML open, close, and empty tags
					if HTMLBehind(bufr, pos) {
						continue
					}
				}
				// found end of XML tag, break
				break
			}
		}

		// trim back to last > character, save remainder for next buffer
		if pos > -1 {
			pos++
			rdr.Remainder = string(bufr[pos:])
			return string(bufr[:pos]), false, false
		}

		// no > found, signal need to continue reading long content
		return string(bufr[:]), true, false
	}

	// read next buffer
	line, cont, closed := nextBuffer()

	if closed {
		// no sentinel in remainder at end of file
		return ""
	}

	// if buffer does not end with > character
	if cont {
		var buff bytes.Buffer

		// keep reading long content blocks
		for {
			if line != "" {
				buff.WriteString(line)
			}
			if !cont {
				// last buffer ended with sentinel
				break
			}
			line, cont, closed = nextBuffer()
			if closed {
				// no sentinel in multi-block buffer at end of file
				return ""
			}
		}

		// concatenate blocks
		line = buff.String()
	}

	// trimming spaces here would throw off line tracking

	// optionally compress/cleanup tags/attributes and contents
	if rdr.Docompress {
		line = CompressRunsOfSpaces(line)
	}
	if rdr.Docleanup {
		if HasBadSpace(line) {
			line = CleanupBadSpaces(line)
		}
	}

	return line
}

// PARSE XML BLOCK STREAM INTO STRINGS FROM <PATTERN> TO </PATTERN>

// PartitionPattern splits XML input by pattern and sends individual records to a callback
func PartitionPattern(pat, star string, rdr *XMLReader, proc func(int, int64, string)) {

	if pat == "" || rdr == nil || proc == nil {
		return
	}

	type Scanner struct {
		Pattern   string
		PatLength int
		CharSkip  [256]int
	}

	// initialize <pattern> to </pattern> scanner
	newScanner := func(pattern string) *Scanner {

		if pattern == "" {
			return nil
		}

		scr := &Scanner{Pattern: pattern}

		patlen := len(pattern)
		scr.PatLength = patlen

		// position of last character in pattern
		last := patlen - 1

		// initialize bad character displacement table
		for i := range scr.CharSkip {
			scr.CharSkip[i] = patlen
		}
		for i := 0; i < last; i++ {
			ch := pattern[i]
			scr.CharSkip[ch] = last - i
		}

		return scr
	}

	// check surroundings of match candidate
	isAnElement := func(text string, lf, rt, mx int) bool {

		if (lf >= 0 && text[lf] == '<') || (lf > 0 && text[lf] == '/' && text[lf-1] == '<') {
			if (rt < mx && (text[rt] == '>' || text[rt] == ' ')) || (rt+1 < mx && text[rt] == '/' && text[rt+1] == '>') {
				return true
			}
		}

		return false
	}

	// modified Boyer-Moore-Horspool search function
	findNextMatch := func(scr *Scanner, text string, offset int) (int, int, int) {

		if scr == nil || text == "" {
			return -1, -1, -1
		}

		// copy values into local variables for speed
		txtlen := len(text)
		pattern := scr.Pattern[:]
		patlen := scr.PatLength
		max := txtlen - patlen
		last := patlen - 1
		skip := scr.CharSkip[:]

		i := offset

		for i <= max {
			j := last
			k := i + last
			for j >= 0 && text[k] == pattern[j] {
				j--
				k--
			}
			// require match candidate to be element name, i.e., <pattern ... >, </pattern ... >, or <pattern ... />
			if j < 0 && isAnElement(text, i-1, i+patlen, txtlen) {
				// find positions of flanking brackets
				lf := i - 1
				for lf > 0 && text[lf] != '<' {
					lf--
				}
				rt := i + patlen
				for rt < txtlen && text[rt] != '>' {
					rt++
				}
				return i + 1, lf, rt + 1
			}
			// find character in text above last character in pattern
			ch := text[i+last]
			// displacement table can shift pattern by one or more positions
			i += skip[ch]
		}

		return -1, -1, -1
	}

	type PatternType int

	const (
		NOPATTERN PatternType = iota
		STARTPATTERN
		SELFPATTERN
		STOPPATTERN
	)

	// find next element with pattern name
	nextPattern := func(scr *Scanner, text string, pos int) (PatternType, int, int) {

		if scr == nil || text == "" {
			return NOPATTERN, 0, 0
		}

		prev := pos

		for {
			next, start, stop := findNextMatch(scr, text, prev)
			if next < 0 {
				return NOPATTERN, 0, 0
			}

			prev = next + 1

			if text[start+1] == '/' {
				return STOPPATTERN, stop, prev
			} else if text[stop-2] == '/' {
				return SELFPATTERN, start, prev
			} else {
				return STARTPATTERN, start, prev
			}
		}
	}

	// -pattern Object construct

	doNormal := func() {

		// current depth of -pattern objects
		level := 0

		begin := 0
		inPattern := false

		line := ""
		var accumulator bytes.Buffer

		match := NOPATTERN
		pos := 0
		next := 0

		offset := int64(0)

		rec := 0

		scr := newScanner(pat)
		if scr == nil {
			return
		}

		for {

			begin = 0
			next = 0

			line = rdr.NextBlock()
			if line == "" {
				return
			}

			for {
				match, pos, next = nextPattern(scr, line, next)
				if match == STARTPATTERN {
					if level == 0 {
						inPattern = true
						begin = pos
						offset = rdr.Position + int64(pos)
					}
					level++
				} else if match == STOPPATTERN {
					level--
					if level == 0 {
						inPattern = false
						accumulator.WriteString(line[begin:pos])
						// read and process one -pattern object at a time
						str := accumulator.String()
						if str != "" {
							rec++
							proc(rec, offset, str[:])
						}
						// reset accumulator
						accumulator.Reset()
					}
				} else if match == SELFPATTERN {
				} else {
					if inPattern {
						accumulator.WriteString(line[begin:])
					}
					break
				}
			}
		}
	}

	// -pattern Parent/* construct now works with catenated files, but not if components
	// are recursive or self-closing objects, process those through xtract -format first

	doStar := func() {

		// current depth of -pattern objects
		level := 0

		begin := 0
		inPattern := false

		line := ""
		var accumulator bytes.Buffer

		match := NOPATTERN
		pos := 0
		next := 0

		offset := int64(0)

		rec := 0

		scr := newScanner(pat)
		if scr == nil {
			return
		}

		last := pat

		// read to first <pattern> element
		for {

			next = 0

			line = rdr.NextBlock()
			if line == "" {
				break
			}

			match, pos, next = nextPattern(scr, line, next)
			if match == STARTPATTERN {
				break
			}
		}

		if match != STARTPATTERN {
			return
		}

		// find next element in XML
		nextElement := func(text string, pos int) string {

			txtlen := len(text)

			tag := ""
			for i := pos; i < txtlen; i++ {
				if text[i] == '<' {
					tag = text[i+1:]
					break
				}
			}
			if tag == "" {
				return ""
			}
			if tag[0] == '/' {
				if strings.HasPrefix(tag[1:], pat) {
					//should be </pattern> at end, want to continue if catenated files
					return "/"
				}
				return ""
			}
			for i, ch := range tag {
				if ch == '>' || ch == ' ' || ch == '/' {
					return tag[0:i]
				}
			}

			return ""
		}

		// read and process heterogeneous objects immediately below <pattern> parent
		for {
			tag := nextElement(line, next)
			if tag == "" {

				begin = 0
				next = 0

				line = rdr.NextBlock()
				if line == "" {
					break
				}

				tag = nextElement(line, next)
			}
			if tag == "" {
				return
			}

			// check for catenated parent set files
			if tag[0] == '/' {
				scr = newScanner(pat)
				if scr == nil {
					return
				}
				last = pat
				// confirm end </pattern> just found
				match, pos, next = nextPattern(scr, line, next)
				if match != STOPPATTERN {
					return
				}
				// now look for a new start <pattern> tag
				for {
					match, pos, next = nextPattern(scr, line, next)
					if match == STARTPATTERN {
						break
					}
					next = 0
					line = rdr.NextBlock()
					if line == "" {
						break
					}
				}
				if match != STARTPATTERN {
					return
				}
				// continue with processing loop
				continue
			}

			if tag != last {
				scr = newScanner(tag)
				if scr == nil {
					return
				}
				last = tag
			}

			for {
				match, pos, next = nextPattern(scr, line, next)
				if match == STARTPATTERN {
					if level == 0 {
						inPattern = true
						begin = pos
						offset = rdr.Position + int64(pos)
					}
					level++
				} else if match == STOPPATTERN {
					level--
					if level == 0 {
						inPattern = false
						accumulator.WriteString(line[begin:pos])
						// read and process one -pattern/* object at a time
						str := accumulator.String()
						if str != "" {
							rec++
							proc(rec, offset, str[:])
						}
						// reset accumulator
						accumulator.Reset()
						break
					}
				} else {
					if inPattern {
						accumulator.WriteString(line[begin:])
					}

					begin = 0
					next = 0

					line = rdr.NextBlock()
					if line == "" {
						break
					}
				}
			}
		}
	}

	// call appropriate handler
	if star == "" {
		doNormal()
	} else if star == "*" {
		doStar()
	}
}

// XML VALIDATION AND FORMATTING FUNCTIONS

// ProcessXMLStream tokenizes and runs designated operations on an entire XML file
func ProcessXMLStream(in *XMLReader, tbls *Tables, args []string, action SpecialType) {

	if in == nil || tbls == nil {
		return
	}

	blockCount := 0

	// token parser variables
	Text := ""
	Txtlen := 0
	Idx := 0
	Line := 1

	// variables to track comments or CDATA sections that span reader blocks
	Which := NOTAG
	SkipTo := ""

	plainText := (!tbls.DoStrict && !tbls.DoMixed)

	// get next XML token
	nextToken := func(idx int) (TagType, string, string, int, int) {

		if Text == "" {
			// if buffer is empty, read next block
			Text = in.NextBlock()
			Txtlen = len(Text)
			Idx = 0
			idx = 0
			blockCount++
		}

		if Text == "" {
			return ISCLOSED, "", "", Line, 0
		}

		// lookup table array pointers
		inBlank := &tbls.AltBlank
		inFirst := &tbls.InFirst
		inElement := &tbls.InElement

		text := Text[:]
		txtlen := Txtlen
		line := Line

		if Which != NOTAG && SkipTo != "" {
			which := Which
			// previous block ended inside CDATA object or comment
			start := idx
			found := strings.Index(text[:], SkipTo)
			if found < 0 {
				// no stop signal found in next block
				// count lines
				for i := 0; i < txtlen; i++ {
					if text[i] == '\n' {
						line++
					}
				}
				Line = line
				str := text[:]
				if HasFlankingSpace(str) {
					str = strings.TrimSpace(str)
				}
				// signal end of current block
				Text = ""
				// leave Which and SkipTo values unchanged as another continuation signal
				// send CDATA or comment contents
				return which, str[:], "", Line, 0
			}
			// otherwise adjust position past end of skipTo string and return to normal processing
			idx += found
			// count lines
			for i := 0; i < idx; i++ {
				if text[i] == '\n' {
					line++
				}
			}
			Line = line
			str := text[start:idx]
			if HasFlankingSpace(str) {
				str = strings.TrimSpace(str)
			}
			idx += len(SkipTo)
			// clear tracking variables
			Which = NOTAG
			SkipTo = ""
			// send CDATA or comment contents
			return which, str[:], "", Line, idx
		}

		// all blocks end with > character, acts as sentinel to check if past end of text
		if idx >= txtlen {
			// signal end of current block, will read next block on next call
			Text = ""
			Line = line
			return NOTAG, "", "", Line, 0
		}

		// skip past leading blanks
		ch := text[idx]
		for {
			for inBlank[ch] {
				idx++
				ch = text[idx]
			}
			if ch != '\n' {
				break
			}
			line++
			idx++
			ch = text[idx]
		}
		Line = line

		start := idx

		if ch == '<' && (plainText || HTMLAhead(text, idx) == 0) {

			// at start of element
			idx++
			ch = text[idx]

			// check for legal first character of element
			if inFirst[ch] {

				// read element name
				start = idx
				idx++

				ch = text[idx]
				for inElement[ch] {
					idx++
					ch = text[idx]
				}

				str := text[start:idx]

				switch ch {
				case '>':
					// end of element
					idx++

					return STARTTAG, str[:], "", Line, idx
				case '/':
					// self-closing element without attributes
					idx++
					ch = text[idx]
					if ch != '>' {
						fmt.Fprintf(os.Stderr, "\nSelf-closing element missing right angle bracket, line %d\n", line)
					}
					idx++

					return SELFTAG, str[:], "", Line, idx
				case '\n':
					line++
					fallthrough
				case ' ', '\t', '\r', '\f':
					// attributes
					idx++
					start = idx
					ch = text[idx]
					for {
						for ch != '<' && ch != '>' && ch != '\n' {
							idx++
							ch = text[idx]
						}
						if ch != '\n' {
							break
						}
						line++
						idx++
						ch = text[idx]
					}
					Line = line
					if ch != '>' {
						fmt.Fprintf(os.Stderr, "\nAttributes not followed by right angle bracket, line %d\n", line)
					}
					if text[idx-1] == '/' {
						// self-closing
						atr := text[start : idx-1]
						idx++
						return SELFTAG, str[:], atr[:], Line, idx
					}
					atr := text[start:idx]
					idx++
					return STARTTAG, str[:], atr[:], Line, idx
				default:
					fmt.Fprintf(os.Stderr, "\nUnexpected punctuation '%c' in XML element, line %d\n", ch, line)
					return STARTTAG, str[:], "", Line, idx
				}

			} else {

				// punctuation character immediately after first angle bracket
				switch ch {
				case '/':
					// at start of end tag
					idx++
					start = idx
					ch = text[idx]
					// expect legal first character of element
					if inFirst[ch] {
						idx++
						ch = text[idx]
						for inElement[ch] {
							idx++
							ch = text[idx]
						}
						str := text[start:idx]
						if ch != '>' {
							fmt.Fprintf(os.Stderr, "\nUnexpected characters after end element name, line %d\n", line)
						}
						idx++

						return STOPTAG, str[:], "", Line, idx
					}
					fmt.Fprintf(os.Stderr, "\nUnexpected punctuation '%c' in XML element, line %d\n", ch, line)
				case '?':
					// skip ?xml and ?processing instructions
					idx++
					ch = text[idx]
					for ch != '>' {
						idx++
						ch = text[idx]
					}
					idx++
					return NOTAG, "", "", Line, idx
				case '!':
					// skip !DOCTYPE, !comment, and ![CDATA[
					idx++
					start = idx
					ch = text[idx]
					Which = NOTAG
					SkipTo = ""
					if ch == '[' && strings.HasPrefix(text[idx:], "[CDATA[") {
						Which = CDATATAG
						SkipTo = "]]>"
						start += 7
					} else if ch == '-' && strings.HasPrefix(text[idx:], "--") {
						Which = COMMENTTAG
						SkipTo = "-->"
						start += 2
					} else if strings.HasPrefix(text[idx:], "DOCTYPE") {
						Which = DOCTYPETAG
						SkipTo = ">"
					}
					if Which != NOTAG && SkipTo != "" {
						which := Which
						// CDATA or comment block may contain internal angle brackets
						found := strings.Index(text[idx:], SkipTo)
						if found < 0 {
							// string stops in middle of CDATA or comment
							// count lines
							for i := start; i < txtlen; i++ {
								if text[i] == '\n' {
									line++
								}
							}
							Line = line
							str := text[start:]
							if HasFlankingSpace(str) {
								str = strings.TrimSpace(str)
							}
							// signal end of current block
							Text = ""
							// leave Which and SkipTo values unchanged as another continuation signal
							// send CDATA or comment contents
							return which, str[:], "", Line, 0
						}
						// adjust position past end of CDATA or comment
						idx += found
						// count lines
						for i := start; i < idx; i++ {
							if text[i] == '\n' {
								line++
							}
						}
						Line = line
						str := text[start:idx]
						if HasFlankingSpace(str) {
							str = strings.TrimSpace(str)
						}
						idx += len(SkipTo)
						// clear tracking variables
						Which = NOTAG
						SkipTo = ""
						// send CDATA or comment contents
						return which, str[:], "", Line, idx
					}
					// otherwise just skip to next right angle bracket
					for ch != '>' {
						if ch == '\n' {
							line++
						}
						idx++
						ch = text[idx]
					}
					Line = line
					idx++
					return NOTAG, "", "", Line, idx
				default:
					fmt.Fprintf(os.Stderr, "\nUnexpected punctuation '%c' in XML element, line %d\n", ch, line)
				}
			}

		} else if ch != '>' {

			// at start of contents
			start = idx

			// find end of contents
			for {
				for ch != '<' && ch != '>' && ch != '\n' {
					idx++
					ch = text[idx]
				}
				if ch == '<' && !plainText {
					// optionally allow HTML text formatting elements and super/subscripts
					advance := HTMLAhead(text, idx)
					if advance > 0 {
						idx += advance
						ch = text[idx]
						continue
					}
				}
				if ch != '\n' {
					break
				}
				line++
				idx++
				ch = text[idx]
			}
			Line = line

			// trim back past trailing blanks
			lst := idx - 1
			ch = text[lst]
			for inBlank[ch] && lst > start {
				lst--
				ch = text[lst]
			}

			str := text[start : lst+1]

			return CONTENTTAG, str[:], "", Line, idx
		}

		// signal end of current block, will read next block on next call
		Text = ""
		Line = line
		return NOTAG, "", "", Line, 0
	}

	// common output buffer
	var buffer bytes.Buffer
	count := 0

	// processOutline displays outline of XML structure
	processOutline := func() {

		indent := 0

		for {
			tag, name, _, _, idx := nextToken(Idx)
			Idx = idx

			switch tag {
			case STARTTAG:
				if name == "eSummaryResult" ||
					name == "eLinkResult" ||
					name == "eInfoResult" ||
					name == "PubmedArticleSet" ||
					name == "DocumentSummarySet" ||
					name == "INSDSet" ||
					name == "Entrezgene-Set" ||
					name == "TaxaSet" {
					break
				}
				for i := 0; i < indent; i++ {
					buffer.WriteString("  ")
				}
				buffer.WriteString(name)
				buffer.WriteString("\n")
				indent++
			case SELFTAG:
				for i := 0; i < indent; i++ {
					buffer.WriteString("  ")
				}
				buffer.WriteString(name)
				buffer.WriteString("\n")
			case STOPTAG:
				indent--
			case DOCTYPETAG:
			case NOTAG:
			case ISCLOSED:
				txt := buffer.String()
				if txt != "" {
					// print final buffer
					fmt.Fprintf(os.Stdout, "%s", txt)
				}
				return
			default:
			}

			count++
			if count > 1000 {
				count = 0
				txt := buffer.String()
				if txt != "" {
					// print current buffered output
					fmt.Fprintf(os.Stdout, "%s", txt)
				}
				buffer.Reset()
			}
		}
	}

	// processSynopsis displays paths to XML elements
	processSynopsis := func() {

		// synopsisLevel recursive definition
		var synopsisLevel func(string) bool

		synopsisLevel = func(parent string) bool {

			for {
				tag, name, _, _, idx := nextToken(Idx)
				Idx = idx

				switch tag {
				case STARTTAG:
					if name == "eSummaryResult" ||
						name == "eLinkResult" ||
						name == "eInfoResult" ||
						name == "PubmedArticleSet" ||
						name == "DocumentSummarySet" ||
						name == "INSDSet" ||
						name == "Entrezgene-Set" ||
						name == "TaxaSet" {
						break
					}
					if parent != "" {
						buffer.WriteString(parent)
						buffer.WriteString("/")
					}
					buffer.WriteString(name)
					buffer.WriteString("\n")
					path := parent
					if path != "" {
						path += "/"
					}
					path += name
					if synopsisLevel(path) {
						return true
					}
				case SELFTAG:
					if parent != "" {
						buffer.WriteString(parent)
						buffer.WriteString("/")
					}
					buffer.WriteString(name)
					buffer.WriteString("\n")
				case STOPTAG:
					// break recursion
					return false
				case DOCTYPETAG:
				case NOTAG:
				case ISCLOSED:
					txt := buffer.String()
					if txt != "" {
						// print final buffer
						fmt.Fprintf(os.Stdout, "%s", txt)
					}
					return true
				default:
				}

				count++
				if count > 1000 {
					count = 0
					txt := buffer.String()
					if txt != "" {
						// print current buffered output
						fmt.Fprintf(os.Stdout, "%s", txt)
					}
					buffer.Reset()
				}
			}
		}

		for {
			// may have concatenated XMLs, loop through all
			if synopsisLevel("") {
				return
			}
		}
	}

	// processVerify checks for well-formed XML
	processVerify := func() {

		type VerifyType int

		const (
			_ VerifyType = iota
			START
			STOP
			CHAR
			OTHER
		)

		// skip past command name
		args = args[1:]

		pttrn := ""

		if len(args) > 0 {
			pttrn = args[0]
			args = args[1:]
		}

		// if pattern supplied, report maximum nesting depth and record spanning the most blocks (undocumented)
		maxDepth := 0
		depthLine := 0
		maxBlocks := 0
		blockLine := 0
		startLine := 0

		// warn if HTML tags are not well-formed
		unbalancedHTML := func(text string) bool {

			var arry []string

			idx := 0
			txtlen := len(text)

			inTag := false
			start := 0

			for idx < txtlen {
				ch := text[idx]
				if ch == '<' {
					if inTag {
						return true
					}
					inTag = true
					start = idx
				} else if ch == '>' {
					if !inTag {
						return true
					}
					inTag = false
					curr := text[start+1 : idx]
					if strings.HasPrefix(curr, "/") {
						curr = curr[1:]
						if len(arry) < 1 {
							return true
						}
						prev := arry[len(arry)-1]
						if curr != prev {
							return true
						}
						arry = arry[:len(arry)-1]
					} else {
						arry = append(arry, curr)
					}
				}
				idx++
			}

			if inTag {
				return true
			}

			if len(arry) > 0 {
				return true
			}

			return false
		}

		// verifyLevel recursive definition
		var verifyLevel func(string, int)

		// verify integrity of XML object nesting (well-formed)
		verifyLevel = func(parent string, level int) {

			status := START
			for {
				// use alternative low-level tokenizer
				tag, name, _, line, idx := nextToken(Idx)
				Idx = idx

				if level > maxDepth {
					maxDepth = level
					depthLine = line
				}

				switch tag {
				case STARTTAG:
					if status == CHAR {
						fmt.Fprintf(os.Stdout, "<%s> not expected after contents, line %d\n", name, line)
					}
					if name == pttrn {
						blockCount = 1
						startLine = line
					}
					verifyLevel(name, level+1)
					// returns here after recursion
					status = STOP
				case SELFTAG:
					status = OTHER
				case STOPTAG:
					if name == pttrn {
						if blockCount > maxBlocks {
							maxBlocks = blockCount
							blockLine = startLine
						}
					}
					if parent != name && parent != "" {
						fmt.Fprintf(os.Stdout, "Expected </%s>, found </%s>, line %d\n", parent, name, line)
					}
					if level < 1 {
						fmt.Fprintf(os.Stdout, "Unexpected </%s> at end of XML, line %d\n", name, line)
					}
					// break recursion
					return
				case CONTENTTAG:
					if status != START {
						fmt.Fprintf(os.Stdout, "Contents not expected before </%s>, line %d\n", parent, line)
					}
					if tbls.DoStrict || tbls.DoMixed {
						if unbalancedHTML(name) {
							fmt.Fprintf(os.Stdout, "Unbalanced mixed-content tags, line %d\n", line)
						}
					}
					status = CHAR
				case CDATATAG, COMMENTTAG:
					status = OTHER
				case DOCTYPETAG:
				case NOTAG:
				case ISCLOSED:
					if level > 0 {
						fmt.Fprintf(os.Stdout, "Unexpected end of data\n")
					}
					return
				default:
					status = OTHER
				}
			}
		}

		verifyLevel("", 0)

		if pttrn != "" {
			fmt.Fprintf(os.Stdout, "Maximum nesting (%d levels) at line %d\n", maxDepth, depthLine)
			fmt.Fprintf(os.Stdout, "Longest pattern (%d blocks) at line %d\n", maxBlocks, blockLine)
		}
	}

	// processFilter modifies XML content, comments, or CDATA
	processFilter := func() {

		// skip past command name
		args = args[1:]

		max := len(args)
		if max < 1 {
			fmt.Fprintf(os.Stderr, "\nERROR: Insufficient command-line arguments supplied to xtract -filter\n")
			os.Exit(1)
		}

		pttrn := args[0]

		args = args[1:]
		max--

		if max < 2 {
			fmt.Fprintf(os.Stderr, "\nERROR: No object name supplied to xtract -filter\n")
			os.Exit(1)
		}

		type ActionType int

		const (
			NOACTION ActionType = iota
			DORETAIN
			DOREMOVE
			DOENCODE
			DODECODE
			DOSHRINK
			DOEXPAND
			DOACCENT
		)

		action := args[0]

		what := NOACTION
		switch action {
		case "retain":
			what = DORETAIN
		case "remove":
			what = DOREMOVE
		case "encode":
			what = DOENCODE
		case "decode":
			what = DODECODE
		case "shrink":
			what = DOSHRINK
		case "expand":
			what = DOEXPAND
		case "accent":
			what = DOACCENT
		default:
			fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized action '%s' supplied to xtract -filter\n", action)
			os.Exit(1)
		}

		trget := args[1]

		which := NOTAG
		switch trget {
		case "attribute", "attributes":
			which = ATTRIBTAG
		case "content", "contents":
			which = CONTENTTAG
		case "cdata", "CDATA":
			which = CDATATAG
		case "comment", "comments":
			which = COMMENTTAG
		case "object":
			// object normally retained
			which = OBJECTTAG
		case "container":
			which = CONTAINERTAG
		default:
			fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized target '%s' supplied to xtract -filter\n", trget)
			os.Exit(1)
		}

		inPattern := false
		prevName := ""

		for {
			tag, name, attr, _, idx := nextToken(Idx)
			Idx = idx

			switch tag {
			case STARTTAG:
				prevName = name
				if name == pttrn {
					inPattern = true
					if which == CONTAINERTAG && what == DOREMOVE {
						continue
					}
				}
				if inPattern && which == OBJECTTAG && what == DOREMOVE {
					continue
				}
				buffer.WriteString("<")
				buffer.WriteString(name)
				if attr != "" {
					if which != ATTRIBTAG || what != DOREMOVE {
						attr = strings.TrimSpace(attr)
						attr = CompressRunsOfSpaces(attr)
						buffer.WriteString(" ")
						buffer.WriteString(attr)
					}
				}
				buffer.WriteString(">\n")
			case SELFTAG:
				if inPattern && which == OBJECTTAG && what == DOREMOVE {
					continue
				}
				buffer.WriteString("<")
				buffer.WriteString(name)
				if attr != "" {
					if which != ATTRIBTAG || what != DOREMOVE {
						attr = strings.TrimSpace(attr)
						attr = CompressRunsOfSpaces(attr)
						buffer.WriteString(" ")
						buffer.WriteString(attr)
					}
				}
				buffer.WriteString("/>\n")
			case STOPTAG:
				if name == pttrn {
					inPattern = false
					if which == OBJECTTAG && what == DOREMOVE {
						continue
					}
					if which == CONTAINERTAG && what == DOREMOVE {
						continue
					}
				}
				if inPattern && which == OBJECTTAG && what == DOREMOVE {
					continue
				}
				buffer.WriteString("</")
				buffer.WriteString(name)
				buffer.WriteString(">\n")
			case CONTENTTAG:
				if inPattern && which == OBJECTTAG && what == DOREMOVE {
					continue
				}
				if inPattern && which == CONTENTTAG && what == DOEXPAND {
					var words []string
					if strings.Contains(name, "|") {
						words = strings.FieldsFunc(name, func(c rune) bool {
							return c == '|'
						})
					} else if strings.Contains(name, ",") {
						words = strings.FieldsFunc(name, func(c rune) bool {
							return c == ','
						})
					} else {
						words = strings.Fields(name)
					}
					between := ""
					for _, item := range words {
						max := len(item)
						for max > 1 {
							ch := item[max-1]
							if ch != '.' && ch != ',' && ch != ':' && ch != ';' {
								break
							}
							// trim trailing punctuation
							item = item[:max-1]
							// continue checking for runs of punctuation at end
							max--
						}
						if HasFlankingSpace(item) {
							item = strings.TrimSpace(item)
						}
						if item != "" {
							if between != "" {
								buffer.WriteString(between)
							}
							buffer.WriteString(item)
							buffer.WriteString("\n")
							between = "</" + prevName + ">\n<" + prevName + ">\n"
						}
					}
					continue
				}
				if inPattern && which == tag {
					switch what {
					case DORETAIN:
						// default behavior for content - can use -filter X retain content as a no-op
					case DOREMOVE:
						continue
					case DOENCODE:
						name = html.EscapeString(name)
					case DODECODE:
						name = html.UnescapeString(name)
					case DOSHRINK:
						name = CompressRunsOfSpaces(name)
					case DOACCENT:
						if IsNotASCII(name) {
							name = DoAccentTransform(name)
						}
					default:
						continue
					}
				}
				// content normally printed
				if HasFlankingSpace(name) {
					name = strings.TrimSpace(name)
				}
				buffer.WriteString(name)
				buffer.WriteString("\n")
			case CDATATAG, COMMENTTAG:
				if inPattern && which == OBJECTTAG && what == DOREMOVE {
					continue
				}
				if inPattern && which == tag {
					switch what {
					case DORETAIN:
						// cdata and comment require explicit retain command
					case DOREMOVE:
						continue
					case DOENCODE:
						name = html.EscapeString(name)
					case DODECODE:
						name = html.UnescapeString(name)
					case DOSHRINK:
						name = CompressRunsOfSpaces(name)
					case DOACCENT:
						if IsNotASCII(name) {
							name = DoAccentTransform(name)
						}
					default:
						continue
					}
					// cdata and comment normally removed
					if HasFlankingSpace(name) {
						name = strings.TrimSpace(name)
					}
					buffer.WriteString(name)
					buffer.WriteString("\n")
				}
			case DOCTYPETAG:
			case NOTAG:
			case ISCLOSED:
				txt := buffer.String()
				if txt != "" {
					// print final buffer
					fmt.Fprintf(os.Stdout, "%s", txt)
				}
				return
			default:
			}

			count++
			if count > 1000 {
				count = 0
				txt := buffer.String()
				if txt != "" {
					// print current buffered output
					fmt.Fprintf(os.Stdout, "%s", txt)
				}
				buffer.Reset()
			}
		}
	}

	// processFormat reformats XML for ease of reading
	processFormat := func() {

		// skip past command name
		args = args[1:]

		copyRecrd := false
		compRecrd := false
		flushLeft := false
		wrapAttrs := false
		ret := "\n"
		frst := true

		xml := ""
		customDoctype := false
		doctype := ""

		// look for [copy|compact|flush|indent|expand] specification
		if len(args) > 0 {
			inSwitch := true

			switch args[0] {
			case "compact", "compacted", "compress", "compressed", "terse", "*":
				// compress to one record per line
				compRecrd = true
				ret = ""
			case "flush", "flushed", "left":
				// suppress line indentation
				flushLeft = true
			case "expand", "expanded", "verbose", "@":
				// each attribute on its own line
				wrapAttrs = true
			case "indent", "indented", "normal":
				// default behavior
			case "copy":
				// fast block copy
				copyRecrd = true
			default:
				// if not any of the controls, will check later for -xml and -doctype arguments
				inSwitch = false
			}

			if inSwitch {
				// skip past first argument
				args = args[1:]
			}
		}

		// copy with processing flags
		if copyRecrd {

			for {
				str := in.NextBlock()
				if str == "" {
					break
				}

				if tbls.DoStrict {
					if HasMarkup(str) {
						str = RemoveUnicodeMarkup(str)
					}
					if HasAngleBracket(str) {
						str = DoHTMLReplace(str)
					}
				}
				if tbls.DoMixed {
					if HasMarkup(str) {
						str = SimulateUnicodeMarkup(str)
					}
					if HasAngleBracket(str) {
						str = DoHTMLRepair(str)
					}
					str = DoTrimFlankingHTML(str)
				}
				if tbls.DeAccent {
					if IsNotASCII(str) {
						str = DoAccentTransform(str)
					}
				}
				if tbls.DoASCII {
					if IsNotASCII(str) {
						str = UnicodeToASCII(str)
					}
				}

				os.Stdout.WriteString(str)
			}
			os.Stdout.WriteString("\n")
			return
		}

		// look for -xml and -doctype arguments (undocumented)
		for len(args) > 0 {

			switch args[0] {
			case "-xml":
				args = args[1:]
				// -xml argument must be followed by value to use in xml line
				if len(args) < 1 || strings.HasPrefix(args[0], "-") {
					fmt.Fprintf(os.Stderr, "\nERROR: -xml argument is missing\n")
					os.Exit(1)
				}
				xml = args[0]
				args = args[1:]
			case "-doctype":
				customDoctype = true
				args = args[1:]
				if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
					// if -doctype argument followed by value, use instead of DOCTYPE line
					doctype = args[0]
					args = args[1:]
				}
			default:
				fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized option after -format command\n")
				os.Exit(1)
			}
		}

		type FormatType int

		const (
			NOTSET FormatType = iota
			START
			STOP
			CHAR
			OTHER
		)

		// array to speed up indentation
		indentSpaces := []string{
			"",
			"  ",
			"    ",
			"      ",
			"        ",
			"          ",
			"            ",
			"              ",
			"                ",
			"                  ",
		}

		indent := 0

		// parent used to detect first start tag, will place in doctype line unless overridden by -doctype argument
		parent := ""

		status := NOTSET

		// delay printing right bracket of start tag to support self-closing tag style
		needsRightBracket := ""

		// delay printing start tag if no attributes, suppress empty start-end pair if followed by end
		justStartName := ""
		justStartIndent := 0

		// indent a specified number of spaces
		doIndent := func(indt int) {
			if compRecrd || flushLeft {
				return
			}
			i := indt
			for i > 9 {
				buffer.WriteString("                    ")
				i -= 10
			}
			if i < 0 {
				return
			}
			buffer.WriteString(indentSpaces[i])
		}

		// handle delayed start tag
		doDelayedName := func() {
			if needsRightBracket != "" {
				buffer.WriteString(">")
				needsRightBracket = ""
			}
			if justStartName != "" {
				doIndent(justStartIndent)
				buffer.WriteString("<")
				buffer.WriteString(justStartName)
				buffer.WriteString(">")
				justStartName = ""
			}
		}

		closingTag := ""

		// print attributes
		printAttributes := func(attr string) {

			attr = strings.TrimSpace(attr)
			attr = CompressRunsOfSpaces(attr)
			if tbls.DeAccent {
				if IsNotASCII(attr) {
					attr = DoAccentTransform(attr)
				}
			}
			if tbls.DoASCII {
				if IsNotASCII(attr) {
					attr = UnicodeToASCII(attr)
				}
			}

			if wrapAttrs {

				start := 0
				idx := 0

				attlen := len(attr)

				for idx < attlen {
					ch := attr[idx]
					if ch == '=' {
						str := attr[start:idx]
						buffer.WriteString("\n")
						doIndent(indent)
						buffer.WriteString(" ")
						buffer.WriteString(str)
						// skip past equal sign and leading double quote
						idx += 2
						start = idx
					} else if ch == '"' {
						str := attr[start:idx]
						buffer.WriteString("=\"")
						buffer.WriteString(str)
						buffer.WriteString("\"")
						// skip past trailing double quote and (possible) space
						idx += 2
						start = idx
					} else {
						idx++
					}
				}

				buffer.WriteString("\n")
				doIndent(indent)

			} else {

				buffer.WriteString(" ")
				buffer.WriteString(attr)
			}
		}

		for {
			tag, name, attr, _, idx := nextToken(Idx)
			Idx = idx

			switch tag {
			case STARTTAG:
				doDelayedName()
				if status == START {
					buffer.WriteString(ret)
				}
				// remove internal copies of </parent><parent> tags
				if parent != "" && name == parent && indent == 1 {
					continue
				}

				// detect first start tag, print xml and doctype parent
				if indent == 0 && parent == "" {
					parent = name

					// check for xml line explicitly set in argument
					if xml != "" {
						xml = strings.TrimSpace(xml)
						if strings.HasPrefix(xml, "<") {
							xml = xml[1:]
						}
						if strings.HasPrefix(xml, "?") {
							xml = xml[1:]
						}
						if strings.HasPrefix(xml, "xml") {
							xml = xml[3:]
						}
						if strings.HasPrefix(xml, " ") {
							xml = xml[1:]
						}
						if strings.HasSuffix(xml, "?>") {
							xlen := len(xml)
							xml = xml[:xlen-2]
						}
						xml = strings.TrimSpace(xml)

						buffer.WriteString("<?xml ")
						buffer.WriteString(xml)
						buffer.WriteString("?>")
					} else {
						buffer.WriteString("<?xml version=\"1.0\"?>")
					}

					buffer.WriteString("\n")

					// check for doctype taken from XML file or explicitly set in argument
					if doctype != "" {
						doctype = strings.TrimSpace(doctype)
						if strings.HasPrefix(doctype, "<") {
							doctype = doctype[1:]
						}
						if strings.HasPrefix(doctype, "!") {
							doctype = doctype[1:]
						}
						if strings.HasPrefix(doctype, "DOCTYPE") {
							doctype = doctype[7:]
						}
						if strings.HasPrefix(doctype, " ") {
							doctype = doctype[1:]
						}
						if strings.HasSuffix(doctype, ">") {
							dlen := len(doctype)
							doctype = doctype[:dlen-1]
						}
						doctype = strings.TrimSpace(doctype)

						buffer.WriteString("<!DOCTYPE ")
						buffer.WriteString(doctype)
						buffer.WriteString(">")
					} else {
						buffer.WriteString("<!DOCTYPE ")
						buffer.WriteString(parent)
						buffer.WriteString(">")
					}

					buffer.WriteString("\n")

					// now filtering internal </parent><parent> tags, so queue printing of closing tag
					closingTag = fmt.Sprintf("</%s>\n", parent)
					// already past </parent><parent> test, so opening tag will print normally
				}

				// check for attributes
				if attr != "" {
					doIndent(indent)

					buffer.WriteString("<")
					buffer.WriteString(name)

					printAttributes(attr)

					needsRightBracket = name

				} else {
					justStartName = name
					justStartIndent = indent
				}

				if compRecrd && frst && indent == 0 {
					frst = false
					doDelayedName()
					buffer.WriteString("\n")
				}

				indent++

				status = START
			case SELFTAG:
				doDelayedName()
				if status == START {
					buffer.WriteString(ret)
				}

				// suppress self-closing tag without attributes
				if attr != "" {
					doIndent(indent)

					buffer.WriteString("<")
					buffer.WriteString(name)

					printAttributes(attr)

					buffer.WriteString("/>")
					buffer.WriteString(ret)
				}

				status = STOP
			case STOPTAG:
				// if end immediately follows start, turn into self-closing tag if there were attributes, otherwise suppress empty tag
				if needsRightBracket != "" {
					if status == START && name == needsRightBracket {
						// end immediately follows start, produce self-closing tag
						buffer.WriteString("/>")
						buffer.WriteString(ret)
						needsRightBracket = ""
						indent--
						status = STOP
						break
					}
					buffer.WriteString(">")
					needsRightBracket = ""
				}
				if justStartName != "" {
					if status == START && name == justStartName {
						// end immediately follows delayed start with no attributes, suppress
						justStartName = ""
						indent--
						status = STOP
						break
					}
					doIndent(justStartIndent)
					buffer.WriteString("<")
					buffer.WriteString(justStartName)
					buffer.WriteString(">")
					justStartName = ""
				}

				// remove internal copies of </parent><parent> tags
				if parent != "" && name == parent && indent == 1 {
					continue
				}
				indent--
				if status == CHAR {
					buffer.WriteString("</")
					buffer.WriteString(name)
					buffer.WriteString(">")
					buffer.WriteString(ret)
				} else if status == START {
					buffer.WriteString("</")
					buffer.WriteString(name)
					buffer.WriteString(">")
					buffer.WriteString(ret)
				} else {
					doIndent(indent)

					buffer.WriteString("</")
					buffer.WriteString(name)
					buffer.WriteString(">")
					buffer.WriteString(ret)
				}
				status = STOP
				if compRecrd && indent == 1 {
					buffer.WriteString("\n")
				}
			case CONTENTTAG:
				doDelayedName()
				if len(name) > 0 && IsNotJustWhitespace(name) {
					if tbls.DoStrict {
						if HasMarkup(name) {
							name = RemoveUnicodeMarkup(name)
						}
						if HasAngleBracket(name) {
							name = DoHTMLReplace(name)
						}
					}
					if tbls.DoMixed {
						if HasMarkup(name) {
							name = SimulateUnicodeMarkup(name)
						}
						if HasAngleBracket(name) {
							name = DoHTMLRepair(name)
						}
						name = DoTrimFlankingHTML(name)
					}
					if tbls.DeAccent {
						if IsNotASCII(name) {
							name = DoAccentTransform(name)
						}
					}
					if tbls.DoASCII {
						if IsNotASCII(name) {
							name = UnicodeToASCII(name)
						}
					}
					if HasFlankingSpace(name) {
						name = strings.TrimSpace(name)
					}
					buffer.WriteString(name)
					status = CHAR
				}
			case CDATATAG, COMMENTTAG:
				// ignore
			case DOCTYPETAG:
				if customDoctype && doctype == "" {
					doctype = name
				}
			case NOTAG:
			case ISCLOSED:
				doDelayedName()
				if closingTag != "" {
					buffer.WriteString(closingTag)
				}
				txt := buffer.String()
				if txt != "" {
					// print final buffer
					fmt.Fprintf(os.Stdout, "%s", txt)
				}
				return
			default:
				doDelayedName()
				status = OTHER
			}

			count++
			if count > 1000 {
				count = 0
				txt := buffer.String()
				if txt != "" {
					// print current buffered output
					fmt.Fprintf(os.Stdout, "%s", txt)
				}
				buffer.Reset()
			}
		}
	}

	// ProcessXMLStream

	// call specific function
	switch action {
	case DOFORMAT:
		processFormat()
	case DOOUTLINE:
		processOutline()
	case DOSYNOPSIS:
		processSynopsis()
	case DOVERIFY:
		processVerify()
	case DOFILTER:
		processFilter()
	default:
	}
}

// INSDSEQ EXTRACTION COMMAND GENERATOR

// e.g., xtract -insd complete mat_peptide "%peptide" product peptide

// ProcessINSD generates extraction commands for GenBank/RefSeq records in INSDSet format
func ProcessINSD(args []string, isPipe, addDash, doIndex bool) []string {

	// legal GenBank / GenPept / RefSeq features

	features := []string{
		"-10_signal",
		"-35_signal",
		"3'clip",
		"3'UTR",
		"5'clip",
		"5'UTR",
		"allele",
		"assembly_gap",
		"attenuator",
		"Bond",
		"C_region",
		"CAAT_signal",
		"CDS",
		"centromere",
		"conflict",
		"D_segment",
		"D-loop",
		"enhancer",
		"exon",
		"gap",
		"GC_signal",
		"gene",
		"iDNA",
		"intron",
		"J_segment",
		"LTR",
		"mat_peptide",
		"misc_binding",
		"misc_difference",
		"misc_feature",
		"misc_recomb",
		"misc_RNA",
		"misc_signal",
		"misc_structure",
		"mobile_element",
		"modified_base",
		"mRNA",
		"mutation",
		"N_region",
		"ncRNA",
		"old_sequence",
		"operon",
		"oriT",
		"polyA_signal",
		"polyA_site",
		"precursor_RNA",
		"prim_transcript",
		"primer_bind",
		"promoter",
		"propeptide",
		"protein_bind",
		"Protein",
		"RBS",
		"Region",
		"regulatory",
		"rep_origin",
		"repeat_region",
		"repeat_unit",
		"rRNA",
		"S_region",
		"satellite",
		"scRNA",
		"sig_peptide",
		"Site",
		"snoRNA",
		"snRNA",
		"source",
		"stem_loop",
		"STS",
		"TATA_signal",
		"telomere",
		"terminator",
		"tmRNA",
		"transit_peptide",
		"tRNA",
		"unsure",
		"V_region",
		"V_segment",
		"variation",
	}

	// legal GenBank / GenPept / RefSeq qualifiers

	qualifiers := []string{
		"allele",
		"altitude",
		"anticodon",
		"artificial_location",
		"bio_material",
		"bond_type",
		"bound_moiety",
		"breed",
		"calculated_mol_wt",
		"cell_line",
		"cell_type",
		"chloroplast",
		"chromoplast",
		"chromosome",
		"citation",
		"clone_lib",
		"clone",
		"coded_by",
		"codon_start",
		"codon",
		"collected_by",
		"collection_date",
		"compare",
		"cons_splice",
		"country",
		"cultivar",
		"culture_collection",
		"cyanelle",
		"db_xref",
		"derived_from",
		"dev_stage",
		"direction",
		"EC_number",
		"ecotype",
		"encodes",
		"endogenous_virus",
		"environmental_sample",
		"estimated_length",
		"evidence",
		"exception",
		"experiment",
		"focus",
		"frequency",
		"function",
		"gap_type",
		"gdb_xref",
		"gene_synonym",
		"gene",
		"germline",
		"haplogroup",
		"haplotype",
		"host",
		"identified_by",
		"inference",
		"insertion_seq",
		"isolate",
		"isolation_source",
		"kinetoplast",
		"lab_host",
		"label",
		"lat_lon",
		"linkage_evidence",
		"locus_tag",
		"macronuclear",
		"map",
		"mating_type",
		"metagenome_source",
		"metagenomic",
		"mitochondrion",
		"mobile_element_type",
		"mobile_element",
		"mod_base",
		"mol_type",
		"name",
		"nat_host",
		"ncRNA_class",
		"non_functional",
		"note",
		"number",
		"old_locus_tag",
		"operon",
		"organelle",
		"organism",
		"partial",
		"PCR_conditions",
		"PCR_primers",
		"peptide",
		"phenotype",
		"plasmid",
		"pop_variant",
		"product",
		"protein_id",
		"proviral",
		"pseudo",
		"pseudogene",
		"rearranged",
		"recombination_class",
		"region_name",
		"regulatory_class",
		"replace",
		"ribosomal_slippage",
		"rpt_family",
		"rpt_type",
		"rpt_unit_range",
		"rpt_unit_seq",
		"rpt_unit",
		"satellite",
		"segment",
		"sequenced_mol",
		"serotype",
		"serovar",
		"sex",
		"site_type",
		"specific_host",
		"specimen_voucher",
		"standard_name",
		"strain",
		"structural_class",
		"sub_clone",
		"sub_species",
		"sub_strain",
		"tag_peptide",
		"tissue_lib",
		"tissue_type",
		"trans_splicing",
		"transcript_id",
		"transcription",
		"transgenic",
		"transl_except",
		"transl_table",
		"translation",
		"transposon",
		"type_material",
		"UniProtKB_evidence",
		"usedin",
		"variety",
		"virion",
	}

	// legal INSDSeq XML fields

	insdtags := []string{
		"INSDAltSeqData_items",
		"INSDAltSeqData",
		"INSDAltSeqItem_first-accn",
		"INSDAltSeqItem_gap-comment",
		"INSDAltSeqItem_gap-length",
		"INSDAltSeqItem_gap-linkage",
		"INSDAltSeqItem_gap-type",
		"INSDAltSeqItem_interval",
		"INSDAltSeqItem_isgap",
		"INSDAltSeqItem_isgap@value",
		"INSDAltSeqItem_last-accn",
		"INSDAltSeqItem_value",
		"INSDAltSeqItem",
		"INSDAuthor",
		"INSDComment_paragraphs",
		"INSDComment_type",
		"INSDComment",
		"INSDCommentParagraph",
		"INSDFeature_intervals",
		"INSDFeature_key",
		"INSDFeature_location",
		"INSDFeature_operator",
		"INSDFeature_partial3",
		"INSDFeature_partial3@value",
		"INSDFeature_partial5",
		"INSDFeature_partial5@value",
		"INSDFeature_quals",
		"INSDFeature_xrefs",
		"INSDFeature",
		"INSDFeatureSet_annot-source",
		"INSDFeatureSet_features",
		"INSDFeatureSet",
		"INSDInterval_accession",
		"INSDInterval_from",
		"INSDInterval_interbp",
		"INSDInterval_interbp@value",
		"INSDInterval_iscomp",
		"INSDInterval_iscomp@value",
		"INSDInterval_point",
		"INSDInterval_to",
		"INSDInterval",
		"INSDKeyword",
		"INSDQualifier_name",
		"INSDQualifier_value",
		"INSDQualifier",
		"INSDReference_authors",
		"INSDReference_consortium",
		"INSDReference_journal",
		"INSDReference_position",
		"INSDReference_pubmed",
		"INSDReference_reference",
		"INSDReference_remark",
		"INSDReference_title",
		"INSDReference_xref",
		"INSDReference",
		"INSDSecondary-accn",
		"INSDSeq_accession-version",
		"INSDSeq_alt-seq",
		"INSDSeq_comment-set",
		"INSDSeq_comment",
		"INSDSeq_contig",
		"INSDSeq_create-date",
		"INSDSeq_create-release",
		"INSDSeq_database-reference",
		"INSDSeq_definition",
		"INSDSeq_division",
		"INSDSeq_entry-version",
		"INSDSeq_feature-set",
		"INSDSeq_feature-table",
		"INSDSeq_keywords",
		"INSDSeq_length",
		"INSDSeq_locus",
		"INSDSeq_moltype",
		"INSDSeq_organism",
		"INSDSeq_other-seqids",
		"INSDSeq_primary-accession",
		"INSDSeq_primary",
		"INSDSeq_project",
		"INSDSeq_references",
		"INSDSeq_secondary-accessions",
		"INSDSeq_segment",
		"INSDSeq_sequence",
		"INSDSeq_source-db",
		"INSDSeq_source",
		"INSDSeq_strandedness",
		"INSDSeq_struc-comments",
		"INSDSeq_taxonomy",
		"INSDSeq_topology",
		"INSDSeq_update-date",
		"INSDSeq_update-release",
		"INSDSeq_xrefs",
		"INSDSeq",
		"INSDSeqid",
		"INSDSet",
		"INSDStrucComment_items",
		"INSDStrucComment_name",
		"INSDStrucComment",
		"INSDStrucCommentItem_tag",
		"INSDStrucCommentItem_url",
		"INSDStrucCommentItem_value",
		"INSDStrucCommentItem",
		"INSDXref_dbname",
		"INSDXref_id",
		"INSDXref",
	}

	checkAgainstVocabulary := func(str, objtype string, arry []string) {

		if str == "" || arry == nil {
			return
		}

		// skip past pound, percent, or caret character at beginning of string
		if len(str) > 1 {
			switch str[0] {
			case '#', '%', '^':
				str = str[1:]
			default:
			}
		}

		for _, txt := range arry {
			if str == txt {
				return
			}
			if strings.ToUpper(str) == strings.ToUpper(txt) {
				fmt.Fprintf(os.Stderr, "\nERROR: Incorrect capitalization of '%s' %s, change to '%s'\n", str, objtype, txt)
				os.Exit(1)
			}
		}

		fmt.Fprintf(os.Stderr, "\nERROR: Item '%s' is not a legal -insd %s\n", str, objtype)
		os.Exit(1)
	}

	var acc []string

	max := len(args)
	if max < 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Insufficient command-line arguments supplied to xtract -insd\n")
		os.Exit(1)
	}

	if doIndex {
		if isPipe {
			acc = append(acc, "-head", "<IdxDocumentSet>", "-tail", "</IdxDocumentSet>")
			acc = append(acc, "-hd", "  <IdxDocument>\n", "-tl", "  </IdxDocument>")
			acc = append(acc, "-pattern", "INSDSeq", "-pfx", "    <IdxUid>", "-sfx", "</IdxUid>\n")
			acc = append(acc, "-element", "INSDSeq_accession-version", "-clr", "-rst", "-tab", "\n")
		} else {
			acc = append(acc, "-head", "\"<IdxDocumentSet>\"", "-tail", "\"</IdxDocumentSet>\"")
			acc = append(acc, "-hd", "\"  <IdxDocument>\\n\"", "-tl", "\"  </IdxDocument>\"")
			acc = append(acc, "-pattern", "INSDSeq", "-pfx", "\"    <IdxUid>\"", "-sfx", "\"</IdxUid>\\n\"")
			acc = append(acc, "-element", "INSDSeq_accession-version", "-clr", "-rst", "-tab", "\\n")
		}
	} else {
		acc = append(acc, "-pattern", "INSDSeq", "-ACCN", "INSDSeq_accession-version")
	}

	if doIndex {
		if isPipe {
			acc = append(acc, "-group", "INSDSeq", "-lbl", "    <IdxSearchFields>\n")
		} else {
			acc = append(acc, "-group", "INSDSeq", "-lbl", "\"    <IdxSearchFields>\\n\"")
		}
	}

	printAccn := true

	// collect descriptors

	if strings.HasPrefix(args[0], "INSD") {

		if doIndex {
			acc = append(acc, "-clr", "-indices")
		} else {
			if isPipe {
				acc = append(acc, "-clr", "-pfx", "\\n", "-element", "&ACCN")
				acc = append(acc, "-group", "INSDSeq", "-sep", "|", "-element")
			} else {
				acc = append(acc, "-clr", "-pfx", "\"\\n\"", "-element", "\"&ACCN\"")
				acc = append(acc, "-group", "INSDSeq", "-sep", "\"|\"", "-element")
			}
			printAccn = false
		}

		for {
			if len(args) < 1 {
				return acc
			}
			str := args[0]
			if !strings.HasPrefix(args[0], "INSD") {
				break
			}
			checkAgainstVocabulary(str, "element", insdtags)
			acc = append(acc, str)
			args = args[1:]
		}

	} else if strings.HasPrefix(strings.ToUpper(args[0]), "INSD") {

		// report capitalization or vocabulary failure
		checkAgainstVocabulary(args[0], "element", insdtags)

		// program should not get to this point, but warn and exit anyway
		fmt.Fprintf(os.Stderr, "\nERROR: Item '%s' is not a legal -insd %s\n", args[0], "element")
		os.Exit(1)
	}

	// collect qualifiers

	partial := false
	complete := false

	if args[0] == "+" || args[0] == "complete" {
		complete = true
		args = args[1:]
		max--
	} else if args[0] == "-" || args[0] == "partial" {
		partial = true
		args = args[1:]
		max--
	}

	if max < 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: No feature key supplied to xtract -insd\n")
		os.Exit(1)
	}

	acc = append(acc, "-group", "INSDFeature")

	// limit to designated features

	feature := args[0]

	fcmd := "-if"

	// can specify multiple features separated by plus sign (e.g., CDS+mRNA) or comma (e.g., CDS,mRNA)
	plus := strings.Split(feature, "+")
	for _, pls := range plus {
		comma := strings.Split(pls, ",")
		for _, cma := range comma {

			checkAgainstVocabulary(cma, "feature", features)
			acc = append(acc, fcmd, "INSDFeature_key", "-equals", cma)

			fcmd = "-or"
		}
	}

	if max < 2 {
		// still need at least one qualifier even on legal feature
		fmt.Fprintf(os.Stderr, "\nERROR: Feature '%s' must be followed by at least one qualifier\n", feature)
		os.Exit(1)
	}

	args = args[1:]

	if complete {
		acc = append(acc, "-unless", "INSDFeature_partial5", "-or", "INSDFeature_partial3")
	} else if partial {
		acc = append(acc, "-if", "INSDFeature_partial5", "-or", "INSDFeature_partial3")
	}

	if printAccn {
		if doIndex {
		} else {
			if isPipe {
				acc = append(acc, "-clr", "-pfx", "\\n", "-element", "&ACCN")
			} else {
				acc = append(acc, "-clr", "-pfx", "\"\\n\"", "-element", "\"&ACCN\"")
			}
		}
	}

	for _, str := range args {
		if strings.HasPrefix(str, "INSD") {

			checkAgainstVocabulary(str, "element", insdtags)
			if doIndex {
				acc = append(acc, "-block", "INSDFeature", "-clr", "-indices")
			} else {
				if isPipe {
					acc = append(acc, "-block", "INSDFeature", "-sep", "|", "-element")
				} else {
					acc = append(acc, "-block", "INSDFeature", "-sep", "\"|\"", "-element")
				}
			}
			acc = append(acc, str)
			if addDash {
				acc = append(acc, "-block", "INSDFeature", "-unless", str)
				if strings.HasSuffix(str, "@value") {
					if isPipe {
						acc = append(acc, "-lbl", "false")
					} else {
						acc = append(acc, "-lbl", "\"false\"")
					}
				} else {
					if isPipe {
						acc = append(acc, "-lbl", "\\-")
					} else {
						acc = append(acc, "-lbl", "\"\\-\"")
					}
				}
			}

		} else if strings.HasPrefix(str, "#INSD") {

			checkAgainstVocabulary(str, "element", insdtags)
			if doIndex {
				acc = append(acc, "-block", "INSDFeature", "-clr", "-indices")
			} else {
				if isPipe {
					acc = append(acc, "-block", "INSDFeature", "-sep", "|", "-element")
					acc = append(acc, str)
				} else {
					acc = append(acc, "-block", "INSDFeature", "-sep", "\"|\"", "-element")
					ql := fmt.Sprintf("\"%s\"", str)
					acc = append(acc, ql)
				}
			}

		} else if strings.HasPrefix(strings.ToUpper(str), "#INSD") || strings.HasPrefix(strings.ToUpper(str), "#INSD") {

			// report capitalization or vocabulary failure
			checkAgainstVocabulary(str, "element", insdtags)

		} else {

			acc = append(acc, "-block", "INSDQualifier")

			checkAgainstVocabulary(str, "qualifier", qualifiers)
			if len(str) > 2 && str[0] == '%' {
				acc = append(acc, "-if", "INSDQualifier_name", "-equals", str[1:])
				if doIndex {
					if isPipe {
						acc = append(acc, "-clr", "-indices", "%INSDQualifier_value")
					} else {
						acc = append(acc, "-clr", "-indices", "\"%INSDQualifier_value\"")
					}
				} else {
					if isPipe {
						acc = append(acc, "-element", "%INSDQualifier_value")
					} else {
						acc = append(acc, "-element", "\"%INSDQualifier_value\"")
					}
				}
				if addDash {
					acc = append(acc, "-block", "INSDFeature", "-unless", "INSDQualifier_name", "-equals", str[1:])
					if isPipe {
						acc = append(acc, "-lbl", "\\-")
					} else {
						acc = append(acc, "-lbl", "\"\\-\"")
					}
				}
			} else {
				if doIndex {
					acc = append(acc, "-if", "INSDQualifier_name", "-equals", str)
					acc = append(acc, "-clr", "-indices", "INSDQualifier_value")
				} else {
					acc = append(acc, "-if", "INSDQualifier_name", "-equals", str)
					acc = append(acc, "-element", "INSDQualifier_value")
				}
				if addDash {
					acc = append(acc, "-block", "INSDFeature", "-unless", "INSDQualifier_name", "-equals", str)
					if isPipe {
						acc = append(acc, "-lbl", "\\-")
					} else {
						acc = append(acc, "-lbl", "\"\\-\"")
					}
				}
			}
		}
	}

	if doIndex {
		if isPipe {
			acc = append(acc, "-group", "INSDSeq", "-clr", "-lbl", "    </IdxSearchFields>\n")
		} else {
			acc = append(acc, "-group", "INSDSeq", "-clr", "-lbl", "\"    </IdxSearchFields>\\n\"")
		}
	}

	return acc
}

// HYDRA CITATION MATCHER COMMAND GENERATOR

// ProcessHydra generates extraction commands for NCBI's in-house citation matcher (undocumented)
func ProcessHydra(isPipe bool) []string {

	var acc []string

	// acceptable scores are 0.8 or higher, exact match on "1" rejects low value in scientific notation with minus sign present

	acc = append(acc, "-pattern", "Id")
	acc = append(acc, "-if", "@score", "-equals", "1")
	acc = append(acc, "-or", "@score", "-starts-with", "0.9")
	acc = append(acc, "-or", "@score", "-starts-with", "0.8")
	acc = append(acc, "-element", "Id")

	return acc
}

// ENTREZ2INDEX COMMAND GENERATOR

// ProcessE2Index generates extraction commands to create input for Entrez2Index (undocumented)
func ProcessE2Index(args []string, isPipe bool) []string {

	var acc []string

	max := len(args)
	if max < 3 {
		fmt.Fprintf(os.Stderr, "\nERROR: Insufficient command-line arguments supplied to xtract -e2index\n")
		os.Exit(1)
	}

	patrn := args[0]
	ident := args[1]

	args = args[2:]

	if isPipe {
		acc = append(acc, "-head", "<IdxDocumentSet>", "-tail", "</IdxDocumentSet>")
		acc = append(acc, "-hd", "  <IdxDocument>\\n", "-tl", "  </IdxDocument>")
		acc = append(acc, "-pattern")
		ql := fmt.Sprintf("\"%s\"", patrn)
		acc = append(acc, ql)
		acc = append(acc, "-pfx", "    <IdxUid>", "-sfx", "</IdxUid>\\n")
		acc = append(acc, "-element")
		ql = fmt.Sprintf("\"%s\"", ident)
		acc = append(acc, ql)
		acc = append(acc, "-clr", "-rst", "-tab", "")
		acc = append(acc, "-lbl", "    <IdxSearchFields>\\n")
		acc = append(acc, "-indices")
		for _, str := range args {
			ql = fmt.Sprintf("\"%s\"", str)
			acc = append(acc, ql)
		}
		acc = append(acc, "-clr", "-lbl", "    </IdxSearchFields>\\n")
	} else {
		acc = append(acc, "-head", "\"<IdxDocumentSet>\"", "-tail", "\"</IdxDocumentSet>\"")
		acc = append(acc, "-hd", "\"  <IdxDocument>\\n\"", "-tl", "\"  </IdxDocument>\"")
		acc = append(acc, "-pattern")
		ql := fmt.Sprintf("\"%s\"", patrn)
		acc = append(acc, ql)
		acc = append(acc, "-pfx", "\"    <IdxUid>\"", "-sfx", "\"</IdxUid>\\n\"")
		acc = append(acc, "-element")
		ql = fmt.Sprintf("\"%s\"", ident)
		acc = append(acc, ql)
		acc = append(acc, "-clr", "-rst", "-tab", "\"\"")
		acc = append(acc, "-lbl", "\"    <IdxSearchFields>\\n\"")
		acc = append(acc, "-indices")
		for _, str := range args {
			ql = fmt.Sprintf("\"%s\"", str)
			acc = append(acc, ql)
		}
		acc = append(acc, "-clr", "-lbl", "\"    </IdxSearchFields>\\n\"")
	}

	return acc
}

// COLLECT AND FORMAT REQUESTED XML VALUES

// ParseAttributes is only run if attribute values are requested in element statements
func ParseAttributes(attrb string) []string {

	if attrb == "" {
		return nil
	}

	attlen := len(attrb)

	// count equal signs
	num := 0
	for i := 0; i < attlen; i++ {
		if attrb[i] == '=' {
			num += 2
		}
	}
	if num < 1 {
		return nil
	}

	// allocate array of proper size
	arry := make([]string, num)
	if arry == nil {
		return nil
	}

	start := 0
	idx := 0
	itm := 0

	// place tag and value in successive array slots
	for idx < attlen && itm < num {
		ch := attrb[idx]
		if ch == '=' {
			// skip past possible leading blanks
			for start < attlen {
				ch = attrb[start]
				if ch == ' ' || ch == '\n' || ch == '\t' || ch == '\r' || ch == '\f' {
					start++
				} else {
					break
				}
			}
			// =
			arry[itm] = attrb[start:idx]
			itm++
			// skip past equal sign and leading double quote
			idx += 2
			start = idx
		} else if ch == '"' {
			// "
			arry[itm] = attrb[start:idx]
			itm++
			// skip past trailing double quote and (possible) space
			idx += 2
			start = idx
		} else {
			idx++
		}
	}

	return arry
}

// ExploreElements returns matching element values to callback
func ExploreElements(curr *Node, mask, prnt, match, attrib string, wildcard bool, level int, proc func(string, int)) {

	if curr == nil || proc == nil {
		return
	}

	// **/Object performs deep exploration of recursive data (*/Object also supported)
	deep := false
	if prnt == "**" || prnt == "*" {
		prnt = ""
		deep = true
	}

	// exploreElements recursive definition
	var exploreElements func(curr *Node, skip string, lev int)

	exploreElements = func(curr *Node, skip string, lev int) {

		if !deep && curr.Name == skip {
			// do not explore within recursive object
			return
		}

		// wildcard matches any namespace prefix
		if curr.Name == match ||
			(wildcard && strings.HasPrefix(match, ":") && strings.HasSuffix(curr.Name, match)) ||
			(match == "" && attrib != "") {

			if prnt == "" ||
				curr.Parent == prnt ||
				(wildcard && strings.HasPrefix(prnt, ":") && strings.HasSuffix(curr.Parent, prnt)) {

				if attrib != "" {
					if curr.Attributes != "" && curr.Attribs == nil {
						// parse attributes on-the-fly if queried
						curr.Attribs = ParseAttributes(curr.Attributes)
					}
					for i := 0; i < len(curr.Attribs)-1; i += 2 {
						// attributes now parsed into array as [ tag, value, tag, value, tag, value, ... ]
						if curr.Attribs[i] == attrib ||
							(wildcard && strings.HasPrefix(attrib, ":") && strings.HasSuffix(curr.Attribs[i], attrib)) {
							proc(curr.Attribs[i+1], level)
							return
						}
					}

				} else if curr.Contents != "" {

					str := curr.Contents[:]

					if HasAmpOrNotASCII(str) {
						// processing of <, >, &, ", and ' characters is now delayed until element contents is requested
						str = html.UnescapeString(str)
					}

					proc(str, level)
					return

				} else if curr.Children != nil {

					// for XML container object, send empty string to callback to increment count
					proc("", level)
					// and continue exploring

				} else if curr.Attributes != "" {

					// for self-closing object, indicate presence by sending empty string to callback
					proc("", level)
					return
				}
			}
		}

		for chld := curr.Children; chld != nil; chld = chld.Next {
			// inner exploration is subject to recursive object exclusion
			exploreElements(chld, mask, lev+1)
		}
	}

	exploreElements(curr, "", level)
}

// PrintSubtree supports compression styles selected by -element "*" through "****"
func PrintSubtree(node *Node, style IndentType, printAttrs bool, proc func(string)) {

	if node == nil || proc == nil {
		return
	}

	// WRAPPED is SUBTREE plus each attribute on its own line
	wrapped := false
	if style == WRAPPED {
		style = SUBTREE
		wrapped = true
	}

	// INDENT is offset by two spaces to allow for parent tag, SUBTREE is not offset
	initial := 1
	if style == SUBTREE {
		style = INDENT
		initial = 0
	}

	// array to speed up indentation
	indentSpaces := []string{
		"",
		"  ",
		"    ",
		"      ",
		"        ",
		"          ",
		"            ",
		"              ",
		"                ",
		"                  ",
	}

	// indent a specified number of spaces
	doIndent := func(indt int) {
		i := indt
		for i > 9 {
			proc("                    ")
			i -= 10
		}
		if i < 0 {
			return
		}
		proc(indentSpaces[i])
	}

	// doSubtree recursive definition
	var doSubtree func(*Node, int)

	doSubtree = func(curr *Node, depth int) {

		// suppress if it would be an empty self-closing tag
		if !IsNotJustWhitespace(curr.Attributes) && curr.Contents == "" && curr.Children == nil {
			return
		}

		if style == INDENT {
			doIndent(depth)
		}

		proc("<")
		proc(curr.Name)

		if printAttrs {

			attr := strings.TrimSpace(curr.Attributes)
			attr = CompressRunsOfSpaces(attr)

			if attr != "" {

				if wrapped {

					start := 0
					idx := 0

					attlen := len(attr)

					for idx < attlen {
						ch := attr[idx]
						if ch == '=' {
							str := attr[start:idx]
							proc("\n")
							doIndent(depth)
							proc(" ")
							proc(str)
							// skip past equal sign and leading double quote
							idx += 2
							start = idx
						} else if ch == '"' {
							str := attr[start:idx]
							proc("=\"")
							proc(str)
							proc("\"")
							// skip past trailing double quote and (possible) space
							idx += 2
							start = idx
						} else {
							idx++
						}
					}

					proc("\n")
					doIndent(depth)

				} else {

					proc(" ")
					proc(attr)
				}
			}
		}

		// see if suitable for for self-closing tag
		if curr.Contents == "" && curr.Children == nil {
			proc("/>")
			if style != COMPACT {
				proc("\n")
			}
			return
		}

		proc(">")

		if curr.Contents != "" {

			proc(curr.Contents[:])

		} else {

			if style != COMPACT {
				proc("\n")
			}

			for chld := curr.Children; chld != nil; chld = chld.Next {
				doSubtree(chld, depth+1)
			}

			if style == INDENT {
				i := depth
				for i > 9 {
					proc("                    ")
					i -= 10
				}
				proc(indentSpaces[i])
			}
		}

		proc("<")
		proc("/")
		proc(curr.Name)
		proc(">")

		if style != COMPACT {
			proc("\n")
		}
	}

	doSubtree(node, initial)
}

// ProcessClause handles comma-separated -element arguments
func ProcessClause(curr *Node, stages []*Step, mask, prev, pfx, sfx, sep, def string, status OpType, index, level int, variables map[string]string) (string, bool) {

	if curr == nil || stages == nil {
		return "", false
	}

	// processElement handles individual -element constructs
	processElement := func(acc func(string)) {

		if acc == nil {
			return
		}

		// element names combined with commas are treated as a prefix-separator-suffix group
		for _, stage := range stages {

			stat := stage.Type
			item := stage.Value
			prnt := stage.Parent
			match := stage.Match
			attrib := stage.Attrib
			wildcard := stage.Wild

			// exploreElements is a wrapper for ExploreElements, obtaining most arguments as closures
			exploreElements := func(proc func(string, int)) {
				ExploreElements(curr, mask, prnt, match, attrib, wildcard, level, proc)
			}

			switch stat {
			case ELEMENT, TERMS, WORDS, PAIRS, LETTERS, INDICES, VALUE, LEN, SUM, MIN, MAX, SUB, AVG, DEV:
				exploreElements(func(str string, lvl int) {
					if str != "" {
						acc(str)
					}
				})
			case FIRST:
				single := ""

				exploreElements(func(str string, lvl int) {
					if single == "" {
						single = str
					}
				})

				if single != "" {
					acc(single)
				}
			case LAST:
				single := ""

				exploreElements(func(str string, lvl int) {
					single = str
				})

				if single != "" {
					acc(single)
				}
			case ENCODE:
				exploreElements(func(str string, lvl int) {
					if str != "" {
						str = html.EscapeString(str)
						acc(str)
					}
				})
			case UPPER:
				exploreElements(func(str string, lvl int) {
					if str != "" {
						str = strings.ToUpper(str)
						acc(str)
					}
				})
			case LOWER:
				exploreElements(func(str string, lvl int) {
					if str != "" {
						str = strings.ToLower(str)
						acc(str)
					}
				})
			case TITLE:
				exploreElements(func(str string, lvl int) {
					if str != "" {
						str = strings.ToLower(str)
						str = strings.Title(str)
						acc(str)
					}
				})
			case VARIABLE:
				// use value of stored variable
				val, ok := variables[match]
				if ok {
					acc(val)
				}
			case NUM, COUNT:
				count := 0

				exploreElements(func(str string, lvl int) {
					count++
				})

				// number of element objects
				val := strconv.Itoa(count)
				acc(val)
			case LENGTH:
				length := 0

				exploreElements(func(str string, lvl int) {
					length += len(str)
				})

				// length of element strings
				val := strconv.Itoa(length)
				acc(val)
			case DEPTH:
				exploreElements(func(str string, lvl int) {
					// depth of each element in scope
					val := strconv.Itoa(lvl)
					acc(val)
				})
			case INDEX:
				// -element "+" prints index of current XML object
				val := strconv.Itoa(index)
				acc(val)
			case INC:
				// -inc, or component of -0-based, -1-based, or -ucsc-based
				exploreElements(func(str string, lvl int) {
					if str != "" {
						num, err := strconv.Atoi(str)
						if err == nil {
							// increment value
							num++
							val := strconv.Itoa(num)
							acc(val)
						}
					}
				})
			case DEC:
				// -dec, or component of -0-based, -1-based, or -ucsc-based
				exploreElements(func(str string, lvl int) {
					if str != "" {
						num, err := strconv.Atoi(str)
						if err == nil {
							// decrement value
							num--
							val := strconv.Itoa(num)
							acc(val)
						}
					}
				})
			case STAR:
				// -element "*" prints current XML subtree on a single line
				style := SINGULARITY
				printAttrs := true

				for _, ch := range item {
					if ch == '*' {
						style++
					} else if ch == '@' {
						printAttrs = false
					}
				}
				if style > WRAPPED {
					style = WRAPPED
				}
				if style < COMPACT {
					style = COMPACT
				}

				var buffer bytes.Buffer

				PrintSubtree(curr, style, printAttrs,
					func(str string) {
						if str != "" {
							buffer.WriteString(str)
						}
					})

				txt := buffer.String()
				if txt != "" {
					acc(txt)
				}
			case DOLLAR:
				for chld := curr.Children; chld != nil; chld = chld.Next {
					acc(chld.Name)
				}
			case ATSIGN:
				if curr.Attributes != "" && curr.Attribs == nil {
					curr.Attribs = ParseAttributes(curr.Attributes)
				}
				for i := 0; i < len(curr.Attribs)-1; i += 2 {
					acc(curr.Attribs[i])
				}
			default:
			}
		}
	}

	ok := false

	// format results in buffer
	var buffer bytes.Buffer

	buffer.WriteString(prev)
	buffer.WriteString(pfx)
	between := ""

	switch status {
	case ELEMENT, ENCODE, UPPER, LOWER, TITLE, VALUE, NUM, INC, DEC, ZEROBASED, ONEBASED, UCSCBASED:
		processElement(func(str string) {
			if str != "" {
				ok = true
				buffer.WriteString(between)
				buffer.WriteString(str)
				between = sep
			}
		})
	case FIRST:
		single := ""

		processElement(func(str string) {
			ok = true
			if single == "" {
				single = str
			}
		})

		if single != "" {
			buffer.WriteString(between)
			buffer.WriteString(single)
			between = sep
		}
	case LAST:
		single := ""

		processElement(func(str string) {
			ok = true
			single = str
		})

		if single != "" {
			buffer.WriteString(between)
			buffer.WriteString(single)
			between = sep
		}
	case TERMS:
		processElement(func(str string) {
			if str != "" {
				words := strings.Fields(str)
				for _, item := range words {
					max := len(item)
					for max > 1 {
						ch := item[max-1]
						if ch != '.' && ch != ',' && ch != ':' && ch != ';' {
							break
						}
						// trim trailing period, comma, colon, and semicolon
						item = item[:max-1]
						// continue checking for runs of punctuation at end
						max--
					}
					ok = true
					buffer.WriteString(between)
					buffer.WriteString(item)
					between = sep
				}
			}
		})
	case WORDS:
		processElement(func(str string) {
			if str != "" {
				words := strings.FieldsFunc(str, func(c rune) bool {
					return !unicode.IsLetter(c) && !unicode.IsDigit(c)
				})
				for _, item := range words {
					item = strings.ToLower(item)
					ok = true
					buffer.WriteString(between)
					buffer.WriteString(item)
					between = sep
				}
			}
		})
	case PAIRS:
		processElement(func(str string) {
			if str != "" {
				words := strings.FieldsFunc(str, func(c rune) bool {
					return !unicode.IsLetter(c) && !unicode.IsDigit(c)
				})
				if len(words) > 1 {
					past := ""
					for _, item := range words {
						item = strings.ToLower(item)
						plock.RLock()
						isSW := isStopWord[item]
						plock.RUnlock()
						if isSW {
							past = ""
							continue
						}
						if past != "" {
							ok = true
							buffer.WriteString(between)
							buffer.WriteString(past + " " + item)
							between = sep
						}
						past = item
					}
				}
			}
		})
	case LETTERS:
		processElement(func(str string) {
			if str != "" {
				for _, ch := range str {
					ok = true
					buffer.WriteString(between)
					buffer.WriteRune(ch)
					between = sep
				}
			}
		})
	case INDICES:
		var term []string
		var pair []string

		addToIndex := func(item, past string) string {

			if item == "" {
				return ""
			}
			plock.RLock()
			isSW := isStopWord[item]
			plock.RUnlock()
			if isSW {
				// skip if stop word, interrupts overlapping word pair chain
				return ""
			}
			ok = true
			item = html.EscapeString(item)
			// add single term
			term = append(term, item)
			if past != "" {
				// add informative adjacent word pair
				pair = append(pair, past+" "+item)
			}

			return item
		}

		processElement(func(str string) {
			if str != "" {
				if IsNotASCII(str) {
					str = DoAccentTransform(str)
				}
				str = strings.ToLower(str)
				if HasBadSpace(str) {
					str = CleanupBadSpaces(str)
				}
				if HasMarkup(str) {
					str = RemoveUnicodeMarkup(str)
				}
				if HasAngleBracket(str) {
					str = DoHTMLReplace(str)
				}

				// break terms at spaces, allowing hyphenated terms
				terms := strings.Fields(str)
				for _, item := range terms {
					item = html.UnescapeString(item)
					// allow parentheses in chemical formula
					item = TrimPunctuation(item)
					// skip numbers
					if IsAllNumeric(item) {
						continue
					}
					// index single term
					addToIndex(item, "")
				}

				// break words at non-alphanumeric punctuation
				words := strings.FieldsFunc(str, func(c rune) bool {
					return !unicode.IsLetter(c) && !unicode.IsDigit(c)
				})
				past := ""
				for _, item := range words {
					// skip anything starting with a digit
					if len(item) < 1 || unicode.IsDigit(rune(item[0])) {
						past = ""
						continue
					}
					// index word and adjacent word pairs
					past = addToIndex(item, past)
				}
			}
		})

		if ok {
			// sort arrays of words and pairs
			sort.Slice(term, func(i, j int) bool { return term[i] < term[j] })
			sort.Slice(pair, func(i, j int) bool { return pair[i] < pair[j] })

			last := ""
			for _, item := range term {
				if item == last {
					// skip duplicate entry
					continue
				}
				buffer.WriteString("      <NORM>")
				buffer.WriteString(item)
				buffer.WriteString("</NORM>\n")
				last = item
			}

			last = ""
			for _, item := range pair {
				if item == last {
					// skip duplicate entry
					continue
				}
				buffer.WriteString("      <PAIR>")
				buffer.WriteString(item)
				buffer.WriteString("</PAIR>\n")
				last = item
			}
		}
	case LEN:
		length := 0

		processElement(func(str string) {
			ok = true
			length += len(str)
		})

		// length of element strings
		val := strconv.Itoa(length)
		buffer.WriteString(between)
		buffer.WriteString(val)
		between = sep
	case SUM:
		sum := 0

		processElement(func(str string) {
			value, err := strconv.Atoi(str)
			if err == nil {
				sum += value
				ok = true
			}
		})

		if ok {
			// sum of element values
			val := strconv.Itoa(sum)
			buffer.WriteString(between)
			buffer.WriteString(val)
			between = sep
		}
	case MIN:
		min := 0

		processElement(func(str string) {
			value, err := strconv.Atoi(str)
			if err == nil {
				if !ok || value < min {
					min = value
				}
				ok = true
			}
		})

		if ok {
			// minimum of element values
			val := strconv.Itoa(min)
			buffer.WriteString(between)
			buffer.WriteString(val)
			between = sep
		}
	case MAX:
		max := 0

		processElement(func(str string) {
			value, err := strconv.Atoi(str)
			if err == nil {
				if !ok || value > max {
					max = value
				}
				ok = true
			}
		})

		if ok {
			// maximum of element values
			val := strconv.Itoa(max)
			buffer.WriteString(between)
			buffer.WriteString(val)
			between = sep
		}
	case SUB:
		first := 0
		second := 0
		count := 0

		processElement(func(str string) {
			value, err := strconv.Atoi(str)
			if err == nil {
				count++
				if count == 1 {
					first = value
				} else if count == 2 {
					second = value
				}
			}
		})

		if count == 2 {
			// must have exactly 2 elements
			ok = true
			// difference of element values
			val := strconv.Itoa(first - second)
			buffer.WriteString(between)
			buffer.WriteString(val)
			between = sep
		}
	case AVG:
		sum := 0
		count := 0

		processElement(func(str string) {
			value, err := strconv.Atoi(str)
			if err == nil {
				sum += value
				count++
				ok = true
			}
		})

		if ok {
			// average of element values
			avg := int(float64(sum) / float64(count))
			val := strconv.Itoa(avg)
			buffer.WriteString(between)
			buffer.WriteString(val)
			between = sep
		}
	case DEV:
		count := 0
		mean := 0.0
		m2 := 0.0

		processElement(func(str string) {
			value, err := strconv.Atoi(str)
			if err == nil {
				// Welford algorithm for one-pass standard deviation
				count++
				x := float64(value)
				delta := x - mean
				mean += delta / float64(count)
				m2 += delta * (x - mean)
			}
		})

		if count > 1 {
			// must have at least 2 elements
			ok = true
			// standard deviation of element values
			vrc := m2 / float64(count-1)
			dev := int(math.Sqrt(vrc))
			val := strconv.Itoa(dev)
			buffer.WriteString(between)
			buffer.WriteString(val)
			between = sep
		}
	default:
	}

	// use default value if nothing written
	if !ok && def != "" {
		ok = true
		buffer.WriteString(def)
	}

	buffer.WriteString(sfx)

	if !ok {
		return "", false
	}

	txt := buffer.String()

	return txt, true
}

// ProcessInstructions performs extraction commands on a subset of XML
func ProcessInstructions(commands []*Operation, curr *Node, mask, tab, ret string, index, level int, variables map[string]string, accum func(string)) (string, string) {

	if accum == nil {
		return tab, ret
	}

	sep := "\t"
	pfx := ""
	sfx := ""

	def := ""

	col := "\t"
	lin := "\n"

	varname := ""

	// process commands
	for _, op := range commands {

		str := op.Value

		switch op.Type {
		case ELEMENT, FIRST, LAST, ENCODE, UPPER, LOWER, TITLE, TERMS, WORDS, PAIRS, LETTERS, INDICES,
			NUM, LEN, SUM, MIN, MAX, INC, DEC, SUB, AVG, DEV, ZEROBASED, ONEBASED, UCSCBASED:
			txt, ok := ProcessClause(curr, op.Stages, mask, tab, pfx, sfx, sep, def, op.Type, index, level, variables)
			if ok {
				tab = col
				ret = lin
				accum(txt)
			}
		case TAB:
			col = str
		case RET:
			lin = str
		case PFX:
			pfx = str
		case SFX:
			sfx = str
		case SEP:
			sep = str
		case LBL:
			lbl := str
			accum(tab)
			accum(lbl)
			tab = col
			ret = lin
		case PFC:
			// preface clears previous tab and sets prefix in one command
			pfx = str
			fallthrough
		case CLR:
			// clear previous tab after the fact
			tab = ""
		case RST:
			pfx = ""
			sfx = ""
			sep = "\t"
			def = ""
		case DEF:
			def = str
		case VARIABLE:
			varname = str
		case VALUE:
			length := len(str)
			if length > 1 && str[0] == '(' && str[length-1] == ')' {
				// set variable from literal text inside parentheses, e.g., -COM "(, )"
				variables[varname] = str[1 : length-1]
				// -if "&VARIABLE" will succeed if set to blank with empty parentheses "()"
			} else if str == "" {
				// -if "&VARIABLE" will fail if initialized with empty string ""
				delete(variables, varname)
			} else {
				txt, ok := ProcessClause(curr, op.Stages, mask, "", pfx, sfx, sep, def, op.Type, index, level, variables)
				if ok {
					variables[varname] = txt
				}
			}
			varname = ""
		default:
		}
	}

	return tab, ret
}

// CONDITIONAL EXECUTION USES -if AND -unless STATEMENT, WITH SUPPORT FOR DEPRECATED -match AND -avoid STATEMENTS

// ConditionsAreSatisfied tests a set of conditions to determine if extraction should proceed
func ConditionsAreSatisfied(conditions []*Operation, curr *Node, mask string, index, level int, variables map[string]string) bool {

	if curr == nil {
		return false
	}

	required := 0
	observed := 0
	forbidden := 0
	isMatch := false
	isAvoid := false

	// test string or numeric constraints
	testConstraint := func(str string, constraint *Step) bool {

		if str == "" || constraint == nil {
			return false
		}

		val := constraint.Value
		stat := constraint.Type

		switch stat {
		case EQUALS, CONTAINS, STARTSWITH, ENDSWITH, ISNOT:
			// substring test on element values
			str = strings.ToUpper(str)
			val = strings.ToUpper(val)

			switch stat {
			case EQUALS:
				if str == val {
					return true
				}
			case CONTAINS:
				if strings.Contains(str, val) {
					return true
				}
			case STARTSWITH:
				if strings.HasPrefix(str, val) {
					return true
				}
			case ENDSWITH:
				if strings.HasSuffix(str, val) {
					return true
				}
			case ISNOT:
				if str != val {
					return true
				}
			default:
			}
		case GT, GE, LT, LE, EQ, NE:
			// second argument of numeric test can be element specifier
			if constraint.Parent != "" || constraint.Match != "" || constraint.Attrib != "" {
				ch := val[0]
				// pound, percent, and caret prefixes supported as potentially useful for data QA (undocumented)
				switch ch {
				case '#':
					count := 0
					ExploreElements(curr, mask, constraint.Parent, constraint.Match, constraint.Attrib, constraint.Wild, level, func(stn string, lvl int) {
						count++
					})
					val = strconv.Itoa(count)
				case '%':
					length := 0
					ExploreElements(curr, mask, constraint.Parent, constraint.Match, constraint.Attrib, constraint.Wild, level, func(stn string, lvl int) {
						if stn != "" {
							length += len(stn)
						}
					})
					val = strconv.Itoa(length)
				case '^':
					depth := 0
					ExploreElements(curr, mask, constraint.Parent, constraint.Match, constraint.Attrib, constraint.Wild, level, func(stn string, lvl int) {
						depth = lvl
					})
					val = strconv.Itoa(depth)
				default:
					ExploreElements(curr, mask, constraint.Parent, constraint.Match, constraint.Attrib, constraint.Wild, level, func(stn string, lvl int) {
						if stn != "" {
							_, errz := strconv.Atoi(stn)
							if errz == nil {
								val = stn
							}
						}
					})
				}
			}

			// numeric tests on element values
			x, errx := strconv.Atoi(str)
			y, erry := strconv.Atoi(val)

			// both arguments must resolve to integers
			if errx != nil || erry != nil {
				return false
			}

			switch stat {
			case GT:
				if x > y {
					return true
				}
			case GE:
				if x >= y {
					return true
				}
			case LT:
				if x < y {
					return true
				}
			case LE:
				if x <= y {
					return true
				}
			case EQ:
				if x == y {
					return true
				}
			case NE:
				if x != y {
					return true
				}
			default:
			}
		default:
		}

		return false
	}

	// matchFound tests individual conditions
	matchFound := func(stages []*Step) bool {

		if stages == nil || len(stages) < 1 {
			return false
		}

		stage := stages[0]

		var constraint *Step

		if len(stages) > 1 {
			constraint = stages[1]
		}

		status := stage.Type
		prnt := stage.Parent
		match := stage.Match
		attrib := stage.Attrib
		wildcard := stage.Wild

		found := false
		number := ""

		// exploreElements is a wrapper for ExploreElements, obtaining most arguments as closures
		exploreElements := func(proc func(string, int)) {
			ExploreElements(curr, mask, prnt, match, attrib, wildcard, level, proc)
		}

		switch status {
		case ELEMENT:
			exploreElements(func(str string, lvl int) {
				// match to XML container object sends empty string, so do not check for str != "" here
				// test every selected element individually if value is specified
				if constraint == nil || testConstraint(str, constraint) {
					found = true
				}
			})
		case VARIABLE:
			// use value of stored variable
			str, ok := variables[match]
			if ok {
				//  -if &VARIABLE -equals VALUE is the supported construct
				if constraint == nil || testConstraint(str, constraint) {
					found = true
				}
			}
		case COUNT:
			count := 0

			exploreElements(func(str string, lvl int) {
				count++
				found = true
			})

			// number of element objects
			number = strconv.Itoa(count)
		case LENGTH:
			length := 0

			exploreElements(func(str string, lvl int) {
				length += len(str)
				found = true
			})

			// length of element strings
			number = strconv.Itoa(length)
		case DEPTH:
			depth := 0

			exploreElements(func(str string, lvl int) {
				depth = lvl
				found = true
			})

			// depth of last element in scope
			number = strconv.Itoa(depth)
		case INDEX:
			// index of explored parent object
			number = strconv.Itoa(index)
			found = true
		default:
		}

		if number == "" {
			return found
		}

		if constraint == nil || testConstraint(number, constraint) {
			return true
		}

		return false
	}

	// test conditional arguments
	for _, op := range conditions {

		switch op.Type {
		// -if tests for presence of element (deprecated -match can test element:value)
		case IF, MATCH:
			// checking for failure here allows for multiple -if [ -and / -or ] clauses
			if isMatch && observed < required {
				return false
			}
			if isAvoid && forbidden > 0 {
				return false
			}
			required = 0
			observed = 0
			forbidden = 0
			isMatch = true
			isAvoid = false
			// continue on to next two cases
			fallthrough
		case AND:
			required++
			// continue on to next case
			fallthrough
		case OR:
			if matchFound(op.Stages) {
				observed++
				// record presence of forbidden element if in -unless clause
				forbidden++
			}
		// -unless tests for absence of element, or presence but with failure of subsequent value test (deprecated -avoid can test element:value)
		case UNLESS, AVOID:
			if isMatch && observed < required {
				return false
			}
			if isAvoid && forbidden > 0 {
				return false
			}
			required = 0
			observed = 0
			forbidden = 0
			isMatch = false
			isAvoid = true
			if matchFound(op.Stages) {
				forbidden++
			}
		default:
		}
	}

	if isMatch && observed < required {
		return false
	}
	if isAvoid && forbidden > 0 {
		return false
	}

	return true
}

// RECURSIVELY PROCESS EXPLORATION COMMANDS AND XML DATA STRUCTURE

// ProcessCommands visits XML nodes, performs conditional tests, and executes data extraction instructions
func ProcessCommands(cmds *Block, curr *Node, tab, ret string, index, level int, variables map[string]string, accum func(string)) (string, string) {

	if accum == nil {
		return tab, ret
	}

	prnt := cmds.Parent
	match := cmds.Match

	// leading colon indicates namespace prefix wildcard
	wildcard := false
	if strings.HasPrefix(prnt, ":") || strings.HasPrefix(match, ":") {
		wildcard = true
	}

	// **/Object performs deep exploration of recursive data
	deep := false
	if prnt == "**" {
		prnt = "*"
		deep = true
	}

	// closure passes local variables to callback, which can modify caller tab and ret values
	processNode := func(node *Node, idx, lvl int) {

		// apply -if or -unless tests
		if ConditionsAreSatisfied(cmds.Conditions, node, match, idx, lvl, variables) {

			// execute data extraction commands
			if len(cmds.Commands) > 0 {
				tab, ret = ProcessInstructions(cmds.Commands, node, match, tab, ret, idx, lvl, variables, accum)
			}

			// process sub commands on child node
			for _, sub := range cmds.Subtasks {
				tab, ret = ProcessCommands(sub, node, tab, ret, 1, lvl, variables, accum)
			}

		} else {

			// execute commands after -else statement
			if len(cmds.Failure) > 0 {
				tab, ret = ProcessInstructions(cmds.Failure, node, match, tab, ret, idx, lvl, variables, accum)
			}
		}
	}

	// exploreNodes recursive definition
	var exploreNodes func(*Node, int, int, func(*Node, int, int)) int

	// exploreNodes visits all nodes that match the selection criteria
	exploreNodes = func(curr *Node, indx, levl int, proc func(*Node, int, int)) int {

		if curr == nil || proc == nil {
			return indx
		}

		// match is "*" for heterogeneous data constructs, e.g., -group PubmedArticleSet/*
		// wildcard matches any namespace prefix
		if curr.Name == match ||
			match == "*" ||
			(wildcard && strings.HasPrefix(match, ":") && strings.HasSuffix(curr.Name, match)) {

			if prnt == "" ||
				curr.Parent == prnt ||
				(wildcard && strings.HasPrefix(prnt, ":") && strings.HasSuffix(curr.Parent, prnt)) {

				proc(curr, indx, levl)
				indx++

				if !deep {
					// do not explore within recursive object
					return indx
				}
			}
		}

		// clearing prnt "*" now allows nested exploration within recursive data, e.g., -pattern Taxon -block */Taxon
		if prnt == "*" {
			prnt = ""
		}

		// explore child nodes
		for chld := curr.Children; chld != nil; chld = chld.Next {
			indx = exploreNodes(chld, indx, levl+1, proc)
		}

		return indx
	}

	// apply -position test

	if cmds.Position == "" {

		exploreNodes(curr, index, level, processNode)

	} else {

		var single *Node
		lev := 0
		ind := 0

		if cmds.Position == "first" {

			exploreNodes(curr, index, level,
				func(node *Node, idx, lvl int) {
					if single == nil {
						single = node
						ind = idx
						lev = lvl
					}
				})

		} else if cmds.Position == "last" {

			exploreNodes(curr, index, level,
				func(node *Node, idx, lvl int) {
					single = node
					ind = idx
					lev = lvl
				})

		} else {

			// use numeric position
			number, err := strconv.Atoi(cmds.Position)
			if err == nil {

				pos := 0

				exploreNodes(curr, index, level,
					func(node *Node, idx, lvl int) {
						pos++
						if pos == number {
							single = node
							ind = idx
							lev = lvl
						}
					})

			} else {

				fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized position '%s'\n", cmds.Position)
				os.Exit(1)
			}
		}

		if single != nil {
			processNode(single, ind, lev)
		}
	}

	return tab, ret
}

// PROCESS ONE XML COMPONENT RECORD

// ProcessQuery calls XML combined tokenizer parser on a partitioned string
func ProcessQuery(Text, parent string, index int, cmds *Block, tbls *Tables, action SpecialType) string {

	if Text == "" || tbls == nil {
		return ""
	}

	// node farm variables
	FarmPos := 0
	FarmMax := tbls.FarmSize
	FarmItems := make([]Node, FarmMax)

	// allocate multiple nodes in a large array for memory management efficiency
	nextNode := func(strt, attr, prnt string) *Node {

		// if farm array slots used up, allocate new array
		if FarmPos >= FarmMax {
			FarmItems = make([]Node, FarmMax)
			FarmPos = 0
		}

		if FarmItems == nil {
			return nil
		}

		// take node from next available slot in farm array
		node := &FarmItems[FarmPos]

		node.Name = strt[:]
		node.Attributes = attr[:]
		node.Parent = prnt[:]

		FarmPos++

		return node
	}

	// token parser variables
	Txtlen := len(Text)
	Idx := 0

	plainText := (!tbls.DoStrict && !tbls.DoMixed)

	// get next XML token
	nextToken := func(idx int) (TagType, string, string, int) {

		// lookup table array pointers
		inBlank := &tbls.InBlank
		inFirst := &tbls.InFirst
		inElement := &tbls.InElement

		text := Text[:]
		txtlen := Txtlen

		// XML string ends with > character, acts as sentinel to check if past end of text
		if idx >= txtlen {
			// signal end of XML string
			return ISCLOSED, "", "", 0
		}

		// skip past leading blanks
		ch := text[idx]
		for inBlank[ch] {
			idx++
			ch = text[idx]
		}

		start := idx

		if ch == '<' && (plainText || HTMLAhead(text, idx) == 0) {

			// at start of element
			idx++
			ch = text[idx]

			// check for legal first character of element
			if inFirst[ch] {

				// read element name
				start = idx
				idx++

				ch = text[idx]
				for inElement[ch] {
					idx++
					ch = text[idx]
				}

				str := text[start:idx]

				switch ch {
				case '>':
					// end of element
					idx++

					return STARTTAG, str[:], "", idx
				case '/':
					// self-closing element without attributes
					idx++
					ch = text[idx]
					if ch != '>' {
						fmt.Fprintf(os.Stderr, "\nSelf-closing element missing right angle bracket\n")
					}
					idx++

					return SELFTAG, str[:], "", idx
				case ' ', '\t', '\n', '\r', '\f':
					// attributes
					idx++
					start = idx
					ch = text[idx]
					for ch != '<' && ch != '>' {
						idx++
						ch = text[idx]
					}
					if ch != '>' {
						fmt.Fprintf(os.Stderr, "\nAttributes not followed by right angle bracket\n")
					}
					if text[idx-1] == '/' {
						// self-closing
						atr := text[start : idx-1]
						idx++
						return SELFTAG, str[:], atr[:], idx
					}
					atr := text[start:idx]
					idx++
					return STARTTAG, str[:], atr[:], idx
				default:
					fmt.Fprintf(os.Stderr, "\nUnexpected punctuation '%c' in XML element\n", ch)
					return STARTTAG, str[:], "", idx
				}

			} else {

				// punctuation character immediately after first angle bracket
				switch ch {
				case '/':
					// at start of end tag
					idx++
					start = idx
					ch = text[idx]
					// expect legal first character of element
					if inFirst[ch] {
						idx++
						ch = text[idx]
						for inElement[ch] {
							idx++
							ch = text[idx]
						}
						str := text[start:idx]
						if ch != '>' {
							fmt.Fprintf(os.Stderr, "\nUnexpected characters after end element name\n")
						}
						idx++

						return STOPTAG, str[:], "", idx
					}
					fmt.Fprintf(os.Stderr, "\nUnexpected punctuation '%c' in XML element\n", ch)
				case '?':
					// skip ?xml and ?processing instructions
					idx++
					ch = text[idx]
					for ch != '>' {
						idx++
						ch = text[idx]
					}
					idx++
				case '!':
					// skip !DOCTYPE, !comment, and ![CDATA[
					idx++
					start = idx
					ch = text[idx]
					which := NOTAG
					skipTo := ""
					if ch == '[' && strings.HasPrefix(text[idx:], "[CDATA[") {
						which = CDATATAG
						skipTo = "]]>"
						start += 7
					} else if ch == '-' && strings.HasPrefix(text[idx:], "--") {
						which = COMMENTTAG
						skipTo = "-->"
						start += 2
					}
					if which != NOTAG && skipTo != "" {
						// CDATA or comment block may contain internal angle brackets
						found := strings.Index(text[idx:], skipTo)
						if found < 0 {
							// string stops in middle of CDATA or comment
							return ISCLOSED, "", "", idx
						}
						// adjust position past end of CDATA or comment
						idx += found + len(skipTo)
					} else {
						// otherwise just skip to next right angle bracket
						for ch != '>' {
							idx++
							ch = text[idx]
						}
						idx++
					}
				default:
					fmt.Fprintf(os.Stderr, "\nUnexpected punctuation '%c' in XML element\n", ch)
				}
			}

		} else if ch != '>' {

			// at start of contents
			start = idx

			// find end of contents
			for {
				for ch != '<' && ch != '>' {
					idx++
					ch = text[idx]
				}
				if ch == '<' && !plainText {
					// optionally allow HTML text formatting elements and super/subscripts
					advance := HTMLAhead(text, idx)
					if advance > 0 {
						idx += advance
						ch = text[idx]
						continue
					}
				}
				break
			}

			// trim back past trailing blanks
			lst := idx - 1
			ch = text[lst]
			for inBlank[ch] && lst > start {
				lst--
				ch = text[lst]
			}

			str := text[start : lst+1]

			return CONTENTTAG, str[:], "", idx
		}

		return NOTAG, "", "", idx
	}

	// Parse tokens into tree structure for exploration

	// parseLevel recursive definition
	var parseLevel func(string, string, string) (*Node, bool)

	// parse XML tags into tree structure for searching
	parseLevel = func(strt, attr, prnt string) (*Node, bool) {

		ok := true

		// obtain next node from farm
		node := nextNode(strt, attr, prnt)
		if node == nil {
			return nil, false
		}

		var lastNode *Node

		for {
			tag, name, attr, idx := nextToken(Idx)
			if tag == ISCLOSED {
				break
			}
			Idx = idx

			switch tag {
			case STARTTAG:
				// read sub tree
				obj, ok := parseLevel(name, attr, node.Name)
				if !ok {
					break
				}

				// adding next child to end of linked list gives better performance than appending to slice of nodes
				if node.Children == nil {
					node.Children = obj
				}
				if lastNode != nil {
					lastNode.Next = obj
				}
				lastNode = obj
			case STOPTAG:
				// pop out of recursive call
				return node, ok
			case CONTENTTAG:
				if tbls.DoStrict {
					if HasMarkup(name) {
						name = RemoveUnicodeMarkup(name)
					}
					if HasAngleBracket(name) {
						name = DoHTMLReplace(name)
					}
				}
				if tbls.DoMixed {
					if HasMarkup(name) {
						name = SimulateUnicodeMarkup(name)
					}
					if HasAngleBracket(name) {
						name = DoHTMLReplace(name)
					}
					name = DoTrimFlankingHTML(name)
				}
				if tbls.DeAccent {
					if IsNotASCII(name) {
						name = DoAccentTransform(name)
					}
				}
				if tbls.DoASCII {
					if IsNotASCII(name) {
						name = UnicodeToASCII(name)
					}
				}
				node.Contents = name
			case SELFTAG:
				if attr == "" {
					// ignore if self-closing tag has no attributes
					continue
				}

				// self-closing tag has no contents, just create child node
				obj := nextNode(name, attr, node.Name)

				if node.Children == nil {
					node.Children = obj
				}
				if lastNode != nil {
					lastNode.Next = obj
				}
				lastNode = obj
				// continue on same level
			default:
			}
		}

		return node, ok
	}

	// perform data extraction driven by command-line arguments
	doQuery := func() string {

		if cmds == nil {
			return ""
		}

		// exit from function will collect garbage of node structure for current XML object
		tag, name, attr, idx := nextToken(Idx)

		// loop until start tag
		for {
			if tag == ISCLOSED {
				break
			}

			Idx = idx

			if tag == STARTTAG {
				break
			}

			tag, name, attr, idx = nextToken(Idx)
		}

		pat, ok := parseLevel(name, attr, parent)

		if !ok {
			return ""
		}

		// exit from function will also free map of recorded variables for current -pattern
		variables := make(map[string]string)

		var buffer bytes.Buffer

		ok = false

		if tbls.Hd != "" {
			buffer.WriteString(tbls.Hd[:])
		}

		// start processing at top of command tree and top of XML subregion selected by -pattern
		_, ret := ProcessCommands(cmds, pat, "", "", index, 1, variables,
			func(str string) {
				if str != "" {
					ok = true
					buffer.WriteString(str)
				}
			})

		if tbls.Tl != "" {
			buffer.WriteString(tbls.Tl[:])
		}

		if ret != "" {
			ok = true
			buffer.WriteString(ret)
		}

		txt := buffer.String()

		// remove leading newline (-insd -pfx artifact)
		if txt != "" && txt[0] == '\n' {
			txt = txt[1:]
		}

		if !ok {
			return ""
		}

		// return consolidated result string
		return txt
	}

	// Stream tokens to obtain value of single index element

	// parseIndex recursive definition
	var parseIndex func(string, string, string) string

	// parse XML tags looking for trie index element
	parseIndex = func(strt, attr, prnt string) string {

		// check for attribute index match
		if attr != "" && tbls.Attrib != "" && strings.Contains(attr, tbls.Attrib) {
			if strt == tbls.Match || tbls.Match == "" {
				if tbls.Parent == "" || prnt == tbls.Parent {
					attribs := ParseAttributes(attr)
					for i := 0; i < len(attribs)-1; i += 2 {
						if attribs[i] == tbls.Attrib {
							return attribs[i+1]
						}
					}
				}
			}
		}

		for {
			tag, name, attr, idx := nextToken(Idx)
			if tag == ISCLOSED {
				break
			}
			Idx = idx

			switch tag {
			case STARTTAG:
				id := parseIndex(name, attr, strt)
				if id != "" {
					return id
				}
			case SELFTAG:
			case STOPTAG:
				// break recursion
				return ""
			case CONTENTTAG:
				// check for content index match
				if strt == tbls.Match || tbls.Match == "" {
					if tbls.Parent == "" || prnt == tbls.Parent {
						return name
					}
				}
			default:
			}
		}

		return ""
	}

	// just return indexed identifier
	doIndex := func() string {

		if tbls.Index == "" {
			return ""
		}

		tag, name, attr, idx := nextToken(Idx)

		// loop until start tag
		for {
			if tag == ISCLOSED {
				break
			}

			Idx = idx

			if tag == STARTTAG {
				break
			}

			tag, name, attr, idx = nextToken(Idx)
		}

		return parseIndex(name, attr, parent)
	}

	// ProcessQuery

	// call specific function
	switch action {
	case DOQUERY:
		return doQuery()
	case DOINDEX:
		return doIndex()
	default:
	}

	return ""
}

// CONVERT IDENTIFIER TO DIRECTORY PATH FOR LOCAL FILE ARCHIVE

// MakeArchiveTrie allows a short prefix of letters with an optional underscore, and splits the remainder into character pairs
func MakeArchiveTrie(str string, arry [132]rune) string {

	if len(str) > 64 {
		return ""
	}

	max := 4
	k := 0
	for _, ch := range str {
		if unicode.IsLetter(ch) {
			k++
			continue
		}
		if ch == '_' {
			k++
			max = 6
		}
		break
	}

	// prefix is up to three letters if followed by digits, or up to four letters if followed by an underscore
	pfx := str[:k]
	if len(pfx) < max {
		str = str[k:]
	} else {
		pfx = ""
	}

	i := 0

	if pfx != "" {
		for _, ch := range pfx {
			arry[i] = ch
			i++
		}
		arry[i] = '/'
		i++
	}

	between := 0
	doSlash := false

	// remainder is divided in character pairs, e.g., NP_/06/00/51 for NP_060051.2
	for _, ch := range str {
		// break at period separating accession from version
		if ch == '.' {
			break
		}
		if doSlash {
			arry[i] = '/'
			i++
			doSlash = false
		}
		arry[i] = ch
		i++
		between++
		if between > 1 {
			doSlash = true
			between = 0
		}
	}

	return strings.ToUpper(string(arry[:i]))
}

// CONVERT TERM TO DIRECTORY PATH FOR POSTINGS FILE STORAGE

// MakePostingsTrie splits a string into characters, separated by path delimiting slashes
func MakePostingsTrie(str string, arry [516]rune) string {

	if len(str) > 256 {
		return ""
	}

	i := 0
	doSlash := false
	for _, ch := range str {
		if doSlash {
			arry[i] = '/'
			i++
		}
		if ch == ' ' {
			ch = '_'
		}
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
			ch = '_'
		}
		arry[i] = ch
		i++
		doSlash = true
	}

	return strings.ToLower(string(arry[:i]))
}

// UNSHUFFLER USES HEAP TO RESTORE OUTPUT OF MULTIPLE CONSUMERS TO ORIGINAL RECORD ORDER

type Extract struct {
	Index int
	Ident string
	Text  string
}

type ExtractHeap []Extract

// methods that satisfy heap.Interface
func (h ExtractHeap) Len() int {
	return len(h)
}
func (h ExtractHeap) Less(i, j int) bool {
	return h[i].Index < h[j].Index
}
func (h ExtractHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}
func (h *ExtractHeap) Push(x interface{}) {
	*h = append(*h, x.(Extract))
}
func (h *ExtractHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// CONCURRENT CONSUMER GOROUTINES PARSE AND PROCESS PARTITIONED XML OBJECTS

// ReadBlocks -> SplitPattern => StreamTokens => ParseXML => ProcessQuery -> MergeResults

// process with single goroutine calls defer close(out) so consumer(s) can range over channel
// process with multiple instances calls defer wg.Done(), separate goroutine uses wg.Wait() to delay close(out)

func CreateProducer(pat, star string, rdr *XMLReader, tbls *Tables) <-chan Extract {

	if rdr == nil || tbls == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create producer channel\n")
		os.Exit(1)
	}

	// xmlProducer sends partitioned XML strings through channel
	xmlProducer := func(pat, star string, rdr *XMLReader, out chan<- Extract) {

		// close channel when all records have been processed
		defer close(out)

		// partition all input by pattern and send XML substring to available consumer through channel
		PartitionPattern(pat, star, rdr,
			func(rec int, ofs int64, str string) {
				out <- Extract{rec, "", str}
			})
	}

	// launch single producer goroutine
	go xmlProducer(pat, star, rdr, out)

	return out
}

func CreateUIDReader(in io.Reader, tbls *Tables) <-chan Extract {

	if in == nil || tbls == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create uid reader channel\n")
		os.Exit(1)
	}

	// uidReader reads uids from input stream and sends through channel
	uidReader := func(in io.Reader, out chan<- Extract) {

		// close channel when all records have been processed
		defer close(out)

		scanr := bufio.NewScanner(in)

		idx := 0
		for scanr.Scan() {

			// read lines of identifiers
			file := scanr.Text()
			idx++

			out <- Extract{idx, "", file}
		}
	}

	// launch single uid reader goroutine
	go uidReader(in, out)

	return out
}

func CreateConsumers(cmds *Block, tbls *Tables, parent string, inp <-chan Extract) <-chan Extract {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create consumer channel\n")
		os.Exit(1)
	}

	// xmlConsumer reads partitioned XML from channel and calls parser for processing
	xmlConsumer := func(cmds *Block, tbls *Tables, parent string, wg *sync.WaitGroup, inp <-chan Extract, out chan<- Extract) {

		// report when this consumer has no more records to process
		defer wg.Done()

		// read partitioned XML from producer channel
		for ext := range inp {

			idx := ext.Index
			text := ext.Text

			if text == "" {
				// should never see empty input data
				out <- Extract{idx, "", text}
				continue
			}

			str := ProcessQuery(text[:], parent, idx, cmds, tbls, DOQUERY)

			// send even if empty to get all record counts for reordering
			out <- Extract{idx, "", str}
		}
	}

	var wg sync.WaitGroup

	// launch multiple consumer goroutines
	for i := 0; i < tbls.NumServe; i++ {
		wg.Add(1)
		go xmlConsumer(cmds, tbls, parent, &wg, inp, out)
	}

	// launch separate anonymous goroutine to wait until all consumers are done
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func CreateExaminers(tbls *Tables, parent string, inp <-chan Extract) <-chan Extract {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create examiner channel\n")
		os.Exit(1)
	}

	// xmlExaminer reads partitioned XML from channel and returns unique identifier
	xmlExaminer := func(tbls *Tables, wg *sync.WaitGroup, inp <-chan Extract, out chan<- Extract) {

		// report when this examiner has no more records to process
		defer wg.Done()

		// read partitioned XML from producer channel
		for ext := range inp {

			idx := ext.Index
			text := ext.Text

			if text == "" {
				// should never see empty input data
				out <- Extract{idx, "", text}
				continue
			}

			id := ProcessQuery(text[:], parent, 0, nil, tbls, DOINDEX)

			// send even if empty to get all record counts for reordering
			out <- Extract{idx, id, text}
		}
	}

	var wg sync.WaitGroup

	// launch multiple examiner goroutines
	for i := 0; i < tbls.NumServe; i++ {
		wg.Add(1)
		go xmlExaminer(tbls, &wg, inp, out)
	}

	// launch separate anonymous goroutine to wait until all examiners are done
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func CreateUnshuffler(tbls *Tables, inp <-chan Extract) <-chan Extract {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create unshuffler channel\n")
		os.Exit(1)
	}

	// xmlUnshuffler restores original order with heap
	xmlUnshuffler := func(inp <-chan Extract, out chan<- Extract) {

		// close channel when all records have been processed
		defer close(out)

		// initialize empty heap
		hp := &ExtractHeap{}
		heap.Init(hp)

		// index of next desired result
		next := 1

		delay := 0

		for ext := range inp {

			// push result onto heap
			heap.Push(hp, ext)

			// read several values before checking to see if next record to print has been processed
			if delay < tbls.HeapSize {
				delay++
				continue
			}

			delay = 0

			for hp.Len() > 0 {

				// remove lowest item from heap, use interface type assertion
				curr := heap.Pop(hp).(Extract)

				if curr.Index > next {

					// record should be printed later, push back onto heap
					heap.Push(hp, curr)
					// and go back to waiting on input channel
					break
				}

				// send even if empty to get all record counts for reordering
				out <- Extract{curr.Index, curr.Ident, curr.Text}

				// prevent ambiguous -limit filter from clogging heap (deprecated)
				if curr.Index == next {
					// increment index for next expected match
					next++
				}

				// keep checking heap to see if next result is already available
			}
		}

		// send remainder of heap to output
		for hp.Len() > 0 {
			curr := heap.Pop(hp).(Extract)

			out <- Extract{curr.Index, curr.Ident, curr.Text}
		}
	}

	// launch single unshuffler goroutine
	go xmlUnshuffler(inp, out)

	return out
}

func CreateUniquer(tbls *Tables, inp <-chan Extract) <-chan Extract {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create uniquer channel\n")
		os.Exit(1)
	}

	// xmlUniquer removes adjacent records with the same identifier
	xmlUniquer := func(inp <-chan Extract, out chan<- Extract) {

		// close channel when all records have been processed
		defer close(out)

		// remember previous record
		prev := Extract{}

		for curr := range inp {

			// compare adjacent record identifiers
			if prev.Text != "" && prev.Ident != curr.Ident {

				// if identifiers are different, send previous to output channel
				out <- prev
			}

			// now remember this record
			prev = curr
		}

		if prev.Text != "" {

			// send last record
			out <- prev
		}
	}

	// launch single uniquer goroutine
	go xmlUniquer(inp, out)

	return out
}

func CreateDeleter(tbls *Tables, dltd string, inp <-chan Extract) <-chan Extract {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create deleter channel\n")
		os.Exit(1)
	}

	// map to track UIDs to skip
	shouldSkip := make(map[string]bool)

	checkMap := false

	if dltd != "" && dltd != "-" {
		fmt.Fprintf(os.Stderr, "\nEnter CreateDeleter Scanner\n")
		checkMap = true

		skipFile, err := os.Open(dltd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: Unable to read skip file\n")
			os.Exit(1)
		}

		scanr := bufio.NewScanner(skipFile)

		for scanr.Scan() {

			// read lines of identifiers
			id := scanr.Text()

			// add to exclusion map
			shouldSkip[id] = true
		}

		skipFile.Close()
		fmt.Fprintf(os.Stderr, "\nLeave CreateDeleter Scanner\n")
	}

	// xmlDeleter removes records listed as deleted
	xmlDeleter := func(inp <-chan Extract, out chan<- Extract) {

		// close channel when all records have been processed
		defer close(out)

		for curr := range inp {

			// check if identifier was deleted
			if checkMap && shouldSkip[curr.Ident] {
				continue
			}

			// send to output channel
			out <- curr
		}
	}

	// launch single deleter goroutine
	go xmlDeleter(inp, out)

	return out
}

func CreateStashers(tbls *Tables, inp <-chan Extract) <-chan string {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan string, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create stasher channel\n")
		os.Exit(1)
	}

	sfx := ".xml"
	if tbls.Zipp {
		sfx = ".xml.gz"
	}

	type StasherType int

	const (
		OKAY StasherType = iota
		WAIT
		BAIL
	)

	// mutex to protect access to inUse map
	var flock sync.Mutex

	// map to track files currently being written
	inUse := make(map[string]int)

	// lockFile function prevents colliding writes
	lockFile := func(id string, index int) StasherType {
		// map is non-reentrant, protect with mutex
		flock.Lock()
		// multiple return paths, schedule the unlock command up front
		defer flock.Unlock()

		idx, ok := inUse[id]

		if ok {
			if idx < index {
				// later version is being written by another goroutine, skip this
				return BAIL
			}
			// earlier version is being written by another goroutine, wait
			return WAIT
		}

		// okay to write file, mark in use to prevent collision
		inUse[id] = index
		return OKAY
	}

	// freeFile function removes entry from inUse map
	freeFile := func(id string) {
		flock.Lock()
		// free entry in map, later versions of same record can now be written
		delete(inUse, id)
		flock.Unlock()
	}

	// trimLeft function reformats output, efficiently skipping leading spaces on each line
	trimLeft := func(text string) string {

		if text == "" {
			return ""
		}

		var buffer bytes.Buffer

		max := len(text)
		idx := 0
		inBlank := &tbls.InBlank

		for idx < max {

			// skip past leading blanks and empty lines
			for idx < max {
				ch := text[idx]
				if !inBlank[ch] {
					break
				}
				idx++
			}

			start := idx

			// skip to next newline
			for idx < max {
				if text[idx] == '\n' {
					break
				}
				idx++
			}

			str := text[start:idx]

			if str == "" {
				continue
			}

			// skip processing instruction
			if strings.HasPrefix(str, "<?") && strings.HasSuffix(str, "?>") {
				continue
			}

			// trim spaces next to angle bracket
			str = strings.Replace(str, "> ", ">", -1)
			str = strings.Replace(str, " <", "<", -1)

			buffer.WriteString(str[:])
			buffer.WriteString("\n")
		}

		return buffer.String()
	}

	// stashRecord saves individual XML record to archive file accessed by trie
	stashRecord := func(text, id string, index int) string {

		var arry [132]rune
		trie := MakeArchiveTrie(id, arry)
		if trie == "" {
			return ""
		}

		attempts := 5
		keepChecking := true

		for keepChecking {
			// check if file is not being written by another goroutine
			switch lockFile(id, index) {
			case OKAY:
				// okay to save this record now
				keepChecking = false
			case WAIT:
				// earlier version is being saved, wait one second and try again
				time.Sleep(time.Second)
				attempts--
				if attempts < 1 {
					// cannot get lock after several attempts
					fmt.Fprintf(os.Stderr, "\nERROR: Unable to save '%s'\n", id)
					return ""
				}
			case BAIL:
				// later version is being saved, skip this one
				return ""
			default:
			}
		}

		// delete lock after writing file
		defer freeFile(id)

		dpath := path.Join(tbls.Stash, trie)
		if dpath == "" {
			return ""
		}
		_, err := os.Stat(dpath)
		if err != nil && os.IsNotExist(err) {
			err = os.MkdirAll(dpath, os.ModePerm)
		}
		if err != nil {
			fmt.Println(err.Error())
			return ""
		}
		fpath := path.Join(dpath, id+sfx)
		if fpath == "" {
			return ""
		}

		// overwrites and truncates existing file
		fl, err := os.Create(fpath)
		if err != nil {
			fmt.Println(err.Error())
			return ""
		}

		// remove leading spaces on each line
		str := trimLeft(text)

		res := ""

		if tbls.Hash {
			// calculate hash code for verification table
			hsh := crc32.NewIEEE()
			hsh.Write([]byte(str))
			val := hsh.Sum32()
			res = strconv.FormatUint(uint64(val), 10)
		}

		if tbls.Zipp {

			zpr, err := gzip.NewWriterLevel(fl, gzip.BestCompression)

			if err == nil {
				bfr := bufio.NewWriter(zpr)

				// compress and copy record to file
				bfr.WriteString(str)
				if !strings.HasSuffix(str, "\n") {
					bfr.WriteString("\n")
				}
				bfr.Flush()
			}

			zpr.Close()

		} else {

			// copy record to file
			fl.WriteString(str)
			if !strings.HasSuffix(str, "\n") {
				fl.WriteString("\n")
			}
		}

		err = fl.Sync()
		if err != nil {
			fmt.Println(err.Error())
		}
		fl.Close()

		return res
	}

	// xmlStasher reads from channel and calls stashRecord
	xmlStasher := func(wg *sync.WaitGroup, inp <-chan Extract, out chan<- string) {

		defer wg.Done()

		for ext := range inp {

			hsh := stashRecord(ext.Text, ext.Ident, ext.Index)
			res := ext.Ident
			if tbls.Hash {
				res += "\t" + hsh
			}
			res += "\n"

			out <- res
		}
	}

	var wg sync.WaitGroup

	// launch multiple stasher goroutines
	for i := 0; i < tbls.NumServe; i++ {
		wg.Add(1)
		go xmlStasher(&wg, inp, out)
	}

	// launch separate anonymous goroutine to wait until all stashers are done
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func CreateFetchers(tbls *Tables, inp <-chan Extract) <-chan Extract {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create fetcher channel\n")
		os.Exit(1)
	}

	sfx := ".xml"
	if tbls.Zipp {
		sfx = ".xml.gz"
	}

	// xmlFetcher reads XML from file
	xmlFetcher := func(tbls *Tables, wg *sync.WaitGroup, inp <-chan Extract, out chan<- Extract) {

		// report when more records to process
		defer wg.Done()

		var buf bytes.Buffer

		for ext := range inp {

			idx := ext.Index
			file := ext.Text

			var arry [132]rune
			trie := MakeArchiveTrie(file, arry)
			if trie == "" {
				continue
			}

			fpath := path.Join(tbls.Stash, trie, file+sfx)
			if fpath == "" {
				continue
			}

			iszip := tbls.Zipp

			inFile, err := os.Open(fpath)

			// if failed to find ".xml" file, try ".xml.gz" without requiring -gzip
			if err != nil && os.IsNotExist(err) && !tbls.Zipp {
				iszip = true
				fpath := path.Join(tbls.Stash, trie, file+".xml.gz")
				if fpath == "" {
					continue
				}
				inFile, err = os.Open(fpath)
			}
			if err != nil {
				continue
			}

			buf.Reset()

			brd := bufio.NewReader(inFile)

			if iszip {

				zpr, err := gzip.NewReader(brd)

				if err == nil {
					// copy and decompress cached file contents
					buf.ReadFrom(zpr)
				}

				zpr.Close()

			} else {

				// copy cached file contents
				buf.ReadFrom(brd)
			}

			inFile.Close()

			str := buf.String()

			out <- Extract{idx, "", str}
		}
	}

	var wg sync.WaitGroup

	// launch multiple fetcher goroutines
	for i := 0; i < tbls.NumServe; i++ {
		wg.Add(1)
		go xmlFetcher(tbls, &wg, inp, out)
	}

	// launch separate anonymous goroutine to wait until all fetchers are done
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func CreateTermListReader(in io.Reader, tbls *Tables) <-chan Extract {

	if in == nil || tbls == nil {
		return nil
	}

	out := make(chan Extract, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create term list reader channel\n")
		os.Exit(1)
	}

	// termReader reads uids and terms from input stream and sends through channel
	termReader := func(in io.Reader, out chan<- Extract) {

		// close channel when all records have been processed
		defer close(out)

		var buffer bytes.Buffer

		uid := ""
		term := ""
		prev := ""
		count := 0

		scanr := bufio.NewScanner(in)

		idx := 0
		for scanr.Scan() {

			// read lines of uid and term groups
			line := scanr.Text()
			idx++

			uid, term = SplitInTwoAt(line, "\t", LEFT)

			if prev != "" && prev != term {

				str := buffer.String()
				out <- Extract{idx, prev, str}

				buffer.Reset()
				count = 0
			}

			buffer.WriteString(uid)
			buffer.WriteString("\n")
			count++

			prev = term
		}

		if count > 0 {

			str := buffer.String()
			out <- Extract{idx, term, str}

			buffer.Reset()
		}
	}

	// launch single term reader goroutine
	go termReader(in, out)

	return out
}

func CreatePosters(tbls *Tables, inp <-chan Extract) <-chan string {

	if tbls == nil || inp == nil {
		return nil
	}

	out := make(chan string, tbls.ChanDepth)
	if out == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create poster channel\n")
		os.Exit(1)
	}

	// savePosting writes individual postings list to file accessed by radix trie
	savePosting := func(text, id string, index int) {

		var arry [516]rune
		trie := MakePostingsTrie(id, arry)
		if trie == "" {
			return
		}

		dpath := path.Join(tbls.Posting, trie)
		if dpath == "" {
			return
		}
		_, err := os.Stat(dpath)
		if err != nil && os.IsNotExist(err) {
			err = os.MkdirAll(dpath, os.ModePerm)
		}
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fpath := path.Join(dpath, "uids.txt")
		if fpath == "" {
			return
		}

		// appends if file exists, otherwise creates
		fl, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fl.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			fl.WriteString("\n")
		}

		err = fl.Sync()
		if err != nil {
			fmt.Println(err.Error())
		}
		fl.Close()
	}

	// xmlPoster reads from channel and calls savePosting
	xmlPoster := func(wg *sync.WaitGroup, inp <-chan Extract, out chan<- string) {

		defer wg.Done()

		for ext := range inp {

			savePosting(ext.Text, ext.Ident, ext.Index)

			out <- ext.Ident
		}
	}

	var wg sync.WaitGroup

	// launch multiple poster goroutines
	for i := 0; i < tbls.NumServe; i++ {
		wg.Add(1)
		go xmlPoster(&wg, inp, out)
	}

	// launch separate anonymous goroutine to wait until all posters are done
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// MAIN FUNCTION

// e.g., xtract -pattern PubmedArticle -element MedlineCitation/PMID -block Author -sep " " -element Initials,LastName

func main() {

	// skip past executable name
	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: No command-line arguments supplied to xtract\n")
		os.Exit(1)
	}

	// CONCURRENCY, CLEANUP, AND DEBUGGING FLAGS

	// do these first because -defcpu and -maxcpu can be sent from wrapper before other arguments

	ncpu := runtime.NumCPU()
	if ncpu < 1 {
		ncpu = 1
	}

	// wrapper can limit maximum number of processors to use (undocumented)
	maxProcs := ncpu
	defProcs := 0

	// concurrent performance tuning parameters, can be overridden by -proc and -cons
	numProcs := 0
	serverRatio := 4

	// number of servers usually calculated by -cons server ratio, but can be overridden by -serv
	numServers := 0

	// number of channels usually equals number of servers, but can be overridden by -chan
	chanDepth := 0

	// miscellaneous tuning parameters
	heapSize := 16
	farmSize := 64

	// garbage collector control can be set by environment variable or default value with -gogc 0
	goGc := 600

	// XML data cleanup
	doCompress := false
	doCleanup := false
	doStrict := false
	doMixed := false
	deAccent := false
	doASCII := false

	// -flag sets -strict or -mixed cleanup flags from argument
	flgs := ""

	// read data from file instead of stdin
	fileName := ""

	// debugging
	dbug := false
	mpty := false
	idnt := false
	stts := false
	timr := false

	// profiling
	prfl := false

	// element to use as local data index
	indx := ""

	// phrase to find anywhere in XML
	phrs := ""

	// path for local data indexed as trie
	stsh := ""

	// file of UIDs to skip
	dltd := ""

	// path for postings files indexed as trie
	pstg := ""

	// use gzip compression on local data files
	zipp := false

	// print UIDs and hash values
	hshv := false

	// convert UIDs to directory trie
	trei := false

	// compare input record against stash
	cmpr := false
	cmprType := ""
	ignr := ""

	// flag missing identifiers
	msng := false

	// repeat the specified extraction 5 times for each -proc from 1 to nCPU
	trial := false

	// get numeric value
	getNumericArg := func(name string, zer, min, max int) int {

		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "\nERROR: %s is missing\n", name)
			os.Exit(1)
		}
		value, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s (%s) is not an integer\n", name, args[1])
			os.Exit(1)
		}
		// skip past first of two arguments
		args = args[1:]

		// special case for argument value of 0
		if value < 1 {
			return zer
		}
		// limit value to between specified minimum and maximum
		if value < min {
			return min
		}
		if value > max {
			return max
		}
		return value
	}

	inSwitch := true

	// get concurrency, cleanup, and debugging flags in any order
	for {

		inSwitch = true

		switch args[0] {
		// concurrency override arguments can be passed in by local wrapper script (undocumented)
		case "-maxcpu":
			maxProcs = getNumericArg("Maximum number of processors", 1, 1, ncpu)
		case "-defcpu":
			defProcs = getNumericArg("Default number of processors", ncpu, 1, ncpu)
		// performance tuning flags
		case "-proc":
			numProcs = getNumericArg("Number of processors", ncpu, 1, ncpu)
		case "-cons":
			serverRatio = getNumericArg("Parser to processor ratio", 4, 1, 32)
		case "-serv":
			numServers = getNumericArg("Concurrent parser count", 0, ncpu, 128)
		case "-chan":
			chanDepth = getNumericArg("Communication channel depth", 0, ncpu, 128)
		case "-heap":
			heapSize = getNumericArg("Unshuffler heap size", 8, 8, 64)
		case "-farm":
			farmSize = getNumericArg("Node buffer length", 4, 4, 2048)
		case "-gogc":
			goGc = getNumericArg("Garbage collection percentage", 0, 100, 1000)
		// read data from file
		case "-input":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Input file name is missing\n")
				os.Exit(1)
			}
			fileName = args[1]
			// skip past first of two arguments
			args = args[1:]
		// data element for indexing
		case "-index":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Index element is missing\n")
				os.Exit(1)
			}
			indx = args[1]
			// skip past first of two arguments
			args = args[1:]
		// local directory path for indexing
		case "-archive", "-stash":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Archive path is missing\n")
				os.Exit(1)
			}
			stsh = args[1]
			// skip past first of two arguments
			args = args[1:]
		// UIDs to ignore
		case "-skip":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Skip file is missing\n")
				os.Exit(1)
			}
			dltd = args[1]
			// skip past first of two arguments
			args = args[1:]
		// local directory path for postings files (undocumented)
		case "-posting", "-postings":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Posting path is missing\n")
				os.Exit(1)
			}
			pstg = args[1]
			// skip past first of two arguments
			args = args[1:]
		// file with selected indexes for removing duplicates
		case "-phrase":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Selection phrase is missing\n")
				os.Exit(1)
			}
			phrs = args[1]
			// skip past first of two arguments
			args = args[1:]
		case "-gzip":
			zipp = true
		case "-hash":
			hshv = true
		case "-trie", "-tries":
			trei = true
		// data cleanup flags
		case "-compress":
			doCompress = true
		case "-spaces", "-cleanup":
			doCleanup = true
		case "-strict":
			doStrict = true
		case "-mixed", "-relaxed":
			doMixed = true
		case "-accent", "-plain":
			deAccent = true
		case "-ascii":
			doASCII = true
		case "-flag", "-flags":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Flags argument is missing\n")
				os.Exit(1)
			}
			flgs = args[1]
			// skip past first of two arguments
			args = args[1:]
		// debugging flags
		case "-prepare":
			cmpr = true
			if len(args) > 1 {
				next := args[1]
				// if next argument is not another flag
				if next != "" && next[0] != '-' {
					// get optional data source specifier
					cmprType = next
					// skip past first of two arguments
					args = args[1:]
				}
			}
		case "-ignore":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: -ignore value is missing\n")
				os.Exit(1)
			}
			ignr = args[1]
			// skip past first of two arguments
			args = args[1:]
		case "-missing":
			msng = true
		case "-debug":
			dbug = true
		case "-empty":
			mpty = true
		case "-ident":
			idnt = true
		case "-stats", "-stat":
			stts = true
		case "-timer":
			timr = true
		case "-profile":
			prfl = true
		case "-trial", "-trials":
			trial = true
		default:
			// if not any of the controls, set flag to break out of for loop
			inSwitch = false
		}

		if !inSwitch {
			break
		}

		// skip past argument
		args = args[1:]

		if len(args) < 1 {
			break
		}
	}

	// -flag allows script to set -strict or -mixed from argument
	switch flgs {
	case "strict":
		doStrict = true
	case "mixed":
		doMixed = true
	case "none", "default":
	default:
		if flgs != "" {
			fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized -flag value '%s'\n", flgs)
			os.Exit(1)
		}
	}

	// reality checks on number of processors to use
	// performance degrades if capacity is above maximum number of partitions per second (context switching?)
	if numProcs == 0 {
		if defProcs > 0 {
			numProcs = defProcs
		} else {
			// best performance measurement with current code is obtained when 4 to 6 processors are assigned,
			// varying slightly among queries on PubmedArticle, gene DocumentSummary, and INSDSeq sequence records
			numProcs = 4
		}
	}
	if numProcs > ncpu {
		numProcs = ncpu
	}
	if numProcs > maxProcs {
		numProcs = maxProcs
	}

	// allow simultaneous threads for multiplexed goroutines
	runtime.GOMAXPROCS(numProcs)

	// adjust garbage collection target percentage
	if goGc >= 100 {
		debug.SetGCPercent(goGc)
	}

	// explicit -serv argument overrides -cons ratio
	if numServers > 0 {
		serverRatio = numServers / numProcs
		// if numServers / numProcs is not a whole number, do not print serverRatio in -stats
		if numServers != numProcs*serverRatio {
			serverRatio = 0
		}
	} else {
		numServers = numProcs * serverRatio
	}
	// server limits
	if numServers > 128 {
		numServers = 128
	} else if numServers < 1 {
		numServers = numProcs
	}

	// explicit -chan argument overrides default to number of servers
	if chanDepth == 0 {
		chanDepth = numServers
	}

	// -stats prints number of CPUs and performance tuning values if no other arguments (undocumented)
	if stts && len(args) < 1 {

		fmt.Fprintf(os.Stderr, "CPUs %d\n", ncpu)
		fmt.Fprintf(os.Stderr, "Proc %d\n", numProcs)
		if serverRatio > 0 {
			fmt.Fprintf(os.Stderr, "Cons %d\n", serverRatio)
		}
		fmt.Fprintf(os.Stderr, "Serv %d\n", numServers)
		fmt.Fprintf(os.Stderr, "Chan %d\n", chanDepth)
		fmt.Fprintf(os.Stderr, "Heap %d\n", heapSize)
		fmt.Fprintf(os.Stderr, "Farm %d\n", farmSize)
		if goGc >= 100 {
			fmt.Fprintf(os.Stderr, "Gogc %d\n", goGc)
		}
		fi, err := os.Stdin.Stat()
		if err == nil {
			mode := fi.Mode().String()
			fmt.Fprintf(os.Stderr, "Mode %s\n", mode)
		}
		fmt.Fprintf(os.Stderr, "\n")

		return
	}

	// if copying from local files accessed by identifier, add dummy argument to bypass length tests
	if stsh != "" && indx == "" {
		args = append(args, "-dummy")
	} else if trei || cmpr || pstg != "" {
		args = append(args, "-dummy")
	}

	// expand -archive ~/ to home directory path
	if stsh != "" {

		if stsh[:2] == "~/" {
			cur, err := user.Current()
			if err == nil {
				hom := cur.HomeDir
				stsh = strings.Replace(stsh, "~/", hom+"/", 1)
			}
		}
	}

	// expand -posting ~/ to home directory path
	if pstg != "" {

		if pstg[:2] == "~/" {
			cur, err := user.Current()
			if err == nil {
				hom := cur.HomeDir
				pstg = strings.Replace(pstg, "~/", hom+"/", 1)
			}
		}
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Insufficient command-line arguments supplied to xtract\n")
		os.Exit(1)
	}

	// DOCUMENTATION COMMANDS

	inSwitch = true

	switch args[0] {
	case "-version":
		fmt.Printf("%s\n", xtractVersion)
	case "-help":
		fmt.Printf("xtract %s\n%s\n", xtractVersion, xtractHelp)
	case "-examples", "-example":
		fmt.Printf("xtract %s\n%s\n", xtractVersion, xtractExamples)
	case "-extras", "-extra":
		fmt.Printf("xtract %s\n%s\n", xtractVersion, xtractExtras)
	case "-advanced":
		fmt.Printf("xtract %s\n%s\n", xtractVersion, xtractAdvanced)
	case "-internal", "-internals":
		fmt.Printf("xtract %s\n%s\n", xtractVersion, xtractInternal)
	case "-sample", "-samples":
		// -sample [pubmed|protein|gene] sends specified sample record to stdout (undocumented)
		testType := ""
		if len(args) > 1 {
			testType = args[1]
		}
		switch testType {
		case "pubmed":
			fmt.Printf("%s\n", pubMedArtSample)
		case "protein", "sequence", "insd":
			fmt.Printf("%s\n", insdSeqSample)
		case "gene", "docsum":
			fmt.Printf("%s\n", geneDocSumSample)
		default:
			fmt.Printf("%s\n", pubMedArtSample)
		}
	case "-keys":
		fmt.Printf("%s\n", keyboardShortcuts)
	case "-unix":
		fmt.Printf("%s\n", unixCommands)
	default:
		// if not any of the documentation commands, keep going
		inSwitch = false
	}

	if inSwitch {
		return
	}

	// INITIALIZE TABLES

	tbls := InitTables()
	if tbls == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Problem creating token streamer lookup tables\n")
		os.Exit(1)
	}

	// additional fields passed in master table
	tbls.ChanDepth = chanDepth
	tbls.FarmSize = farmSize
	tbls.HeapSize = heapSize
	tbls.NumServe = numServers

	// base location of local file archive
	tbls.Stash = stsh
	// use compression for local archive files
	tbls.Zipp = zipp
	// generate hash table on stash or fetch
	tbls.Hash = hshv
	// base location of local postings directory
	tbls.Posting = pstg

	if indx != "" {

		// parse parent/element@attribute index
		prnt, match := SplitInTwoAt(indx, "/", RIGHT)
		match, attrib := SplitInTwoAt(match, "@", LEFT)

		// save fields for matching identifiers
		tbls.Index = indx
		tbls.Parent = prnt
		tbls.Match = match
		tbls.Attrib = attrib
	}

	// transformation properties also saved in table
	tbls.DoStrict = doStrict
	tbls.DoMixed = doMixed
	tbls.DeAccent = deAccent
	tbls.DoASCII = doASCII

	// FILE NAME CAN BE SUPPLIED WITH -input COMMAND

	in := os.Stdin

	// check for data being piped into stdin
	isPipe := false
	fi, err := os.Stdin.Stat()
	if err == nil {
		isPipe = bool((fi.Mode() & os.ModeNamedPipe) != 0)
	}

	usingFile := false

	if fileName != "" {

		inFile, err := os.Open(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: Unable to open input file '%s'\n", fileName)
			os.Exit(1)
		}

		defer inFile.Close()

		// use indicated file instead of stdin
		in = inFile
		usingFile = true

		if isPipe && runtime.GOOS != "windows" {
			mode := fi.Mode().String()
			fmt.Fprintf(os.Stderr, "\nERROR: Input data from both stdin and file '%s', mode is '%s'\n", fileName, mode)
			os.Exit(1)
		}
	}

	// check for -input command after extraction arguments
	for _, str := range args {
		if str == "-input" {
			fmt.Fprintf(os.Stderr, "\nERROR: Misplaced -input command\n")
			os.Exit(1)
		}
	}

	// DEBUGGING

	// test reading from input pipe or file (undocumented)
	if args[0] == "-echo" {
		const XMLBUFSIZE = 65536 + 16384
		buffr := make([]byte, XMLBUFSIZE)
		for {
			n, err := in.Read(buffr)
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "ERR: %s, N: %d\n", err, n)
					break
				}
				if n == 0 {
					// EOF and no more data
					break
				}
			}
			if n == 0 {
				fmt.Fprintf(os.Stderr, "N: zero\n")
				continue
			}
			fmt.Fprintf(os.Stdout, "%s", buffr[:n])
		}
		return
	}

	// CREATE XML BLOCK READER FROM STDIN OR FILE

	rdr := NewXMLReader(in, doCompress, doCleanup, doStrict || doMixed)
	if rdr == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create XML Block Reader\n")
		os.Exit(1)
	}

	// DEBUGGING

	// test reading blocks from xml reader (undocumented)
	if args[0] == "-read" {
		for {
			str := rdr.NextBlock()
			if str == "" {
				fmt.Fprintf(os.Stderr, "\n\nSTR: empty\n")
				break
			}
			fmt.Fprintf(os.Stdout, "%s", str)
		}
		fmt.Fprintf(os.Stdout, "\n")
		return
	}

	// SEQUENCE RECORD EXTRACTION COMMAND GENERATOR

	// -insd simplifies extraction of INSDSeq qualifiers
	if args[0] == "-insd" || args[0] == "-insd-" || args[0] == "-insd-idx" {

		addDash := true
		doIndex := false
		// -insd- variant suppresses use of dash as placeholder for missing qualifiers (undocumented)
		if args[0] == "-insd-" {
			addDash = false
		}
		// -insd-idx variant creates word and word pair index using -indices command (undocumented)
		if args[0] == "-insd-idx" {
			doIndex = true
			addDash = false
		}

		args = args[1:]

		insd := ProcessINSD(args, isPipe || usingFile, addDash, doIndex)

		if !isPipe && !usingFile {
			// no piped input, so write output instructions
			fmt.Printf("xtract")
			for _, str := range insd {
				fmt.Printf(" %s", str)
			}
			fmt.Printf("\n")
			return
		}

		// data in pipe, so replace arguments, execute dynamically
		args = insd
	}

	// CITATION MATCHER EXTRACTION COMMAND GENERATOR

	// -hydra filters HydraResponse output by relevance score (undocumented)
	if args[0] == "-hydra" {

		hydra := ProcessHydra(isPipe || usingFile)

		if !isPipe && !usingFile {
			// no piped input, so write output instructions
			fmt.Printf("xtract")
			for _, str := range hydra {
				fmt.Printf(" %s", str)
			}
			fmt.Printf("\n")
			return
		}

		// data in pipe, so replace arguments, execute dynamically
		args = hydra
	}

	// EXPERIMENTAL ENTREZ2INDEX COMMAND GENERATOR

	// -e2index shortcut for experimental indexing code (undocumented)
	if args[0] == "-e2index" {

		args = args[1:]

		res := ProcessE2Index(args, isPipe || usingFile)

		if !isPipe && !usingFile {
			// no piped input, so write output instructions
			fmt.Printf("xtract")
			for _, str := range res {
				fmt.Printf(" %s", str)
			}
			fmt.Printf("\n")
			return
		}

		// data in pipe, so replace arguments, execute dynamically
		args = res
	}

	// CONFIRM INPUT DATA AVAILABILITY AFTER RUNNING COMMAND GENERATORS

	if fileName == "" && runtime.GOOS != "windows" {

		fromStdin := bool((fi.Mode() & os.ModeCharDevice) == 0)
		if !isPipe || !fromStdin {
			mode := fi.Mode().String()
			fmt.Fprintf(os.Stderr, "\nERROR: No data supplied to xtract from stdin or file, mode is '%s'\n", mode)
			os.Exit(1)
		}
	}

	if !usingFile && !isPipe {

		fmt.Fprintf(os.Stderr, "\nERROR: No XML input data supplied to xtract\n")
		os.Exit(1)
	}

	// START PROFILING IF REQUESTED

	if prfl {

		f, err := os.Create("cpu.pprof")
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: Unable to create profile output file\n")
			os.Exit(1)
		}

		pprof.StartCPUProfile(f)

		defer pprof.StopCPUProfile()
	}

	// SPECIAL FORMATTING COMMANDS

	inSwitch = true
	action := NOPROCESS

	switch args[0] {
	case "-format":
		action = DOFORMAT
	case "-outline":
		action = DOOUTLINE
	case "-synopsis":
		action = DOSYNOPSIS
	case "-verify", "-validate":
		action = DOVERIFY
	case "-filter":
		action = DOFILTER
	default:
		// if not any of the formatting commands, keep going
		inSwitch = false
	}

	if inSwitch {
		ProcessXMLStream(rdr, tbls, args, action)
		return
	}

	// INITIALIZE PROCESS TIMER AND RECORD COUNT

	startTime := time.Now()
	recordCount := 0
	byteCount := 0

	// print processing rate and program duration
	printDuration := func(name string) {

		stopTime := time.Now()
		duration := stopTime.Sub(startTime)
		seconds := float64(duration.Nanoseconds()) / 1e9

		if recordCount >= 1000000 {
			fmt.Fprintf(os.Stderr, "\nXtract processed %d million %s in %.3f seconds", recordCount/1000000, name, seconds)
		} else {
			fmt.Fprintf(os.Stderr, "\nXtract processed %d %s in %.3f seconds", recordCount, name, seconds)
		}

		if seconds >= 0.001 && recordCount > 0 {
			rate := int(float64(recordCount) / seconds)
			if rate >= 1000000 {
				fmt.Fprintf(os.Stderr, " (%d mega%s/second", rate/1000000, name)
			} else {
				fmt.Fprintf(os.Stderr, " (%d %s/second", rate, name)
			}
			if byteCount > 0 {
				rate := int(float64(byteCount) / seconds)
				if rate >= 1000000 {
					fmt.Fprintf(os.Stderr, ", %d megabytes/second", rate/1000000)
				} else if rate >= 1000 {
					fmt.Fprintf(os.Stderr, ", %d kilobytes/second", rate/1000)
				} else {
					fmt.Fprintf(os.Stderr, ", %d bytes/second", rate)
				}
			}
			fmt.Fprintf(os.Stderr, ")")
		}

		fmt.Fprintf(os.Stderr, "\n\n")
	}

	// SPECIFY STRINGS TO GO BEFORE AND AFTER ENTIRE OUTPUT OR EACH RECORD

	head := ""
	tail := ""

	hd := ""
	tl := ""

	for {

		inSwitch = true

		switch args[0] {
		case "-head":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Pattern missing after -head command\n")
				os.Exit(1)
			}
			head = ConvertSlash(args[1])
		case "-tail":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Pattern missing after -tail command\n")
				os.Exit(1)
			}
			tail = ConvertSlash(args[1])
		case "-hd":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Pattern missing after -hd command\n")
				os.Exit(1)
			}
			hd = ConvertSlash(args[1])
		case "-tl":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "\nERROR: Pattern missing after -tl command\n")
				os.Exit(1)
			}
			tl = ConvertSlash(args[1])
		default:
			// if not any of the controls, set flag to break out of for loop
			inSwitch = false
		}

		if !inSwitch {
			break
		}

		// skip past arguments
		args = args[2:]

		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "\nERROR: Insufficient command-line arguments supplied to xtract\n")
			os.Exit(1)
		}
	}

	// per-record head and tail passed in master table
	tbls.Hd = hd
	tbls.Tl = tl

	// PRODUCE DIRECTORY SUBPATH FROM IDENTIFIER

	// -trie converts identifier to directory subpath plus file name (undocumented)
	if trei {

		scanr := bufio.NewScanner(rdr.Reader)

		sfx := ".xml"
		if zipp {
			sfx = ".xml.gz"
		}

		// read lines of identifiers
		for scanr.Scan() {

			file := scanr.Text()
			var arry [132]rune
			trie := MakeArchiveTrie(file, arry)
			if trie == "" || file == "" {
				continue
			}

			fpath := path.Join(trie, file+sfx)
			if fpath == "" {
				continue
			}

			os.Stdout.WriteString(fpath)
			os.Stdout.WriteString("\n")
		}

		return
	}

	// CREATE POSTINGS FILES USING TRIE ON TERM CHARACTERS

	// -posting produces postings files (undocumented)
	if pstg != "" {

		trml := CreateTermListReader(rdr.Reader, tbls)
		pstr := CreatePosters(tbls, trml)

		if trml == nil || pstr == nil {
			fmt.Fprintf(os.Stderr, "\nERROR: Unable to create postings generator\n")
			os.Exit(1)
		}

		// drain output channel
		for _ = range pstr {

			recordCount++
			runtime.Gosched()
		}

		debug.FreeOSMemory()

		if timr {
			printDuration("terms")
		}

		return
	}

	// CHECK FOR MISSING RECORDS IN LOCAL DIRECTORY INDEXED BY TRIE ON IDENTIFIER

	// -archive plus -missing checks for missing records
	if stsh != "" && msng {

		scanr := bufio.NewScanner(rdr.Reader)

		sfx := ".xml"
		if zipp {
			sfx = ".xml.gz"
		}

		// read lines of identifiers
		for scanr.Scan() {

			file := scanr.Text()
			var arry [132]rune
			trie := MakeArchiveTrie(file, arry)
			if trie == "" || file == "" {
				continue
			}

			fpath := path.Join(stsh, trie, file+sfx)
			if fpath == "" {
				continue
			}

			_, err := os.Stat(fpath)

			// if failed to find ".xml" file, try ".xml.gz" without requiring -gzip
			if err != nil && os.IsNotExist(err) && !zipp {
				fpath := path.Join(stsh, trie, file+".xml.gz")
				if fpath == "" {
					continue
				}
				_, err = os.Stat(fpath)
			}
			if err != nil && os.IsNotExist(err) {
				// record is missing from local file cache
				os.Stdout.WriteString(file)
				os.Stdout.WriteString("\n")
			}
		}

		return
	}

	// RETRIEVE XML COMPONENT RECORDS FROM LOCAL DIRECTORY INDEXED BY TRIE ON IDENTIFIER

	// -archive without -index retrieves XML files in trie-based directory structure
	if stsh != "" && indx == "" {

		uidq := CreateUIDReader(rdr.Reader, tbls)
		strq := CreateFetchers(tbls, uidq)
		unsq := CreateUnshuffler(tbls, strq)

		if uidq == nil || strq == nil || unsq == nil {
			fmt.Fprintf(os.Stderr, "\nERROR: Unable to create stash reader\n")
			os.Exit(1)
		}

		if head != "" {
			os.Stdout.WriteString(head)
			os.Stdout.WriteString("\n")
		}

		// drain output channel
		for curr := range unsq {

			str := curr.Text

			if str == "" {
				continue
			}

			recordCount++

			if hd != "" {
				os.Stdout.WriteString(hd)
				os.Stdout.WriteString("\n")
			}

			if hshv {
				// calculate hash code for verification table
				hsh := crc32.NewIEEE()
				hsh.Write([]byte(curr.Text))
				val := hsh.Sum32()
				res := strconv.FormatUint(uint64(val), 10)
				txt := curr.Ident + "\t" + res + "\n"
				os.Stdout.WriteString(txt)
			} else {
				// send result to output
				os.Stdout.WriteString(curr.Text)
				if !strings.HasSuffix(curr.Text, "\n") {
					os.Stdout.WriteString("\n")
				}
			}

			if tl != "" {
				os.Stdout.WriteString(tl)
				os.Stdout.WriteString("\n")
			}
		}

		if tail != "" {
			os.Stdout.WriteString(tail)
			os.Stdout.WriteString("\n")
		}

		debug.FreeOSMemory()

		if timr {
			printDuration("records")
		}

		return
	}

	// ENSURE PRESENCE OF PATTERN ARGUMENT

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Insufficient command-line arguments supplied to xtract\n")
		os.Exit(1)
	}

	// allow -record as synonym of -pattern (undocumented)
	if args[0] == "-record" || args[0] == "-Record" {
		args[0] = "-pattern"
	}

	// make sure top-level -pattern command is next
	if args[0] != "-pattern" && args[0] != "-Pattern" {
		fmt.Fprintf(os.Stderr, "\nERROR: No -pattern in command-line arguments\n")
		os.Exit(1)
	}
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "\nERROR: Item missing after -pattern command\n")
		os.Exit(1)
	}

	topPat := args[1]
	if topPat == "" {
		fmt.Fprintf(os.Stderr, "\nERROR: Item missing after -pattern command\n")
		os.Exit(1)
	}
	if strings.HasPrefix(topPat, "-") {
		fmt.Fprintf(os.Stderr, "\nERROR: Misplaced %s command\n", topPat)
		os.Exit(1)
	}

	// look for -pattern Parent/* construct for heterogeneous data, e.g., -pattern PubmedArticleSet/*
	topPattern, star := SplitInTwoAt(topPat, "/", LEFT)
	if topPattern == "" {
		return
	}

	parent := ""
	if star == "*" {
		parent = topPattern
	} else if star != "" {
		fmt.Fprintf(os.Stderr, "\nERROR: -pattern Parent/Child construct is not supported\n")
		os.Exit(1)
	}

	// COMPARE XML UPDATES TO LOCAL DIRECTORY, RETAIN NEW OR SUBSTANTIVELY CHANGED RECORDS

	// -prepare plus -archive plus -index plus -pattern compares XML files against stash
	if stsh != "" && indx != "" && cmpr {

		doReport := false
		if cmprType == "" || cmprType == "report" {
			doReport = true
		} else if cmprType != "release" {
			fmt.Fprintf(os.Stderr, "\nERROR: -prepare argument must be release or report\n")
			os.Exit(1)
		}

		if head != "" {
			os.Stdout.WriteString(head)
			os.Stdout.WriteString("\n")
		}

		PartitionPattern(topPattern, star, rdr,
			func(rec int, ofs int64, str string) {
				recordCount++

				id := ProcessQuery(str[:], parent, rec, nil, tbls, DOINDEX)
				if id == "" {
					return
				}

				var arry [132]rune
				trie := MakeArchiveTrie(id, arry)
				if trie == "" {
					return
				}

				fpath := path.Join(stsh, trie, id+".xml")
				if fpath == "" {
					return
				}

				// print new or updated XML record
				printRecord := func(stn string, isNew bool) {

					if stn == "" {
						return
					}

					if doReport {
						if isNew {
							os.Stdout.WriteString("NW ")
							os.Stdout.WriteString(id)
							os.Stdout.WriteString("\n")
						} else {
							os.Stdout.WriteString("UP ")
							os.Stdout.WriteString(id)
							os.Stdout.WriteString("\n")
						}
						return
					}

					if hd != "" {
						os.Stdout.WriteString(hd)
						os.Stdout.WriteString("\n")
					}

					os.Stdout.WriteString(stn)
					os.Stdout.WriteString("\n")

					if tl != "" {
						os.Stdout.WriteString(tl)
						os.Stdout.WriteString("\n")
					}
				}

				_, err := os.Stat(fpath)
				if err != nil && os.IsNotExist(err) {
					// new record
					printRecord(str, true)
					return
				}
				if err != nil {
					return
				}

				buf, err := ioutil.ReadFile(fpath)
				if err != nil {
					return
				}

				txt := string(buf[:])
				if strings.HasSuffix(txt, "\n") {
					tlen := len(txt)
					txt = txt[:tlen-1]
				}

				// check for optional -ignore argument
				if ignr != "" {

					// ignore differences inside specified object
					ltag := "<" + ignr + ">"
					sleft, _ := SplitInTwoAt(str, ltag, LEFT)
					tleft, _ := SplitInTwoAt(txt, ltag, LEFT)

					rtag := "</" + ignr + ">"
					_, srght := SplitInTwoAt(str, rtag, RIGHT)
					_, trght := SplitInTwoAt(txt, rtag, RIGHT)

					if sleft == tleft && srght == trght {
						if doReport {
							os.Stdout.WriteString("NO ")
							os.Stdout.WriteString(id)
							os.Stdout.WriteString("\n")
						}
						return
					}

				} else {

					// compare entirety of objects
					if str == txt {
						if doReport {
							os.Stdout.WriteString("NO ")
							os.Stdout.WriteString(id)
							os.Stdout.WriteString("\n")
						}
						return
					}
				}

				// substantively modified record
				printRecord(str, false)
			})

		if tail != "" {
			os.Stdout.WriteString(tail)
			os.Stdout.WriteString("\n")
		}

		if timr {
			printDuration("records")
		}

		return
	}

	// SAVE XML COMPONENT RECORDS TO LOCAL DIRECTORY INDEXED BY TRIE ON IDENTIFIER

	// -archive plus -index plus -pattern saves XML files in trie-based directory structure
	if stsh != "" && indx != "" {

		xmlq := CreateProducer(topPattern, star, rdr, tbls)
		idnq := CreateExaminers(tbls, parent, xmlq)
		unsq := CreateUnshuffler(tbls, idnq)
		unqq := CreateUniquer(tbls, unsq)
		delq := unqq
		if dltd != "" {
			// only create deleter if -skip argument is present
			delq = CreateDeleter(tbls, dltd, unqq)
		}
		stsq := CreateStashers(tbls, delq)

		if xmlq == nil || idnq == nil || unsq == nil || unqq == nil || delq == nil || stsq == nil {
			fmt.Fprintf(os.Stderr, "\nERROR: Unable to create stash generator\n")
			os.Exit(1)
		}

		// drain output channel
		for str := range stsq {

			if hshv {
				// print table of UIDs and hash values
				os.Stdout.WriteString(str)
			}

			recordCount++
			runtime.Gosched()
		}

		debug.FreeOSMemory()

		if timr {
			printDuration("records")
		}

		return
	}

	// GENERATE RECORD INDEX ON XML INPUT FILE

	// -index plus -pattern prints record identifier, file offset, and XML size
	if indx != "" {

		lbl := ""
		// check for optional filename label after -pattern argument (undocumented)
		if len(args) > 3 && args[2] == "-lbl" {
			lbl = args[3]

			lbl = strings.TrimSpace(lbl)
			if strings.HasPrefix(lbl, "medline") {
				lbl = lbl[7:]
			}
			if strings.HasSuffix(lbl, ".xml.gz") {
				xlen := len(lbl)
				lbl = lbl[:xlen-7]
			}
			lbl = strings.TrimSpace(lbl)
		}

		// legend := "ID\tREC\tOFST\tSIZE"

		PartitionPattern(topPattern, star, rdr,
			func(rec int, ofs int64, str string) {
				recordCount++

				id := ProcessQuery(str[:], parent, rec, nil, tbls, DOINDEX)
				if id == "" {
					return
				}
				if lbl != "" {
					fmt.Printf("%s\t%d\t%d\t%d\t%s\n", id, rec, ofs, len(str), lbl)
				} else {
					fmt.Printf("%s\t%d\t%d\t%d\n", id, rec, ofs, len(str))
				}
			})

		if timr {
			printDuration("records")
		}

		return
	}

	// FILTER XML RECORDS BY PRESENCE OF ONE OR MORE PHRASES

	// -phrase plus -pattern filters by phrase in XML
	if phrs != "" && len(args) == 2 {

		// cleanupPhrase splits at punctuation, but leaves < and > in to avoid false positives
		cleanupPhrase := func(str string, keepPlus bool) string {

			var buffer bytes.Buffer

			for _, ch := range str {
				if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
					buffer.WriteRune(ch)
				} else if ch == '<' || ch == '>' {
					buffer.WriteRune(' ')
					buffer.WriteRune(ch)
					buffer.WriteRune(' ')
				} else if ch == '+' && keepPlus {
					buffer.WriteRune(' ')
					buffer.WriteRune(ch)
					buffer.WriteRune(' ')
				} else {
					buffer.WriteRune(' ')
				}
			}

			return buffer.String()
		}

		phrs = cleanupPhrase(phrs, true)
		phrs = strings.TrimSpace(phrs)
		phrs = CompressRunsOfSpaces(phrs)
		phrs = RemoveUnicodeMarkup(phrs)
		phrs = strings.ToUpper(phrs)

		// multiple phrases separated by plus sign
		clauses := strings.Split(phrs, " + ")

		if head != "" {
			os.Stdout.WriteString(head)
			os.Stdout.WriteString("\n")
		}

		PartitionPattern(topPattern, star, rdr,
			func(rec int, ofs int64, str string) {
				recordCount++

				srch := cleanupPhrase(str[:], false)
				srch = strings.ToUpper(srch)
				srch = CompressRunsOfSpaces(srch)
				srch = RemoveUnicodeMarkup(srch)
				srch = strings.ToUpper(srch)

				for _, item := range clauses {
					// require presence of each clause
					if !strings.Contains(srch, item) {
						return
					}
				}

				if hd != "" {
					os.Stdout.WriteString(hd)
					os.Stdout.WriteString("\n")
				}

				// write selected record
				os.Stdout.WriteString(str)
				if !strings.HasSuffix(str, "\n") {
					os.Stdout.WriteString("\n")
				}

				if tl != "" {
					os.Stdout.WriteString(tl)
					os.Stdout.WriteString("\n")
				}
			})

		if tail != "" {
			os.Stdout.WriteString(tail)
			os.Stdout.WriteString("\n")
		}

		if timr {
			printDuration("records")
		}

		return
	}

	// PARSE AND VALIDATE EXTRACTION ARGUMENTS

	// parse nested exploration instruction from command-line arguments
	cmds := ParseArguments(args, topPattern)
	if cmds == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Problem parsing command-line arguments\n")
		os.Exit(1)
	}

	// PERFORMANCE TIMING COMMAND

	// -stats with an extraction command prints XML size and processing time for each record
	if stts {

		legend := "REC\tOFST\tSIZE\tTIME"

		PartitionPattern(topPattern, star, rdr,
			func(rec int, ofs int64, str string) {
				beginTime := time.Now()
				ProcessQuery(str[:], parent, rec, cmds, tbls, DOQUERY)
				endTime := time.Now()
				duration := endTime.Sub(beginTime)
				micro := int(float64(duration.Nanoseconds()) / 1e3)
				if legend != "" {
					fmt.Printf("%s\n", legend)
					legend = ""
				}
				fmt.Printf("%d\t%d\t%d\t%d\n", rec, ofs, len(str), micro)
			})

		return
	}

	// PERFORMANCE OPTIMIZATION FUNCTION

	// -trial -input fileName runs the specified extraction for each -proc from 1 to nCPU
	if trial && fileName != "" {

		legend := "CPU\tRATE\tDEV"

		for numServ := 1; numServ <= ncpu; numServ++ {

			tbls.NumServe = numServ

			runtime.GOMAXPROCS(numServ)

			sum := 0
			count := 0
			mean := 0.0
			m2 := 0.0

			// calculate mean and standard deviation of processing rate
			for trials := 0; trials < 5; trials++ {

				inFile, err := os.Open(fileName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nERROR: Unable to open input file '%s'\n", fileName)
					os.Exit(1)
				}

				rdr := NewXMLReader(inFile, doCompress, doCleanup, doStrict || doMixed)
				if rdr == nil {
					fmt.Fprintf(os.Stderr, "\nERROR: Unable to read input file\n")
					os.Exit(1)
				}

				xmlq := CreateProducer(topPattern, star, rdr, tbls)
				tblq := CreateConsumers(cmds, tbls, parent, xmlq)

				if xmlq == nil || tblq == nil {
					fmt.Fprintf(os.Stderr, "\nERROR: Unable to create servers\n")
					os.Exit(1)
				}

				begTime := time.Now()
				recordCount = 0

				for _ = range tblq {
					recordCount++
					runtime.Gosched()
				}

				inFile.Close()

				debug.FreeOSMemory()

				endTime := time.Now()
				expended := endTime.Sub(begTime)
				secs := float64(expended.Nanoseconds()) / 1e9

				if secs >= 0.000001 && recordCount > 0 {
					speed := int(float64(recordCount) / secs)
					sum += speed
					count++
					x := float64(speed)
					delta := x - mean
					mean += delta / float64(count)
					m2 += delta * (x - mean)
				}
			}

			if legend != "" {
				fmt.Printf("%s\n", legend)
				legend = ""
			}
			if count > 1 {
				vrc := m2 / float64(count-1)
				dev := int(math.Sqrt(vrc))
				fmt.Printf("%d\t%d\t%d\n", numServ, sum/count, dev)
			}
		}

		return
	}

	// PROCESS SINGLE SELECTED RECORD IF -pattern ARGUMENT IS IMMEDIATELY FOLLOWED BY -position COMMAND

	if cmds.Visit == topPat && cmds.Position != "" {

		qry := ""
		idx := 0

		if cmds.Position == "first" {

			PartitionPattern(topPattern, star, rdr,
				func(rec int, ofs int64, str string) {
					if rec == 1 {
						qry = str
						idx = rec
					}
				})

		} else if cmds.Position == "last" {

			PartitionPattern(topPattern, star, rdr,
				func(rec int, ofs int64, str string) {
					qry = str
					idx = rec
				})

		} else {

			// use numeric position
			number, err := strconv.Atoi(cmds.Position)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: Unrecognized position '%s'\n", cmds.Position)
				os.Exit(1)
			}

			PartitionPattern(topPattern, star, rdr,
				func(rec int, ofs int64, str string) {
					if rec == number {
						qry = str
						idx = rec
					}
				})
		}

		if qry == "" {
			return
		}

		// clear position on top node to prevent condition test failure
		cmds.Position = ""

		// process single selected record
		res := ProcessQuery(qry[:], parent, idx, cmds, tbls, DOQUERY)

		if res != "" {
			fmt.Printf("%s\n", res)
		}

		return
	}

	// LAUNCH PRODUCER, CONSUMER, AND UNSHUFFLER SERVERS

	// launch producer goroutine to partition XML by pattern
	xmlq := CreateProducer(topPattern, star, rdr, tbls)

	// launch consumer goroutines to parse and explore partitioned XML objects
	tblq := CreateConsumers(cmds, tbls, parent, xmlq)

	// launch unshuffler goroutine to restore order of results
	unsq := CreateUnshuffler(tbls, tblq)

	if xmlq == nil || tblq == nil || unsq == nil {
		fmt.Fprintf(os.Stderr, "\nERROR: Unable to create servers\n")
		os.Exit(1)
	}

	// PERFORMANCE SUMMARY

	if dbug {

		// drain results, but suppress extraction output
		for ext := range unsq {
			byteCount += len(ext.Text)
			recordCount++
			runtime.Gosched()
		}

		// force garbage collection, return memory to operating system
		debug.FreeOSMemory()

		// print processing parameters as XML object
		stopTime := time.Now()
		duration := stopTime.Sub(startTime)
		seconds := float64(duration.Nanoseconds()) / 1e9

		// Threads is a more easily explained concept than GOMAXPROCS
		fmt.Printf("<Xtract>\n")
		fmt.Printf("  <Threads>%d</Threads>\n", numProcs)
		fmt.Printf("  <Parsers>%d</Parsers>\n", numServers)
		fmt.Printf("  <Time>%.3f</Time>\n", seconds)
		if seconds >= 0.001 && recordCount > 0 {
			rate := int(float64(recordCount) / seconds)
			fmt.Printf("  <Rate>%d</Rate>\n", rate)
		}
		fmt.Printf("</Xtract>\n")

		return
	}

	// DRAIN OUTPUT CHANNEL TO EXECUTE EXTRACTION COMMANDS, RESTORE OUTPUT ORDER WITH HEAP

	var buffer bytes.Buffer
	count := 0
	okay := false

	// printResult prints output for current pattern, handles -empty and -ident flags, and periodically flushes buffer
	printResult := func(curr Extract) {

		str := curr.Text

		if mpty {

			if str == "" {

				okay = true

				idx := curr.Index
				val := strconv.Itoa(idx)
				buffer.WriteString(val[:])
				buffer.WriteString("\n")

				count++
			}

		} else if str != "" {

			okay = true

			if idnt {
				idx := curr.Index
				val := strconv.Itoa(idx)
				buffer.WriteString(val[:])
				buffer.WriteString("\t")
			}

			// save output to byte buffer
			buffer.WriteString(str[:])

			count++
		}

		if count > 1000 {
			count = 0
			txt := buffer.String()
			if txt != "" {
				// print current buffer
				os.Stdout.WriteString(txt[:])
			}
			buffer.Reset()
		}
	}

	if head != "" {
		buffer.WriteString(head[:])
		buffer.WriteString("\n")
	}

	// drain unshuffler channel
	for curr := range unsq {

		// send result to output
		printResult(curr)

		recordCount++
	}

	if tail != "" {
		buffer.WriteString(tail[:])
		buffer.WriteString("\n")
	}

	// do not print head or tail if no extraction output
	if okay {
		txt := buffer.String()
		if txt != "" {
			// print final buffer
			os.Stdout.WriteString(txt[:])
		}
	}
	buffer.Reset()

	// force garbage collection and return memory before calculating processing rate
	debug.FreeOSMemory()

	if timr {
		printDuration("records")
	}
}
