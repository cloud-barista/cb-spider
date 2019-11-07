RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/vnic?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{
    "Name": "CB-VNic",
    "VNetName": "CB-Subnet",
    "SecurityGroupIds": ["CB-SecGroup"],
    "PublicIPid": "CB-PublicIP"
}' |json_pp
