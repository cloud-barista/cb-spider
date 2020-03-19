#!/bin/bash
source ../setup.env

num=0
for NAME in "${CONNECT_NAMES[@]}"
do
        #ID=`curl -sX GET http://$RESTSERVER:1024/spider/publicip?connection_name=${NAME} |json_pp |grep "\"Name\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
	ID=publicipt${num}-powerkim
        curl -sX DELETE http://$RESTSERVER:1024/spider/publicip/${ID}?connection_name=${NAME} &

	num=`expr $num + 1`
done
