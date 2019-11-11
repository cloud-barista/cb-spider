#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	echo ========================== $NAME
	VM_ID=02c71331-558e-4ce7-b61a-cb8f8faaf270

	curl -sX GET http://$RESTSERVER:1024/vm/$VM_ID?connection_name=$NAME |json_pp
done
