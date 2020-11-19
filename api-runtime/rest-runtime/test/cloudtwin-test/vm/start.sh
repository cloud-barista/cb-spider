#!/bin/bash
source ../setup.env

num=0
for NAME in "${CLOUD_NAMES[@]}"
do
CONNECT_NAME=cloudtwin-$NAME-config

        echo ========================== $CONNECT_NAME

	echo curl -sX POST http://$RESTSERVER:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONNECT_NAME}'", "VMName": "vm-powerkim01" }' 

	curl -sX POST http://$RESTSERVER:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONNECT_NAME}'", "VMName": "vm-powerkim01" }' &

	sleep 2
        num=`expr $num + 1`
done

