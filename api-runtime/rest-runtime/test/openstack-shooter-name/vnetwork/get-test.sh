#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
        NAMEID=cb-vnet
        curl -sX GET http://$RESTSERVER:1024/vnetwork/${NAMEID}?connection_name=${NAME} |json_pp &
done

