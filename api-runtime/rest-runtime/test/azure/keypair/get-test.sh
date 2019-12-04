RESTSERVER=localhost

KEY_NAME=CB-KeyPair
curl -X GET http://$RESTSERVER:1024/keypair/$KEY_NAME?connection_name=azure-config01 |json_pp
