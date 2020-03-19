RESTSERVER=localhost

# [동작]
# 내부에서 VPC를 자동으로 생성하고 Subnet도 자동으로 생성함.

# [참고]
# 반드시 소문자로 기입

#정상 동작
curl -X POST http://$RESTSERVER:1024/spider/vnetwork?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{"Name":"cb-vnet"}' |json_pp
