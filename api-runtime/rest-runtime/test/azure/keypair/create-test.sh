RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/keypair?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-KeyPair" }' |json_pp
