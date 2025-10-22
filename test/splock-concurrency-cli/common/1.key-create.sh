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

KEYPAIR_NAME=${KEYPAIR_NAME}-$2

echo "============== before create KeyPair: '${KEYPAIR_NAME}'"

$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair create -i json -o json -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${KEYPAIR_NAME}'"
      }
    }' 2> /dev/null

echo "============== after create KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

