#!/bin/bash

if [ "$1" = "" ]; then
	echo 
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo 
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/sg-rules-validation-cli/common
source $SETUP_PATH/setup.env $1

VM_NAME=${VM_NAME}-1
echo "============== before get status VM: '${VM_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf --cname "${CONN_CONFIG}" vm getstatus -n "${VM_NAME}" 2> /dev/null
echo "============== after get status VM: '${VM_NAME}'"

echo -e "\n\n"

