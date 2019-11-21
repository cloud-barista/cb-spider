#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	ID=publicipt01-powerkim
	curl -sX DELETE http://$RESTSERVER:1024/publicip/${ID}?connection_name=${NAME} &
done
