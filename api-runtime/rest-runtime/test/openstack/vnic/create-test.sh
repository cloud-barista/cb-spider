RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/vnic?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
    "Name": "CB-VNic",
    "VNetId": "a425b31e-e68e-4043-b733-e34e48e6b8ad",
    "SecurityGroupIds": [ "fbcc9efc-feb7-4f55-a70d-7745c6880c14" ]
}'
