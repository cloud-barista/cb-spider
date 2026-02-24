#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "#############################################"
echo -e "# TEST: $0 "
echo -e "#############################################"

source ../common/setup.env $1
source setup.env $1

echo "============== before AddRules: '${SG_NAME}' --- outbound:TCP/22/22"
#### @todo Change this command with spctl
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "outbound",
                                        "IPProtocol": "TCP",
                                        "FromPort": "22",
                                        "ToPort": "22"
                                }
                        ]
                }
        }' |json_pp

echo "============== after AddRules: '${SG_NAME}' --- outbound:TCP/22/22"

if [ "$SLEEP" ]; then
        sleep $SLEEP
else
        sleep 1
fi

echo -e "\n\n"

echo "============== before AddRules: '${SG_NAME}' --- outbound:UDP/1/65535"
#### @todo Change this command with spctl
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "outbound",
                                        "IPProtocol": "UDP",
                                        "FromPort": "1",
                                        "ToPort": "65535"
                                }
                        ]
                }
        }' |json_pp

echo "============== after AddRules: '${SG_NAME}' --- outbound:UDP/1/65535"

if [ "$SLEEP" ]; then
        sleep $SLEEP
else
        sleep 1
fi

echo "============== before AddRules: '${SG_NAME}' --- outbound:ICMP/-1/-1"
#### @todo Change this command with spctl
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "outbound",
                                        "IPProtocol": "ICMP",
                                        "FromPort": "-1",
                                        "ToPort": "-1"
                                }
                        ]
                }
        }' |json_pp

echo "============== after AddRules: '${SG_NAME}' --- outbound:TCP/-1/-1"

if [ "$SLEEP" ]; then
        sleep $SLEEP
else
        sleep 1
fi
