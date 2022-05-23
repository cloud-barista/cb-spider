#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# Try to terminate $1 VMs: $2 "
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1


for (( num=1; num <= $2; num++ ))
do
	./single_clear_vm.sh $1 $num &

        if [ `expr $num % 10` = 0 ]; then # tencent RequestLimitExceeded = 10/sec
                sleep 3
        fi
done

echo -e "\n\n"

