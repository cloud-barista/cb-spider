RESTSERVER=localhost

KEY_NAME=CB-KeyPair
curl -X DELETE http://$RESTSERVER:1024/spider/keypair/$KEY_NAME?connection_name=openstack-config01 |json_pp
