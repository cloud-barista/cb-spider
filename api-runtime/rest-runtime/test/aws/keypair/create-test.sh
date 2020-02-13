RESTSERVER=localhost

#정상 동작
curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbkeypair01" }' |json_pp