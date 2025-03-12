#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/splock-concurrency-cli/common
source $SETUP_PATH/setup.env $1

SG_NAME=${SG_NAME}-$2


echo "============== before AddRules: '${SG_NAME}' --- inbound:TCP/22/22"
#### @todo Change this command with spctl
curl -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "inbound",
                                        "IPProtocol": "TCP",
                                        "FromPort": "22",
                                        "ToPort": "22"
                                }
                        ]
                }
        }' |json_pp

echo "============== after AddRules: '${SG_NAME}' --- inbound:TCP/22/22"

echo -e "\n\n"

echo "============== before AddRules: '${SG_NAME}' --- inbound:UDP/1/65535"
#### @todo Change this command with spctl
curl -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
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

echo "============== after AddRules: '${SG_NAME}' --- inbound:UDP/1/65535"

echo -e "\n\n"

echo "============== before AddRules: '${SG_NAME}' --- inbound:ICMP/-1/-1"
#### @todo Change this command with spctl
curl -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
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

echo "============== after AddRules: '${SG_NAME}' --- inbound:ICMP/-1/-1"


echo -e "\n\n"

