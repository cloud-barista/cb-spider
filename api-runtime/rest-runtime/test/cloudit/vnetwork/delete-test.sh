RESTSERVER=localhost

# [참고]
# Cloudit에서 서브넷 조회 시 서브넷 CIDR를 기준으로 조회

VNETWORK_NAME=CB-Subnet
curl -X DELETE http://$RESTSERVER:1024/spider/vnetwork/$VNETWORK_NAME?connection_name=cloudit-config01
