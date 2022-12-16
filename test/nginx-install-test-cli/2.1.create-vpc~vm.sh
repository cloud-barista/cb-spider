#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

# Create: VPC/Subnet => SG => Key => VM
cd 1.vpc-test/
./1.vpc-create.sh $1

cd ../2.sg-test
./1.sg-create.sh $1

cd ../3.key-test
./1.key-create.sh $1

cd ../4.vm-test
./1.vm-start.sh $1

cd ..

