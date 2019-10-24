RESTSERVER=192.168.130.8

curl -X POST http://$RESTSERVER:1024/publicip?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{ "Name": "mcb-test-pubip" }'
