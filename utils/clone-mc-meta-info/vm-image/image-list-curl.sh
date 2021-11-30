#!/bin/sh

mkdir -p data
ofile_name=./data/image_list_curl_$1_`(date +%Y.%m.%d.%H)`.json
sfile_name=./data/image_list_curl_$1_`(date +%Y.%m.%d.%H)`.stat

starttime=$( /bin/date +%s )

curl -sX GET http://localhost:1024/spider/vmimage -H 'Content-Type: application/json' -d '{ "ConnectionName": "'$1'"}' |json_pp > $ofile_name 

stoptime=$( /bin/date +%s )
runtime=$(($stoptime-$starttime))

image_count=$(grep SystemId  $ofile_name |wc -l)
image_info_size=$(ls -hs $ofile_name |awk '{print $1}')
echo ""
echo "------------------------------------------------------------------------------------------------"
echo "  Region: $1 ,   Image #: $image_count ,   Image Info Size: $image_info_size ,   Elapsed time: $runtime sec"
echo "------------------------------------------------------------------------------------------------"

# write image count and elapsed time
echo "------------------------------------------------------------------------------------------------" >> $sfile_name
echo "  Region: $1 ,  Image #: $image_count ,   Image Info Size: $image_info_size ,   Elapsed time: $runtime sec" >> $sfile_name
