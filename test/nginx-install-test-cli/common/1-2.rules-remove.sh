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


echo -e "\n\n"

echo "============== before RemoveRules: '${SG_NAME}' --- inbound:UDP/1/65535"
#### @todo Change this command with spctl
curl -sX DELETE http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "inbound",
                                        "IPProtocol": "UDP",
                                        "FromPort": "1",
                                        "ToPort": "65535"
                                }
                        ]
                }
        }' |json_pp

echo "============== after RemoveRules: '${SG_NAME}' --- inbound:UDP/1/65535"

echo -e "\n\n"

echo "============== before RemoveRules: '${SG_NAME}' --- inbound:ICMP/-1/-1"
#### @todo Change this command with spctl
curl -sX DELETE http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "inbound",
                                        "IPProtocol": "ICMP",
                                        "FromPort": "-1",
                                        "ToPort": "-1"
                                }
                        ]
                }
        }' |json_pp

echo "============== after RemoveRules: '${SG_NAME}' --- inbound:ICMP/-1/-1"


echo -e "\n\n"

