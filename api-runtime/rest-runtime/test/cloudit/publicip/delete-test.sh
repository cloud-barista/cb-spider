RESTSERVER=localhost

# [참고]
# Cloudit에서 PublicIP 조회 시 IP 주소를 기준으로 조회

PUBLICIP_ADDR=182.252.135.44
curl -X DELETE http://$RESTSERVER:1024/publicip/$PUBLICIP_ADDR?connection_name=cloudit-config01
