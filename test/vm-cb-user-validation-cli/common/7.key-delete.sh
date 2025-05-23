#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/vm-cb-user-validation-cli/common
source $SETUP_PATH/setup.env $1

echo "============== before delete KeyPair: '${KEYPAIR_NAME}'"
$CLIPATH/spctl  keypair delete -n "${KEYPAIR_NAME}" -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\"
    }"
echo "============== after delete KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"

