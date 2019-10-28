RESTSERVER=localhost

#정상 동작

# [참고] VNetName / PublicIPid 값은 내부적으로 이용하지 않음.
# - SubnetId가 필요한데 자동으로 1개만 생성되기 때문에 전달받는 VNetName 정보는 사용하지 않고 자동으로 생성된 값을 내부에서 검색해서 사용함.
# - vNic에 PublicIP를 할당하지 않고 VM 생성 시 전달 받아서 처리함.
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{ "Name": "vnic01", "VNetName": "vnet-subnet01", "SecurityGroupIds": [ "sg-0c7d44489619771b5", "sg-05178dba7d1ef476f" ], "PublicIPid": "publicIP01" }' |json_pp
