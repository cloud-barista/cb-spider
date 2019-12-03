RESTSERVER=localhost

#정상 동작
#Name으로 조회해야 함.
curl -X GET http://$RESTSERVER:1024/vnetwork/CB-VNet-Subnet?connection_name=aws-config01 |json_pp
