RESTSERVER=node12

curl -X POST http://$RESTSERVER:1024/vnic?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "vnic01", "VNetName": "vnet-subnet01", "SecurityGroupIds": [ "security_01", "security_02" ], "PublicIPid": "publicIP01" }'
