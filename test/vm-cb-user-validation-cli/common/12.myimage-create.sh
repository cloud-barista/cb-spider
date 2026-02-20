#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/vm-cb-user-validation-cli/common
source $SETUP_PATH/setup.env $1

VM_NAME=${VM_NAME}-1

$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" vm snapshot -d \
    "{
        \"ConnectionName\":\"${CONN_CONFIG}\",
        \"ReqInfo\": {
            \"Name\": \"${MYIMAGE_NAME}\",
            \"SourceVM\": \"${VM_NAME}\"
        }
    }"

echo -e "\n\n"
