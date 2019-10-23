RESTSERVER=192.168.130.8

curl -X POST http://$RESTSERVER:1024/vm?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CB-VM",
    "ImageId": "b3411d30-b054-4c02-8bd1-67cd78aecd63",
    "VirtualNetworkId": "43dcec05-a3a4-47dc-a342-1f673cb3f39d",
    "SecurityGroupIds": [
      "c2b6e1d6-dbb1-4100-818c-0f75cc89470c"
    ],
    "VMSpecId": 2,
    "KeyPairName": "CB-KeyPair"
}'
