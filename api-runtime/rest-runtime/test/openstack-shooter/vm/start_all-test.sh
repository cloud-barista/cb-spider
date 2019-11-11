#!/bin/bash
source ../setup.env

num=0
for NAME in "${CONNECT_NAMES[@]}"
do
        echo ========================== $NAME
        VNET_ID=`curl -sX GET http://$RESTSERVER:1024/vnetwork?connection_name=${NAME} |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
        PIP_ID=`curl -sX GET http://$RESTSERVER:1024/publicip?connection_name=${NAME} |json_pp |grep "\"PublicIP\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
        SG_ID1=` curl -sX GET http://$RESTSERVER:1024/securitygroup?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
        SG_ID2=`curl -sX GET http://$RESTSERVER:1024/securitygroup?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`

        echo ${VNET_ID}, ${PIP_ID}, ${SG_ID1}, ${SG_ID2}

        curl -sX POST http://$RESTSERVER:1024/vm?connection_name=${NAME} -H 'Content-Type: application/json' -d '{
            "VMName": "vm-powerkim01",
                "ImageId": "'${IMG_IDS[num]}'",
                "VirtualNetworkId": "'${VNET_ID}'",
                "PublicIPId": "'${PIP_ID}'",
                "SecurityGroupIds": [
                "'${SG_ID1}'",  "'${SG_ID2}'"
		    ], 
		"VMSpecId": "ba7c426b-29b4-4e6f-833c-8c36b8566c37",
                 "KeyPairName": "mcb-keypair-powerkim",
                "VMUserId": "cb-user"
        }' |json_pp &


        num=`expr $num + 1`
done

