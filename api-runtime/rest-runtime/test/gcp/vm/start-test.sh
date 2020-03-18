RESTSERVER=localhost

#정상 동작

#[참고]
# - NetworkInterfaceId는 현재 전달 받아도 내부에서 처리하지 않음. (지금은 GCP API에서 자동으로 생성되는 vNic에 전달 받은 PublicIP를 할당 함.)
curl -X POST http://$RESTSERVER:1024/spider/vm?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{
    "VMName": "vm01", 
    "ImageId": "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024",
    "VirtualNetworkId": "cb-vnet",
    "NetworkInterfaceId": "",
    "PublicIPId": "gcppublicip1", 
    "SecurityGroupIds": [ "security01" ],
    "KeyPairName": "mcb-keypair",
    "VMSpecId": "f1-micro"
}' |json_pp
