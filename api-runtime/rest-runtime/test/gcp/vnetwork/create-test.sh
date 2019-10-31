RESTSERVER=localhost

#정상 동작

# [동작]
# 내부에서 VPC를 자동으로 생서하고 Subnet도 자동으로 생성함.

# [참고]
# v net 만들때  default subnet 자동 생성
# 반듯이 소문자
curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{"Name":"cb-vnet"}' |json_pp
