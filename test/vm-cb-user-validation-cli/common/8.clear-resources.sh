#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)   # this common/ directory (independent of CBSPIDER_ROOT)
source $SETUP_PATH/setup.env $1

echo "============== before delete KeyPair: '${KEYPAIR_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD"  keypair delete -n "${KEYPAIR_NAME}" -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\"
    }"
echo "============== after delete KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

echo "============== before delete SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD"  securitygroup delete -n "${SG_NAME}" -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\"
    }"
echo "============== after delete SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

echo "============== before delete VPC/Subnet: '${VPC_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD"  vpc delete -n "${VPC_NAME}" -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\"
    }"
echo "============== after delete VPC/Subnet: '${VPC_NAME}'"

echo -e "\n\n"

