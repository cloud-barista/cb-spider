RESTSERVER=localhost

#정상 동작

# [참고] VNetName / PublicIPid 값은 내부적으로 이용하지 않음.
# - SubnetId가 필요한데 자동으로 1개만 생성되기 때문에 전달받는 VNetName 정보는 사용하지 않고 자동으로 생성된 값을 내부에서 검색해서 사용함.
# - PublicIp의 경우 vNic에 PublicIP를 할당하지 않고 VM 생성 시 전달 받은 정보를 이용해서 처리하기 때문에 처리 로직 제거 함.

#curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbvnic01", "VNetName": "CB-VNet-Subnet", "SecurityGroupIds": [ "cbsg01-in", "cbsg01-out" ], "PublicIPid": "cbpublicip01" }' |json_pp
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbvnic01", "VNetName": "CB-VNet-Subnet", "SecurityGroupIds": [ "cbsg01-in" ], "PublicIPid": "cbpublicip01" }' |json_pp
