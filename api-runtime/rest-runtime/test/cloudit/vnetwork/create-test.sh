RESTSERVER=localhost

# CB-Subnet 이름으로 네트워크 생성 및 반환 처리

curl -X POST http://$RESTSERVER:1024/spider/vnetwork?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{"Name":"CB-Subnet"}' |json_pp
