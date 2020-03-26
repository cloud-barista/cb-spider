#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
        echo ========================== $NAME

	VM_ID=vm-powerkim01
	echo ....terminate ${VM_ID} ...
	curl -sX DELETE http://$RESTSERVER:1024/spider/vm/${VM_ID} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'" }' &
done

