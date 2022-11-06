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
echo -e "# Try to delete $1 KeyPair"
echo -e "###########################################################"
echo -e "\n\n"

../common/7.key-delete.sh $1
rm -f ./${KEYPAIR_NAME}.pem

#### Check sync called
ret=`../common/2.key-get.sh $1 2>&1 | grep NameId`

if [ "$ret" ];then
        echo -e "\n-------------------------------------------------------------- $0 $1 : fail"
else
        echo -e "\n-------------------------------------------------------------- $0 $1 : pass"
fi

echo -e "\n\n"

