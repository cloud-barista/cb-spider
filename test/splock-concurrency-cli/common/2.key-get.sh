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

KEYPAIR_NAME=${KEYPAIR_NAME}-$2

echo "============== before get KeyPair: '${KEYPAIR_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair get --cname "${CONN_CONFIG}" -n "${KEYPAIR_NAME}" 2> /dev/null
echo "============== after get KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

