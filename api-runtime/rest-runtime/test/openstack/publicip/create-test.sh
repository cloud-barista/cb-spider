RESTSERVER=127.0.0.1

curl -X POST http://$RESTSERVER:1024/publicip?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{ "Name": "public1" }'
