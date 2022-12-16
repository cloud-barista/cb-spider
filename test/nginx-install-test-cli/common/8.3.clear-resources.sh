#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/nginx-install-test-cli/common
source $SETUP_PATH/setup.env $1


echo "============== before delete VPC/Subnet: '${VPC_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf vpc delete --cname "${CONN_CONFIG}" -n "${VPC_NAME}" 2> /dev/null
echo "============== after delete VPC/Subnet: '${VPC_NAME}'"

echo -e "\n\n"

