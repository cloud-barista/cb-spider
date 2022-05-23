#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# 1.create: VPC/Subnet => SG01 => Keypair(save private key)"
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1


### 1.create: VPC/Subnet => SG01 => Keypair(save private key)
../common/1.prepare-resources.sh $1

echo -e "\n\n"

