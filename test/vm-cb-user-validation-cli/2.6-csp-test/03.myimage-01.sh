#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|ncpvpc|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# 1.create: MyImage"
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1


### 1.create: MyImage
../common/12.myimage-create.sh $1

echo -e "\n\n"

