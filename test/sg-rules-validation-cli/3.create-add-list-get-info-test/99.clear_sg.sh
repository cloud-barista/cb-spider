#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# Delete SG "
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1

echo -e "# Try to delete test Security Group"
for (( i=1; i <= 30; i++ ))
do
        ret=`../common/7.sg-delete.sh $1`
        echo -e "$ret"

        result=`echo -e "$ret" |grep "does not exist"`
        if [ "$result" ];then
                break;
        else
                sleep 2
        fi
done


echo -e "###########################################################"
echo -e "# Deleted SG "
echo -e "###########################################################"

echo -e "\n\n"
