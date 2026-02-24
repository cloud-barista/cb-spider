#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/nginx-install-test-cli/common
source $SETUP_PATH/setup.env $1

echo "============== before get KeyPair: '${KEYPAIR_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" keypair get -c "${CONN_CONFIG}" -n "${KEYPAIR_NAME}" 2> /dev/null
echo "============== after get KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

echo "============== before get SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" securitygroup get -c "${CONN_CONFIG}" -n "${SG_NAME}" 2> /dev/null
echo "============== after get SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

echo "============== before get VPC/Subnet: '${VPC_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" vpc get -c "${CONN_CONFIG}" -n "${VPC_NAME}" 2> /dev/null
echo "============== after get VPC/Subnet: '${VPC_NAME}'"

echo -e "\n\n"

