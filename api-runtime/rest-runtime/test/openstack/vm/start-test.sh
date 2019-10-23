RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/vm?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CB-VM",
    "ImageId": "c430f613-2b0d-48c8-9983-cdbc6826cca5",
    "VirtualNetworkId": "c430f613-2b0d-48c8-9983-cdbc6826cca5",
    "SecurityGroupIds": [
        "fbcc9efc-feb7-4f55-a70d-7745c6880c14"
    ],
    "VMSpecId": "2",
    "KeyPairName": "CB-KeyPair",
    "PublicIPId": "1b61f689-dcd9-4037-9da7-dcb9a06c2c5e"
}'
