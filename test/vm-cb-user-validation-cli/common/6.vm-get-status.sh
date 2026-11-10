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

VM_NAME=${VM_NAME}-$2
echo "============== before get status VM: '${VM_NAME}'"
$CLIPATH/spctl  --cname "${CONN_CONFIG}" vm getstatus -n "${VM_NAME}" 2> /dev/null
echo "============== after get status VM: '${VM_NAME}'"

echo -e "\n\n"

