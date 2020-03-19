#!/bin/bash
source ../setup.env


num=0
for NAME in "${CONNECT_NAMES[@]}"
do
        curl -sX POST http://$RESTSERVER:1024/spider/publicip -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'", "ReqInfo": { "Name": "publicipt'${num}'-powerkim" }}' |json_pp &

	num=`expr $num + 1`
done

