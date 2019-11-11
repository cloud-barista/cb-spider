RESTSERVER=localhost

# Cloudit VM 자원 생성 순서
# VNet -> Sec -> VM(VNic 자동 생성, VM 생성 후 PublicIP 자동 생성)
# VM 삭제 시 VNic 자동 삭제, PublicIP 자동 삭제

# Cloudit VM 생성 테스트 시 리소스 이름 정보

# ImageId = CentOS 7.6
# VirtualNetworkId(서브넷 ID와 매핑) = CB-Subnet
# SecurityGroupIds = CB-SecGroup
# VMSpecId = micro-1

# Cloudit의 경우 KeyPair 미지원 (root 계정 패스워드 로그인 방식 지원)
# VMUserPasswd = "etriETRI!@"

curl -X POST http://$RESTSERVER:1024/vm?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "CBVm",
    "ImageId": "2813a6ed-66f8-4041-9cfa-2cb5cb416520",
    "VirtualNetworkId": "10.0.4.0",
    "SecurityGroupIds": [
        "064616d0-74fe-4840-8a42-8af4ce24e96a"
    ],
    "VMSpecId": "1c38e438-ede9-4df5-8775-2ce791698924",
    "VMUserPasswd": "etriETRI!@"
}' |json_pp
