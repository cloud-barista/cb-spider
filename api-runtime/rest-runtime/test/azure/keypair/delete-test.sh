RESTSERVER=localhost

curl -X DELETE http://$RESTSERVER:1024/keypair/CB-KeyPair?connection_name=azure-config01 |json_pp
