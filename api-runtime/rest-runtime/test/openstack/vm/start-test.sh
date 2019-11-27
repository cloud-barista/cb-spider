RESTSERVER=localhost

# OpenStack VM 생성 테스트 시 리소스 이름 정보

# ImageId = ubuntu16.04
# VirtualNetworkId(서브넷 ID와 매핑) = CB-Subnet
# SecurityGroupIds = CB-SecGroup
# VMSpecId = m1.small
# KeyPairName(Name이 ID 처럼 사용) =  CB-KeyPair

curl -X POST http://$RESTSERVER:1024/vm?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CBVm2",
    "ImageId": "ubuntu16.04",
    "SecurityGroupIds": [
        "CB-SecGroup"
    ],
    "VMSpecId": "m1.small",
    "KeyPairName": "CB-KeyPair"
}'
