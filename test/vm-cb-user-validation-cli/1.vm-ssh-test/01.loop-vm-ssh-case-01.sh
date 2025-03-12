#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|ncpvpc|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|ncpvpc|nhncloud number'
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


for (( num=1; num <= $2; num++ ))
do
	# ex) ./vm-ssh-test.sh aws 1
	./vm-ssh-test.sh $1 $num &
	if [ `expr $num % 10` = 0 ]; then # tencent RequestLimitExceeded = 10/sec
		sleep 3
	fi
	echo -e "\n\n"
done

echo -e "\n\n"

