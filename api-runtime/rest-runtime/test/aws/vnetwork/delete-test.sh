RESTSERVER=localhost

curl -X DELETE http://$RESTSERVER:1024/vnetwork/cb-vnet-subnet01?connection_name=aws-config01 |json_pp
