RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/vm?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "vm01", 
        "ImageId": "ami-047f7b46bd6dd5d84",
        "VirtualNetworkId": "subnet-08ec382e9cf881ab6",
        "NetworkInterfaceId": "",
        "PublicIPId": "52.79.99.38", 
    "SecurityGroupIds": [
        "sg-07ae6fb4cddad34e4"
    ],
        "VMSpecId": "t2.micro",
        "KeyPairName": "CB-KeyPairTest",
        "VMUserId": "",
        "VMUserPasswd": ""
}' |json_pp
