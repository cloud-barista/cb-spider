#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
        curl -sX POST http://$RESTSERVER:1024/spider/vnetwork -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'", "ReqInfo": {"Name":"cb-vnet"}}' |json_pp &
done

