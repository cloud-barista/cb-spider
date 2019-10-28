RESTSERVER=localhost

#정상 동작
curl -X POST http://$RESTSERVER:1024/keypair?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair" }' |json_pp