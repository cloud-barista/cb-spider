#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/splock-concurrency-cli/common
source $SETUP_PATH/setup.env $1

echo "============== before delete KeyPair: '${KEYPAIR_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair delete --cname "${CONN_CONFIG}" -n "${KEYPAIR_NAME}" 2> /dev/null
echo "============== after delete KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

echo "============== before delete SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf security delete --cname "${CONN_CONFIG}" -n "${SG_NAME}" 2> /dev/null
echo "============== after delete SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

echo "============== before delete VPC/Subnet: '${VPC_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf vpc delete --cname "${CONN_CONFIG}" -n "${VPC_NAME}" 2> /dev/null
echo "============== after delete VPC/Subnet: '${VPC_NAME}'"

echo -e "\n\n"

