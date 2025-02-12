#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncpvpc|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# 1.destroy: All Resources in Connection                   "
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1

../common/10.destroy.sh $1

echo -e "\n\n"

