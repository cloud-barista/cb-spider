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


echo "============== before list VM Status"

$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" vm-status list -c "${CONN_CONFIG}" 

echo "============== after list VM Status"

echo -e "\n\n"

