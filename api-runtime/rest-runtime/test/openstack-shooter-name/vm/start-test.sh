#!/bin/bash
source ../setup.env

NAME=${CONNECT_NAMES[0]}

IMG_ID=`curl -sX GET http://$RESTSERVER:1024/vmimage/${IMG_IDS[0]}?connection_name=${NAME} |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
VNET_ID=`curl -sX GET http://$RESTSERVER:1024/vnetwork/cb-vnet?connection_name=${NAME} |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
SG_ID1=`curl -sX GET http://$RESTSERVER:1024/securitygroup/security01-powerkim?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`

echo ${IMG_ID} ${VNET_ID}, ${SG_ID1}

curl -sX POST http://$RESTSERVER:1024/vm?connection_name=${NAME} -H 'Content-Type: application/json' -d '{
    "VMName": "vm-powerkim01",
	"ImageId": "'${IMG_ID}'",
	"VirtualNetworkId": "'${VNET_ID}'",
	"SecurityGroupIds": [ "'${SG_ID1}'" ],
	"VMSpecId": "ba7c426b-29b4-4e6f-833c-8c36b8566c37",
	 "KeyPairName": "mcb-keypair-powerkim",
	"VMUserId": "root"
}' | json_pp
