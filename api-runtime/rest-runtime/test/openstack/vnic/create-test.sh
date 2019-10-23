RESTSERVER=192.168.130.8

curl -X POST http://$RESTSERVER:1024/vnic?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
"Name": "CB-VNic",
"VNetId": "fe284dbf-e9f4-4add-a03f-9249cc30a2ac",
"SecurityGroupIds": [ "34585b5e-5ea8-49b5-b38b-0d395689c994", "6d4085c1-e915-487d-9e83-7a5b64f27237" ],
}'
