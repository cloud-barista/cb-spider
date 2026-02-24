#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/splock-concurrency-cli/common
source $SETUP_PATH/setup.env $1

VM_NAME=${VM_NAME}-$2

echo "============== before start VM: '${VM_NAME}'"

$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" vm start -d \
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

