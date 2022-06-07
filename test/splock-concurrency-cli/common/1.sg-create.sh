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

VPC_NAME=${VPC_NAME}-$2
SG_NAME=${SG_NAME}-$2

echo "============== before create SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl --config $CLIPATH/spctl.conf security create -i json -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${SG_NAME}'",
        "VPCName": "'${VPC_NAME}'",
        "SecurityRules": [
          {
            "Direction" : "inbound",
            "IPProtocol" : "all",
            "FromPort": "-1",
            "ToPort" : "-1"
          }
        ]
      }
    }' 2> /dev/null
echo "============== after create SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

