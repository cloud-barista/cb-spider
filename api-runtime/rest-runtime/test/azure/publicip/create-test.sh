RESTSERVER=node12

curl -X POST http://$RESTSERVER:1024/publicip?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "publicipt01" }'
