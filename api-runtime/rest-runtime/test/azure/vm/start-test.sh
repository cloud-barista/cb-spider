RESTSERVER=localhost

# CBVm 이름으로 VM 생성 및 반환 처리
# VM 생성 후 VMUserId에 설정한 "cb-user" 사용자 계정으로 CB-Keypair를 사용해서 SSH 접속 가능

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
