#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud|ncpvpc|ktvpc'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

source $1/setup.env

# Create: VPC/Subnet => SG => Key => VM1, VM2
./common/prepare-nlb-test.sh $1
