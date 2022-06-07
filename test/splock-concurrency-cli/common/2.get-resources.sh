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

echo "============== before get KeyPair: '${KEYPAIR_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair get --cname "${CONN_CONFIG}" -n "${KEYPAIR_NAME}" 2> /dev/null
echo "============== after get KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

echo "============== before get SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf security get --cname "${CONN_CONFIG}" -n "${SG_NAME}" 2> /dev/null
echo "============== after get SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

echo "============== before get VPC/Subnet: '${VPC_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf vpc get --cname "${CONN_CONFIG}" -n "${VPC_NAME}" 2> /dev/null
echo "============== after get VPC/Subnet: '${VPC_NAME}'"

echo -e "\n\n"

