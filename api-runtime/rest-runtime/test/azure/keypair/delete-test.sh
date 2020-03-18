RESTSERVER=localhost

KEY_NAME=CB-KeyPair
curl -X DELETE http://$RESTSERVER:1024/spider/keypair/$KEY_NAME?connection_name=azure-config01 |json_pp
