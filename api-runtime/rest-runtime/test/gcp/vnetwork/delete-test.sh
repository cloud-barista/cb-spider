RESTSERVER=localhost

#[참고]
# 삭제되는 시점에 따라서 일부 리소스의 의존성 때문에 삭제가 안되는 경우가 있음.
curl -X DELETE http://$RESTSERVER:1024/vnetwork/cb-vnet?connection_name=gcp-config01 |json_pp
