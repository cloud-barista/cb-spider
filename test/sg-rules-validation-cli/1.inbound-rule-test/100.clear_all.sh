#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# terminate VM "
echo -e "# delete resources: Keypair => SG01 => VPC/Subnet "
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1

./99.clear_vm.sh $1

for (( num=1; num <= 3; num++ ))
do
	../common/8.clear-resources.sh $1
	sleep 1
done

rm ./${KEYPAIR_NAME}.pem

echo -e "\n\n"

