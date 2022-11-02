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
echo -e "# Try to reboot $1 VM"
echo -e "###########################################################"
echo -e "\n\n"


../common/3-3.vm-reboot.sh $1 

#### Check sync called
for i in {1..30}
do
	ret=`../common/2.vm-getstatus.sh $1 2>&1 | grep Running`

        if [ "$ret" ];then
                echo -e "\n-------------------------------------------------------------- $0 $1 : pass"
                exit 0
        else
                echo -e "\n-------------------------------------------------------------- $0 $1 : one more try"
                sleep 2
        fi
done

echo -e "\n-------------------------------------------------------------- $0 $1 : fail"


echo -e "\n\n"

