#!/bin/bash
source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
#	ID1=`curl -sX GET http://$RESTSERVER:1024/securitygroup?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
	ID1=`curl -sX GET http://$RESTSERVER:1024/securitygroup/security01-powerkim?connection_name=azure-northeu-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
	PUBLICIP_ID='/subscriptions/f1548292-2be3-4acd-84a4-6df079160846/resourceGroups/CB-GROUP-POWERKIM/providers/Microsoft.Network/publicIPAddresses/publicipt01-powerkim'
	curl -sX POST http://$RESTSERVER:1024/vnic?connection_name=${NAME} -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'" ], "PublicIPid": "'${PUBLICIP_ID}'" }' |json_pp &
done
