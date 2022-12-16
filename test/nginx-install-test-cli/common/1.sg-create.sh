#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/nginx-install-test-cli/common
source $SETUP_PATH/setup.env $1

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
            "IPProtocol" : "TCP",
            "FromPort": "22",
            "ToPort" : "22"
          },
          {
            "Direction" : "inbound",
            "IPProtocol" : "TCP",
            "FromPort": "80",
            "ToPort" : "80"
          }
        ]
      }
    }' 2> /dev/null
echo "============== after create SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

