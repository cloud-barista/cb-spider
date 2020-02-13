RESTSERVER=localhost

# 기본 네트워크인 CB-VNet 하위에 서브넷 생성
# 서브넷 생성 시 자동으로 인터넷 게이트웨이 라우터 및 라우터 인터페이스 생성
# CB-Subnet 이름으로 네트워크 생성 및 반환 처리

curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{"Name":"CB-Subnet"}' |json_pp
