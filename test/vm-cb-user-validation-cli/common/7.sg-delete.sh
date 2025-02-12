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


echo "============== before delete SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl  securitygroup delete -n "${SG_NAME}" -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\"
    }"
echo "============== after delete SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

