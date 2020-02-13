RESTSERVER=localhost

#정상 동작

#[참고]
# - ~~Id는 AWS 기준의 Id가 아닌 각 객체를 생성할 때 사용한 Name임.
# - VirtualNetworkId는 현재 CB-VNet-Subnet 고정임.
# - NetworkInterfaceId는 전달할 필요 없으며, 전달할 경우 eth1에 할당됨.
curl -X POST http://$RESTSERVER:1024/vm?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "cbvm01", 
        "ImageId": "ami-047f7b46bd6dd5d84",
        "VirtualNetworkId": "CB-VNet-Subnet",
        "NetworkInterfaceId": "",
        "PublicIPId": "cbpublicip01", 
    "SecurityGroupIds": [
        "cbsg01-in"
    ],
        "VMSpecId": "t2.micro",
        "KeyPairName": "cbkeypair01",
        "VMUserId": "",
        "VMUserPasswd": ""
}' |json_pp
