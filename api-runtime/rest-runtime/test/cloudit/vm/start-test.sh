RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/vm?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CBVM",
    "ImageId": "914007c0-02ea-4794-9250-02b08d6e2db7",
    "VirtualNetworkId": "10.0.0.0",
    "SecurityGroupIds": [
        "b2be62e7-fd29-43ff-b008-08ae736e092a"
    ],
    "VMSpecId": "1c38e438-ede9-4df5-8775-2ce791698924",
    "VMUserPasswd": "qwe1212!Q"
}'
