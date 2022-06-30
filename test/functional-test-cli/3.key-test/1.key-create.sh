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
echo -e "# Try to create $1 KEY"
echo -e "###########################################################"
echo -e "\n\n"


../common/1.key-create.sh $1

#### Check sync called
ret=`../common/2.key-get.sh $1 2>&1 | grep NameId`

if [ "$ret" ];then
        echo -e "\n-------------------------------------------------------------- $0 $1 : pass"
else
        echo -e "\n-------------------------------------------------------------- $0 $1 : fail"
fi

echo -e "\n\n"
