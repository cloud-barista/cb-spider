#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

source ../common/setup.env $1
source setup.env $1

echo -e "\n\n"
echo -e "###########################################################"
echo -e "# Try to Remove subnet $1 "
echo -e "###########################################################"
echo -e "\n\n"


../common/1-2.subnet-remove.sh $1 

#### Check sync called
# the name of Added Subnet: SUBNET_NAME=${SUBNET_NAME}-Added
ret=`../common/2.vpc-get.sh $1 2>&1 | grep Added`

if [ "$ret" ];then
        echo -e "\n-------------------------------------------------------------- $0 $1 : fail"
else
        echo -e "\n-------------------------------------------------------------- $0 $1 : pass"
fi

echo -e "\n\n"

