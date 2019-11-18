RESTSERVER=localhost

# OpenStack VM 생성 테스트 시 리소스 이름 정보

# ImageId = ubuntu16.04
# VirtualNetworkId(서브넷 ID와 매핑) = CB-Subnet
# SecurityGroupIds = CB-SecGroup
# VMSpecId = m1.small
# KeyPairName(Name이 ID 처럼 사용) =  CB-KeyPair
# PublicIPId(IP가 ID 처럼 사용) = 182.252.135.78

curl -X POST http://$RESTSERVER:1024/vm?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CBVm",
    "ImageId": "c14a9728-eb03-4813-9e1a-8f57fe62b4fb",
    "VirtualNetworkId": "aaf3f27d-cb31-404b-b8ed-3ad83833f00c",
    "SecurityGroupIds": [
        "40ed4afc-2f40-4e94-88f5-f017f54ad64f"
    ],
    "VMSpecId": "ba7c426b-29b4-4e6f-833c-8c36b8566c37",
    "KeyPairName": "CB-KeyPair"
}'
