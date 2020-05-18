#!/bin/bash
source ../setup.env


num=0
for NAME in "${CONNECT_NAMES[@]}"
do
        echo ========================== $NAME
	curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' \
		-d '{ "ConnectionName": "'${CONN_CONFIG}'", \
		"ReqInfo": { "Name": "vm-01", "ImageName": "'${IMAGE_NAME}'", \
		"VPCName": "vpc-01", "SubnetName": "subnet-01", 
		"SecurityGroupNames": [ "sg-01" ], "VMSpecName": "'${SPEC_NAME}'", 
		"KeyPairName": "keypair-01"} }' |json_pp

        num=`expr $num + 1`
done

