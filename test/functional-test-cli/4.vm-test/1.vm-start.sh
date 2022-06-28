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
echo -e "# Try to start $1 VM"
echo -e "###########################################################"
echo -e "\n\n"


../common/1.vm-create.sh $1

#### Check sync called and Make sure cb-user prepared
P_IP=`../common/./6.vm-get.sh $1 |grep PublicIP: |awk '{print $2}'`
ssh-keygen -f "/home/ubuntu/.ssh/known_hosts" -R $P_IP 2> /dev/null

SSH_CMD="ssh -i ../3.key-test/${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no -o ConnectTimeout=10 cb-user@$P_IP whoami"

### for debug
#$SSH_CMD

#### Check SSH Call by cb-user
ret=`$SSH_CMD 2>&1 | grep cb-user`
#echo $SSH_CMD
echo 
echo $ret
echo 
if [ "$ret" = "cb-user"  ];then
        echo -e "\n-------------------------------------------------------------- $0 $1 : pass"
else
        echo -e "\n-------------------------------------------------------------- $0 $1 : fail"
fi

echo -e "\n\n"
