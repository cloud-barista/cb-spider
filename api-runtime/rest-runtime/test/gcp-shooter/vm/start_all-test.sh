#!/bin/bash
source ../setup.env

num=0
for NAME in "${CONNECT_NAMES[@]}"
do
        echo ========================== $NAME
        VNET_ID=cb-vnet
        PIP_ID=publicipt01-powerkim
        SG_ID1=security01-powerkim
        #echo ${VNET_ID}, ${PIP_ID}, ${SG_ID}, ${VNIC_ID}

        curl -sX POST http://$RESTSERVER:1024/vm?connection_name=${NAME} -H 'Content-Type: application/json' -d '{
            "VMName": "vm-powerkim01",
                "ImageId": "'${IMG_IDS[num]}'",
                "VirtualNetworkId": "'${VNET_ID}'",
		"NetworkInterfaceId": "",
                "PublicIPId": "'${PIP_ID}'",
            "SecurityGroupIds": [ "'${SG_ID1}'" ],
                "VMSpecId": "f1-micro",
                 "KeyPairName": "mcb-keypair-powerkim",
                "VMUserId": "cb-user"
        }' |json_pp &


        num=`expr $num + 1`
done

