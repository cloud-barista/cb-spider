#!/bin/sh

mkdir -p az-data
ofile_name=./az-data/image_list_az_`(date +%Y.%m.%d.%H)`.json
sfile_name=./az-data/image_list_az_`(date +%Y.%m.%d.%H)`.stat

starttime=$( /bin/date +%s )

az vm image list --all |json_pp > $ofile_name

stoptime=$( /bin/date +%s )
runtime=$(($stoptime-$starttime))

image_count=$(grep SystemId  $ofile_name |wc -l)
image_info_size=$(ls -hs $ofile_name |awk '{print $1}')
echo ""
echo "------------------------------------------------------------------------------------------------"
echo "  Region: ## ,   Image #: $image_count ,   Image Info Size: $image_info_size ,   Elapsed time: $runtime sec"
echo "------------------------------------------------------------------------------------------------"

# write image count and elapsed time
echo "------------------------------------------------------------------------------------------------" >> $sfile_name
echo "  Region: ## ,  Image #: $image_count ,   Image Info Size: $image_info_size ,   Elapsed time: $runtime sec" >> $sfile_name
