RESTSERVER=localhost

# [참고]
# Cloudit에서 서브넷 조회 시 서브넷 CIDR를 기준으로 조회

VNETWORK_NAME=CB-Subnet
#curl -X GET http://$RESTSERVER:1024/vnetwork/$VNETWORK_ID?connection_name=cloudit-config01 | json_pp
curl -X GET http://$RESTSERVER:1024/vnetwork/$VNETWORK_NAME?connection_name=cloudit-config01 | json_pp
