RESTSERVER=localhost

#정상 동작

#[참고]
# - NetworkInterfaceId는 현재 전달 받아도 내부에서 처리하지 않음. (지금은 GCP API에서 자동으로 생성되는 vNic에 전달 받은 PublicIP를 할당 함.)
# - PublicIPId : PublicIP 생성 시 사용한 Name 필드 값이 아닌 생성 후 전달 받은 Name(AllocateID) 필드의 값을 입력해야 함.
curl -X POST http://$RESTSERVER:1024/vm?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "vm01", 
    "VirtualNetworkId": "cb-vnet",
    "NetworkInterfaceId": "",
    "PublicIPId": "publicip-vm01", 
    "VMSpecId": "f1-micro"
}' |json_pp
