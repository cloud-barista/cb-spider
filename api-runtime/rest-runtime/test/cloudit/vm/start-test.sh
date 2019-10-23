RESTSERVER=192.168.130.8

curl -X POST http://$RESTSERVER:1024/vm?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "mcb-vm",
    "ImageId": "mcb-test-img",
    "VirtualNetworkId": "mcb-test-vnet",
    "NetworkInterfaceId": "025e5edc-54ad-4b98-9292-6eeca4c36a6d",
    "PublicIPId": "mcb-test-pubip",
    "SecurityGroupIds": [
        "mcb-test-sg"
    ],

    "VMSpecId": "1c38e438-ede9-4df5-8775-2ce791698924",

    "VMUserId": "cloudit",
    "VMUserPasswd": "cloudit"
}'
