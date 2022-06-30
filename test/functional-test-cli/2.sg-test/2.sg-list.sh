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
echo -e "# Try to list $1 SGs"
echo -e "###########################################################"
echo -e "\n\n"


# ex) ../common/3.sg-list.sh aws
../common/3.sg-list.sh $1 

echo -e "\n\n"

