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


echo "============== before list SecurityGroup"

$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" securitygroup list -c "${CONN_CONFIG}" 2> /dev/null

echo "============== after list SecurityGroup"


echo -e "\n\n"

