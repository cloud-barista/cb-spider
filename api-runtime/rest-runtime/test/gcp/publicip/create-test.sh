RESTSERVER=localhost

#정상 동작
curl -X POST http://$RESTSERVER:1024/publicip?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{ "Name": "gcppublicip1" }' |json_pp
