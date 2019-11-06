#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	ID=vnic01-powerkim
	curl -sX DELETE http://$RESTSERVER:1024/vnic/${ID}?connection_name=${NAME} |json_pp 
done
