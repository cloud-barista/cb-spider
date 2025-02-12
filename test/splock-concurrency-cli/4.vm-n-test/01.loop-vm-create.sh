#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

source ../common/setup.env $1
source setup.env $1

rm -f $RESULT_FNAME
rm -f $TERMINATED_FNAME

echo -e "\n\n"
echo -e "###########################################################"
echo -e "# Try to create $1 VMs: $2"
echo -e "###########################################################"
echo -e "\n\n"


while true
do
	for (( num=1; num <= $2; num++ ))
	do
		for (( i=1; i <= 3; i++ ))
		do
			../common/1.vm-create.sh $1 $num $i &
			#if [ `expr $num % 10` = 0 ]; then # tencent RequestLimitExceeded = 10/sec
			#	sleep 3
			#fi
			echo -e "\n\n"
		done
	done
	sleep 5 
done

echo -e "\n\n"

