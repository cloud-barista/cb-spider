#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

# delete: VM => Key => SG => VPC/Subnet
cd 4.vm-test
./9.vm-terminate.sh $1

cd ../3.key-test
./4.key-delete.sh $1

cd ../2.sg-test
./6.sg-delete.sh $1

cd ../1.vpc-test/
./6.vpc-delete.sh $1

cd ..
