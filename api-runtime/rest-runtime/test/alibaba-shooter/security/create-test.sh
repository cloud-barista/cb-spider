#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
#	NAME=${CONNECT_NAMES[0]}
        curl -sX POST http://$RESTSERVER:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'", "ReqInfo": { "Name": "security01-powerkim", "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"} ] } }' |json_pp &

done
