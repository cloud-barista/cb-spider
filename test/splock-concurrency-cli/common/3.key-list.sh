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


echo "============== before list KeyPair"

$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair list --cname "${CONN_CONFIG}" 2> /dev/null

echo "============== after list KeyPair"


echo -e "\n\n"

