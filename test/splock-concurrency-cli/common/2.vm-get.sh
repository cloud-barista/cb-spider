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

VM_NAME=${VM_NAME}-$2-$3

echo "============== before get VM: '${VM_NAME}'"

$CLIPATH/spctl --config $CLIPATH/spctl.conf vm get --cname "${CONN_CONFIG}" -n "${VM_NAME}" 

echo "============== after get VM: '${VM_NAME}'"


echo -e "\n\n"

