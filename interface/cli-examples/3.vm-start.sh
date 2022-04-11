#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

source ./setup.env $1

VM_NAME=${VM_NAME}-1
echo "============== before start VM: '${VM_NAME}'"

#### This script doesn't work because CloudIt has an Image Namme with Space.
#time $CLIPATH/spctl --config $CLIPATH/spctl.conf vm start -i json -d \
#    '{
#      "ConnectionName":"'${CONN_CONFIG}'",
#      "ReqInfo": {
#        "Name": "'${VM_NAME}'",
#        "ImageName": "'${IMAGE_NAME}'",
#        "VPCName": "'${VPC_NAME}'",
#        "SubnetName": "'${SUBNET_NAME}'",
#        "SecurityGroupNames": [ "'${SG_NAME}'" ],
#        "VMSpecName": "'${SPEC_NAME}'",
#        "KeyPairName": "'${KEYPAIR_NAME}'"
#      }
#    }' 2> /dev/null

time $CLIPATH/spctl --config $CLIPATH/spctl.conf vm start -i json -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\",
      \"ReqInfo\": {
        \"Name\": \"${VM_NAME}\",
        \"ImageName\": \"${IMAGE_NAME}\",
        \"VPCName\": \"${VPC_NAME}\",
        \"SubnetName\": \"${SUBNET_NAME}\",
        \"SecurityGroupNames\": [ \"${SG_NAME}\" ],
        \"VMSpecName\": \"${SPEC_NAME}\",
        \"KeyPairName\": \"${KEYPAIR_NAME}\"
      }
    }" 2> /dev/null

echo "============== after start VM: '${VM_NAME}'"

echo -e "\n\n"

