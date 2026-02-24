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


echo "============== before suspend VM: '${VM_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" vm control -n "${VM_NAME}" --action suspend -d '{"ConnectionName":"'"${CONN_CONFIG}"'"}' 2> /dev/null
echo "============== after suspend VM: '${VM_NAME}'"

echo -e "\n\n"

