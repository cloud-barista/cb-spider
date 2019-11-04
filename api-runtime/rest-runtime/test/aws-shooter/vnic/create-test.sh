#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	ID1=`curl -sX GET http://$RESTSERVER:1024/securitygroup?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
	ID2=`curl -sX GET http://$RESTSERVER:1024/securitygroup?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
	curl -sX POST http://$RESTSERVER:1024/vnic?connection_name=${NAME} -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp &
done
