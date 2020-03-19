#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
        ID=cb-vnet
        curl -sX DELETE http://$RESTSERVER:1024/spider/vnetwork/${ID}?connection_name=${NAME} |json_pp &
done

