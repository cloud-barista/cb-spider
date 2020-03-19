#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	echo ========================== $NAME
	curl -sX GET http://$RESTSERVER:1024/spider/vmstatus/vm-powerkim01?connection_name=$NAME |json_pp &
done
