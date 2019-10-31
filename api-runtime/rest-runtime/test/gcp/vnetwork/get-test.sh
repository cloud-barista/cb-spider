RESTSERVER=localhost

#정상 동작


#[참고]
#VPC 생성때 만든 이름을 사용함 name
curl -X GET http://$RESTSERVER:1024/vnetwork/cb-vnet?connection_name=gcp-config01 |json_pp
