#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn|ncp|ktvpc'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

source $1/setup.env

./common/removevm-nlb-test.sh $1
