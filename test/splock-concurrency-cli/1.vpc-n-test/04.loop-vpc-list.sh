#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

#if [ "$2" = "" ]; then
#        echo
#        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
#        echo -e '\n\tex) '$0' aws 5'
#        echo
#        exit 0;
#fi

source ../common/setup.env $1
source setup.env $1

rm -f $RESULT_FNAME
rm -f $TERMINATED_FNAME

echo -e "\n\n"
echo -e "###########################################################"
echo -e "# Try to list $1 VPCs"
echo -e "###########################################################"
echo -e "\n\n"


while true
do
	# ex) ../common/3.vpc-list.sh aws
	../common/3.vpc-list.sh $1 
	#if [ `expr $num % 10` = 0 ]; then # tencent RequestLimitExceeded = 10/sec
	#	sleep 3
	#fi
	sleep 2
	echo -e "\n\n"
done

echo -e "\n\n"

