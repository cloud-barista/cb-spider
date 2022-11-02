#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

source ../common/setup.env $1
source setup.env $1

echo -e "\n\n"
echo -e "###########################################################"
echo -e "# Try to delete $1 VPC"
echo -e "###########################################################"
echo -e "\n\n"

# ex) ../common/7.vpc-delete.sh aws
../common/7.vpc-delete.sh $1

##### Check sync called
ret=`../common/2.vpc-get.sh $1 2>&1 | grep NameId`

if [ "$ret" ];then
        echo -e "\n-------------------------------------------------------------- $0 $1 : fail"
else
        echo -e "\n-------------------------------------------------------------- $0 $1 : pass"
fi
echo -e "\n\n"

