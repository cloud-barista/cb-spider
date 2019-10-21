RESTSERVER=node12

curl -X POST http://$RESTSERVER:1024/vm?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "vm01", 
        "ImageId": "image01",
        "VirtualNetworkId": "cb-vnet-subnet01",
        "NetworkInterfaceId": "",
        "PublicIPId": "", 
    "SecurityGroupIds": [
        "security_01"
    ],

        "VMSpecId": "f1.micro",

        "KeyPairName": "keyname01",
        "VMUserId": "cb-user",
        "VMUserPasswd": "cb-user"
}'
