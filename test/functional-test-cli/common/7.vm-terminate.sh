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


echo "============== before terminate VM: '${VM_NAME}'"
$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" vm terminate -c "${CONN_CONFIG}" -n "${VM_NAME}" 2> /dev/null
echo "============== after terminate VM: '${VM_NAME}'"

echo -e "\n\n"

