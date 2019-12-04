RESTSERVER=localhost

# OpenStack VM 생성 테스트 시 리소스 이름 정보

# ImageId = ubuntu16.04
# VirtualNetworkId(Not Required) : 내부적으로 CB-VNet (가상 네트워크) ID 설정
# SecurityGroupIds = CB-SecGroup
# VMSpecId = m1.small
# KeyPairName = CB-KeyPair

curl -X POST http://$RESTSERVER:1024/vm?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CBVm",
    "ImageId": "ubuntu16.04",
    "SecurityGroupIds": [
        "CB-SecGroup"
    ],
    "VMSpecId": "m1.small",
    "KeyPairName": "CB-KeyPair"
}' |json_pp
