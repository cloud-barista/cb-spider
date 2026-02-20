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

VPC_NAME=${VPC_NAME}-$2
SUBNET_NAME=${SUBNET_NAME}-$2-$3
# SUBNET_CIDR=192.168.0.0/24
SUBNET_CIDR=`echo $SUBNET_CIDR | sed 's/0\./'$3'\./g'`

echo "============== before remove Subnet: '${VPC_NAME}' : '${SUBNET_NAME}'"

$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" subnet remove --VPCName ${VPC_NAME} --SubnetName ${SUBNET_NAME} -d '{"ConnectionName":"'"${CONN_CONFIG}"'"}'

echo "============== after remove Subnet: '${VPC_NAME}' : '${SUBNET_NAME}'"

echo -e "\n\n"

