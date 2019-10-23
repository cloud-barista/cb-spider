RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/keypair?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-Keypair" }'
