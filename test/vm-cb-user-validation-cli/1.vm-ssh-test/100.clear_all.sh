#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|ncpvpc|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|ncpvpc|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# Terminate VM "
echo -e "# Delete resources: Keypair => SG01 => VPC/Subnet "
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1

./99.clear_vm.sh $1 $2

echo -e "# Try to delete test key"
for (( i=1; i <= 120; i++ ))
do
        ret=`../common/7.key-delete.sh $1`
        echo -e "$ret"

        result=`echo -e "$ret" |grep "does not exist"`
        if [ "$result" ];then
                break;
        else
                sleep 2
        fi
done


echo -e "# Try to delete test Security Group"
for (( i=1; i <= 120; i++ ))
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

echo -e "# Try to delete test VPC/Subnet"
for (( i=1; i <= 120; i++ ))
do
        ret=`../common/7.vpc-delete.sh $1`
        echo -e "$ret"

        result=`echo -e "$ret" |grep "does not exist"`
        if [ "$result" ];then
                break;
        else
                sleep 2
        fi
done


rm -f ./${KEYPAIR_NAME}.pem

echo -e "###########################################################"
echo -e "# Terminated VM "
echo -e "# Deleted resources: Keypair => SG01 => VPC/Subnet "
echo -e "###########################################################"

echo -e "\n\n"

