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


echo "============== before list VPC/Subnet"

$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" vpc list -c "${CONN_CONFIG}" 2> /dev/null

echo "============== after list VPC/Subnet"


echo -e "\n\n"

