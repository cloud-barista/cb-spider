RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/vm?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CBVm",
    "ImageId": "Canonical:UbuntuServer:18.04-LTS:18.04.201804262",
    "VirtualNetworkId": "CB-Subnet",
    "PublicIPId": "CB-PublicIP",
    "SecurityGroupIds": ["CB-SecGroup"],
    "VMSpecId": "Standard_B1ls",
    "VMUserId": "cb-user",
    "KeyPairName" : "CB-Keypair"
}' |json_pp
