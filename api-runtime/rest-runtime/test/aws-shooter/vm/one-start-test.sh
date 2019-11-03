source ../setup.image

source ../setup.env

VNET_ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-singapore-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
PIP_ID=`curl -X GET http://$RESTSERVER:1024/publicip?connection_name=aws-singapore-config |json_pp |grep "\"Name\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
SG_ID1=` curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-singapore-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
SG_ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-singapore-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
VNIC_ID=`curl -X GET http://$RESTSERVER:1024/vnic?connection_name=aws-singapore-config |json_pp |grep "eni" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`

#echo ${VNET_ID}, ${PIP_ID}, ${SG_ID}, ${VNIC_ID}

RETURN=`curl -X POST http://$RESTSERVER:1024/vm?connection_name=aws-singapore-config -H 'Content-Type: application/json' -d '{
    "VMName": "vm-powerkim01",
        "ImageId": "'${SINGAPORE_IMG_ID1}'",
        "VirtualNetworkId": "'${VNET_ID}'",
        "NetworkInterfaceId": "'${VNIC_ID}'",
        "PublicIPId": "'${PIP_ID}'",
    "SecurityGroupIds": [
        "'${SG_ID1}'",  "'${SG_ID2}'"
    ],
        "VMSpecId": "t2.micro",
        "KeyPairName": "mcb-keypair-powerkim",
        "VMUserId": "",
        "VMUserPasswd": ""
}'`

echo ${RETURN} |json_pp

PUBLIC_IP=`echo ${RETURN} |json_pp |grep PublicIP |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
ssh-keygen -f "/root/.ssh/known_hosts" -R ${PUBLIC_IP}

