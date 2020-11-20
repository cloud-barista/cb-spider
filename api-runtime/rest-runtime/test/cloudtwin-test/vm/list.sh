#!/bin/bash
source ../setup.env

num=0
for NAME in "${CLOUD_NAMES[@]}"
do
CONNECT_NAME=cloudtwin-$NAME-config

        echo ========================== $CONNECT_NAME

	curl -sX GET http://$RESTSERVER:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONNECT_NAME}'"}' |json_pp

	sleep 1
        num=`expr $num + 1`
done

