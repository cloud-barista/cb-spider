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

VM_NAME=${VM_NAME}-1

$CLIPATH/spctl nlb create -d \
    "{
        \"ConnectionName\":\"${CONN_CONFIG}\",
        \"ReqInfo\": {
            \"HealthChecker\": {
                \"Port\": \"22\",
                \"Protocol\": \"TCP\"
            },
            \"Listener\": {
                \"Port\": \"22\",
                \"Protocol\": \"TCP\"
            },
            \"Name\": \"${NLB_NAME}\",
            \"Scope\": \"REGION\",
            \"Type\": \"PUBLIC\",
            \"VMGroup\": {
                \"Port\": \"22\",
                \"Protocol\": \"TCP\",
                \"VMs\": [
                    \"${VM_NAME}\"
                ]
            },
            \"VPCName\": \"${VPC_NAME}\"
        }
    }"

echo -e "\n\n"
