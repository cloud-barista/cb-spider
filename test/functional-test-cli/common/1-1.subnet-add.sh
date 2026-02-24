#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/functional-test-cli/common
source $SETUP_PATH/setup.env $1

SUBNET_NAME=${SUBNET_NAME}-Added
# SUBNET_CIDR=192.168.0.0/24
SUBNET_CIDR=`echo $SUBNET_CIDR | sed 's/0\./1\./g'`

# for OpenStack
if [ $SUBNET_CIDR_ADD ]; then
	SUBNET_CIDR=$SUBNET_CIDR_ADD
fi

echo "============== before add Subnet: '${VPC_NAME}' : '${SUBNET_NAME}'"

$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" subnet add --VPCName "${VPC_NAME}" -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${SUBNET_NAME}'",
        "IPv4_CIDR": "'${SUBNET_CIDR}'"
      }
    }'

echo "============== after add Subnet: '${VPC_NAME}' : '${SUBNET_NAME}'"

echo -e "\n\n"

