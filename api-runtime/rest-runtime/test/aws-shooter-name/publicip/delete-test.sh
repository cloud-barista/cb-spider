#!/bin/bash
source ../setup.env

num=0
for NAME in "${CONNECT_NAMES[@]}"
do
	ID=publicipt${num}-powerkim
        curl -sX DELETE http://$RESTSERVER:1024/spider/publicip/${ID} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'" }' &

	num=`expr $num + 1`
done
